# progress-wrap Design

_Date: 2026-03-03_

## Overview

`progress-wrap` is a CLI tool that wraps an arbitrary shell command, parses a
progress percentage from its output, maintains state across periodic invocations,
and appends a progress bar with an ETA estimate to stdout.

Typical usage:

```
progress-wrap --state /tmp/my.state [--config parser.toml] [--reset] \
              [--estimator ema|kalman] \
              [--parse-regex PATTERN] [--parse-jq EXPRESSION] \
              COMMAND [ARGS...]
```

## CLI Flags

| Flag | Description |
|---|---|
| `--state FILE` | Path to JSON state file (required) |
| `--config FILE` | Path to TOML parser config (optional) |
| `--reset` | Wipe state before running |
| `--estimator ema\|kalman` | Estimator to use (default: `ema`) |
| `--parse-regex PATTERN` | Ad-hoc regex parser (highest priority) |
| `--parse-jq EXPRESSION` | Ad-hoc jq parser (highest priority) |

## Architecture

Language: **Go**. Modular package layout:

```
progress-bar-wrapper/
├── main.go
├── cmd/
│   └── root.go                 # cobra CLI wiring
├── runner/
│   └── runner.go               # exec command, capture stdout+stderr
├── parser/
│   ├── parser.go               # Parser interface + priority selector
│   ├── builtin/
│   │   ├── builtin_parsers.toml  # embedded built-in parser entries
│   │   └── builtin.go            # //go:embed loader
│   └── config/
│       └── config.go           # load user TOML config, same schema
├── state/
│   └── state.go                # read/write JSON state file
├── estimator/
│   ├── estimator.go            # Estimator interface
│   ├── ema.go                  # EMA velocity estimator
│   └── kalman.go               # Kalman filter (stub → full)
├── display/
│   └── display.go              # format progress bar + ETA line
└── go.mod
```

## Parser System

### Interface

```go
type Parser interface {
    Parse(output []byte) (progress float64, found bool, err error)
}
```

### Selection priority (first match wins)

1. `--parse-regex` / `--parse-jq` CLI flag (ad-hoc, no config needed)
2. Entries from `--config FILE`, matched by `command_regex`
3. Built-in entries (embedded in binary), matched by `command_regex`

### Matching

Each parser entry carries a `command_regex` field matched against the full
command string (e.g. `"weka status -J"`). Entries are tested in declaration
order; the first match is used. A missing `command_regex` acts as a wildcard
fallback within that source.

### TOML config schema

```toml
[[parsers]]
command_regex = '^weka status rebuild'
type          = "jq"
expression    = ".rebuilt_blocks / .total_blocks * 100"

[[parsers]]
command_regex = '^weka status -J'
type          = "jq"
expression    = ".result.status.progress * 100"

[[parsers]]
command_regex = '^weka status'
type          = "regex"
pattern       = 'Progress:\s*(\d+(?:\.\d+)?)%'
group         = 1          # default: 1

# Entry without command_regex = fallback for this config source
[[parsers]]
type    = "regex"
pattern = '(\d+(?:\.\d+)?)\s*%'
```

### Parser types

- **`regex`**: applies `pattern` to each line of output, returns the float from
  capture group `group` (default 1).
- **`jq`**: parses output as JSON, evaluates `expression` via
  [`gojq`](https://github.com/itchyny/gojq). Supports arithmetic
  (e.g. `.done / .total * 100`).

### Built-ins

Stored in `parser/builtin/builtin_parsers.toml`, embedded at compile time via
`//go:embed`. Ships with parsers for `weka status` variants. Adding a new
built-in requires only a new `[[parsers]]` entry in that file.

## State File

Path specified by `--state`. Format: JSON with **RFC3339Nano** timestamps
(nanosecond precision — important when the command is invoked every second).

```json
{
  "command":    "weka status",
  "started_at": "2026-03-03T10:00:00.000000000Z",
  "updated_at": "2026-03-03T10:05:00.123456789Z",
  "samples": [
    {"time": "2026-03-03T10:00:00.000000000Z", "progress": 0.10},
    {"time": "2026-03-03T10:02:00.500000000Z", "progress": 0.20},
    {"time": "2026-03-03T10:05:00.123456789Z", "progress": 0.45}
  ],
  "estimator": {
    "type":         "ema",
    "ema_velocity": 0.0025
  }
}
```

- Sample history is capped at **500 entries** (oldest dropped) to prevent
  unbounded growth.
- Estimator state is persisted so velocity survives across invocations.

### Reset logic

| Trigger | Action |
|---|---|
| `--reset` flag | Wipe state file before running |
| `new_progress < last_progress - 5%` | Auto-wipe (new run detected) |

The 5% backward-progress threshold is configurable.

## Estimator

### Interface

```go
type Estimator interface {
    Update(progress float64, t time.Time)
    ETA() (eta time.Time, ok bool)  // ok=false until enough data
    State() EstimatorState          // serializable state
}
```

### EMA (default)

Exponential moving average over instantaneous velocity:

```
velocity_i = α × (Δprogress / Δtime) + (1-α) × velocity_{i-1}
ETA        = now + (1.0 - current_progress) / velocity
```

- Default α = 0.2 (configurable via `--ema-alpha`)
- Requires ≥ 2 samples before emitting an ETA

### Kalman filter (next phase)

State vector `[progress, velocity]`, measurement `progress` at time `t`.
Models progress as linear with process noise — naturally handles jitter and
outliers. Bootstrapped from existing sample history on first use.

Enabled via `--estimator kalman`. Becomes the default once implemented.

## Output

Command stdout and stderr are passed through unchanged. After the command
completes, `progress-wrap` appends a single summary line to **stdout**:

```
[====================          ] 45.3%  ETA: 14m32s  (avg velocity: 0.50%/s)
```

- Progress bar width adapts to terminal width (fallback: 80 columns)
- ETA formatted as `Xh Ym Zs`
- Shows `--` when fewer than 2 samples available
- Shows `overdue` when ETA is in the past (stalled / estimator drift)
- Velocity shown in `%/s` for transparency

## Error Handling

- If the wrapped command fails (non-zero exit), `progress-wrap` exits with the
  same code; state is still updated with whatever progress was parsed.
- If no parser matches, a warning is printed to stderr and state is unchanged.
- If the state file is corrupt/unreadable, treat as missing (start fresh);
  print a warning to stderr.
- State file writes are atomic (write to `.tmp` then rename).

## Testing Strategy

- Unit tests for each parser type (regex + jq) with sample outputs
- Unit tests for EMA estimator (convergence, edge cases)
- Unit tests for state read/write/reset logic
- Integration test: wrap a script that prints incrementing percentages,
  verify ETA converges
