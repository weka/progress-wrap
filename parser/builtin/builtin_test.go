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

	cases := []struct {
		name   string
		output string
		want   float64
	}{
		{
			name:   "redistribution",
			output: "       status: REDISTRIBUTING\n                Data redistribution in progress (42.0%)\n   protection: 3+2\n",
			want:   0.42,
		},
		{
			name:   "rebuild",
			output: "       status: REBUILDING\n                Rebuild in progress (23.3806%)\n   protection: 3+2\n",
			want:   0.233806,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog, found, err := p.Parse([]byte(tc.output))
			require.NoError(t, err)
			assert.True(t, found)
			assert.InDelta(t, tc.want, prog, 1e-6)
		})
	}
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
