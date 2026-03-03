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
