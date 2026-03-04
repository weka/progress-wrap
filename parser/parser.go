package parser

import (
	"fmt"
	"regexp"
)

// Parser extracts a progress value in [0,1] from command output.
type Parser interface {
	Parse(output []byte) (progress float64, found bool, err error)
}

// Entry pairs an optional command regex with a Parser and an optional
// estimator hint. If CommandRegex is empty, the entry matches any command.
// If Estimator is empty, the global --estimator flag is used.
type Entry struct {
	CommandRegex string
	Parser       Parser
	Estimator    string // optional: overrides the global --estimator flag
	compiled     *regexp.Regexp
}

// NewEntry creates an Entry with CommandRegex pre-compiled.
// Returns an error if CommandRegex is not a valid regular expression.
func NewEntry(commandRegex string, p Parser) (Entry, error) {
	e := Entry{CommandRegex: commandRegex, Parser: p}
	if commandRegex != "" {
		re, err := regexp.Compile(commandRegex)
		if err != nil {
			return Entry{}, fmt.Errorf("invalid command_regex %q: %w", commandRegex, err)
		}
		e.compiled = re
	}
	return e, nil
}

// matches reports whether cmdStr matches the entry's CommandRegex.
// Entries created via NewEntry have the regex pre-compiled; others compile
// lazily with a safe fallback (an invalid regex never matches).
func (e *Entry) matches(cmdStr string) bool {
	if e.CommandRegex == "" {
		return true
	}
	if e.compiled == nil {
		re, err := regexp.Compile(e.CommandRegex)
		if err != nil {
			return false
		}
		e.compiled = re
	}
	return e.compiled.MatchString(cmdStr)
}

// Select scans sources in order and returns a pointer to the first Entry
// whose CommandRegex matches cmdStr. Returns nil if no match is found.
func Select(cmdStr string, sources ...[]Entry) *Entry {
	for _, entries := range sources {
		for i := range entries {
			if entries[i].matches(cmdStr) {
				return &entries[i]
			}
		}
	}
	return nil
}
