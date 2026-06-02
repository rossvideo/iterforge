# Evaluator Contract (template harness)

In a scaffolded project, the evaluator (`evals.Evaluate`) returns a `Result` and
the shared runner (`iterforge run`) decides pass/fail and records the run. The
evaluator owns *what* to measure; the runner owns *how* a run is gated and
logged.

## Result

```go
type Result struct {
    Total     int                // examples or fields evaluated
    Passed    int
    Failed    int
    Accuracy  float64            // the primary score
    LatencyMS float64
    Metrics   map[string]float64 // named signals, surfaced by the summarizer
    Gates     map[string]bool    // named hard gates, enforced by the runner
}
```

- **Accuracy** is the primary score the runner gates against `min_accuracy`.
- **Metrics** are additional named signals (e.g. `email_accuracy`,
  `format_ok_rate`). They do not gate; the summarizer prints them per run.
- **Gates** are named pass/fail checks the runner treats as hard gates *in
  addition to* the primary score. Any `false` gate fails the run.

## Runner gating rule

A run passes only if **both** hold:

1. `Accuracy >= min_accuracy` (from `policy.yaml`);
2. every entry in `Gates` is `true`.

Failures are recorded in the result's `failures` array (`accuracy ... < ...`
and `gate failed: <name>`), and the runner exits non-zero. Gate names are
sorted so output is deterministic.

## Per-template signals

| Template | Metrics | Gates |
|----------|---------|-------|
| text-normalization | `accuracy` | — |
| extraction | `field_accuracy`, `email_accuracy`, `phone_accuracy` | — |
| prompt-optimization | `accuracy`, `format_ok_rate` | `format_valid` |

`format_valid` fails if any model output is empty, multi-line, or has
surrounding whitespace — a format-regression guard that catches a chatty model
even when its answer text would otherwise match.

## Main tool

The IterForge tool itself (`internal/cli`, `internal/evaluator`, `internal/resultlog`)
uses the same model:

- `evals.Result` carries `Metrics` and `Gates` like the templates.
- `internal/evaluator` defines the canonical `Output{score, passed, metrics,
  gates, failure_reason}` and `Parse([]byte)` for reading **external** evaluator
  JSON (a "score" field is required). `Output.GateFailures()` produces the
  sorted `gate failed: <name>` messages the runner folds into the result.
- `resultlog.Record` persists `metrics` and `gates`, and `iterforge summarize`
  prints the metric map per run.

The runner enforces evaluator gates with the same precedence as the integrity
guardrails: frozen-change > results-tamper > golden-hardcoding > hard gate
(score/latency/failures **and** evaluator gates).

### External evaluator JSON

`Parse` accepts:

```json
{
  "score": 0.875,
  "passed": true,
  "metrics": {"accuracy": 0.875, "latency_ms_p95": 12.4},
  "gates": {"schema_valid": true, "within_budget": true},
  "failure_reason": ""
}
```

This is the seam for swapping the in-process toy evaluator for a real one
without changing the runner.

### Wiring an external evaluator

Set `commands.evaluate` in `policy.yaml`:

```yaml
commands:
  evaluate: go run ./cmd/myeval
```

When set, `iterforge run` runs the command (`sh -c`), passes the golden-set path in
`IF_GOLDEN_SET`, and `evaluator.Parse`s its stdout — using that `score`,
`metrics`, and `gates` instead of the in-process evaluator. The score gate and
every reported gate are enforced as usual. When `commands.evaluate` is unset,
the in-process evaluator is used (unchanged default).

`experiment.timeout_seconds` bounds the external command. The command runs in
its own process group; on deadline the runner kills the whole group (so child
processes like `sleep` can't keep the stdout pipe open and block past the
deadline) and records the run with `decision = rejected_evaluator_timeout` (and
an `evaluator timed out after Ns` failure) rather than hanging or aborting
without an audit entry. A non-zero exit or unparseable output is a fatal error
(no record), since it indicates a broken evaluator rather than a rejected
candidate.

## Adding a signal

In your evaluator, populate `Metrics` for anything you want visible, and add a
`Gates` entry for anything that must hold regardless of score (schema validity,
required fields present, no unsafe output, latency/cost budgets). No runner
change is needed — the runner enforces whatever gates the evaluator reports.
