package jqparser

import (
	"encoding/json"
	"fmt"
	"strings"

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
		// gojq raises an error when operating on null (e.g. null * 100).
		// Treat null-operand errors as not-found rather than hard errors.
		if strings.Contains(err.Error(), "null") {
			return 0, false, nil
		}
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
