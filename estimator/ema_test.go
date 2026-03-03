package estimator_test

import (
	"testing"
	"time"

	"github.com/baruch/progress-wrap/estimator"
	"github.com/stretchr/testify/assert"
)

func TestEMA_NotEnoughSamples(t *testing.T) {
	e := estimator.NewEMA(0.2)
	e.Update(0.10, time.Now())
	_, ok := e.ETA()
	assert.False(t, ok, "need >= 2 samples")
}

func TestEMA_ETAAfterTwoSamples(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.00, t0)
	e.Update(0.10, t0.Add(10*time.Second)) // 1%/s velocity
	eta, ok := e.ETA()
	assert.True(t, ok)
	// remaining = 0.90, velocity = 0.01/s → ETA ≈ t0 + 100s
	assert.WithinDuration(t, t0.Add(100*time.Second), eta, 5*time.Second)
}

func TestEMA_VelocitySmoothing(t *testing.T) {
	e := estimator.NewEMA(0.5)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.1, t0.Add(10*time.Second)) // instant v = 0.01/s
	e.Update(0.3, t0.Add(20*time.Second)) // instant v = 0.02/s
	_, ok := e.ETA()
	assert.True(t, ok)
}

func TestEMA_NegativeVelocity(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.5, t0)
	e.Update(0.3, t0.Add(10*time.Second)) // going backward
	_, ok := e.ETA()
	assert.False(t, ok, "negative velocity should not produce ETA")
}

func TestEMA_StateRoundtrip(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.1, t0.Add(10*time.Second))
	s := e.State()
	assert.Equal(t, "ema", s.Type)
	assert.Greater(t, s.EMAVelocity, 0.0)
}

func TestEMA_RestoreFromState(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.2, t0.Add(10*time.Second))

	s := e.State()
	e2 := estimator.NewEMAFromState(s, 0.2, 0.2, t0.Add(10*time.Second))
	eta, ok := e2.ETA()
	assert.True(t, ok)
	assert.False(t, eta.IsZero())
}
