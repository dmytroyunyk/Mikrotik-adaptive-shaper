package collector

import (
	"context"
	"log"
	"time"

	"github.com/dmytroyunyk/adaptive-shaper/models"
	"github.com/dmytroyunyk/adaptive-shaper/routeros"
)

type Collector struct {
	client   *routeros.Client
	interval time.Duration

	prev map[string]uint64
}

func New(client *routeros.Client, interval time.Duration) *Collector {
	return &Collector{
		client:   client,
		interval: interval,
		prev:     make(map[string]uint64),
	}
}

func (c *Collector) Run(ctx context.Context, out chan<- models.Snapshot) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			snap, err := c.poll(ctx)
			if err != nil {
				log.Printf("collector: poll failed: %v", err)
				continue
			}
			select {
			case out <- snap:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

func (c *Collector) poll(ctx context.Context) (models.Snapshot, error) {
	stats, err := c.client.QueueStats(ctx)
	if err != nil {
		return models.Snapshot{}, err
	}

	seconds := c.interval.Seconds()
	queues := make([]models.QueueRate, 0, len(stats))
	var totalBps uint64

	for name, st := range stats {
		prevBytes, seen := c.prev[name]

		var bps uint64
		if seen && st.Bytes >= prevBytes {
			bps = uint64(float64(st.Bytes-prevBytes) / seconds)
		}

		c.prev[name] = st.Bytes
		totalBps += bps

		queues = append(queues, models.QueueRate{
			Name:        name,
			Class:       classForQueue(name),
			BytesPerSec: bps,
		})
	}
	sums, err := c.client.SourceConnections(ctx)
	if err != nil {
		return models.Snapshot{}, err
	}

	sources := make([]models.SourceStat, 0, len(sums))
	for _, s := range sums {
		sources = append(sources, models.SourceStat{
			Addr:        s.Addr,
			Connections: s.Total,
			TCPCount:    s.TCPCount,
			UDPCount:    s.UDPCount,
			BytesPerSec: s.RateBps,
			Class:       models.ClassUnknown,
		})
	}

	return models.Snapshot{
		Timestamp: time.Now(),
		Sources:   sources,
		Queues:    queues,
		TotalMbit: float64(totalBps*8) / 1_000_000,
	}, nil
}

func classForQueue(name string) models.TrafficClass {
	switch name {
	case routeros.QueueRealtime:
		return models.ClassRealtime
	case routeros.QueueInteractive:
		return models.ClassInteractive
	case routeros.QueueBulk:
		return models.ClassBulk
	default:
		return models.ClassUnknown
	}
}
