# Policy Schema

`policy.yaml` defines the bounded loop: what is mutable, what is frozen, the
primary metric, and the hard gates. The harness parses a small YAML subset (no
external dependencies) via `internal/policy`.

## Fields

| Field | Type | Required | Default | Notes |
|-------|------|----------|---------|-------|
| `primary_metric` | string | yes | `accuracy` | Name of the score that drives keep/revert. Must be non-empty. |
| `score_direction` | string | yes | `higher_is_better` | One of `higher_is_better` or `lower_is_better`. |
| `minimum_delta` | float | no | `0.001` | Smallest score change treated as progress. Must be `>= 0`. |
| `hard_gates.min_accuracy` | float | no | `0.80` | Gate floor. |
| `hard_gates.max_latency_ms` | float | no | `100` | Gate ceiling. |
| `hard_gates.max_failures` | int | no | `2` | Gate ceiling. |
| `experiment.timeout_seconds` | int | no | `30` | Per-experiment timeout. |
| `experiment.results_path` | string | yes | `logs/results.jsonl` | Append-only result log. Must be non-empty. |
| `experiment.golden_set_path` | string | yes | `evals/golden_set.jsonl` | Evaluator input. Must be non-empty. |
| `experiment.mutable_paths` | list | yes | — | At least one editable path. |
| `experiment.frozen_paths` | list | yes | — | At least one frozen path. |

## Example

```yaml
primary_metric: accuracy
score_direction: higher_is_better
minimum_delta: 0.001
hard_gates:
  min_accuracy: 0.80
  max_latency_ms: 100
  max_failures: 2
experiment:
  timeout_seconds: 30
  results_path: logs/results.jsonl
  golden_set_path: evals/golden_set.jsonl
  mutable_paths:
    - candidates/
  frozen_paths:
    - evals/
    - cmd/
    - internal/
    - policy.yaml
```

## Validation

Validate the policy before running experiments:

```bash
go run ./cmd/iterforge validate-policy   # or: make validate
```

`make check` runs validation first, then formatting, `go vet`, and tests.

Validation fails fast with one actionable message per problem, e.g.:

```
error: invalid policy:
  - primary_metric must not be empty
  - score_direction must be "higher_is_better" or "lower_is_better", got "bigger"
  - at least one frozen path is required (experiment.frozen_paths)
```

A non-numeric value (e.g. `minimum_delta: abc`) is reported by the loader
rather than crashing the harness.
