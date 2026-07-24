package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/dmytroyunyk/adaptive-shaper/classifier"
	"github.com/dmytroyunyk/adaptive-shaper/collector"
	"github.com/dmytroyunyk/adaptive-shaper/config"
	"github.com/dmytroyunyk/adaptive-shaper/controller"
	"github.com/dmytroyunyk/adaptive-shaper/metrics"
	"github.com/dmytroyunyk/adaptive-shaper/models"
	"github.com/dmytroyunyk/adaptive-shaper/routeros"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	interval, err := cfg.Agent.Parsed()
	if err != nil {
		log.Fatalf("config: bad interval: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	client := routeros.New(
		cfg.RouterOS.Host, cfg.RouterOS.Username,
		cfg.RouterOS.Password, cfg.RouterOS.Port,
	)
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("routeros: connect: %v", err)
	}
	defer client.Close()

	if err := client.EnsureTree(ctx, cfg.Shaper.Interface,
		cfg.Shaper.UplinkMbit, cfg.Shaper.RealtimeMbit, cfg.Shaper.BulkMbit,
	); err != nil {
		log.Fatalf("routeros: ensure tree: %v", err)
	}
	if err := client.EnsureMangle(ctx, cfg.Shaper.Interface); err != nil {
		log.Fatalf("routeros: ensure mangle: %v", err)
	}

	coll := collector.New(client, interval)

	clsf := classifier.New(classifier.Thresholds{
		UDPRatioRealtime: cfg.Classifier.UDPRatioRealtime,
		BulkMinConns:     cfg.Classifier.BulkMinConns,
		BulkMinBps:       cfg.Classifier.BulkMinBps,
	})

	mtr := metrics.New()

	ctrl := controller.New(client,
		controller.Thresholds{
			HighWatermark: cfg.Controller.HighWatermark,
			HoldTicks:     cfg.Controller.HoldTicks,
		},
		cfg.Controller.StepMbit,
		cfg.Shaper.UplinkMbit, cfg.Shaper.RealtimeMbit, cfg.Shaper.BulkMbit,
	)

	raw := make(chan models.Snapshot)
	toMetrics := make(chan models.Snapshot)
	toController := make(chan models.Snapshot)

	go func() {
		if err := coll.Run(ctx, raw); err != nil && ctx.Err() == nil {
			log.Printf("collector stopped: %v", err)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case snap := <-raw:
				snap = clsf.Classify(snap)

				select {
				case toMetrics <- snap:
				case <-ctx.Done():
					return
				}
				select {
				case toController <- snap:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	go mtr.Consume(ctx, toMetrics)
	go func() {
		if err := mtr.Serve(ctx, ":9090"); err != nil && ctx.Err() == nil {
			log.Printf("metrics server stopped: %v", err)
		}
	}()

	go func() {
		if err := ctrl.Run(ctx, toController); err != nil && ctx.Err() == nil {
			log.Printf("controller stopped: %v", err)
		}
	}()

	log.Printf("adaptive-shaper running (interval=%s, metrics=:9090)", interval)

	<-ctx.Done()
	log.Printf("shutting down...")

	time.Sleep(500 * time.Millisecond)
}
