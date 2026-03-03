package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// Kalman is a placeholder for the Kalman-filter estimator.
// It satisfies the Estimator interface but always returns ok=false until
// the full implementation is added.
type Kalman struct{}

func NewKalman() *Kalman { return &Kalman{} }

func (k *Kalman) Update(_ float64, _ time.Time) {}

func (k *Kalman) ETA() (time.Time, bool) { return time.Time{}, false }

func (k *Kalman) State() state.EstimatorState {
	return state.EstimatorState{Type: TypeKalman}
}
