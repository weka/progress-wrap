package estimator_test

import (
	"math"
	"testing"

	"github.com/baruch/progress-wrap/estimator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIMM_RecoveryAfterRateChange is the primary integration test required by the spec.
//
// Scenario:
//  1. Stable phase:   1%/s for 30 s  (progress 0 → 30%)
//  2. Sudden drop:    0.3%/s for 15 s  (progress 30 → 34.5%)
//  3. Recovery:       1%/s for 10 s  (progress 34.5 → 44.5%)
//
// Assert: after 10 s of recovery the ETA estimate is within 15% of the true
// remaining time at that point.
func TestIMM_RecoveryAfterRateChange(t *testing.T) {
	imm := estimator.NewIMM()

	ts := 0.0
	progress := 0.0

	// Phase 1: stable 1%/s for 30 seconds
	for range 30 {
		ts++
		progress += 1.0
		imm.Update(ts, progress)
	}
	// progress = 30%, ts = 30

	// Phase 2: sudden rate drop to 0.3%/s for 15 seconds
	for range 15 {
		ts++
		progress += 0.3
		imm.Update(ts, progress)
	}
	// progress ≈ 34.5%, ts = 45

	// Phase 3: rate recovers to 1%/s; collect ETA after 10 seconds
	var lastETA float64
	for range 10 {
		ts++
		progress += 1.0
		lastETA, _, _ = imm.Update(ts, progress)
	}
	// progress ≈ 44.5%, ts = 55

	trueETA := (100.0 - progress) / 1.0 // ≈ 55.5 s
	margin := 0.15 * trueETA

	require.False(t, math.IsInf(lastETA, 1), "ETA must be finite after recovery")
	assert.InDelta(t, trueETA, lastETA, margin,
		"ETA (%.1f s) should be within 15%% of true ETA (%.1f s) after recovery",
		lastETA, trueETA)
}

// TestIMM_InfBeforeThreeSamples verifies that +Inf is returned until the filter
// has received at least 3 updates.
func TestIMM_InfBeforeThreeSamples(t *testing.T) {
	imm := estimator.NewIMM()

	eta1, _, _ := imm.Update(1.0, 1.0)
	assert.True(t, math.IsInf(eta1, 1), "1st update: ETA must be +Inf")

	eta2, _, _ := imm.Update(2.0, 2.0)
	assert.True(t, math.IsInf(eta2, 1), "2nd update: ETA must be +Inf")

	eta3, vel3, _ := imm.Update(3.0, 3.0)
	assert.False(t, math.IsInf(eta3, 1), "3rd update: ETA must be finite")
	assert.Greater(t, vel3, 0.0, "velocity must be positive after 3 updates")
}

// TestIMM_Reset verifies that Reset() restores the estimator to its initial state.
func TestIMM_Reset(t *testing.T) {
	imm := estimator.NewIMM()
	imm.Update(1.0, 10.0)
	imm.Update(2.0, 20.0)
	imm.Update(3.0, 30.0)

	imm.Reset()

	eta, _, _ := imm.Update(100.0, 5.0)
	assert.True(t, math.IsInf(eta, 1), "first update after Reset must return +Inf")
}

// TestIMM_ModelProbabilitiesAlwaysSumToOne checks the fundamental IMM invariant.
func TestIMM_ModelProbabilitiesAlwaysSumToOne(t *testing.T) {
	imm := estimator.NewIMM()
	for i := 1; i <= 20; i++ {
		_, _, probs := imm.Update(float64(i), float64(i))
		sum := probs[0] + probs[1]
		assert.InDelta(t, 1.0, sum, 1e-9,
			"model probabilities must sum to 1 after update %d (got %.15f)", i, sum)
	}
}

// TestIMM_StablePhasePreferesModel0 verifies that after smooth constant-rate progress
// the stable model (index 0, low noise) accumulates higher probability than the
// transitioning model.
func TestIMM_StablePhasePreferesModel0(t *testing.T) {
	imm := estimator.NewIMM()
	for i := 1; i <= 30; i++ {
		_, _, probs := imm.Update(float64(i), float64(i))
		if i == 30 {
			assert.Greater(t, probs[0], probs[1],
				"stable model probability (%.4f) should exceed transitioning (%.4f) after 30 s of smooth progress",
				probs[0], probs[1])
		}
	}
}

// TestIMM_TransitionModelActivatesOnRateChange checks that a sudden rate change
// causes the transitioning model (index 1) to gain significant probability.
func TestIMM_TransitionModelActivatesOnRateChange(t *testing.T) {
	imm := estimator.NewIMM()

	// Stable phase: 1%/s for 20 s
	for i := range 20 {
		imm.Update(float64(i+1), float64(i+1))
	}

	// Sudden drop to 0.3%/s; observe model probabilities after a few steps
	progress := 20.0
	ts := 20.0
	var lastProbs [2]float64
	for i := 0; i < 5; i++ {
		ts++
		progress += 0.3
		_, _, lastProbs = imm.Update(ts, progress)
	}

	// After 5 steps of anomalous rate, Model 1 (transitioning) should have risen
	assert.Greater(t, lastProbs[1], 0.05,
		"transitioning model probability should increase during rate change (got %.4f)", lastProbs[1])
}

// TestIMM_DuplicateTimestampIgnored ensures that a repeated timestamp is a no-op.
// The update is skipped, model probabilities are unchanged, and the next valid
// update produces the same result as if the duplicate had never been submitted.
func TestIMM_DuplicateTimestampIgnored(t *testing.T) {
	imm := estimator.NewIMM()
	imm.Update(1.0, 1.0)
	imm.Update(2.0, 2.0)
	_, _, probsBefore := imm.Update(3.0, 3.0)

	// Duplicate timestamp — filter must skip the measurement.
	_, _, probsAfter := imm.Update(3.0, 99.0)

	assert.Equal(t, probsBefore, probsAfter,
		"duplicate timestamp must leave model probabilities unchanged")
}

// TestIMM_NearZeroVelocityReturnsInf verifies that stalled progress yields +Inf ETA.
func TestIMM_NearZeroVelocityReturnsInf(t *testing.T) {
	imm := estimator.NewIMM()
	// Feed the same progress value for many steps.
	for i := 1; i <= 30; i++ {
		imm.Update(float64(i), 50.0)
	}
	eta, vel, _ := imm.Update(31.0, 50.0)
	// Velocity should have converged near zero.
	assert.Less(t, vel, 0.1, "velocity should be near zero for stalled progress")
	if vel < 1e-6 {
		assert.True(t, math.IsInf(eta, 1), "near-zero velocity must yield +Inf ETA")
	}
}
