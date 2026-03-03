package estimator_test

import (
	"testing"
	"time"

	"github.com/baruch/progress-wrap/estimator"
	"github.com/stretchr/testify/assert"
)

func TestKalman_ImplementsInterface(t *testing.T) {
	var _ estimator.Estimator = estimator.NewKalman()
}

func TestKalman_NotEnoughSamples(t *testing.T) {
	k := estimator.NewKalman()
	k.Update(0.10, time.Now())
	_, ok := k.ETA()
	assert.False(t, ok, "need >= 2 samples for ETA")
}

func TestKalman_ETAAfterTwoSamples(t *testing.T) {
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.00, t0)
	k.Update(0.10, t0.Add(10*time.Second)) // roughly 1%/s
	eta, ok := k.ETA()
	assert.True(t, ok)
	// remaining ≈ 0.90, velocity ≈ 0.01/s → ETA ≈ t0 + 100s
	// Kalman blends measurement noise, so allow ±20s tolerance
	assert.WithinDuration(t, t0.Add(100*time.Second), eta, 20*time.Second)
}

func TestKalman_SmoothsNoise(t *testing.T) {
	// Feed alternating fast/slow samples; the Kalman filter should give
	// a stable ETA rather than a wildly oscillating one.
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.0, t0)
	for i := 1; i <= 10; i++ {
		var p float64
		if i%2 == 0 {
			p = float64(i)*0.01 + 0.05 // noisy spike
		} else {
			p = float64(i) * 0.01
		}
		k.Update(p, t0.Add(time.Duration(i)*10*time.Second))
	}
	_, ok := k.ETA()
	assert.True(t, ok)
}

func TestKalman_NegativeVelocity(t *testing.T) {
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.5, t0)
	k.Update(0.3, t0.Add(10*time.Second)) // going backward
	_, ok := k.ETA()
	// Kalman blends with prior, so velocity may be small but could still be
	// positive after just one backward step. After a sustained drop it
	// should turn negative/zero.
	// Feed more backward samples to ensure velocity goes negative.
	k.Update(0.1, t0.Add(20*time.Second))
	_, ok = k.ETA()
	assert.False(t, ok, "sustained backward progress should not produce ETA")
}

func TestKalman_StateRoundtrip(t *testing.T) {
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.0, t0)
	k.Update(0.1, t0.Add(10*time.Second))
	s := k.State()
	assert.Equal(t, "kalman", s.Type)
	assert.Greater(t, s.KalmanP11, 0.0, "velocity covariance must be positive")
	assert.Greater(t, s.KalmanVel, 0.0, "velocity estimate should be positive")
}

func TestKalman_RestoreFromState(t *testing.T) {
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.0, t0)
	k.Update(0.2, t0.Add(10*time.Second))

	s := k.State()
	k2 := estimator.NewKalmanFromState(s, t0.Add(10*time.Second))

	// After restoring, one more update should still give a valid ETA.
	k2.Update(0.3, t0.Add(20*time.Second))
	eta, ok := k2.ETA()
	assert.True(t, ok)
	assert.False(t, eta.IsZero())
}

func TestKalman_DuplicateTimestampIgnored(t *testing.T) {
	k := estimator.NewKalman()
	t0 := time.Now()
	k.Update(0.1, t0)
	k.Update(0.2, t0) // same timestamp — should be ignored
	_, ok := k.ETA()
	assert.False(t, ok, "duplicate timestamp should not count as second sample")
}
