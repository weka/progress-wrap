package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/baruch/progress-wrap/state"
)

// debugLogger writes structured diagnostic entries to a file.
// All methods are no-ops when the logger is disabled (path == "").
type debugLogger struct {
	f *os.File
}

func newDebugLogger(path string) (*debugLogger, error) {
	if path == "" {
		return &debugLogger{}, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open debug log %q: %w", path, err)
	}
	return &debugLogger{f: f}, nil
}

func (d *debugLogger) enabled() bool { return d.f != nil }

func (d *debugLogger) close() {
	if d.f != nil {
		d.f.Close()
	}
}

func (d *debugLogger) writeln(format string, args ...any) {
	if d.f == nil {
		return
	}
	fmt.Fprintf(d.f, format+"\n", args...)
}

// Header writes the run separator and key invocation metadata.
func (d *debugLogger) Header(t time.Time, cmdStr, statePath, estimatorType string) {
	if !d.enabled() {
		return
	}
	d.writeln("")
	d.writeln("=== %s UTC ===", t.UTC().Format("2006-01-02 15:04:05"))
	d.writeln("Command:   %s", cmdStr)
	d.writeln("State:     %s", statePath)
	d.writeln("Estimator: %s", estimatorType)
}

// Capture logs the raw bytes captured from one stream (stdout or stderr).
func (d *debugLogger) Capture(stream string, data []byte) {
	if !d.enabled() {
		return
	}
	const limit = 4096
	d.writeln("--- %s (%d bytes) ---", stream, len(data))
	if len(data) == 0 {
		d.writeln("(empty)")
		return
	}
	preview := data
	truncated := false
	if len(preview) > limit {
		preview = preview[:limit]
		truncated = true
	}
	// Ensure content ends with a newline before the next writeln.
	fmt.Fprint(d.f, string(preview))
	if len(preview) > 0 && preview[len(preview)-1] != '\n' {
		fmt.Fprintln(d.f)
	}
	if truncated {
		d.writeln("...(truncated at %d bytes)", limit)
	}
}

// ParseResult logs the outcome of parsing one stream.
func (d *debugLogger) ParseResult(stream string, progress float64, found bool, err error) {
	if !d.enabled() {
		return
	}
	switch {
	case err != nil:
		d.writeln("parse %-6s -> error: %v", stream, err)
	case found:
		d.writeln("parse %-6s -> matched: %.4f (%.2f%%)", stream, progress, progress*100)
	default:
		d.writeln("parse %-6s -> no match", stream)
	}
}

// StateContext logs critical state information that determines whether an ETA
// can be computed.
func (d *debugLogger) StateContext(s *state.State, found bool, progress float64) {
	if !d.enabled() {
		return
	}
	d.writeln("--- state context ---")
	if s == nil || len(s.Samples) == 0 {
		d.writeln("samples:   0  (no history — ETA not yet computable)")
		return
	}

	n := len(s.Samples)
	first := s.Samples[0]
	last := s.Samples[n-1]
	elapsed := last.Time.Sub(first.Time)

	d.writeln("samples:   %d", n)
	d.writeln("first:     %s  %.2f%%", first.Time.UTC().Format("15:04:05 UTC"), first.Progress*100)
	d.writeln("last:      %s  %.2f%%", last.Time.UTC().Format("15:04:05 UTC"), last.Progress*100)
	d.writeln("elapsed:   %s", elapsed.Round(time.Second))

	if found {
		delta := progress - last.Progress
		d.writeln("new:       %.2f%%  (delta from last: %+.2f%%)", progress*100, delta*100)
	}

	est := s.Estimator
	d.writeln("estimator: type=%s  velocity=%.4f%%/s  accel=%+.4f%%/s²",
		est.Type, est.Velocity*100, est.Acceleration*100)

	// Summarise how "warm" the estimator is.
	switch {
	case n < 2:
		d.writeln("ETA:       not yet (need ≥ 2 samples, have %d)", n)
	case est.Velocity == 0:
		d.writeln("ETA:       not yet (velocity is zero)")
	default:
		remaining := (1.0 - last.Progress) / est.Velocity
		d.writeln("ETA:       ~%.0fs remaining (raw estimate from last velocity)", remaining)
	}

	// IMM model probabilities if available.
	if est.IMMSnapshot != nil {
		snap := est.IMMSnapshot
		d.writeln("IMM:       μ=[%.3f, %.3f]  count=%d",
			snap.Mu[0], snap.Mu[1], snap.Count)
		d.writeln("           model0 x=[%.4f, %.6f, %.8f]",
			snap.M0X[0], snap.M0X[1], snap.M0X[2])
		d.writeln("           model1 x=[%.4f, %.6f, %.8f]",
			snap.M1X[0], snap.M1X[1], snap.M1X[2])
	}

	if strings.Contains(est.Type, "kalman") || est.KalmanP11 > 0 {
		d.writeln("Kalman:    pos=%.4f vel=%.6f P=[%.2e %.2e; %.2e %.2e]",
			est.KalmanPos, est.KalmanVel,
			est.KalmanP00, est.KalmanP01, est.KalmanP01, est.KalmanP11)
	}
}
