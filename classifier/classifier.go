package classifier

import "github.com/dmytroyunyk/adaptive-shaper/models"

type Classifier struct {
	thresholds Thresholds
}

func New(t Thresholds) *Classifier {
	return &Classifier{thresholds: t}
}

func (c *Classifier) Classify(snap models.Snapshot) models.Snapshot {
	for i := range snap.Sources {
		snap.Sources[i].Class = classify(snap.Sources[i], c.thresholds)
	}
	return snap
}
