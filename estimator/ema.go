package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// EMA is an exponential-moving-average velocity estimator.
type EMA struct {
	alpha    float64
	velocity float64
	lastTime time.Time
	lastProg float64
	count    int
}

// NewEMA creates an EMA estimator with smoothing factor alpha (0 < alpha <= 1).
func NewEMA(alpha float64) *EMA {
	return &EMA{alpha: alpha}
}

// NewEMAFromState restores an EMA estimator from persisted state.
func NewEMAFromState(s state.EstimatorState, alpha, lastProg float64, lastTime time.Time) *EMA {
	return &EMA{
		alpha:    alpha,
		velocity: s.EMAVelocity,
		lastTime: lastTime,
		lastProg: lastProg,
		count:    2, // enough to emit ETA
	}
}

// Update records a new progress observation.
func (e *EMA) Update(progress float64, t time.Time) {
	if e.count == 0 {
		e.lastTime = t
		e.lastProg = progress
		e.count++
		return
	}
	dt := t.Sub(e.lastTime).Seconds()
	if dt <= 0 {
		return
	}
	instant := (progress - e.lastProg) / dt
	if e.count == 1 {
		e.velocity = instant
	} else {
		e.velocity = e.alpha*instant + (1-e.alpha)*e.velocity
	}
	e.lastTime = t
	e.lastProg = progress
	e.count++
}

// ETA returns the estimated completion time. ok is false if fewer than
// 2 samples have been seen or velocity is non-positive.
func (e *EMA) ETA() (time.Time, bool) {
	if e.count < 2 || e.velocity <= 0 {
		return time.Time{}, false
	}
	remaining := 1.0 - e.lastProg
	secs := remaining / e.velocity
	return e.lastTime.Add(time.Duration(secs * float64(time.Second))), true
}

// State returns the serializable estimator state.
func (e *EMA) State() state.EstimatorState {
	return state.EstimatorState{
		Type:        TypeEMA,
		EMAVelocity: e.velocity,
	}
}
