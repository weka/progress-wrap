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
