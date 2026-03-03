# progress-wrap

## Dev Workflow

Use [Task](https://taskfile.dev) for all development operations. Install with:
```bash
go install github.com/go-task/task/v3/cmd/task@latest
# or: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d
```

### Common commands

| Command | What it does |
|---|---|
| `task` | Full check: tidy → vet → lint → test → build |
| `task build` | Build the `progress-wrap` binary |
| `task test` | Run unit tests |
| `task test:verbose` | Unit tests with -v output |
| `task test:integration` | Build binary + run integration tests |
| `task test:all` | Unit tests + integration tests |
| `task vet` | `go vet ./...` |
| `task lint` | `golangci-lint run ./...` |
| `task tidy` | `go mod tidy` |
| `task clean` | Remove built binary |

### Before committing

Always run `task` (the default) to ensure tidy, vet, lint, tests, and build all pass.

### Linter

`task lint` runs `go vet` (always available) plus `staticcheck` if version-compatible.
`task lint:strict` runs `staticcheck` and fails if it can't run.

`staticcheck` must be installed and built with a matching Go version:
```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Integration tests

Integration tests are tagged with `//go:build integration` and excluded from normal `task test` runs. Run them explicitly with `task test:integration`. They build the binary from source and run it as a subprocess.

## Project layout

```
cmd/             # cobra CLI wiring
runner/          # exec + capture stdout
parser/
  parser.go      # Parser interface + Select()
  regexparser/   # regex-based parser
  jqparser/      # jq-expression parser (supports arithmetic)
  builtin/       # embedded builtin_parsers.toml + loader
  config/        # user TOML config loader (same schema as builtin)
state/           # JSON state file (RFC3339Nano timestamps, atomic writes)
estimator/       # EMA + Kalman stub (Estimator interface)
display/         # progress bar rendering + terminal width detection
```

## Parser config format

`--config myfile.toml` — TOML file, same schema as `parser/builtin/builtin_parsers.toml`:

```toml
[[parsers]]
command_regex = '^myapp status'
type          = "regex"
pattern       = 'Progress:\s*(\d+(?:\.\d+)?)\s*%'
group         = 1   # default: 1

[[parsers]]
command_regex = '^myapp status -J'
type          = "jq"
expression    = ".status.progress * 100"
```

Entries without `command_regex` match any command (fallback).
