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

func TestKalman_ReturnsNotOK(t *testing.T) {
	k := estimator.NewKalman()
	k.Update(0.5, time.Now())
	_, ok := k.ETA()
	assert.False(t, ok, "Kalman stub must return ok=false until fully implemented")
}
