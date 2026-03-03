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

// Parse scans output, returns the float64 from the first match
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
