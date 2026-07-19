package models

import (
	"net/netip"
	"time"
)

type TrafficClass string

const (
	ClassRealtime    TrafficClass = "realtime"
	ClassInteractive TrafficClass = "interactive"
	ClassBulk        TrafficClass = "bulk"
	ClassUnknown     TrafficClass = "unknown"
)

type SourceStat struct {
	Addr       netip.Addr
	Connections int
	TCPCount   int
	UDPCount   int
	Class      TrafficClass
}

type QueueRate struct {
	Name        string
	Class       TrafficClass
	BytesPerSec uint64
}

type Snapshot struct {
	Timestamp time.Time
	Sources   []SourceStat
	Queues    []QueueRate
	TotalMbit float64
}
