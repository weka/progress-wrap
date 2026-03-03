package config_test

import (
	"os"
	"testing"

	"github.com/baruch/progress-wrap/parser"
	"github.com/baruch/progress-wrap/parser/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.toml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestConfig_LoadFile(t *testing.T) {
	content := `
[[parsers]]
command_regex = '^myapp'
type          = "regex"
pattern       = '(\d+)\s*%'
group         = 1
`
	path := writeTempTOML(t, content)
	entries, err := config.LoadFile(path)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadFile("/nonexistent/path.toml")
	assert.Error(t, err)
}

func TestConfig_CommandRegexMatching(t *testing.T) {
	content := `
[[parsers]]
command_regex = '^myapp status'
type          = "regex"
pattern       = '(\d+)\s*%'
group         = 1

[[parsers]]
command_regex = '^myapp'
type          = "regex"
pattern       = '(\d+)\s*done'
group         = 1
`
	path := writeTempTOML(t, content)
	entries, err := config.LoadFile(path)
	require.NoError(t, err)

	p := parser.Select("myapp status", entries)
	require.NotNil(t, p)
	prog, found, _ := p.Parse([]byte("50%"))
	assert.True(t, found)
	assert.InDelta(t, 0.50, prog, 1e-9)
}
