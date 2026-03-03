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
