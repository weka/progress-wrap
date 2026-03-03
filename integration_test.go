//go:build integration

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "progress-wrap")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func TestIntegration_ProgressBarAppended(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	outputs := []string{}
	for i, pct := range []int{10, 30, 50} {
		args := []string{"--state", stateFile, "sh", "-c", "echo 'Progress: " + itoa(pct) + "%'"}
		out, err := exec.Command(binary, args...).CombinedOutput()
		require.NoError(t, err, "run %d failed: %s", i, out)
		outputs = append(outputs, string(out))
	}

	last := outputs[len(outputs)-1]
	assert.Contains(t, last, "Progress: 50%", "should contain original command output")
	assert.Contains(t, last, "%", "should contain progress percentage in bar line")
	assert.Contains(t, last, "ETA:", "should contain ETA label")
}

func TestIntegration_ResetFlag(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	// Build up some state
	for _, pct := range []int{30, 60} {
		args := []string{"--state", stateFile, "sh", "-c", "echo 'Progress: " + itoa(pct) + "%'"}
		exec.Command(binary, args...).Run() //nolint:errcheck
	}

	// Reset and run with low progress
	args := []string{"--state", stateFile, "--reset", "sh", "-c", "echo 'Progress: 5%'"}
	out, err := exec.Command(binary, args...).CombinedOutput()
	require.NoError(t, err)
	// After reset: only 1 sample, so ETA should be "--"
	assert.Contains(t, string(out), "ETA: --")
}

func TestIntegration_ExitCodePropagated(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	args := []string{"--state", stateFile, "--parse-regex", `(\d+)%`, "sh", "-c", "echo '50%'; exit 7"}
	cmd := exec.Command(binary, args...)
	err := cmd.Run()
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	assert.Equal(t, 7, exitErr.ExitCode())
}

func TestIntegration_AdHocRegexParser(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	args := []string{
		"--state", stateFile,
		"--parse-regex", `done:\s*(\d+)`,
		"sh", "-c", "echo 'done: 75'",
	}
	out, err := exec.Command(binary, args...).CombinedOutput()
	require.NoError(t, err)
	outStr := string(out)

	// Should have parsed 75 as 75% progress
	assert.True(t,
		strings.Contains(outStr, "75.0%") || strings.Contains(outStr, "75%"),
		"expected 75%% in output, got: %s", outStr)
}

func TestIntegration_StateFile(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	args := []string{"--state", stateFile, "sh", "-c", "echo 'Progress: 33%'"}
	_, err := exec.Command(binary, args...).CombinedOutput()
	require.NoError(t, err)

	// State file should exist and be valid JSON
	data, err := os.ReadFile(stateFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"command"`)
	assert.Contains(t, string(data), `"samples"`)
}

// TestIntegration_WekaRedist replays the 8 weka status runs from testdata/weka.redist.log
// using injected timestamps. It verifies that the progress bar appears, ETA is unavailable
// on the first run, and ETA is computed from the second run onward.
func TestIntegration_WekaRedist(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "weka.state")

	// fakeBinDir holds a fake "weka" binary rewritten for each run.
	fakeBinDir := t.TempDir()
	wekaBin := filepath.Join(fakeBinDir, "weka")

	type run struct {
		timestamp string
		pct       string
	}

	// Timestamps and percentages taken from testdata/weka.redist.log.
	runs := []run{
		{"2026-03-03T18:32:28Z", "3.84615"},
		{"2026-03-03T18:32:29Z", "10.641"},
		{"2026-03-03T18:32:31Z", "17.9487"},
		{"2026-03-03T18:32:32Z", "32.5641"},
		{"2026-03-03T18:32:33Z", "68.4615"},
		{"2026-03-03T18:32:34Z", "97.6923"},
		{"2026-03-03T18:32:35Z", "100"},
		{"2026-03-03T18:32:36Z", "100"},
	}

	pathEnv := "PATH=" + fakeBinDir + ":" + os.Getenv("PATH")

	for i, r := range runs {
		// Write a fake weka binary that outputs the redistribution progress line.
		script := fmt.Sprintf("#!/bin/sh\necho '                Data redistribution in progress (%s%%)'", r.pct)
		require.NoError(t, os.WriteFile(wekaBin, []byte(script), 0755), "write fake weka binary")

		cmd := exec.Command(binary, "--state", stateFile, "weka", "status")
		cmd.Env = append(os.Environ(), "PROGRESS_WRAP_NOW="+r.timestamp, pathEnv)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "run %d failed: %s", i+1, out)

		outStr := string(out)
		t.Logf("Run %d (t=%s, pct=%s%%):\n%s", i+1, r.timestamp, r.pct, strings.TrimRight(outStr, "\n"))

		assert.Contains(t, outStr, "%", "run %d: progress bar should be present", i+1)

		if i == 0 {
			assert.Contains(t, outStr, "ETA: --", "run 1: ETA unavailable with single sample")
		} else {
			assert.NotContains(t, outStr, "ETA: --", "run %d: ETA should be computed", i+1)
		}
	}
}
