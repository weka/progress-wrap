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

func TestBuiltins_WekaClusterTask(t *testing.T) {
	entries, err := builtin.Load()
	require.NoError(t, err)

	header := "TASK ID  TYPE  STATE    PHASE                        PROGRESS  USER PAUSED  DESCRIPTION                  TIME\n"
	row := "984      FSCK  RUNNING  CHECK_REGISTRY_CHILDREN 1/2     96.32  False        Checking metadata integrity  1:00:52h\n"

	cases := []struct {
		name    string
		command string
		output  string
		want    float64
	}{
		{
			name:    "plain command",
			command: "weka cluster task",
			output:  header + row,
			want:    0.9632,
		},
		{
			name:    "grep by task id",
			command: "weka cluster task | grep -w 984",
			output:  row,
			want:    0.9632,
		},
		{
			name:    "grep by type",
			command: "weka cluster task | grep -w FSCK",
			output:  row,
			want:    0.9632,
		},
		{
			name:    "user paused true",
			command: "weka cluster task",
			output:  header + "985      RESTRIPE  RUNNING  RESTRIPE 2/3     45.00  True         Something  0:10:00h\n",
			want:    0.45,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := parser.Select(tc.command, entries)
			require.NotNil(t, p, "expected a parser for %q", tc.command)

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
