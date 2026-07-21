package controller

type Action int

const (
	ActionNone Action = iota
	ActionBoostRT
	ActionRelaxRT
)

type Thresholds struct {
	HighWatermark float64
	HoldTicks     int
}

type decider struct {
	t          Thresholds
	tightTicks int
	freeTicks  int
}

func newDecider(t Thresholds) *decider {
	return &decider{t: t}
}

func (d *decider) decide(rtUtilization float64) Action {
	if rtUtilization >= d.t.HighWatermark {

		d.tightTicks++
		d.freeTicks = 0
		if d.tightTicks >= d.t.HoldTicks {
			d.tightTicks = 0
			return ActionBoostRT
		}
		return ActionNone
	}

	d.freeTicks++
	d.tightTicks = 0
	if d.freeTicks >= d.t.HoldTicks {
		d.freeTicks = 0
		return ActionRelaxRT
	}
	return ActionNone
}
