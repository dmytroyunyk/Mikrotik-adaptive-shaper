package routeros

import (
	"fmt"
	"log"
	"time"

	"github.com/go-routeros/routeros/v3"
)

const (
	reconnectDelay = 5 * time.Second
	maxRetries     = 0
)

type Client struct {
	host     string
	username string
	password string
	conn     *routeros.Client
}

func New(host, username, password string, port int) *Client {
	return &Client{
		host:     fmt.Sprintf("%s:%d", host, port),
		username: username,
		password: password,
	}
}

func (c *Client) Connect() {
	for {
		log.Printf("routeros: connecting to %s...", c.host)

		conn, err := routeros.Dial(c.host, c.username, c.password)
		if err != nil {
			log.Printf("routeros: connection failed: %v", err)
			log.Printf("routeros: retrying in %s...", reconnectDelay)
			time.Sleep(reconnectDelay)
			continue
		}

		c.conn = conn
		log.Printf("routeros: connected to %s", c.host)
		return
	}
}

func (c *Client) Run(sentence ...string) (*routeros.Reply, error) {
	reply, err := c.conn.RunArgs(sentence)
	if err != nil {
		log.Printf("routeros: connection lost: %v", err)
		c.Connect()

		return c.conn.RunArgs(sentence)
	}
	return reply, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
