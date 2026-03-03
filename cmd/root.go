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
	Use:   "progress-wrap",
	Short: "Wrap a command and show a progress bar with ETA",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRoot,
}

// now returns the current UTC time, or a time parsed from PROGRESS_WRAP_NOW
// if set (used for deterministic testing).
func now() time.Time {
	if v := os.Getenv("PROGRESS_WRAP_NOW"); v != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, v); err == nil {
				return t
			}
		}
	}
	return time.Now().UTC()
}

// Execute runs the root command.
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
	rootCmd.Flags().SetInterspersed(false)
}

func runRoot(cmd *cobra.Command, args []string) error {
	cmdStr := strings.Join(args, " ")

	// Handle --reset
	if flagReset {
		if err := state.Reset(flagState); err != nil {
			return err
		}
	}

	// Load existing state
	s, err := state.Read(flagState)
	if err != nil {
		return err
	}

	// Build parser sources: CLI inline > config file > built-ins
	var sources [][]parser.Entry

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

	if flagConfig != "" {
		entries, err := config.LoadFile(flagConfig)
		if err != nil {
			return err
		}
		sources = append(sources, entries)
	}

	builtins, err := builtin.Load()
	if err != nil {
		return err
	}
	sources = append(sources, builtins)

	selectedParser := parser.Select(cmdStr, sources...)

	// Run the wrapped command
	stdout, exitCode, runErr := runner.Run(args[0], args[1:])
	if runErr != nil {
		return fmt.Errorf("run command: %w", runErr)
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

	if found {
		now := now()

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

		// Build and update estimator before appending the current sample,
		// so that s.Samples[last] is the previous observation and
		// est.Update(progress, now) computes a real time delta.
		var est estimator.Estimator
		switch flagEstimator {
		case "kalman":
			est = estimator.NewKalman()
		default:
			if len(s.Samples) > 0 && s.Estimator.EMAVelocity > 0 {
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

		s.Samples = append(s.Samples, state.Sample{Time: now, Progress: progress})

		// Save state
		if writeErr := state.Write(flagState, s); writeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not write state: %v\n", writeErr)
		}

		// Render and print progress bar
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
