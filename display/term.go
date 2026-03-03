package display

import (
	"os"

	"golang.org/x/term"
)

// TermWidth returns the current terminal width, falling back to 80.
// It probes stderr first (which remains a TTY even when stdout is piped),
// then stdout, then stdin.
func TermWidth() int {
	for _, f := range []*os.File{os.Stderr, os.Stdout, os.Stdin} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return 80
}
