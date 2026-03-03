package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MaxSamples is the maximum number of samples retained in the state file.
const MaxSamples = 500

// AutoResetThreshold is the minimum backward-progress drop that triggers auto-reset.
const AutoResetThreshold = 0.05

// Sample is a single (time, progress) observation.
type Sample struct {
	Time     time.Time `json:"time"`
	Progress float64   `json:"progress"`
}

// EstimatorState holds serializable estimator data.
type EstimatorState struct {
	Type        string  `json:"type"`
	EMAVelocity float64 `json:"ema_velocity,omitempty"`
}

// State is the full contents of a state file.
type State struct {
	Command   string         `json:"command"`
	StartedAt time.Time      `json:"started_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Samples   []Sample       `json:"samples"`
	Estimator EstimatorState `json:"estimator"`
}

// Read loads a state file. Returns nil, nil if the file does not exist or is corrupt.
func Read(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Fprintf(os.Stderr, "warning: state file %q is corrupt, starting fresh\n", path)
		return nil, nil
	}
	return &s, nil
}

// Write serializes s to path atomically (unique temp file in the same
// directory, then rename). Caps Samples at MaxSamples (retaining the
// most recent entries).
func Write(path string, s *State) error {
	samples := s.Samples
	if len(samples) > MaxSamples {
		samples = samples[len(samples)-MaxSamples:]
	}
	snapshot := *s
	snapshot.Samples = samples
	data, err := json.MarshalIndent(&snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tmp := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close state temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// Reset deletes the state file if it exists.
func Reset(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reset state file: %w", err)
	}
	return nil
}

// MarshalJSON encodes Sample with RFC3339Nano timestamp.
func (s Sample) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Time     string  `json:"time"`
		Progress float64 `json:"progress"`
	}
	return json.Marshal(Alias{
		Time:     s.Time.UTC().Format(time.RFC3339Nano),
		Progress: s.Progress,
	})
}

// UnmarshalJSON decodes RFC3339Nano timestamps.
func (s *Sample) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Time     string  `json:"time"`
		Progress float64 `json:"progress"`
	}
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, a.Time)
	if err != nil {
		return fmt.Errorf("parse sample time: %w", err)
	}
	s.Time = t
	s.Progress = a.Progress
	return nil
}

// ShouldAutoReset returns true if newProgress has dropped more than
// AutoResetThreshold below the last recorded progress.
func ShouldAutoReset(s *State, newProgress float64) bool {
	if s == nil || len(s.Samples) == 0 {
		return false
	}
	last := s.Samples[len(s.Samples)-1].Progress
	return last-newProgress > AutoResetThreshold
}
