package routeros

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

type ConnectionSummary struct {
	Addr     netip.Addr
	Total    int
	TCPCount int
	UDPCount int
	RateBps  uint64
}

func (c *Client) SourceConnections(ctx context.Context) (map[netip.Addr]ConnectionSummary, error) {
	reply, err := c.Run(ctx, "/ip/firewall/connection/print")
	if err != nil {
		return nil, fmt.Errorf("connections: print failed: %w", err)
	}

	sums := make(map[netip.Addr]ConnectionSummary)

	for _, sentence := range reply.Re {
		raw := sentence.Map["src-address"]
		if raw == "" {
			continue
		}

		addr, ok := parseSrcAddr(raw)
		if !ok {
			continue
		}

		s := sums[addr]
		s.Addr = addr
		s.Total++
		switch sentence.Map["protocol"] {
		case "tcp":
			s.TCPCount++
		case "udp":
			s.UDPCount++
		}
		s.RateBps += parseRate(sentence.Map["orig-rate"])
		sums[addr] = s
	}

	return sums, nil
}

func parseSrcAddr(raw string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(raw); err == nil {
		return ap.Addr(), true
	}
	if a, err := netip.ParseAddr(raw); err == nil {
		return a, true
	}
	return netip.Addr{}, false
}

func parseRate(raw string) uint64 {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" || raw == "0bps" {
		return 0
	}
	raw = strings.TrimSuffix(raw, "bps")

	mult := uint64(1)
	switch {
	case strings.HasSuffix(raw, "g"):
		mult, raw = 1_000_000_000, strings.TrimSuffix(raw, "g")
	case strings.HasSuffix(raw, "m"):
		mult, raw = 1_000_000, strings.TrimSuffix(raw, "m")
	case strings.HasSuffix(raw, "k"):
		mult, raw = 1_000, strings.TrimSuffix(raw, "k")
	}

	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return uint64(val * float64(mult))
}
