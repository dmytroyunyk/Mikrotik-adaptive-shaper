package routeros

import (
	"context"
	"fmt"

	"github.com/dmytroyunyk/adaptive-shaper/pkg/models"
)

const mangleComment = "adaptive-shaper"

const (
	realtimeMaxPacketSize = 300

	bulkMinPacketSize = 1400
)

func (c *Client) EnsureMangle(ctx context.Context, iface string) error {
	if err := c.clearMangle(ctx); err != nil {
		return err
	}
	if err := c.markConnection(ctx, iface, models.ClassRealtime,
		"=protocol=udp",
		"=packet-size=0-"+itoa(realtimeMaxPacketSize),
	); err != nil {
		return err
	}
	if err := c.markConnection(ctx, iface, models.ClassBulk,
		"=packet-size="+itoa(bulkMinPacketSize)+"-65535",
	); err != nil {
		return err
	}

	if err := c.markConnection(ctx, iface, models.ClassInteractive); err != nil {
		return err
	}

	for _, class := range []models.TrafficClass{
		models.ClassRealtime, models.ClassBulk, models.ClassInteractive,
	} {
		if err := c.markPacket(ctx, class); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) markConnection(ctx context.Context, iface string, class models.TrafficClass, extra ...string) error {
	args := []string{
		"/ip/firewall/mangle/add",
		"=chain=prerouting",
		"=in-interface=" + iface,
		"=action=mark-connection",
		"=new-connection-mark=" + string(class) + "-conn",

		"=connection-state=new",

		"=passthrough=yes",

		"=comment=" + mangleComment,
	}
	args = append(args, extra...)

	if _, err := c.Run(ctx, args...); err != nil {
		return fmt.Errorf("mangle: mark-connection %s failed: %w", class, err)
	}
	return nil
}

func (c *Client) markPacket(ctx context.Context, class models.TrafficClass) error {
	_, err := c.Run(ctx,
		"/ip/firewall/mangle/add",
		"=chain=prerouting",
		"=connection-mark="+string(class)+"-conn",
		"=action=mark-packet",

		"=new-packet-mark="+string(class),

		"=passthrough=no",

		"=comment="+mangleComment,
	)
	if err != nil {
		return fmt.Errorf("mangle: mark-packet %s failed: %w", class, err)
	}
	return nil
}

func (c *Client) clearMangle(ctx context.Context) error {
	reply, err := c.Run(ctx, "/ip/firewall/mangle/print", "?comment="+mangleComment)
	if err != nil {
		return fmt.Errorf("mangle: print failed: %w", err)
	}

	for _, sentence := range reply.Re {
		id := sentence.Map[".id"]
		if id == "" {
			continue
		}
		if _, err := c.Run(ctx, "/ip/firewall/mangle/remove", "=.id="+id); err != nil {
			return fmt.Errorf("mangle: remove %s failed: %w", id, err)
		}
	}
	return nil
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
