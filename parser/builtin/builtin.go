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

// LoadFile parses an external TOML byte slice with the same schema as builtin_parsers.toml.
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
		entry, err := parser.NewEntry(ec.CommandRegex, p)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
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
