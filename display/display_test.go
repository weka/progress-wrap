package display_test

import (
	"strings"
	"testing"
	"time"

	"github.com/baruch/progress-wrap/display"
	"github.com/stretchr/testify/assert"
)

func TestDisplay_ContainsProgressBar(t *testing.T) {
	line := display.Render(0.45, time.Time{}, false, 0.005, 0, 80)
	assert.Contains(t, line, "[")
	assert.Contains(t, line, "]")
	assert.Contains(t, line, "45.0%")
}

func TestDisplay_ETANotAvailable(t *testing.T) {
	line := display.Render(0.10, time.Time{}, false, 0, 0, 80)
	assert.Contains(t, line, "ETA: --")
}

func TestDisplay_ETAFormatted(t *testing.T) {
	eta := time.Now().Add(14*time.Minute + 32*time.Second)
	line := display.Render(0.45, eta, true, 0.005, 0, 80)
	assert.Contains(t, line, "ETA:")
	assert.NotContains(t, line, "ETA: --")
	assert.NotContains(t, line, "overdue")
	// Wall-clock time should appear as (HH:MM:SS)
	assert.Contains(t, line, eta.Local().Format("(15:04:05)"))
}

func TestDisplay_ETAOverdue(t *testing.T) {
	eta := time.Now().Add(-5 * time.Minute)
	line := display.Render(0.80, eta, true, 0.005, 0, 80)
	assert.Contains(t, line, "overdue")
}

func TestDisplay_VelocityShown(t *testing.T) {
	line := display.Render(0.45, time.Time{}, false, 0.005, 0, 80)
	assert.Contains(t, line, "%/s")
}

func TestDisplay_AccelShown(t *testing.T) {
	line := display.Render(0.45, time.Time{}, false, 0.005, 0.0001, 80)
	assert.Contains(t, line, "%/s²")
	assert.Contains(t, line, "+") // positive accel shows explicit + sign
}

func TestDisplay_BarFitsWidth(t *testing.T) {
	line := display.Render(0.50, time.Time{}, false, 0.01, 0, 80)
	assert.LessOrEqual(t, len(line), 80)
}

func TestDisplay_BarFillRatio(t *testing.T) {
	line := display.Render(0.50, time.Time{}, false, 0, 0, 40)
	// Extract bar content between [ and ]
	start := strings.Index(line, "[") + 1
	end := strings.Index(line, "]")
	bar := line[start:end]
	filled := strings.Count(bar, "=")
	total := len(bar)
	ratio := float64(filled) / float64(total)
	assert.InDelta(t, 0.50, ratio, 0.05)
}
