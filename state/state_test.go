package state_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/baruch/progress-wrap/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempPath(t *testing.T) string {
	return filepath.Join(t.TempDir(), "test.state")
}

func TestState_ReadWriteRoundtrip(t *testing.T) {
	path := tempPath(t)
	now := time.Now().UTC().Truncate(time.Nanosecond)

	s := &state.State{
		Command:   "weka status",
		StartedAt: now,
		UpdatedAt: now,
		Samples: []state.Sample{
			{Time: now, Progress: 0.10},
			{Time: now.Add(time.Second), Progress: 0.20},
		},
		Estimator: state.EstimatorState{Type: "ema", EMAVelocity: 0.0025},
	}

	require.NoError(t, state.Write(path, s))

	loaded, err := state.Read(path)
	require.NoError(t, err)
	assert.Equal(t, s.Command, loaded.Command)
	assert.Equal(t, s.Samples[0].Progress, loaded.Samples[0].Progress)
	// Verify nanosecond precision is preserved
	assert.Equal(t, s.Samples[0].Time.UnixNano(), loaded.Samples[0].Time.UnixNano())
}

func TestState_ReadMissingFile(t *testing.T) {
	s, err := state.Read("/nonexistent/path.state")
	require.NoError(t, err)
	assert.Nil(t, s)
}

func TestState_ReadCorruptFile(t *testing.T) {
	path := tempPath(t)
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))
	s, err := state.Read(path)
	require.NoError(t, err) // corrupt = treat as missing, no error
	assert.Nil(t, s)
}

func TestState_WriteIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.state")
	s := &state.State{Command: "test"}
	require.NoError(t, state.Write(path, s))

	// No temp files should remain after a successful write.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp", "leftover temp file: %s", e.Name())
	}
}

func TestState_ParallelWritesSafe(t *testing.T) {
	path := tempPath(t)

	// Pre-create a valid state file.
	initial := &state.State{Command: "test", Samples: []state.Sample{{Progress: 0.1}}}
	require.NoError(t, state.Write(path, initial))

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(i int) {
			defer wg.Done()
			s := &state.State{
				Command: "test",
				Samples: []state.Sample{{Progress: float64(i) / workers}},
			}
			assert.NoError(t, state.Write(path, s))
		}(i)
	}
	wg.Wait()

	// State file must be valid JSON after all concurrent writes.
	result, err := state.Read(path)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestState_SampleCap(t *testing.T) {
	path := tempPath(t)
	s := &state.State{Command: "test"}
	base := time.Now().UTC()
	for i := 0; i < 600; i++ {
		s.Samples = append(s.Samples, state.Sample{
			Time:     base.Add(time.Duration(i) * time.Second),
			Progress: float64(i) / 600.0,
		})
	}
	require.NoError(t, state.Write(path, s))
	loaded, err := state.Read(path)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(loaded.Samples), state.MaxSamples)
}

func TestState_Reset(t *testing.T) {
	path := tempPath(t)
	s := &state.State{Command: "test", Samples: []state.Sample{{Progress: 0.5}}}
	require.NoError(t, state.Write(path, s))

	require.NoError(t, state.Reset(path))
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err))
}

func TestState_ShouldAutoReset_BackwardProgress(t *testing.T) {
	s := &state.State{
		Samples: []state.Sample{{Progress: 0.50}},
	}
	// New progress is > 5% less than last — should reset
	assert.True(t, state.ShouldAutoReset(s, 0.40))
}

func TestState_ShouldAutoReset_SmallDrop(t *testing.T) {
	s := &state.State{
		Samples: []state.Sample{{Progress: 0.50}},
	}
	// Drop is within threshold — no reset
	assert.False(t, state.ShouldAutoReset(s, 0.48))
}

func TestState_ShouldAutoReset_NilState(t *testing.T) {
	assert.False(t, state.ShouldAutoReset(nil, 0.10))
}

func TestState_ShouldAutoReset_NoSamples(t *testing.T) {
	s := &state.State{}
	assert.False(t, state.ShouldAutoReset(s, 0.10))
}
