package routeros

import (
	"context"
	"fmt"
	"net/netip"
)

type ConnectionSummary struct {
	Addr     netip.Addr
	Total    int
	TCPCount int
	UDPCount int
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
