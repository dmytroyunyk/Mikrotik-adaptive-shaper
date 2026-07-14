package models

import "time"

type TrafficClass string

const (
	ClassRealtime    TrafficClass = "realtime"
	ClassInteractive TrafficClass = "interactive"
	ClassBulk        TrafficClass = "bulk"
	ClassUnknown     TrafficClass = "unknown"
)

type Flow struct {
	SrcIP    string
	DstIP    string
	SrcPort  uint16
	DstPort  uint16
	Protocol string

	Class TrafficClass

	BytesPerSec   uint64
	PacketsPerSec uint64

	LastSeen time.Time
}

type Snapshot struct {
	Timestamp time.Time
	Flows     []Flow

	TotalMbit float64
}
