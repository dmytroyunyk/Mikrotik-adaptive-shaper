package controller

import (
	"context"
	"log"

	"github.com/dmytroyunyk/adaptive-shaper/models"
	"github.com/dmytroyunyk/adaptive-shaper/routeros"
)

type Controller struct {
	client  *routeros.Client
	decider *decider

	rtMbit     int
	bulkMbit   int
	uplinkMbit int
	stepMbit   int
}

func New(client *routeros.Client, t Thresholds, stepMbit, uplinkMbit, rtMbit, bulkMbit int) *Controller {
	return &Controller{
		client:     client,
		decider:    newDecider(t),
		rtMbit:     rtMbit,
		bulkMbit:   bulkMbit,
		uplinkMbit: uplinkMbit,
		stepMbit:   stepMbit,
	}
}

func (c *Controller) Run(ctx context.Context, in <-chan models.Snapshot) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case snap := <-in:
			c.step(ctx, snap)
		}
	}
}

func (c *Controller) step(ctx context.Context, snap models.Snapshot) {
	rtBps := queueRate(snap, routeros.QueueRealtime)
	rtCapBps := float64(c.rtMbit) * 1_000_000 / 8
	if rtCapBps == 0 {
		return
	}
	util := float64(rtBps) / rtCapBps

	switch c.decider.decide(util) {
	case ActionBoostRT:
		c.shift(ctx, +c.stepMbit)
	case ActionRelaxRT:
		c.shift(ctx, -c.stepMbit)
	case ActionNone:
	}
}

func (c *Controller) shift(ctx context.Context, delta int) {
	newRT := c.rtMbit + delta
	newBulk := c.bulkMbit - delta

	const minMbit = 50
	if newRT < minMbit || newBulk < minMbit {
		return
	}
	if newRT+newBulk >= c.uplinkMbit {
		return
	}

	if err := c.client.SetLimits(ctx, routeros.QueueRealtime, newRT, c.uplinkMbit); err != nil {
		log.Printf("controller: set realtime failed: %v", err)
		return
	}
	if err := c.client.SetLimits(ctx, routeros.QueueBulk, newBulk, c.uplinkMbit); err != nil {
		log.Printf("controller: set bulk failed: %v", err)
		return
	}

	c.rtMbit = newRT
	c.bulkMbit = newBulk
	log.Printf("controller: shifted realtime=%dMbit bulk=%dMbit", newRT, newBulk)
}

func queueRate(snap models.Snapshot, name string) uint64 {
	for _, q := range snap.Queues {
		if q.Name == name {
			return q.BytesPerSec
		}
	}
	return 0
}
