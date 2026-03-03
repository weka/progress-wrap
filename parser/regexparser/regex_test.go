package regexparser_test

import (
	"testing"

	"github.com/baruch/progress-wrap/parser/regexparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegex_SimplePercent(t *testing.T) {
	p, err := regexparser.New(`(\d+(?:\.\d+)?)\s*%`, 1)
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte("Progress: 45.3%"))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.453, prog, 1e-9)
}

func TestRegex_NoMatch(t *testing.T) {
	p, err := regexparser.New(`(\d+)\s*%`, 1)
	require.NoError(t, err)
	_, found, err := p.Parse([]byte("no percentage here"))
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRegex_MultilinePicksFirst(t *testing.T) {
	p, err := regexparser.New(`(\d+)\s*%`, 1)
	require.NoError(t, err)
	output := []byte("line one\nProgress: 30%\nProgress: 50%\n")
	prog, found, err := p.Parse(output)
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.30, prog, 1e-9)
}

func TestRegex_InvalidGroupIndex(t *testing.T) {
	p, err := regexparser.New(`(\d+)\s*%`, 5)
	require.NoError(t, err)
	_, found, err := p.Parse([]byte("50%"))
	assert.Error(t, err)
	assert.False(t, found)
}

func TestRegex_InvalidPattern(t *testing.T) {
	_, err := regexparser.New(`[invalid`, 1)
	assert.Error(t, err)
}
