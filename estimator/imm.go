package estimator

import (
	"math"

	"github.com/baruch/progress-wrap/state"
)

// IMM noise and switching parameters.
const (
	immQ1      = 1e-5 // stable model: low process noise intensity
	immQ2      = 1e-2 // transitioning model: high process noise intensity
	immR       = 1e-4 // measurement noise variance (shared by both models)
	immMinProb = 1e-6 // probability floor to prevent filter degeneracy
)

// immPi is the Markov transition probability matrix.
// immPi[i][j] = P(model j at k+1 | model i at k).
//
//	Row 0 (stable):       0.98 remain stable, 0.02 switch to transitioning
//	Row 1 (transitioning): 0.40 return to stable, 0.60 remain transitioning
var immPi = [2][2]float64{
	{0.98, 0.02},
	{0.40, 0.60},
}

// immModel holds the Kalman filter state for one model in the IMM.
// State vector x = [position %, velocity %/s, acceleration %/s²].
type immModel struct {
	x [3]float64
	P [3][3]float64
	q float64 // process noise intensity
}

// IMM is an Interacting Multiple Models Kalman filter estimator.
//
// Two models run in parallel:
//   - Model 0 (stable): low process noise; tracks normal operation with smooth drift.
//   - Model 1 (transitioning): high process noise; recovers quickly from sudden changes.
//
// Progress is in percent (0–100); timestamps are in seconds (arbitrary epoch).
type IMM struct {
	models [2]immModel
	mu     [2]float64 // model probabilities (always sum to 1)
	prevTs float64
	count  int
}

// NewIMM creates a new IMM estimator.
func NewIMM() *IMM {
	imm := &IMM{}
	imm.reset()
	return imm
}

func (imm *IMM) reset() {
	imm.mu = [2]float64{0.9, 0.1}
	imm.models[0].q = immQ1
	imm.models[1].q = immQ2
	largeP := [3][3]float64{{100, 0, 0}, {0, 100, 0}, {0, 0, 100}}
	for i := range imm.models {
		imm.models[i].x = [3]float64{}
		imm.models[i].P = largeP
	}
	imm.prevTs = 0
	imm.count = 0
}

// Reset performs a full state reset (e.g. on process restart).
func (imm *IMM) Reset() { imm.reset() }

// Update processes a new measurement and returns:
//   - etaSec: estimated seconds until 100% (+Inf if fewer than 3 updates or velocity ≤ 0)
//   - velPctPerSec: fused velocity estimate in %/s
//   - probs: current model probabilities [stable, transitioning]
//
// ts is the timestamp in seconds; progressPct is in [0, 100].
func (imm *IMM) Update(ts, progressPct float64) (etaSec, velPctPerSec float64, probs [2]float64) {
	if imm.count == 0 {
		for i := range imm.models {
			imm.models[i].x[0] = progressPct
		}
		imm.prevTs = ts
		imm.count++
		return math.Inf(1), 0, imm.mu
	}

	dt := ts - imm.prevTs
	if dt <= 0 {
		return math.Inf(1), 0, imm.mu
	}
	imm.prevTs = ts

	F := immBuildF(dt)

	// ── Step 1: Interaction / Mixing ─────────────────────────────────────────
	// c[j] = Σ_i π[i][j] * μ[i]  (predicted model probability for model j)
	var c [2]float64
	for j := range 2 {
		for i := range 2 {
			c[j] += immPi[i][j] * imm.mu[i]
		}
	}

	// Mixed initial state and covariance for each model j.
	var mixed [2]immModel
	for j := range 2 {
		mixed[j].q = imm.models[j].q

		// Mixing weights: μ_{i|j} = π[i][j] * μ[i] / c[j]
		var muij [2]float64
		for i := range 2 {
			muij[i] = immPi[i][j] * imm.mu[i] / c[j]
		}

		// Mixed mean: x0j = Σ_i μ_{i|j} * x_i
		for i := range 2 {
			for k := range 3 {
				mixed[j].x[k] += muij[i] * imm.models[i].x[k]
			}
		}

		// Mixed covariance: P0j = Σ_i μ_{i|j} * [P_i + (x_i - x0j)(x_i - x0j)^T]
		for i := range 2 {
			diff := vec3Sub(imm.models[i].x, mixed[j].x)
			spread := vec3Outer(diff, diff)
			for r := range 3 {
				for col := range 3 {
					mixed[j].P[r][col] += muij[i] * (imm.models[i].P[r][col] + spread[r][col])
				}
			}
		}
	}

	// ── Step 2: Mode-conditioned predict + update ─────────────────────────────
	var L [2]float64
	for j := range 2 {
		Q := immBuildQ(dt, mixed[j].q)

		// Predict: x̂⁻ = F x̂₀,  P⁻ = F P₀ F^T + Q
		xPred := mat3MulVec(F, mixed[j].x)
		Ppred := mat3Add(mat3Mul(mat3Mul(F, mixed[j].P), mat3T(F)), Q)

		// Innovation (scalar, H = [1, 0, 0]): ν = z - H x̂⁻ = z - x̂⁻[0]
		nu := progressPct - xPred[0]
		S := Ppred[0][0] + immR // H Ppred H^T + R

		// Kalman gain: K = Ppred H^T / S  →  K[r] = Ppred[r][0] / S
		var K [3]float64
		for r := range 3 {
			K[r] = Ppred[r][0] / S
		}

		// Updated state: x̂ = x̂⁻ + K ν
		for r := range 3 {
			imm.models[j].x[r] = xPred[r] + K[r]*nu
		}

		// Joseph-form covariance: P = (I - KH) Ppred (I - KH)^T + K R K^T
		// A = I - K*H;  H = [1,0,0], so A[r][0] = δ_{r,0} - K[r], A[r][c]=δ_{r,c} for c≠0.
		var A [3][3]float64
		for r := range 3 {
			A[r][r] = 1
			A[r][0] -= K[r]
		}
		APAt := mat3Mul(mat3Mul(A, Ppred), mat3T(A))
		KKt := vec3Outer(K, K)
		for r := range 3 {
			for col := range 3 {
				imm.models[j].P[r][col] = APAt[r][col] + immR*KKt[r][col]
			}
		}

		// Gaussian likelihood: L_j = N(ν; 0, S)
		L[j] = math.Exp(-0.5*nu*nu/S) / math.Sqrt(2*math.Pi*S)
	}

	// ── Step 3: Update model probabilities ────────────────────────────────────
	// μ_j ← L_j * c_j, then normalize.
	for j := range 2 {
		imm.mu[j] = L[j] * c[j]
	}
	total := imm.mu[0] + imm.mu[1]
	if total <= 0 {
		// Both likelihoods underflowed; fall back to the prediction weights.
		imm.mu = c
		total = imm.mu[0] + imm.mu[1]
	}
	for j := range imm.mu {
		imm.mu[j] /= total
		if imm.mu[j] < immMinProb {
			imm.mu[j] = immMinProb
		}
	}
	// Renormalize after probability flooring.
	total = imm.mu[0] + imm.mu[1]
	for j := range imm.mu {
		imm.mu[j] /= total
	}

	// ── Step 4: Output fusion ─────────────────────────────────────────────────
	var xFused [3]float64
	for j := range 2 {
		for k := range 3 {
			xFused[k] += imm.mu[j] * imm.models[j].x[k]
		}
	}

	imm.count++
	vel := xFused[1]
	probs = imm.mu

	if imm.count < 3 || vel < 1e-6 {
		return math.Inf(1), vel, probs
	}
	remaining := 100.0 - xFused[0]
	if remaining <= 0 {
		return 0, vel, probs
	}
	return remaining / vel, vel, probs
}

// Snapshot captures the full filter state for persistence.
func (imm *IMM) Snapshot() *state.IMMSnapshot {
	return &state.IMMSnapshot{
		PrevTs: imm.prevTs,
		Mu:     imm.mu,
		M0X:    imm.models[0].x,
		M0P:    imm.models[0].P,
		M1X:    imm.models[1].x,
		M1P:    imm.models[1].P,
		Count:  imm.count,
	}
}

// restoreFromSnapshot loads a previously saved filter state.
func (imm *IMM) restoreFromSnapshot(snap *state.IMMSnapshot) {
	imm.prevTs = snap.PrevTs
	imm.mu = snap.Mu
	imm.models[0].x = snap.M0X
	imm.models[0].P = snap.M0P
	imm.models[1].x = snap.M1X
	imm.models[1].P = snap.M1P
	imm.count = snap.Count
}

// ── Kinematic model helpers ───────────────────────────────────────────────────

// immBuildF constructs the constant-acceleration transition matrix for time step dt.
//
//	F = [[1, dt, dt²/2],
//	     [0,  1, dt   ],
//	     [0,  0,  1   ]]
func immBuildF(dt float64) [3][3]float64 {
	return [3][3]float64{
		{1, dt, dt * dt / 2},
		{0, 1, dt},
		{0, 0, 1},
	}
}

// immBuildQ constructs the discrete process noise covariance for the given
// time step dt and noise intensity q. Derived from a continuous Wiener
// acceleration model (jerk = white noise):
//
//	Q = q * [[dt⁵/20, dt⁴/8, dt³/6],
//	          [dt⁴/8,  dt³/3, dt²/2],
//	          [dt³/6,  dt²/2, dt   ]]
func immBuildQ(dt, q float64) [3][3]float64 {
	dt2 := dt * dt
	dt3 := dt2 * dt
	dt4 := dt3 * dt
	dt5 := dt4 * dt
	return [3][3]float64{
		{q * dt5 / 20, q * dt4 / 8, q * dt3 / 6},
		{q * dt4 / 8, q * dt3 / 3, q * dt2 / 2},
		{q * dt3 / 6, q * dt2 / 2, q * dt},
	}
}

// ── 3×3 matrix arithmetic helpers ────────────────────────────────────────────

func mat3Mul(A, B [3][3]float64) [3][3]float64 {
	var C [3][3]float64
	for i := range 3 {
		for k := range 3 {
			for j := range 3 {
				C[i][j] += A[i][k] * B[k][j]
			}
		}
	}
	return C
}

func mat3T(A [3][3]float64) [3][3]float64 {
	var B [3][3]float64
	for i := range 3 {
		for j := range 3 {
			B[i][j] = A[j][i]
		}
	}
	return B
}

func mat3Add(A, B [3][3]float64) [3][3]float64 {
	var C [3][3]float64
	for i := range 3 {
		for j := range 3 {
			C[i][j] = A[i][j] + B[i][j]
		}
	}
	return C
}

func mat3MulVec(A [3][3]float64, v [3]float64) [3]float64 {
	var u [3]float64
	for i := range 3 {
		for j := range 3 {
			u[i] += A[i][j] * v[j]
		}
	}
	return u
}

func vec3Sub(a, b [3]float64) [3]float64 {
	return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]}
}

func vec3Outer(a, b [3]float64) [3][3]float64 {
	var M [3][3]float64
	for i := range 3 {
		for j := range 3 {
			M[i][j] = a[i] * b[j]
		}
	}
	return M
}
