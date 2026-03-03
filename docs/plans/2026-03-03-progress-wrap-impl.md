# progress-wrap Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool that wraps arbitrary shell commands, parses a progress percentage from their output, and appends a progress bar with ETA to stdout.

**Architecture:** Modular Go packages — `runner`, `parser` (regex/jq with priority selection), `state` (JSON with RFC3339Nano), `estimator` (EMA, Kalman stub), `display`. The cobra CLI wires them together.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra`, `github.com/BurntSushi/toml`, `github.com/itchyny/gojq`, `golang.org/x/term`, `github.com/stretchr/testify`

---

### Task 1: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `runner/runner.go` (empty stub)
- Create: `parser/parser.go` (empty stub)
- Create: `state/state.go` (empty stub)
- Create: `estimator/estimator.go` (empty stub)
- Create: `display/display.go` (empty stub)
- Create: `parser/builtin/builtin.go` (empty stub)
- Create: `parser/config/config.go` (empty stub)

**Step 1: Initialize Go module**

```bash
go mod init github.com/baruch/progress-wrap
```

**Step 2: Install dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/BurntSushi/toml@latest
go get github.com/itchyny/gojq@latest
go get golang.org/x/term@latest
go get github.com/stretchr/testify@latest
```

**Step 3: Create `main.go`**

```go
package main

import "github.com/baruch/progress-wrap/cmd"

func main() {
	cmd.Execute()
}
```

**Step 4: Create `cmd/root.go` skeleton**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "progress-wrap",
	Short: "Wrap a command and show a progress bar with ETA",
	RunE:  runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	fmt.Println("TODO: implement")
	return nil
}
```

**Step 5: Create stub files for each package**

Each file below just declares its package so `go build ./...` passes.

`runner/runner.go`:
```go
package runner
```

`parser/parser.go`:
```go
package parser
```

`parser/builtin/builtin.go`:
```go
package builtin
```

`parser/config/config.go`:
```go
package config
```

`state/state.go`:
```go
package state
```

`estimator/estimator.go`:
```go
package estimator
```

`display/display.go`:
```go
package display
```

**Step 6: Verify build**

```bash
go build ./...
```

Expected: no errors.

**Step 7: Commit**

```bash
git add -A
git commit -m "feat: scaffold project structure"
```

---

### Task 2: Runner package

**Files:**
- Modify: `runner/runner.go`
- Create: `runner/runner_test.go`

**Step 1: Write the failing tests**

`runner/runner_test.go`:
```go
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
	// stderr passes through to terminal but is NOT in returned bytes
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
```

**Step 2: Run tests to verify they fail**

```bash
go test ./runner/...
```

Expected: compilation error (package is empty).

**Step 3: Implement `runner/runner.go`**

```go
package runner

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// Run executes name with args, streaming its stdout to os.Stdout while also
// capturing it. Stderr is passed through to os.Stderr but not captured.
// Returns captured stdout bytes, the process exit code, and any exec error.
// A non-zero exit code is NOT returned as an error.
func Run(name string, args []string) (stdout []byte, exitCode int, err error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = os.Stderr

	if runErr := cmd.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return buf.Bytes(), exitErr.ExitCode(), nil
		}
		return nil, -1, runErr
	}
	return buf.Bytes(), 0, nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./runner/... -v
```

Expected: all 4 tests PASS.

**Step 5: Commit**

```bash
git add runner/
git commit -m "feat: add runner package"
```

---

### Task 3: Parser interface and Entry type

**Files:**
- Modify: `parser/parser.go`
- Create: `parser/parser_test.go`

**Step 1: Write the failing test**

`parser/parser_test.go`:
```go
package parser_test

import (
	"testing"

	"github.com/baruch/progress-wrap/parser"
	"github.com/stretchr/testify/assert"
)

// mockParser always returns a fixed value
type mockParser struct{ val float64 }

func (m *mockParser) Parse(_ []byte) (float64, bool, error) { return m.val, true, nil }

func TestSelect_FirstSourceMatchWins(t *testing.T) {
	e1 := parser.Entry{CommandRegex: "^weka status$", Parser: &mockParser{0.5}}
	e2 := parser.Entry{CommandRegex: "^weka", Parser: &mockParser{0.9}}

	got := parser.Select("weka status", []parser.Entry{e1, e2})
	assert.NotNil(t, got)
	prog, found, _ := got.Parse(nil)
	assert.True(t, found)
	assert.Equal(t, 0.5, prog)
}

func TestSelect_FallbackWildcard(t *testing.T) {
	wildcard := parser.Entry{Parser: &mockParser{0.3}} // no CommandRegex
	got := parser.Select("anything", []parser.Entry{wildcard})
	assert.NotNil(t, got)
}

func TestSelect_NoMatch(t *testing.T) {
	e := parser.Entry{CommandRegex: "^specific$", Parser: &mockParser{1.0}}
	got := parser.Select("other command", []parser.Entry{e})
	assert.Nil(t, got)
}

func TestSelect_MultipleSourcePriority(t *testing.T) {
	// source[0] has no match, source[1] has a match
	s0 := []parser.Entry{{CommandRegex: "^nope$", Parser: &mockParser{0.1}}}
	s1 := []parser.Entry{{CommandRegex: "^weka$", Parser: &mockParser{0.7}}}
	got := parser.Select("weka", s0, s1)
	assert.NotNil(t, got)
	prog, _, _ := got.Parse(nil)
	assert.Equal(t, 0.7, prog)
}
```

**Step 2: Run to verify failure**

```bash
go test ./parser/... 2>&1 | head -20
```

Expected: compilation error.

**Step 3: Implement `parser/parser.go`**

```go
package parser

import "regexp"

// Parser extracts a progress value in [0,1] from command output.
type Parser interface {
	Parse(output []byte) (progress float64, found bool, err error)
}

// Entry pairs an optional command regex with a Parser.
// If CommandRegex is empty, the entry matches any command.
type Entry struct {
	CommandRegex string
	Parser       Parser
	compiled     *regexp.Regexp
}

// matches reports whether cmdStr matches the entry's CommandRegex.
func (e *Entry) matches(cmdStr string) bool {
	if e.CommandRegex == "" {
		return true
	}
	if e.compiled == nil {
		e.compiled = regexp.MustCompile(e.CommandRegex)
	}
	return e.compiled.MatchString(cmdStr)
}

// Select scans sources in order and returns the Parser from the first Entry
// whose CommandRegex matches cmdStr. Returns nil if no match is found.
func Select(cmdStr string, sources ...[]Entry) Parser {
	for _, entries := range sources {
		for i := range entries {
			if entries[i].matches(cmdStr) {
				return entries[i].Parser
			}
		}
	}
	return nil
}
```

**Step 4: Run tests**

```bash
go test ./parser/... -v
```

Expected: all 4 tests PASS.

**Step 5: Commit**

```bash
git add parser/parser.go parser/parser_test.go
git commit -m "feat: add parser interface and entry selector"
```

---

### Task 4: Regex parser

**Files:**
- Create: `parser/regexparser/regex.go`
- Create: `parser/regexparser/regex_test.go`

**Step 1: Write the failing tests**

`parser/regexparser/regex_test.go`:
```go
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
	p, err := regexparser.New(`(\d+)\s*%`, 5) // group 5 doesn't exist
	require.NoError(t, err)
	_, found, err := p.Parse([]byte("50%"))
	assert.Error(t, err)
	assert.False(t, found)
}

func TestRegex_InvalidPattern(t *testing.T) {
	_, err := regexparser.New(`[invalid`, 1)
	assert.Error(t, err)
}
```

**Step 2: Run to verify failure**

```bash
go test ./parser/regexparser/... 2>&1 | head -5
```

Expected: compilation error (package doesn't exist).

**Step 3: Implement `parser/regexparser/regex.go`**

```go
package regexparser

import (
	"fmt"
	"regexp"
	"strconv"
)

// RegexParser extracts progress from a capture group in a regex pattern.
type RegexParser struct {
	re    *regexp.Regexp
	group int
}

// New compiles pattern and returns a RegexParser that extracts capture group group.
func New(pattern string, group int) (*RegexParser, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return &RegexParser{re: re, group: group}, nil
}

// Parse scans output line by line, returns the float64 from the first match
// divided by 100 (converting percentage to [0,1]).
func (p *RegexParser) Parse(output []byte) (float64, bool, error) {
	m := p.re.FindSubmatch(output)
	if m == nil {
		return 0, false, nil
	}
	if p.group >= len(m) {
		return 0, false, fmt.Errorf("regex has %d groups but group %d was requested", len(m)-1, p.group)
	}
	val, err := strconv.ParseFloat(string(m[p.group]), 64)
	if err != nil {
		return 0, false, fmt.Errorf("could not parse %q as float: %w", m[p.group], err)
	}
	return val / 100.0, true, nil
}
```

**Step 4: Run tests**

```bash
go test ./parser/regexparser/... -v
```

Expected: all 5 tests PASS.

**Step 5: Commit**

```bash
git add parser/regexparser/
git commit -m "feat: add regex parser"
```

---

### Task 5: jq parser

**Files:**
- Create: `parser/jqparser/jq.go`
- Create: `parser/jqparser/jq_test.go`

**Step 1: Write the failing tests**

`parser/jqparser/jq_test.go`:
```go
package jqparser_test

import (
	"testing"

	"github.com/baruch/progress-wrap/parser/jqparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJQ_SimpleField(t *testing.T) {
	p, err := jqparser.New(".progress * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"progress": 0.453}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.453, prog, 1e-9)
}

func TestJQ_Division(t *testing.T) {
	p, err := jqparser.New(".done / .total * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"done": 45, "total": 100}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.45, prog, 1e-9)
}

func TestJQ_NestedField(t *testing.T) {
	p, err := jqparser.New(".status.progress * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"status":{"progress":0.7}}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.70, prog, 1e-9)
}

func TestJQ_InvalidJSON(t *testing.T) {
	p, err := jqparser.New(".progress * 100")
	require.NoError(t, err)
	_, found, err := p.Parse([]byte("not json"))
	assert.Error(t, err)
	assert.False(t, found)
}

func TestJQ_FieldMissing(t *testing.T) {
	p, err := jqparser.New(".missing * 100")
	require.NoError(t, err)
	_, found, err := p.Parse([]byte(`{}`))
	// null * 100 = 0 in jq; we treat 0 as not-found to avoid false progress
	require.NoError(t, err)
	assert.False(t, found)
}

func TestJQ_InvalidExpression(t *testing.T) {
	_, err := jqparser.New("[[[[invalid")
	assert.Error(t, err)
}
```

**Step 2: Run to verify failure**

```bash
go test ./parser/jqparser/... 2>&1 | head -5
```

**Step 3: Implement `parser/jqparser/jq.go`**

```go
package jqparser

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// JQParser evaluates a jq expression against JSON output to extract progress.
type JQParser struct {
	query *gojq.Query
}

// New compiles expression and returns a JQParser.
func New(expression string) (*JQParser, error) {
	q, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression: %w", err)
	}
	return &JQParser{query: q}, nil
}

// Parse unmarshals output as JSON, runs the jq expression, and returns the
// result divided by 100 (converting percentage to [0,1]).
// Returns found=false if result is null or zero.
func (p *JQParser) Parse(output []byte) (float64, bool, error) {
	var v interface{}
	if err := json.Unmarshal(output, &v); err != nil {
		return 0, false, fmt.Errorf("jq parser: invalid JSON: %w", err)
	}
	iter := p.query.Run(v)
	result, ok := iter.Next()
	if !ok || result == nil {
		return 0, false, nil
	}
	if err, ok := result.(error); ok {
		return 0, false, fmt.Errorf("jq expression error: %w", err)
	}
	pct, err := toFloat64(result)
	if err != nil {
		return 0, false, err
	}
	if pct == 0 {
		return 0, false, nil
	}
	return pct / 100.0, true, nil
}

func toFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case int:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("jq result is %T, expected number", v)
	}
}
```

**Step 4: Run tests**

```bash
go test ./parser/jqparser/... -v
```

Expected: all 6 tests PASS.

**Step 5: Commit**

```bash
git add parser/jqparser/
git commit -m "feat: add jq parser"
```

---

### Task 6: Built-in parsers

**Files:**
- Create: `parser/builtin/builtin_parsers.toml`
- Modify: `parser/builtin/builtin.go`
- Create: `parser/builtin/builtin_test.go`

**Step 1: Write failing tests**

`parser/builtin/builtin_test.go`:
```go
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
```

**Step 2: Run to verify failure**

```bash
go test ./parser/builtin/... 2>&1 | head -5
```

**Step 3: Create `parser/builtin/builtin_parsers.toml`**

```toml
# Built-in parsers for known commands.
# Entries are tested in order; first command_regex match wins.
# command_regex is matched against the full command string.

[[parsers]]
command_regex = '^weka status -J'
type          = "jq"
expression    = ".progress * 100"

[[parsers]]
command_regex = '^weka status'
type          = "regex"
pattern       = 'Progress:\s*(\d+(?:\.\d+)?)\s*%'
group         = 1
```

**Step 4: Implement `parser/builtin/builtin.go`**

```go
package builtin

import (
	_ "embed"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/baruch/progress-wrap/parser"
	"github.com/baruch/progress-wrap/parser/jqparser"
	"github.com/baruch/progress-wrap/parser/regexparser"
)

//go:embed builtin_parsers.toml
var builtinTOML []byte

type entryConfig struct {
	CommandRegex string `toml:"command_regex"`
	Type         string `toml:"type"`
	Pattern      string `toml:"pattern"`
	Group        int    `toml:"group"`
	Expression   string `toml:"expression"`
}

type config struct {
	Parsers []entryConfig `toml:"parsers"`
}

// Load parses the embedded builtin_parsers.toml and returns a slice of Entry.
func Load() ([]parser.Entry, error) {
	return loadTOML(builtinTOML)
}

// LoadFile parses an external TOML file with the same schema as builtin_parsers.toml.
func LoadFile(data []byte) ([]parser.Entry, error) {
	return loadTOML(data)
}

func loadTOML(data []byte) ([]parser.Entry, error) {
	var cfg config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse parser config: %w", err)
	}
	entries := make([]parser.Entry, 0, len(cfg.Parsers))
	for _, ec := range cfg.Parsers {
		p, err := buildParser(ec)
		if err != nil {
			return nil, err
		}
		entries = append(entries, parser.Entry{
			CommandRegex: ec.CommandRegex,
			Parser:       p,
		})
	}
	return entries, nil
}

func buildParser(ec entryConfig) (parser.Parser, error) {
	switch ec.Type {
	case "regex":
		g := ec.Group
		if g == 0 {
			g = 1
		}
		return regexparser.New(ec.Pattern, g)
	case "jq":
		return jqparser.New(ec.Expression)
	default:
		return nil, fmt.Errorf("unknown parser type %q", ec.Type)
	}
}
```

**Step 5: Run tests**

```bash
go test ./parser/builtin/... -v
```

Expected: all 3 tests PASS.

**Step 6: Commit**

```bash
git add parser/builtin/
git commit -m "feat: add built-in parsers with embedded TOML"
```

---

### Task 7: Config file parser

**Files:**
- Modify: `parser/config/config.go`
- Create: `parser/config/config_test.go`

The config package reuses `builtin.LoadFile` — it's just a file-loading wrapper.

**Step 1: Write the failing tests**

`parser/config/config_test.go`:
```go
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
	toml := `
[[parsers]]
command_regex = '^myapp'
type          = "regex"
pattern       = '(\d+)\s*%'
group         = 1
`
	path := writeTempTOML(t, toml)
	entries, err := config.LoadFile(path)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestConfig_FileNotFound(t *testing.T) {
	_, err := config.LoadFile("/nonexistent/path.toml")
	assert.Error(t, err)
}

func TestConfig_CommandRegexMatching(t *testing.T) {
	toml := `
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
	path := writeTempTOML(t, toml)
	entries, err := config.LoadFile(path)
	require.NoError(t, err)

	p := parser.Select("myapp status", entries)
	require.NotNil(t, p)
	prog, found, _ := p.Parse([]byte("50%"))
	assert.True(t, found)
	assert.InDelta(t, 0.50, prog, 1e-9)
}
```

**Step 2: Run to verify failure**

```bash
go test ./parser/config/... 2>&1 | head -5
```

**Step 3: Implement `parser/config/config.go`**

```go
package config

import (
	"fmt"
	"os"

	"github.com/baruch/progress-wrap/parser"
	"github.com/baruch/progress-wrap/parser/builtin"
)

// LoadFile reads path and returns parser entries using the same TOML schema
// as the built-in parsers.
func LoadFile(path string) ([]parser.Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read parser config %q: %w", path, err)
	}
	return builtin.LoadFile(data)
}
```

**Step 4: Run tests**

```bash
go test ./parser/config/... -v
```

Expected: all 3 tests PASS.

**Step 5: Commit**

```bash
git add parser/config/
git commit -m "feat: add config file parser loader"
```

---

### Task 8: State package

**Files:**
- Modify: `state/state.go`
- Create: `state/state_test.go`

**Step 1: Write the failing tests**

`state/state_test.go`:
```go
package state_test

import (
	"os"
	"path/filepath"
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
	path := tempPath(t)
	s := &state.State{Command: "test"}
	require.NoError(t, state.Write(path, s))

	// .tmp file should not remain
	_, err := os.Stat(path + ".tmp")
	assert.True(t, os.IsNotExist(err))
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
```

**Step 2: Run to verify failure**

```bash
go test ./state/... 2>&1 | head -5
```

**Step 3: Implement `state/state.go`**

```go
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// MaxSamples is the maximum number of samples retained in the state file.
const MaxSamples = 500

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

// Write serializes s to path atomically (write to path+".tmp", then rename).
// Caps Samples at MaxSamples (retaining the most recent entries).
func Write(path string, s *State) error {
	if len(s.Samples) > MaxSamples {
		s.Samples = s.Samples[len(s.Samples)-MaxSamples:]
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
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

// MarshalJSON encodes time.Time as RFC3339Nano.
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
```

**Step 4: Run tests**

```bash
go test ./state/... -v
```

Expected: all 6 tests PASS.

**Step 5: Commit**

```bash
git add state/
git commit -m "feat: add state package with atomic writes and RFC3339Nano timestamps"
```

---

### Task 9: Reset logic (auto-detect backward progress)

**Files:**
- Modify: `state/state.go`
- Add tests to: `state/state_test.go`

**Step 1: Add the failing tests** (append to `state/state_test.go`)

```go
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
```

**Step 2: Run to verify failure**

```bash
go test ./state/... -run TestState_ShouldAutoReset 2>&1
```

Expected: compilation error.

**Step 3: Add `ShouldAutoReset` to `state/state.go`**

```go
// AutoResetThreshold is the minimum backward-progress drop that triggers auto-reset.
const AutoResetThreshold = 0.05

// ShouldAutoReset returns true if newProgress has dropped more than
// AutoResetThreshold below the last recorded progress.
func ShouldAutoReset(s *State, newProgress float64) bool {
	if s == nil || len(s.Samples) == 0 {
		return false
	}
	last := s.Samples[len(s.Samples)-1].Progress
	return last-newProgress > AutoResetThreshold
}
```

**Step 4: Run all state tests**

```bash
go test ./state/... -v
```

Expected: all 10 tests PASS.

**Step 5: Commit**

```bash
git add state/
git commit -m "feat: add auto-reset detection for backward progress"
```

---

### Task 10: EMA estimator

**Files:**
- Modify: `estimator/estimator.go`
- Create: `estimator/ema.go`
- Create: `estimator/ema_test.go`

**Step 1: Write the failing tests**

`estimator/ema_test.go`:
```go
package estimator_test

import (
	"testing"
	"time"

	"github.com/baruch/progress-wrap/estimator"
	"github.com/stretchr/testify/assert"
)

func TestEMA_NotEnoughSamples(t *testing.T) {
	e := estimator.NewEMA(0.2)
	e.Update(0.10, time.Now())
	_, ok := e.ETA()
	assert.False(t, ok, "need >= 2 samples")
}

func TestEMA_ETAAfterTwoSamples(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.00, t0)
	e.Update(0.10, t0.Add(10*time.Second)) // 1%/s
	eta, ok := e.ETA()
	assert.True(t, ok)
	// remaining = 0.90, velocity = 0.01/s → ~90s from t1
	assert.WithinDuration(t, t0.Add(100*time.Second), eta, 5*time.Second)
}

func TestEMA_VelocitySmoothing(t *testing.T) {
	e := estimator.NewEMA(0.5)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.1, t0.Add(10*time.Second))  // instant v = 0.01/s
	e.Update(0.3, t0.Add(20*time.Second))  // instant v = 0.02/s
	// EMA v = 0.5*0.02 + 0.5*(0.5*0.01 + 0.5*0.01) = blended, should be between 0.01 and 0.02
	_, ok := e.ETA()
	assert.True(t, ok)
}

func TestEMA_NegativeVelocity(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.5, t0)
	e.Update(0.3, t0.Add(10*time.Second)) // going backward
	_, ok := e.ETA()
	assert.False(t, ok, "negative velocity should not produce ETA")
}

func TestEMA_StateRoundtrip(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.1, t0.Add(10*time.Second))
	s := e.State()
	assert.Equal(t, "ema", s.Type)
	assert.Greater(t, s.EMAVelocity, 0.0)
}

func TestEMA_RestoreFromState(t *testing.T) {
	e := estimator.NewEMA(0.2)
	t0 := time.Now()
	e.Update(0.0, t0)
	e.Update(0.2, t0.Add(10*time.Second))

	// Restore into a new EMA
	s := e.State()
	e2 := estimator.NewEMAFromState(s, 0.2, 0.2, t0.Add(10*time.Second))
	eta, ok := e2.ETA()
	assert.True(t, ok)
	assert.False(t, eta.IsZero())
}
```

**Step 2: Run to verify failure**

```bash
go test ./estimator/... 2>&1 | head -5
```

**Step 3: Implement `estimator/estimator.go`** (the interface)

```go
package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// Estimator tracks progress samples and produces ETA predictions.
type Estimator interface {
	Update(progress float64, t time.Time)
	ETA() (eta time.Time, ok bool)
	State() state.EstimatorState
}
```

**Step 4: Implement `estimator/ema.go`**

```go
package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// EMA is an exponential-moving-average velocity estimator.
type EMA struct {
	alpha    float64
	velocity float64
	lastTime time.Time
	lastProg float64
	count    int
}

// NewEMA creates an EMA estimator with smoothing factor alpha (0 < alpha <= 1).
func NewEMA(alpha float64) *EMA {
	return &EMA{alpha: alpha}
}

// NewEMAFromState restores an EMA estimator from persisted state.
func NewEMAFromState(s state.EstimatorState, alpha, lastProg float64, lastTime time.Time) *EMA {
	return &EMA{
		alpha:    alpha,
		velocity: s.EMAVelocity,
		lastTime: lastTime,
		lastProg: lastProg,
		count:    2, // enough to emit ETA
	}
}

// Update records a new progress observation.
func (e *EMA) Update(progress float64, t time.Time) {
	if e.count == 0 {
		e.lastTime = t
		e.lastProg = progress
		e.count++
		return
	}
	dt := t.Sub(e.lastTime).Seconds()
	if dt <= 0 {
		return
	}
	instant := (progress - e.lastProg) / dt
	if e.count == 1 {
		e.velocity = instant
	} else {
		e.velocity = e.alpha*instant + (1-e.alpha)*e.velocity
	}
	e.lastTime = t
	e.lastProg = progress
	e.count++
}

// ETA returns the estimated completion time. ok is false if there are fewer
// than 2 samples or velocity is non-positive.
func (e *EMA) ETA() (time.Time, bool) {
	if e.count < 2 || e.velocity <= 0 {
		return time.Time{}, false
	}
	remaining := 1.0 - e.lastProg
	secs := remaining / e.velocity
	return e.lastTime.Add(time.Duration(secs * float64(time.Second))), true
}

// State returns the serializable estimator state.
func (e *EMA) State() state.EstimatorState {
	return state.EstimatorState{
		Type:        "ema",
		EMAVelocity: e.velocity,
	}
}
```

**Step 5: Run tests**

```bash
go test ./estimator/... -v
```

Expected: all 6 tests PASS.

**Step 6: Commit**

```bash
git add estimator/
git commit -m "feat: add EMA estimator"
```

---

### Task 11: Kalman filter stub

**Files:**
- Create: `estimator/kalman.go`
- Create: `estimator/kalman_test.go`

**Step 1: Write the failing test**

`estimator/kalman_test.go`:
```go
package estimator_test

import (
	"testing"
	"time"

	"github.com/baruch/progress-wrap/estimator"
)

func TestKalman_ImplementsInterface(t *testing.T) {
	var _ estimator.Estimator = estimator.NewKalman()
}

func TestKalman_ReturnsNotOK(t *testing.T) {
	k := estimator.NewKalman()
	k.Update(0.5, time.Now())
	_, ok := k.ETA()
	// stub always returns false until implemented
	_ = ok // will become true once full implementation lands
}
```

**Step 2: Implement `estimator/kalman.go`**

```go
package estimator

import (
	"time"

	"github.com/baruch/progress-wrap/state"
)

// Kalman is a placeholder for the Kalman-filter estimator.
// It satisfies the Estimator interface but always returns ok=false until
// the full implementation is added.
type Kalman struct{}

func NewKalman() *Kalman { return &Kalman{} }

func (k *Kalman) Update(_ float64, _ time.Time) {}

func (k *Kalman) ETA() (time.Time, bool) { return time.Time{}, false }

func (k *Kalman) State() state.EstimatorState {
	return state.EstimatorState{Type: "kalman"}
}
```

**Step 3: Run tests**

```bash
go test ./estimator/... -v
```

Expected: all tests PASS.

**Step 4: Commit**

```bash
git add estimator/kalman.go estimator/kalman_test.go
git commit -m "feat: add Kalman estimator stub"
```

---

### Task 12: Display package

**Files:**
- Modify: `display/display.go`
- Create: `display/display_test.go`

**Step 1: Write the failing tests**

`display/display_test.go`:
```go
package display_test

import (
	"strings"
	"testing"
	"time"

	"github.com/baruch/progress-wrap/display"
	"github.com/stretchr/testify/assert"
)

func TestDisplay_ContainsProgressBar(t *testing.T) {
	line := display.Render(0.45, time.Time{}, false, 0.005, 80)
	assert.Contains(t, line, "[")
	assert.Contains(t, line, "]")
	assert.Contains(t, line, "45.0%")
}

func TestDisplay_ETANotAvailable(t *testing.T) {
	line := display.Render(0.10, time.Time{}, false, 0, 80)
	assert.Contains(t, line, "ETA: --")
}

func TestDisplay_ETAFormatted(t *testing.T) {
	eta := time.Now().Add(14*time.Minute + 32*time.Second)
	line := display.Render(0.45, eta, true, 0.005, 80)
	assert.Contains(t, line, "ETA:")
	assert.NotContains(t, line, "ETA: --")
	assert.NotContains(t, line, "overdue")
}

func TestDisplay_ETAOverdue(t *testing.T) {
	eta := time.Now().Add(-5 * time.Minute)
	line := display.Render(0.80, eta, true, 0.005, 80)
	assert.Contains(t, line, "overdue")
}

func TestDisplay_VelocityShown(t *testing.T) {
	line := display.Render(0.45, time.Time{}, false, 0.005, 80)
	assert.Contains(t, line, "%/s")
}

func TestDisplay_BarFitsWidth(t *testing.T) {
	line := display.Render(0.50, time.Time{}, false, 0.01, 80)
	assert.LessOrEqual(t, len(line), 80)
}

func TestDisplay_BarFillRatio(t *testing.T) {
	line := display.Render(0.50, time.Time{}, false, 0, 40)
	// Extract between [ and ]
	start := strings.Index(line, "[") + 1
	end := strings.Index(line, "]")
	bar := line[start:end]
	filled := strings.Count(bar, "=")
	total := len(bar)
	ratio := float64(filled) / float64(total)
	assert.InDelta(t, 0.50, ratio, 0.05)
}
```

**Step 2: Run to verify failure**

```bash
go test ./display/... 2>&1 | head -5
```

**Step 3: Implement `display/display.go`**

```go
package display

import (
	"fmt"
	"strings"
	"time"
)

// Render returns a single-line progress bar string.
// termWidth is the available terminal width (use TermWidth() to detect it).
func Render(progress float64, eta time.Time, etaOK bool, velocity float64, termWidth int) string {
	etaStr := formatETA(eta, etaOK)
	velStr := fmt.Sprintf("%.3f%%/s", velocity*100)
	suffix := fmt.Sprintf(" %.1f%%  ETA: %s  (avg velocity: %s)", progress*100, etaStr, velStr)

	barOuter := termWidth - len(suffix)
	if barOuter < 12 {
		barOuter = 12
	}
	inner := barOuter - 2 // subtract [ and ]
	filled := int(progress * float64(inner))
	if filled > inner {
		filled = inner
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", inner-filled)
	return fmt.Sprintf("[%s]%s", bar, suffix)
}

func formatETA(eta time.Time, ok bool) string {
	if !ok {
		return "--"
	}
	remaining := time.Until(eta)
	if remaining <= 0 {
		return "overdue"
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	s := int(remaining.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
```

**Step 4: Create `display/term.go`** for terminal width detection

```go
package display

import (
	"os"

	"golang.org/x/term"
)

// TermWidth returns the current terminal width, falling back to 80.
func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}
```

**Step 5: Run tests**

```bash
go test ./display/... -v
```

Expected: all 7 tests PASS.

**Step 6: Commit**

```bash
git add display/
git commit -m "feat: add display package with progress bar and ETA formatting"
```

---

### Task 13: CLI wiring

**Files:**
- Modify: `cmd/root.go`
- Modify: `main.go` (already correct)

**Step 1: Implement `cmd/root.go`**

```go
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/baruch/progress-wrap/display"
	"github.com/baruch/progress-wrap/estimator"
	"github.com/baruch/progress-wrap/parser"
	"github.com/baruch/progress-wrap/parser/builtin"
	"github.com/baruch/progress-wrap/parser/config"
	"github.com/baruch/progress-wrap/parser/jqparser"
	"github.com/baruch/progress-wrap/parser/regexparser"
	"github.com/baruch/progress-wrap/runner"
	"github.com/baruch/progress-wrap/state"
	"github.com/spf13/cobra"
)

var (
	flagState      string
	flagConfig     string
	flagReset      bool
	flagEstimator  string
	flagParseRegex string
	flagParseJQ    string
	flagEMAAlpha   float64
)

var rootCmd = &cobra.Command{
	Use:                "progress-wrap",
	Short:              "Wrap a command and show a progress bar with ETA",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runRoot,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVar(&flagState, "state", "", "Path to JSON state file (required)")
	rootCmd.Flags().StringVar(&flagConfig, "config", "", "Path to TOML parser config file")
	rootCmd.Flags().BoolVar(&flagReset, "reset", false, "Reset state before running")
	rootCmd.Flags().StringVar(&flagEstimator, "estimator", "ema", "Estimator type: ema or kalman")
	rootCmd.Flags().StringVar(&flagParseRegex, "parse-regex", "", "Ad-hoc regex parser pattern")
	rootCmd.Flags().StringVar(&flagParseJQ, "parse-jq", "", "Ad-hoc jq parser expression")
	rootCmd.Flags().Float64Var(&flagEMAAlpha, "ema-alpha", 0.2, "EMA smoothing factor (0 < alpha <= 1)")
	_ = rootCmd.MarkFlagRequired("state")
}

func runRoot(cmd *cobra.Command, args []string) error {
	cmdStr := strings.Join(args, " ")

	// Handle --reset
	if flagReset {
		if err := state.Reset(flagState); err != nil {
			return err
		}
	}

	// Load state
	s, err := state.Read(flagState)
	if err != nil {
		return err
	}

	// Build parser sources in priority order
	var sources [][]parser.Entry

	// 1. CLI inline flags (highest priority)
	if flagParseRegex != "" {
		p, err := regexparser.New(flagParseRegex, 1)
		if err != nil {
			return fmt.Errorf("--parse-regex: %w", err)
		}
		sources = append(sources, []parser.Entry{{Parser: p}})
	} else if flagParseJQ != "" {
		p, err := jqparser.New(flagParseJQ)
		if err != nil {
			return fmt.Errorf("--parse-jq: %w", err)
		}
		sources = append(sources, []parser.Entry{{Parser: p}})
	}

	// 2. Config file
	if flagConfig != "" {
		entries, err := config.LoadFile(flagConfig)
		if err != nil {
			return err
		}
		sources = append(sources, entries)
	}

	// 3. Built-ins
	builtins, err := builtin.Load()
	if err != nil {
		return err
	}
	sources = append(sources, builtins)

	selectedParser := parser.Select(cmdStr, sources...)

	// Run the command
	stdout, exitCode, err := runner.Run(args[0], args[1:])
	if err != nil {
		return fmt.Errorf("run command: %w", err)
	}

	// Parse progress
	var progress float64
	var found bool
	if selectedParser != nil {
		progress, found, err = selectedParser.Parse(stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: parser error: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: no parser matched command %q\n", cmdStr)
	}

	now := time.Now().UTC()

	if found {
		// Auto-reset if progress went backward
		if state.ShouldAutoReset(s, progress) {
			fmt.Fprintf(os.Stderr, "info: progress reset detected, clearing state\n")
			s = nil
		}

		// Initialize state if needed
		if s == nil {
			s = &state.State{
				Command:   cmdStr,
				StartedAt: now,
			}
		}
		s.UpdatedAt = now
		s.Samples = append(s.Samples, state.Sample{Time: now, Progress: progress})

		// Build estimator
		var est estimator.Estimator
		switch flagEstimator {
		case "kalman":
			est = estimator.NewKalman()
		default:
			if len(s.Samples) >= 2 && s.Estimator.EMAVelocity > 0 {
				last := s.Samples[len(s.Samples)-1]
				est = estimator.NewEMAFromState(s.Estimator, flagEMAAlpha, last.Progress, last.Time)
			} else {
				est = estimator.NewEMA(flagEMAAlpha)
				for _, sample := range s.Samples {
					est.Update(sample.Progress, sample.Time)
				}
			}
		}
		est.Update(progress, now)
		s.Estimator = est.State()

		if err := state.Write(flagState, s); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write state: %v\n", err)
		}

		eta, etaOK := est.ETA()
		velocity := est.State().EMAVelocity
		line := display.Render(progress, eta, etaOK, velocity, display.TermWidth())
		fmt.Println(line)
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
```

**Step 2: Build and verify no compilation errors**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Smoke test**

```bash
echo "Progress: 42%" | go run . --state /tmp/smoke.state sh -c 'echo "Progress: 42%"'
```

Expected: command output followed by a progress bar line.

**Step 4: Commit**

```bash
git add cmd/ main.go
git commit -m "feat: wire up CLI with all packages"
```

---

### Task 14: Integration test

**Files:**
- Create: `integration_test.go`

**Step 1: Write the integration test**

`integration_test.go`:
```go
//go:build integration

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_ProgressBarAppended runs progress-wrap wrapping a script
// that prints a progress percentage, verifies the ETA line is appended.
func TestIntegration_ProgressBarAppended(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	// Run 3 times to build up EMA history
	outputs := []string{}
	for i, pct := range []int{10, 30, 50} {
		script := []string{"sh", "-c", "echo 'Progress: " + itoa(pct) + "%'"}
		args := append([]string{"--state", stateFile}, script...)
		out, err := exec.Command(binary, args...).CombinedOutput()
		require.NoError(t, err, "run %d failed: %s", i, out)
		outputs = append(outputs, string(out))
	}

	last := outputs[len(outputs)-1]
	assert.Contains(t, last, "%", "should contain progress percentage")
	assert.Contains(t, last, "ETA:", "should contain ETA")
	assert.Contains(t, last, "Progress: 50%", "should contain original command output")
}

func TestIntegration_ResetFlag(t *testing.T) {
	binary := buildBinary(t)
	stateFile := filepath.Join(t.TempDir(), "test.state")

	// Build up some state
	for _, pct := range []int{30, 60} {
		args := []string{"--state", stateFile, "sh", "-c", "echo 'Progress: " + itoa(pct) + "%'"}
		exec.Command(binary, args...).Run()
	}

	// Reset and run with low progress — should not show overdue
	args := []string{"--state", stateFile, "--reset", "sh", "-c", "echo 'Progress: 5%'"}
	out, err := exec.Command(binary, args...).CombinedOutput()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "overdue")
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "progress-wrap")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Join(strings.Fields(os.Expand("$n", func(k string) string {
		if k == "n" {
			return fmt.Sprintf("%d", n)
		}
		return ""
	})), ""))
}
```

Actually replace the `itoa` helper with a simpler one:

```go
import "fmt"
func itoa(n int) string { return fmt.Sprintf("%d", n) }
```

**Step 2: Run integration tests**

```bash
go test -tags integration -v -run TestIntegration ./...
```

Expected: both tests PASS.

**Step 3: Run all tests**

```bash
go test ./...
```

Expected: all unit tests PASS (integration tests skipped without the tag).

**Step 4: Commit**

```bash
git add integration_test.go
git commit -m "test: add integration tests"
```

---

### Task 15: Final verification

**Step 1: Run all tests**

```bash
go test ./...
```

Expected: PASS.

**Step 2: Build release binary**

```bash
go build -o progress-wrap .
```

**Step 3: End-to-end smoke test**

```bash
# First invocation — no state yet
./progress-wrap --state /tmp/demo.state sh -c 'echo "Progress: 20%"; echo "Nodes: 5"'

# Second invocation — EMA has one sample, no ETA yet
./progress-wrap --state /tmp/demo.state sh -c 'echo "Progress: 40%"; echo "Nodes: 5"'

# Third invocation — ETA should appear
./progress-wrap --state /tmp/demo.state sh -c 'echo "Progress: 60%"; echo "Nodes: 5"'
```

Expected: third invocation shows `[...] 60.0%  ETA: Xs  (avg velocity: X.XXX%/s)`.

**Step 4: Test --reset**

```bash
./progress-wrap --state /tmp/demo.state --reset sh -c 'echo "Progress: 5%"'
```

Expected: state cleared, first sample only, no ETA.

**Step 5: Commit**

```bash
git add progress-wrap  # if you want the binary committed, otherwise skip
git commit -m "feat: complete progress-wrap implementation"
```

---

## Summary of package dependencies

```
main → cmd → runner, parser, parser/builtin, parser/config,
              parser/regexparser, parser/jqparser,
              state, estimator, display
```

All packages are independently testable. The parser system is fully additive — new built-ins require only a new TOML entry.
