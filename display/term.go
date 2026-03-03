package display

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

// TermWidth returns the current terminal width using a chain of fallbacks:
//  1. stderr / stdout / stdin — whichever fd is still a TTY (covers the common
//     case where stdout is piped but stderr is not).
//  2. /dev/tty — the process's controlling terminal, works even when all three
//     standard fds are redirected.
//  3. $COLUMNS — set by bash/zsh in interactive shells; survives pipelines.
//  4. 80 — safe default for fully-detached processes (cron, systemd, CI).
func TermWidth() int {
	for _, f := range []*os.File{os.Stderr, os.Stdout, os.Stdin} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if w, _, err := term.GetSize(int(tty.Fd())); err == nil && w > 0 {
			return w
		}
	}
	if v := os.Getenv("COLUMNS"); v != "" {
		if w, err := strconv.Atoi(v); err == nil && w > 0 {
			return w
		}
	}
	return 80
}
