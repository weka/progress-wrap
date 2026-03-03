package display

import (
	"fmt"
	"strings"
	"time"
)

// Render returns a single-line progress bar string sized to termWidth columns.
// velocity is in progress-units/second (e.g. 0.005 = 0.5%/s).
func Render(progress float64, eta time.Time, etaOK bool, velocity float64, termWidth int) string {
	etaStr := formatETA(eta, etaOK)
	velStr := fmt.Sprintf("%.3f%%/s", velocity*100)
	suffix := fmt.Sprintf(" %.1f%%  ETA: %s  (avg velocity: %s)", progress*100, etaStr, velStr)

	barOuter := termWidth - len(suffix)
	if barOuter < 12 {
		barOuter = 12
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
