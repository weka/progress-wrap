package estimator

import (
	"math"
	"time"

	"github.com/baruch/progress-wrap/state"
)

// IMMEstimator adapts IMM to the Estimator interface.
//
// The IMM works in percent (0–100) with float64 Unix-second timestamps;
// this wrapper converts to/from the 0–1 progress and time.Time used by
// the rest of the codebase.
type IMMEstimator struct {
	imm     *IMM
	lastETA time.Time
	etaOK   bool
	vel     float64   // fused velocity in 0-1/s
	accel   float64   // fused acceleration in 0-1/s²
}

// NewIMMEstimator creates a fresh IMMEstimator.
func NewIMMEstimator() *IMMEstimator {
	return &IMMEstimator{imm: NewIMM()}
}

// NewIMMEstimatorFromState restores an IMMEstimator from a persisted snapshot.
func NewIMMEstimatorFromState(snap *state.IMMSnapshot) *IMMEstimator {
	e := &IMMEstimator{imm: NewIMM()}
	e.imm.restoreFromSnapshot(snap)
	return e
}

// Update implements Estimator.
func (e *IMMEstimator) Update(progress float64, t time.Time) {
	ts := float64(t.UnixNano()) / 1e9
	etaSec, vel, _ := e.imm.Update(ts, progress*100)

	e.vel = vel / 100

	// Fused acceleration from the probability-weighted model states (x[2] is in %/s²).
	var accelPct float64
	for j := range 2 {
		accelPct += e.imm.mu[j] * e.imm.models[j].x[2]
	}
	e.accel = accelPct / 100

	if math.IsInf(etaSec, 1) || etaSec < 0 {
		e.etaOK = false
	} else {
		e.lastETA = t.Add(time.Duration(etaSec * float64(time.Second)))
		e.etaOK = true
	}
}

// ETA implements Estimator.
func (e *IMMEstimator) ETA() (time.Time, bool) {
	return e.lastETA, e.etaOK
}

// State implements Estimator.
func (e *IMMEstimator) State() state.EstimatorState {
	st := state.EstimatorState{
		Type:         TypeIMM,
		Velocity:     e.vel,
		Acceleration: e.accel,
	}
	if e.imm.count >= 1 {
		st.IMMSnapshot = e.imm.Snapshot()
	}
	return st
}
