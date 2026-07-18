package metrics

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/dmytroyunyk/adaptive-shaper/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	flowBytesPerSec *prometheus.GaugeVec
	totalMbit       prometheus.Gauge
	reg             *prometheus.Registry
}

func New() *Metrics {
	m := &Metrics{
		reg: prometheus.NewRegistry(),
		flowBytesPerSec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "shaper_flow_bytes_per_second",
				Help: "Current throughput per traffic class in bytes per second.",
			},
			[]string{"class"},
		),
		totalMbit: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "shaper_total_mbit",
				Help: "Agregate uplink throughput across all classes in Mbit/s.",
			},
		),
	}

	m.reg.MustRegister(m.flowBytesPerSec, m.totalMbit)
	return m
}

func (m *Metrics) Update(snap models.Snapshot) {
	for _, f := range snap.Flows {
		m.flowBytesPerSec.
			WithLabelValues(string(f.Class)).
			Set(float64(f.BytesPerSec))
	}
	m.totalMbit.Set(snap.TotalMbit)
}

func (m *Metrics) Consume(ctx context.Context, in <-chan models.Snapshot) {
	for {
		select {
		case <-ctx.Done():
			return
		case snap := <-in:
			m.Update(snap)
		}
	}
}

func (m *Metrics) Serve(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{}))

	srv := &http.Server{Addr: addr, Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("metrics: shutdown error: %v", err)
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}
