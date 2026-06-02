# IterForge (Go)

A Go-native harness for running an autonomous improvement loop with Claude Code or another coding agent. (Go module: `iterforge`.)

The pattern is:

```text
bounded mutable surface + frozen evaluator + cheap trials + clear score + hard gates = autonomous improvement loop
```

Claude Code should normally edit only `candidates/` and `logs/agent_journal.md`. The evaluator, runner, golden set, policy, and summarizer are frozen unless a human explicitly authorizes changes.

Beyond the basic loop, the harness provides:

- **Multi-signal evaluation** — evaluators report a primary score plus a `metrics` map and named `gates`; a failed gate rejects a run regardless of score ([evaluator-contract](docs/evaluator-contract.md)).
- **Guardrails** — frozen-path changes, results-log tampering, golden-set hardcoding, and evaluator timeouts each produce a distinct rejection decision ([guardrails](docs/guardrails.md)).
- **External evaluators** — point `commands.evaluate` at any command emitting standard JSON ([evaluator-contract](docs/evaluator-contract.md)).
- **Scaffolding** — `iterforge init -template <name>` generates a working project ([workflow-templates](docs/workflow-templates.md)).
- **Reporting** — `iterforge summarize` (filters + decision histogram + `-json`) and `iterforge compare` (keep/promote recommendation).

See [docs/cli.md](docs/cli.md) for the full command reference.

## Quick start

```bash
make test
make baseline
make summarize
```

Equivalent raw Go commands (all subcommands are under the single `iterforge` binary):

```bash
go test ./...
go run ./cmd/iterforge run -note "baseline"
go run ./cmd/iterforge summarize
go run ./cmd/iterforge validate-policy
go run ./cmd/iterforge compare
go run ./cmd/iterforge init <name>
```

Then open this directory in Claude Code and prompt:

```text
Read program.md and policy.yaml. Inspect candidates/candidate.go, evals/evaluator.go, and the latest results. Start with one conservative experiment. Do not edit frozen files. Run `make check` and `make run NOTE="<hypothesis>"` after each change. Keep only changes that pass hard gates and improve the primary score.
```

## File layout

```text
Makefile                   Convenient local commands for test/run/summarize/check
program.md                 Claude Code control file
policy.yaml                Experiment policy and gates
cmd/iterforge/             Single CLI binary (run/summarize/validate-policy/compare/init)
internal/cli/              Subcommand implementations
candidates/                Mutable candidate implementation
configs/candidate.yaml     Optional mutable config if you add one
evals/                     Frozen evaluator and golden set
internal/policy/           Frozen lightweight YAML-ish policy parser
logs/results.jsonl         Append-only experiment results
logs/agent_journal.md      Agent notes and hypotheses
docs/                      cli, policy-schema, evaluator-contract, guardrails,
                           result-log, compare, init, workflow-templates,
                           adaptation-guide
```

## Current toy task

The included example optimizes a small text-normalization candidate. The evaluator scores exact output matches across a frozen golden set.

Replace the toy task with your real candidate/evaluator pair:

| Use case | Mutable surface | Frozen evaluator |
|---|---|---|
| RAG optimization | retrieval/chunking logic | recall@k, MRR, citation precision |
| Prompt optimization | prompt/template builder | correctness, faithfulness, latency, cost |
| Extraction | parser/extractor | exact match, F1, schema validity |
| Code repair | target package | unit tests, benchmarks, mutation score |
| Ranking/search | scoring function | NDCG, MRR, calibrated judgments |

## Invariants

1. The agent may edit `candidates/*`.
2. The agent must not edit `evals/*`, `cmd/*`, `internal/*`, `policy.yaml`, or the golden set.
3. Promotion requires all hard gates to pass — the primary score gate **and** every named evaluator gate.
4. Promotion requires the primary score to exceed the incumbent by at least `minimum_delta`.
5. Every run must append structured results to `logs/results.jsonl` (the runner is the only writer; out-of-band edits are detected and rejected).
6. Guardrail violations (frozen-path change, log tamper, golden-set hardcoding, evaluator timeout) override the score and are recorded as the run's `decision`.
