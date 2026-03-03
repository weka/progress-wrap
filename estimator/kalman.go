package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// kalmanQ is the process noise intensity (controls how quickly velocity can change).
const kalmanQ = 1e-4

// kalmanR is the measurement noise variance (how much we trust each position reading).
const kalmanR = 1e-4

// Kalman is a 2D constant-velocity Kalman filter estimator.
// State vector: x = [position, velocity].
// Only position is observed; velocity is inferred from successive measurements.
type Kalman struct {
	pos      float64   // current position estimate
	vel      float64   // current velocity estimate (progress-units/second)
	p00      float64   // covariance (position, position)
	p01      float64   // covariance (position, velocity) = (velocity, position)
	p11      float64   // covariance (velocity, velocity)
	lastTime time.Time // time of last update
	count    int       // number of updates seen
}

// NewKalman creates a new Kalman estimator.
func NewKalman() *Kalman {
	return &Kalman{}
}

// NewKalmanFromState restores a Kalman estimator from persisted state.
func NewKalmanFromState(s state.EstimatorState, lastTime time.Time) *Kalman {
	return &Kalman{
		pos:      s.KalmanPos,
		vel:      s.KalmanVel,
		p00:      s.KalmanP00,
		p01:      s.KalmanP01,
		p11:      s.KalmanP11,
		lastTime: lastTime,
		count:    2, // persisted state is always at least 2 updates old
	}
}

// Update records a new progress observation and runs a Kalman predict+update step.
func (k *Kalman) Update(progress float64, t time.Time) {
	if k.count == 0 {
		// Initialise state with first measurement.
		k.pos = progress
		k.vel = 0
		k.p00 = kalmanR // position uncertainty = measurement noise
		k.p01 = 0
		k.p11 = 1.0 // large initial velocity uncertainty
		k.lastTime = t
		k.count++
		return
	}

	dt := t.Sub(k.lastTime).Seconds()
	if dt <= 0 {
		return
	}

	// --- Predict ---
	// x_pred = F * x,  F = [[1, dt], [0, 1]]
	pPred := k.pos + k.vel*dt
	vPred := k.vel

	// P_pred = F * P * F^T + Q,
	// Q = kalmanQ * [[dt⁴/4, dt³/2], [dt³/2, dt²]]
	dt2 := dt * dt
	dt3 := dt2 * dt
	dt4 := dt3 * dt
	pp00 := k.p00 + 2*k.p01*dt + k.p11*dt2 + kalmanQ*dt4/4
	pp01 := k.p01 + k.p11*dt + kalmanQ*dt3/2
	pp11 := k.p11 + kalmanQ*dt2

	// --- Update ---
	// S = H * P_pred * H^T + R = pp00 + R  (H = [1, 0])
	innVar := pp00 + kalmanR
	// K = P_pred * H^T / S = [pp00, pp01]^T / S
	k0 := pp00 / innVar
	k1 := pp01 / innVar
	// Innovation: y = z - H * x_pred
	y := progress - pPred

	// Updated state: x = x_pred + K * y
	k.pos = pPred + k0*y
	k.vel = vPred + k1*y

	// Updated covariance: P = (I - K*H) * P_pred
	k.p00 = (1 - k0) * pp00
	k.p01 = (1 - k0) * pp01
	k.p11 = -k1*pp01 + pp11

	k.lastTime = t
	k.count++
}

// ETA returns the estimated completion time.
// ok is false if fewer than 2 samples have been seen or velocity is non-positive.
func (k *Kalman) ETA() (time.Time, bool) {
	if k.count < 2 || k.vel <= 0 {
		return time.Time{}, false
	}
	remaining := 1.0 - k.pos
	secs := remaining / k.vel
	return k.lastTime.Add(time.Duration(secs * float64(time.Second))), true
}

// State returns the serializable estimator state.
func (k *Kalman) State() state.EstimatorState {
	return state.EstimatorState{
		Type:      TypeKalman,
		Velocity:  k.vel,
		KalmanPos: k.pos,
		KalmanVel: k.vel,
		KalmanP00: k.p00,
		KalmanP01: k.p01,
		KalmanP11: k.p11,
	}
}
