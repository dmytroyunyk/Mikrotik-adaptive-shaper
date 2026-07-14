package routeros

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-routeros/routeros/v3"
)

const reconnectDelay = 5 * time.Second

type Client struct {
	host     string
	username string
	password string

	mu   sync.Mutex
	conn *routeros.Client
}

func New(host, username, password string, port int) *Client {
	return &Client{
		host:     fmt.Sprintf("%s:%d", host, port),
		username: username,
		password: password,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked(ctx)
}

func (c *Client) connectLocked(ctx context.Context) error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		log.Printf("routeros: connecting to %s...", c.host)

		conn, err := routeros.Dial(c.host, c.username, c.password)
		if err == nil {
			c.conn = conn
			log.Printf("routeros: connected to %s", c.host)
			return nil
		}

		var devErr *routeros.DeviceError
		if errors.As(err, &devErr) {
			return fmt.Errorf("routeros: login rejected, check credentials: %w", err)
		}

		log.Printf("routeros: dial failed: %v — retry in %s", err, reconnectDelay)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *Client) Run(ctx context.Context, sentence ...string) (*routeros.Reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		if err := c.connectLocked(ctx); err != nil {
			return nil, err
		}
	}

	reply, err := c.conn.RunArgs(sentence)
	if err == nil {
		return reply, nil
	}

	var devErr *routeros.DeviceError
	if errors.As(err, &devErr) {
		return nil, fmt.Errorf("routeros: command rejected: %w", err)
	}

	log.Printf("routeros: transport error: %v — reconnecting", err)
	if err := c.connectLocked(ctx); err != nil {
		return nil, err
	}
	return c.conn.RunArgs(sentence)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}
