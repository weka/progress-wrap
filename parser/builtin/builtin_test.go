package builtin_test

import (
	"testing"

	"github.com/baruch/progress-wrap/parser"
	"github.com/baruch/progress-wrap/parser/builtin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltins_Load(t *testing.T) {
	entries, err := builtin.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, entries)
}

func TestBuiltins_WekaStatusRegex(t *testing.T) {
	entries, err := builtin.Load()
	require.NoError(t, err)

	p := parser.Select("weka status", entries)
	require.NotNil(t, p, "expected a parser for 'weka status'")

	// Simulate weka status text output with a Progress: line
	sampleOutput := []byte("Status: OK\nProgress: 42.0%\nNodes: 5\n")
	prog, found, err := p.Parse(sampleOutput)
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.42, prog, 1e-9)
}

func TestBuiltins_WekaStatusJSON(t *testing.T) {
	entries, err := builtin.Load()
	require.NoError(t, err)

	p := parser.Select("weka status -J", entries)
	require.NotNil(t, p, "expected a parser for 'weka status -J'")

	sampleJSON := []byte(`{"progress": 0.65}`)
	prog, found, err := p.Parse(sampleJSON)
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.65, prog, 1e-9)
}
