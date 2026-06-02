# ITERFORGE_GO_EVOLUTION.md

Claude Code instructions for continuing development of the Go implementation of IterForge.

This file is intended to be placed in the root of the Go repository and used as the primary development guide for Claude Code.

---

## 1. Mission

You are continuing development of **IterForge**, a Go-based autonomous improvement harness inspired by Karpathy's `autoresearch`.

Preserve the key insight:

> Program the research loop, not every experiment manually.

IterForge should help engineers define a bounded optimization workflow once, then allow an agent to safely iterate inside that loop.

The core abstraction is:

```text
bounded mutable surface
+ trusted evaluator
+ cheap repeated trials
+ clear score
+ hard gates
+ audit trail
+ keep/revert semantics
= autonomous improvement loop
```

The Go implementation should make this loop reusable for production AI and software workflows.

---

## 2. Product Positioning

IterForge is not a general-purpose autonomous agent framework.

It is an **experiment-loop harness**.

It should be useful for:

- RAG retrieval/chunking optimization;
- prompt-template optimization;
- data extraction pipelines;
- code repair loops;
- ranking/search tuning;
- normalization and transformation pipelines;
- evaluator development;
- model-serving configuration tuning.

The value is not that the agent is clever.

The value is that the loop is:

- bounded;
- reproducible;
- measurable;
- auditable;
- safe to run repeatedly.

---

## 3. Current Expected Go Repository Shape

The repository may start from this structure:

```text
iterforge/
├── README.md
├── ITERFORGE.md
├── EXAMPLE.md
├── ITERFORGE_GO_EVOLUTION.md
├── Makefile
├── go.mod
├── policy.yaml
├── candidates/
│   ├── candidate.go
│   └── candidate_test.go
├── evals/
│   ├── evaluator.go
│   ├── evaluator_test.go
│   └── golden_set.jsonl
├── cmd/
│   ├── runexp/
│   │   └── main.go
│   └── summarize/
│       └── main.go
├── internal/
│   └── policy/
│       └── policy.go
├── logs/
│   ├── agent_journal.md
│   └── results.jsonl
└── docs/
    └── adaptation-guide.md
```

The exact structure may evolve, but the separation of concerns must remain clear.

---

## 4. Architectural Principles

### 4.1 Keep the Loop Explicit

Every experiment should have:

```text
hypothesis
candidate diff
check command
evaluation command
score
pass/fail result
keep/revert decision
journal entry
machine-readable result log
```

If any of these are missing, the experiment loop is incomplete.

### 4.2 Separate Candidate Logic from Evaluator Logic

Candidate logic is the mutable surface.

Evaluator logic is trusted infrastructure.

Do not blur this boundary.

The agent may improve candidate code. It must not change evaluator code during an optimization run.

### 4.3 Prefer Contracts Over Prompts

Do not rely only on natural-language instructions.

Encode critical behavior in:

- policy schema;
- validation;
- CLI commands;
- Makefile targets;
- result logs;
- tests.

### 4.4 Make the Safe Path the Easy Path

The default commands should guide the agent toward correct behavior:

```bash
make check
make run NOTE="<hypothesis>"
make summarize
```

These commands should be stable, documented, and low-friction.

### 4.5 Evidence Beats Explanation

Agent explanations are useful, but they are not evidence.

Evidence comes from:

- evaluator output;
- result logs;
- tests;
- git diff;
- reproducible commands;
- policy validation.

---

## 5. Editable and Frozen Surfaces

When IterForge is being used to optimize a candidate workflow, the distinction should be enforced.

### Mutable Surface

Default mutable paths:

```text
candidates/**
logs/agent_journal.md
```

Possible future mutable paths:

```text
prompts/candidate/**
configs/candidate/**
experiments/current/**
```

### Frozen Surface

Default frozen paths:

```text
evals/**
cmd/**
internal/**
policy.yaml
Makefile
go.mod
go.sum
logs/results.jsonl
golden/**
testdata/eval/**
```

### Important

When developing IterForge itself, you may modify implementation files as needed.

When running an IterForge experiment, the agent must respect the policy-defined mutable/frozen surfaces.

Do not confuse **tool development mode** with **experiment execution mode**.

---

## 6. Go Development Standards

Use idiomatic, production-quality Go.

Prefer:

- small packages;
- clear interfaces;
- explicit errors;
- table-driven tests;
- deterministic behavior;
- stable CLI output;
- no hidden global state;
- standard library first;
- minimal dependencies;
- JSON/JSONL for machine-readable logs;
- YAML for human-authored policy/config;
- non-zero exit codes on failure.

Avoid:

- unnecessary frameworks;
- reflection-heavy designs;
- global mutable state;
- implicit filesystem conventions without documentation;
- swallowing errors;
- writing logs in multiple incompatible formats;
- magic strings scattered across the codebase.

Every exported type or function should earn its existence.

---

## 7. Preferred Commands

Start every session by inspecting the repo:

```bash
git status --short
find . -maxdepth 4 -type f | sort
```

Use Makefile targets where available:

```bash
make help
make fmt
make vet
make test
make check
make baseline
make run NOTE="<hypothesis>"
make summarize
```

Before considering a change complete, run:

```bash
make check
```

If the change affects runner behavior, also run:

```bash
make baseline
make summarize
```

If `make check` does not exist, add it.

Do not claim a command passed unless it was actually run.

---

## 8. Development Loop for Claude Code

For each development session:

1. Read this file.
2. Inspect the repository structure.
3. Inspect `README.md`, `ITERFORGE.md`, `EXAMPLE.md`, and `policy.yaml` if present.
4. Identify the smallest high-value improvement.
5. State the implementation hypothesis.
6. Make a focused change.
7. Run relevant tests/checks.
8. Inspect the diff.
9. Update docs if behavior or commands changed.
10. Summarize what changed and what remains.

Prefer one coherent change per session.

Avoid broad rewrites unless explicitly requested.

---

## 9. Priority Backlog

Implement or improve these in order unless the human operator gives a different priority.

---

### P0: Stable Makefile Contract

Ensure these targets exist:

```bash
make help
make fmt
make vet
make test
make check
make baseline
make run NOTE="<hypothesis>"
make summarize
make clean
```

Expected behavior:

- `make check` runs formatting, vetting, and tests.
- `make run NOTE="..."` runs exactly one scored experiment.
- `make summarize` summarizes `logs/results.jsonl`.
- `make baseline` runs a baseline experiment.
- targets should be documented by `make help`.

---

### P1: Policy Schema and Validator

Add or improve policy validation.

Required policy fields:

```yaml
name: text-normalization-example

editable_paths:
  - candidates/**
  - logs/agent_journal.md

frozen_paths:
  - evals/**
  - cmd/**
  - internal/**
  - policy.yaml
  - Makefile
  - go.mod
  - go.sum
  - logs/results.jsonl

commands:
  check: make check
  evaluate: go run ./cmd/runexp -note "{{note}}"
  summarize: go run ./cmd/summarize

metrics:
  primary:
    name: score
    direction: higher_is_better
    minimum_delta: 0.0001

hard_gates:
  - check_command_must_pass
  - evaluator_must_pass
  - frozen_files_must_not_change
  - no_manual_results_log_edits
  - no_exact_golden_set_hardcoding

logs:
  results: logs/results.jsonl
  journal: logs/agent_journal.md
```

Validator requirements:

- fail fast on missing required fields;
- return actionable error messages;
- validate score direction: `higher_is_better` or `lower_is_better`;
- validate result and journal paths are not empty;
- validate at least one editable path and one frozen path;
- include table-driven tests.

Preferred CLI:

```bash
go run ./cmd/validate-policy
```

or eventually:

```bash
iterforge validate-policy
```

---

### P2: Result Log Contract

`logs/results.jsonl` should be append-only.

Each experiment result should be one JSON object per line.

Minimum schema:

```json
{
  "id": "2026-05-31T14:03:21Z-abc123",
  "timestamp": "2026-05-31T14:03:21Z",
  "note": "trim and lowercase",
  "git_sha": "abc123",
  "dirty": true,
  "check_passed": true,
  "eval_passed": true,
  "score": 0.25,
  "score_direction": "higher_is_better",
  "metrics": {
    "correct": 2,
    "total": 8
  },
  "duration_ms": 187,
  "decision": "candidate_pending_review",
  "failure_reason": ""
}
```

Implementation requirements:

- use stable JSON field names;
- include timestamp;
- include note/hypothesis;
- include score;
- include pass/fail status;
- include metrics map;
- include duration;
- include git metadata where available;
- append, never overwrite;
- preserve valid JSONL.

---

### P3: Runner

The runner should execute one scored experiment.

Preferred command:

```bash
go run ./cmd/runexp -note "hypothesis"
```

Future CLI:

```bash
iterforge run --note "hypothesis"
```

Runner responsibilities:

1. load policy;
2. validate policy;
3. capture timestamp;
4. capture git SHA and dirty state if available;
5. run configured check command or assume caller already ran it, depending on policy;
6. run evaluator;
7. parse evaluator JSON;
8. apply hard gates;
9. append result JSONL;
10. print a concise human-readable summary;
11. return non-zero on hard failure.

Keep the runner deterministic and testable.

Avoid burying logic in `main.go`; put reusable logic under `internal/`.

---

### P4: Evaluator Contract

Evaluators should output machine-readable JSON.

Minimum evaluator output:

```json
{
  "score": 0.875,
  "passed": true,
  "metrics": {
    "correct": 7,
    "total": 8
  },
  "failure_reason": ""
}
```

Preferred evaluator output:

```json
{
  "score": 0.875,
  "score_direction": "higher_is_better",
  "passed": true,
  "hard_gates": {
    "schema_valid": true,
    "latency_within_budget": true,
    "cost_within_budget": true
  },
  "metrics": {
    "correct": 7,
    "total": 8,
    "latency_ms_p95": 12.4,
    "cost_usd": 0.002
  },
  "failure_reason": ""
}
```

The core runner should not be tightly coupled to the toy evaluator.

---

### P5: Summarizer

The summarizer should read `logs/results.jsonl` and report:

- total experiment count;
- latest experiment;
- best experiment;
- best score;
- latest score;
- score direction;
- pass/fail counts;
- trend summary;
- plateau signal if obvious;
- failure reasons;
- recommendation.

Preferred command:

```bash
go run ./cmd/summarize
```

Future CLI:

```bash
iterforge summarize
```

Summarizer should tolerate:

- empty logs;
- malformed lines, with clear error reporting;
- mixed failed/passed runs;
- lower-is-better or higher-is-better metrics.

---

### P6: Candidate Comparator

Add candidate/result comparison.

Future command:

```bash
iterforge compare --baseline <id> --candidate <id>
```

Minimum comparison:

- primary score;
- score delta;
- pass/fail;
- metrics delta;
- duration delta;
- failure reasons;
- decision recommendation.

This should support production review.

---

### P7: Guardrails

Add useful guardrails.

Suggested checks:

- frozen files changed;
- evaluator changed during candidate run;
- golden set changed during candidate run;
- results log manually edited;
- candidate reads evaluator files directly;
- candidate imports forbidden packages;
- network usage attempted when disabled;
- exact golden-set hardcoding patterns appear in candidate code.

These guardrails do not need to be perfect. They should be explicit and useful.

---

### P8: Project Initializer

Add initializer:

```bash
iterforge init <project-name>
```

It should generate a working project:

```text
<project-name>/
├── README.md
├── ITERFORGE.md
├── Makefile
├── policy.yaml
├── candidates/
│   ├── candidate.go
│   └── candidate_test.go
├── evals/
│   ├── evaluator.go
│   ├── evaluator_test.go
│   └── golden_set.jsonl
├── cmd/
│   ├── runexp/
│   └── summarize/
└── logs/
    ├── agent_journal.md
    └── results.jsonl
```

The generated project should pass:

```bash
make check
make baseline
make summarize
```

---

### P9: Workflow Templates

Add templates for real workflows.

Priority templates:

1. text normalization;
2. prompt optimization;
3. RAG retrieval/chunking;
4. extraction pipeline;
5. code repair;
6. ranking/search.

Each template should define:

- mutable surface;
- frozen evaluator;
- golden/eval data shape;
- metrics;
- hard gates;
- example policy;
- example candidate.

Do not add templates until the core policy/runner/logging path is stable.

---

## 10. Package Design Direction

Suggested internal packages:

```text
internal/policy
internal/resultlog
internal/runner
internal/evaluator
internal/gitmeta
internal/guardrails
internal/report
internal/templates
```

Suggested responsibilities:

### internal/policy

- load YAML;
- validate schema;
- expose typed policy struct;
- produce actionable validation errors.

### internal/resultlog

- append JSONL result;
- read JSONL results;
- tolerate empty files;
- reject malformed records with useful errors.

### internal/runner

- coordinate one experiment;
- call checks/evaluator;
- apply hard gates;
- write result log;
- return structured result.

### internal/evaluator

- parse evaluator JSON output;
- normalize metrics;
- validate required evaluator fields.

### internal/gitmeta

- capture git SHA;
- capture dirty state;
- optionally list changed files.

### internal/guardrails

- detect frozen file changes;
- detect suspicious candidate behavior;
- enforce policy-defined boundaries.

### internal/report

- summarize results;
- compare candidates;
- generate Markdown reports.

### internal/templates

- scaffold new project templates.

Keep package APIs small.

---

## 11. CLI Design Direction

Long-term CLI:

```bash
iterforge init <name>
iterforge validate-policy
iterforge check
iterforge run --note "hypothesis"
iterforge summarize
iterforge compare --latest --best
iterforge report
```

Current Makefile-compatible commands may wrap Go commands.

The CLI should:

- print concise summaries by default;
- support JSON output later if useful;
- return meaningful exit codes;
- avoid noisy logs unless verbose mode is requested;
- work well when called by Claude Code.

---

## 12. Testing Strategy

Use table-driven tests.

Prioritize tests for:

- policy validation;
- JSONL result parsing;
- best-score selection;
- score direction handling;
- malformed evaluator output;
- empty logs;
- frozen file detection;
- runner failure modes;
- summarizer output logic.

Examples:

```go
func TestValidatePolicyMissingRequiredFields(t *testing.T) { ... }
func TestBestResultHigherIsBetter(t *testing.T) { ... }
func TestBestResultLowerIsBetter(t *testing.T) { ... }
func TestReadResultsSkipsEmptyLines(t *testing.T) { ... }
func TestEvaluatorRejectsMissingScore(t *testing.T) { ... }
```

Do not rely only on end-to-end manual tests.

---

## 13. Production AI Requirements

IterForge should eventually support production-oriented metrics and gates.

Examples:

### RAG

Metrics:

```text
recall@k
MRR
citation precision
faithfulness
answer correctness
latency p95
cost per query
```

Hard gates:

```text
no uncited claims
minimum citation precision
latency within budget
cost within budget
private holdout not exposed to agent
```

### Prompt Optimization

Metrics:

```text
correctness
schema validity
faithfulness
refusal accuracy
cost
latency
```

Hard gates:

```text
schema must validate
no unsafe completions
no prompt leakage
no format regression
```

### Extraction

Metrics:

```text
field-level F1
exact match
schema validity
false positive rate
```

Hard gates:

```text
output schema valid
required fields present
no hallucinated fields
```

### Code Repair

Metrics:

```text
unit test pass rate
hidden test pass rate
mutation score
benchmark runtime
```

Hard gates:

```text
tests pass
no frozen tests changed
no production API breakage
no benchmark regression beyond threshold
```

Keep this production orientation visible in docs and examples.

---

## 14. Anti-Patterns

Do not turn IterForge into:

- a vague agent orchestration framework;
- a prompt-only convention;
- a benchmark the agent can edit;
- a hidden magic system;
- a tool that trusts explanations over evals;
- a tool that hides failed experiments;
- a system with no keep/revert semantics;
- a system with no hard gates;
- a system that only optimizes one score while ignoring cost, latency, safety, or maintainability.

---

## 15. Documentation Expectations

Keep docs practical.

Important docs:

```text
README.md
ITERFORGE.md
EXAMPLE.md
docs/adaptation-guide.md
docs/policy-schema.md
docs/evaluator-contract.md
docs/workflow-templates.md
```

Docs should answer:

- What problem does this solve?
- What is mutable?
- What is frozen?
- What command runs one experiment?
- What is the score?
- What gates override the score?
- How does the agent decide keep/revert?
- How do I adapt this to my workflow?

---

## 16. First Task If No Task Is Provided

If the human operator gives no specific implementation task, do this first:

```text
Implement or improve policy validation.
```

Acceptance criteria:

- `policy.yaml` has a documented schema.
- missing required fields fail validation.
- invalid score direction fails validation.
- validation errors are actionable.
- tests cover valid and invalid policy examples.
- `make check` passes.
- docs mention how to validate policy.

Preferred command:

```bash
go run ./cmd/validate-policy
```

or if CLI already exists:

```bash
iterforge validate-policy
```

If a validator already exists, improve tests or error messages.

---

## 17. Second Task If Policy Validation Exists

If policy validation is already good, improve result logging.

Acceptance criteria:

- result schema is stable;
- JSONL append path is tested;
- empty logs are handled;
- malformed logs produce actionable errors;
- summarizer can identify best/latest result;
- higher-is-better and lower-is-better are both supported.

---

## 18. Third Task If Result Logging Exists

If policy validation and result logging are already good, improve guardrails.

Acceptance criteria:

- detect changed frozen files using git where available;
- fail or warn clearly depending on policy;
- tests cover common path-matching cases;
- docs explain limitations.

---

## 19. Session Output Format

At the end of each Claude Code session, produce:

```markdown
# Development Summary

Goal:
- <what was attempted>

Changed files:
- <files changed>

Commands run:
- <commands actually run>

Result:
- <pass/fail and important output>

Design impact:
- <how the change improves the IterForge loop>

Production impact:
- <how the change improves safety, reproducibility, usability, or auditability>

Risks / gaps:
- <known limitations>

Next recommended step:
- <one concrete next task>
```

Do not claim success unless commands passed.

Do not omit failed commands.

---

## 20. Review Checklist Before Finishing

Before finishing a change, verify:

```text
[ ] The change preserves mutable/frozen boundary semantics.
[ ] The change supports programmatic experiment loops.
[ ] The change improves reproducibility or safety.
[ ] The change has tests where appropriate.
[ ] The change does not hide failures.
[ ] The command surface remains simple.
[ ] make check passes or failure is clearly reported.
[ ] Docs are updated if behavior changed.
```

---

## 21. Critical Reminder

IterForge should not require the human to manually design each experiment.

The human should define the loop.

The agent should operate within the loop.

The evaluator should decide whether progress occurred.

The logs should preserve the evidence.

The human should decide what gets promoted.

Preserve Karpathy's insight while making the system production-grade:

```text
program the research loop
not every experiment
```
