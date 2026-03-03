package runner_test

import (
	"testing"

	"github.com/baruch/progress-wrap/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_CapturesStdout(t *testing.T) {
	stdout, code, err := runner.Run("echo", []string{"hello world"})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, string(stdout), "hello world")
}

func TestRun_NonZeroExitCode(t *testing.T) {
	_, code, err := runner.Run("sh", []string{"-c", "exit 42"})
	require.NoError(t, err)
	assert.Equal(t, 42, code)
}

func TestRun_StderrNotCaptured(t *testing.T) {
	stdout, code, err := runner.Run("sh", []string{"-c", "echo out; echo err >&2"})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Contains(t, string(stdout), "out")
	assert.NotContains(t, string(stdout), "err")
}

func TestRun_CommandNotFound(t *testing.T) {
	_, _, err := runner.Run("no_such_command_xyz", []string{})
	assert.Error(t, err)
}
