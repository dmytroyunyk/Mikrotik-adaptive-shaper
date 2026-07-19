package classifier

import "github.com/dmytroyunyk/adaptive-shaper/models"

type Thresholds struct {
	UDPRatioRealtime float64
	BulkMinConns     int
	BulkMinBps       uint64
}

func classify(s models.SourceStat, t Thresholds) models.TrafficClass {
	if s.Connections > 2 {
		udpRatio := float64(s.UDPCount) / float64(s.Connections)
		if udpRatio >= t.UDPRatioRealtime {
			return models.ClassRealtime
		}
	}

	if s.Connections >= t.BulkMinConns || s.BytesPerSec >= t.BulkMinBps {
		return models.ClassBulk
	}

	return models.ClassInteractive
}
