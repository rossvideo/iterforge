# Result Log Contract

`logs/results.jsonl` is the append-only audit trail. One JSON object per line,
one line per experiment. The schema lives in `internal/resultlog` (`Record`)
and is shared by the runner (appends) and the summarizer (reads) so the two
never drift.

The log is **frozen** to manual editing. The only sanctioned writer is the
runner (`iterforge run`). Hand-editing it breaks the audit trail and the
`no_manual_results_log_edits` hard gate.

## Schema

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | Sortable id: `YYYYMMDDThhmmssZ` plus short git SHA when available. |
| `timestamp_utc` | string | RFC3339 UTC. |
| `note` | string | The hypothesis passed via `-note`. |
| `git_sha` | string | Commit SHA, omitted outside a git repo. |
| `dirty` | bool | Working tree had uncommitted changes. |
| `primary_metric` | string | From policy. |
| `score` | float | The primary score (currently accuracy). |
| `score_direction` | string | `higher_is_better` / `lower_is_better`, from policy. |
| `check_passed` | bool | Whether `make check` passed (assumed true; check runs around the experiment). |
| `eval_passed` | bool | Whether all hard gates passed. |
| `hard_gates` | object | `{passed, failures, min_accuracy, max_latency_ms, max_failures}`. |
| `duration_ms` | int | Evaluator wall time. |
| `decision` | string | `candidate_pending_review` or `rejected_hard_gate`. |
| `failure_reason` | string | `; `-joined gate failures. |
| `eval` | object | Full evaluator `Result` (totals, accuracy, latency, case results). |

## Reading guarantees

`resultlog.Read`:

- a missing file returns no records and no error;
- blank lines are skipped;
- a malformed line is reported with its line number;
- old records missing newer fields parse fine (additive schema).

`resultlog.Best(records, direction)` selects the best record per score
direction; the summarizer uses it for the "Best score" line and top-runs order.

## Summarizing

`iterforge summarize` reports counts, best/latest score, a decision histogram, top
runs, and failure reasons. Flags scope and format the report:

| Flag | Effect |
|------|--------|
| `-results <path>` | log to read (default `logs/results.jsonl`) |
| `-last N` | only the most recent N runs |
| `-since <RFC3339>` | only runs at or after a timestamp |
| `-failed-only` | only runs that did not pass gates |
| `-json` | emit a machine-readable summary instead of text |

Filters compose in order: time (`-since`) → recency (`-last`) → failures
(`-failed-only`). The `-json` form (records, passed/failed, direction,
latest, best, decisions) is suited to dashboards and CI.

```bash
go run ./cmd/iterforge summarize -since 2026-06-02T00:00:00Z -failed-only
go run ./cmd/iterforge summarize -last 20 -json
```

## Tamper detection

After each append the runner writes `logs/results.jsonl.sha256`, a SHA-256 of
the whole log. The next run recomputes it; a mismatch is treated as an
out-of-band edit (`decision=rejected_results_tampered`). See
[guardrails.md](guardrails.md). The sidecar is harness-managed — do not edit it.
