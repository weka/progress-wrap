package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// Estimator tracks progress samples and produces ETA predictions.
type Estimator interface {
	Update(progress float64, t time.Time)
	ETA() (eta time.Time, ok bool)
	State() state.EstimatorState
}
