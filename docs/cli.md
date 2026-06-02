# CLI

IterForge ships as a single binary, `iterforge`, with subcommands. During
development run it via `go run ./cmd/iterforge <command>`; the Makefile wraps the
common ones.

```
iterforge <command> [flags]

commands:
  init             scaffold a new project (-template, -dir)
  validate-policy  validate policy.yaml (-policy)
  run              run one scored experiment (-note, -policy)
  summarize        summarize results (-last, -since, -failed-only, -json)
  compare          compare two runs (-baseline, -candidate)
```

## Commands

| Command | Purpose | Key flags | Exit codes |
|---------|---------|-----------|------------|
| `init <name>` | Scaffold a project from a template | `-template`, `-dir` | 0 ok, 1 error |
| `validate-policy` | Validate `policy.yaml` | `-policy` | 0 valid, 1 invalid |
| `run` | Run one scored, gated experiment | `-note`, `-policy` | 0 pass, 2 gate/guardrail fail, 1 error |
| `summarize` | Report over the results log | `-last`, `-since`, `-failed-only`, `-json` | 0 ok, 1 error |
| `compare` | Diff two runs, recommend keep/promote | `-baseline`, `-candidate` | 0 promote, 3 otherwise, 1 error |

## Makefile shortcuts

```bash
make run NOTE="hypothesis"   # iterforge run -note ...
make summarize               # iterforge summarize
make compare                 # iterforge compare (BASELINE=/CANDIDATE= optional)
make validate                # iterforge validate-policy
make new NAME=myproj         # iterforge init myproj
```

## Architecture

`cmd/iterforge/main.go` is a thin dispatcher; each subcommand's logic lives in
`internal/cli` as `func Name(args []string) int`, returning the process exit
code. This keeps the binary small and the commands unit-addressable.

See [init.md](init.md), [result-log.md](result-log.md),
[compare.md](compare.md), [policy-schema.md](policy-schema.md), and
[evaluator-contract.md](evaluator-contract.md) for per-area detail.
