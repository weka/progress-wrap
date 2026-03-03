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
	s0 := []parser.Entry{{CommandRegex: "^nope$", Parser: &mockParser{0.1}}}
	s1 := []parser.Entry{{CommandRegex: "^weka$", Parser: &mockParser{0.7}}}
	got := parser.Select("weka", s0, s1)
	assert.NotNil(t, got)
	prog, _, _ := got.Parse(nil)
	assert.Equal(t, 0.7, prog)
}
