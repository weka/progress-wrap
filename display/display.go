package display

import (
	"fmt"
	"strings"
	"time"
)

// minBarWidth is the minimum number of columns reserved for the bar itself
// (the [...] portion, excluding the suffix). Prevents the bar from collapsing
// to nothing on very narrow terminals.
const minBarWidth = 12

// perMinThreshold is the raw velocity (progress-units/second) below which
// the display switches from per-second to per-minute units.
const perMinThreshold = 0.01 // 0.01/s == 1 %/s

// Render returns a single-line progress bar string sized to termWidth columns.
// velocity is in progress-units/second (e.g. 0.005 = 0.5%/s).
// accel is in progress-units/second² (e.g. 0.0001 = 0.01%/s²).
// When velocity is below perMinThreshold both velocity and acceleration are
// displayed in per-minute units for readability.
func Render(progress float64, eta time.Time, etaOK bool, velocity, accel float64, termWidth int) string {
	etaStr := formatETA(eta, etaOK)

	var velStr, accelStr string
	if velocity < perMinThreshold {
		velStr = fmt.Sprintf("%.3f%%/min", velocity*100*60)
		accelStr = fmt.Sprintf("%+.3f%%/min²", accel*100*3600)
	} else {
		velStr = fmt.Sprintf("%.3f%%/s", velocity*100)
		accelStr = fmt.Sprintf("%+.3f%%/s²", accel*100)
	}
	suffix := fmt.Sprintf(" %.1f%%  ETA: %s  (avg velocity: %s  accel: %s)", progress*100, etaStr, velStr, accelStr)

	barOuter := termWidth - len(suffix)
	if barOuter < minBarWidth {
		barOuter = minBarWidth
	}
	inner := barOuter - 2 // subtract [ and ]
	filled := int(progress * float64(inner))
	if filled > inner {
		filled = inner
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", inner-filled)
	return fmt.Sprintf("[%s]%s", bar, suffix)
}

func formatETA(eta time.Time, ok bool) string {
	if !ok {
		return "--"
	}
	remaining := time.Until(eta)
	if remaining <= 0 {
		return "overdue"
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	s := int(remaining.Seconds()) % 60
	var dur string
	if h > 0 {
		dur = fmt.Sprintf("%dh%dm%ds", h, m, s)
	} else if m > 0 {
		dur = fmt.Sprintf("%dm%ds", m, s)
	} else {
		dur = fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%s (%s)", dur, eta.Local().Format("15:04:05"))
}
