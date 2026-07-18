package routeros

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dmytroyunyk/adaptive-shaper/models"
)

const (
	QueueRoot        = "total-upload"
	QueueRealtime    = "rt-upload"
	QueueInteractive = "inter-upload"
	QueueBulk        = "bulk-upload"
)

const (
	PriorityRoot        = 8
	PriorityRealtime    = 1
	PriorityInteractive = 4
	PriorityBulk        = 8
)

func mbitToBps(mbit int) string {
	return strconv.FormatUint(uint64(mbit)*1_000_000, 10)
}

type Queue struct {
	ID         string
	Name       string
	Parent     string
	PacketMark string
	LimitAt    uint64
	MaxLimit   uint64
	Priority   int
}

type QueueStat struct {
	Name    string
	Bytes   uint64
	Rate    uint64
	Dropped uint64
}

func (c *Client) ListQueues(ctx context.Context) ([]Queue, error) {
	reply, err := c.Run(ctx, "/queue/tree/print")
	if err != nil {
		return nil, fmt.Errorf("queues: print failed: %w", err)
	}

	queues := make([]Queue, 0, len(reply.Re))
	for _, sentence := range reply.Re {
		m := sentence.Map

		limitAt, _ := strconv.ParseUint(m["limit-at"], 10, 64)
		maxLimit, _ := strconv.ParseUint(m["max-limit"], 10, 64)
		priority, _ := strconv.Atoi(m["priority"])

		queues = append(queues, Queue{
			ID:         m[".id"],
			Name:       m["name"],
			Parent:     m["parent"],
			PacketMark: m["packet-mark"],
			LimitAt:    limitAt,
			MaxLimit:   maxLimit,
			Priority:   priority,
		})
	}
	return queues, nil
}

func (c *Client) QueueStats(ctx context.Context) (map[string]QueueStat, error) {
	reply, err := c.Run(ctx, "/queue/tree/print", "=stats=")
	if err != nil {
		return nil, fmt.Errorf("queues: stats failed: %w", err)
	}

	stats := make(map[string]QueueStat, len(reply.Re))
	for _, sentence := range reply.Re {
		m := sentence.Map
		name := m["name"]

		bytes, _ := strconv.ParseUint(m["bytes"], 10, 64)
		rate, _ := strconv.ParseUint(m["rate"], 10, 64)
		dropped, _ := strconv.ParseUint(m["dropped"], 10, 64)

		stats[name] = QueueStat{
			Name:    name,
			Bytes:   bytes,
			Rate:    rate,
			Dropped: dropped,
		}
	}
	return stats, nil
}

func (c *Client) findID(ctx context.Context, name string) (string, error) {
	reply, err := c.Run(ctx, "/queue/tree/print", "?name="+name)
	if err != nil {
		return "", fmt.Errorf("queues: lookup %q failed: %w", name, err)
	}
	if len(reply.Re) == 0 {
		return "", fmt.Errorf("queues: %q not found", name)
	}
	return reply.Re[0].Map[".id"], nil
}

func (c *Client) EnsureTree(ctx context.Context, iface string, uplinkMbit, rtMbit, bulkMbit int) error {
	interMbit := uplinkMbit - rtMbit - bulkMbit
	if interMbit <= 0 {
		return fmt.Errorf(
			"queues: bad config — realtime(%d) + bulk(%d) >= uplink(%d)",
			rtMbit, bulkMbit, uplinkMbit,
		)
	}

	existing, err := c.ListQueues(ctx)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(existing))
	for _, q := range existing {
		present[q.Name] = true
	}

	if !present[QueueRoot] {
		if err := c.addQueue(ctx,
			QueueRoot, iface, "",
			mbitToBps(uplinkMbit), mbitToBps(uplinkMbit), PriorityRoot,
		); err != nil {
			return err
		}
	}

	children := []struct {
		name     string
		mark     string
		limitAt  int
		priority int
	}{
		{QueueRealtime, string(models.ClassRealtime), rtMbit, PriorityRealtime},
		{QueueInteractive, string(models.ClassInteractive), interMbit, PriorityInteractive},
		{QueueBulk, string(models.ClassBulk), bulkMbit, PriorityBulk},
	}

	for _, ch := range children {
		if present[ch.name] {
			continue
		}
		if err := c.addQueue(ctx,
			ch.name, QueueRoot, ch.mark,
			mbitToBps(ch.limitAt), mbitToBps(uplinkMbit), ch.priority,
		); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) addQueue(ctx context.Context, name, parent, mark, limitAt, maxLimit string, priority int) error {
	args := []string{
		"/queue/tree/add",
		"=name=" + name,
		"=parent=" + parent,
		"=limit-at=" + limitAt,
		"=max-limit=" + maxLimit,
		"=priority=" + strconv.Itoa(priority),
		"=queue=pcq-upload-default",
	}
	if mark != "" {
		args = append(args, "=packet-mark="+mark)
	}

	if _, err := c.Run(ctx, args...); err != nil {
		return fmt.Errorf("queues: add %q failed: %w", name, err)
	}
	return nil
}

func (c *Client) SetLimits(ctx context.Context, name string, limitAtMbit, maxLimitMbit int) error {
	id, err := c.findID(ctx, name)
	if err != nil {
		return err
	}

	_, err = c.Run(ctx,
		"/queue/tree/set",
		"=.id="+id,
		"=limit-at="+mbitToBps(limitAtMbit),
		"=max-limit="+mbitToBps(maxLimitMbit),
	)
	if err != nil {
		return fmt.Errorf("queues: set %q failed: %w", name, err)
	}
	return nil
}
