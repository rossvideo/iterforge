# Guardrails

Guardrails enforce policy boundaries the score cannot buy its way past. A run
that improves accuracy while mutating the evaluator is not progress — it is a
broken experiment.

## Frozen-path guardrail

Before recording a result, the runner lists changed files (`git status`) and
checks them against `experiment.frozen_paths` in `policy.yaml`. Any match makes
the run fail:

- `hard_gates.passed` becomes `false`;
- each offending path is added to `hard_gates.failures` as `frozen path changed: <path>`;
- `decision` becomes `rejected_frozen_change` (this overrides `rejected_hard_gate`);
- the runner exits non-zero.

`eval_passed` still records whether the evaluator gates alone passed, so the log
distinguishes "good eval, but you touched frozen files" from "eval failed".

## Pattern matching

`internal/guardrails` supports a gitignore-style subset:

| Pattern | Matches |
|---------|---------|
| `evals/`, `internal/` | anything under that directory |
| `cmd/**` | anything under `cmd` |
| `policy.yaml`, `go.sum` | that exact file (not `policy.yaml.bak`) |
| `testdata/*.json` | `path.Match` against the full path and the basename |

Bare names with no slash or glob match the exact path or treat it as a
directory prefix (`internal` matches `internal/policy/policy.go`).

## Golden-set hardcoding guardrail

The runner reads non-test `.go` files under `experiment.mutable_paths` and
scans them for golden-set expected values appearing as exact string literals
(double-quoted or backtick-quoted). A candidate that embeds the answers is
memorizing the eval, not solving the task. On a hit:

- `hard_gates.passed` becomes `false`;
- each hit is added to failures as `golden-set value hardcoded in <file>: "<value>"`;
- `decision` becomes `rejected_golden_hardcoding` (lower precedence than
  `rejected_frozen_change`, higher than `rejected_hard_gate`);
- the runner exits non-zero.

Values shorter than 4 characters are ignored to avoid noisy false positives.

## Results-log tamper guardrail

The results log is append-only and the runner is its only sanctioned writer.
After each append the runner records a SHA-256 of the whole log to a sidecar
(`logs/results.jsonl.sha256`). On the next run it recomputes the hash; a
mismatch means the log was edited out of band:

- the rejected run is still recorded (audit trail);
- `decision` becomes `rejected_results_tampered` (precedence below
  `rejected_frozen_change`, above `rejected_golden_hardcoding`);
- the runner exits non-zero.

First run (no sidecar yet) passes and writes the initial fingerprint.

## Decision precedence

`rejected_frozen_change` > `rejected_results_tampered` >
`rejected_golden_hardcoding` > `rejected_hard_gate` > `candidate_pending_review`.

## Limitations

- Detection relies on git. Outside a git repository, `ChangedFiles` returns
  nothing, so the guardrail passes vacuously. Run experiments in a git repo to
  get enforcement.
- Globbing is a practical subset, not full gitignore semantics (no `!`
  negation, no `**` in the middle of a pattern).
- It detects *that* a frozen file changed, not *what* changed. Pair it with code
  review for intent.

The tamper guardrail's sidecar is itself unsigned: an editor who recomputes the
sidecar hash evades detection. Defeating that needs a hash chain or signature.

## Future guardrails (P7 backlog)

Candidate imports forbidden packages, network use when disabled.
