package models

import "time"

type TrafficClass string

const (
	ClassRealtime    TrafficClass = "realtime"
	ClassInteractive TrafficClass = "interactive"
	ClassBulk        TrafficClass = "bulk"
	ClassUnkown      TrafficClass = "unkown"
)

type Flow struct {
	SrcIP    string
	DstIP    string
	SrcPort  uint16
	DstPort  uint16
	Protocol string

	Class TrafficClass

	BytesPerSec  uint64
	PacketPerSec uint64

	lastSeen time.Time
}

type SnapShot struct {
	Timestamp time.Time
	Flows     []Flow

	TotalMbit float64
}
