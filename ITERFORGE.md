# IterForge / Claude Code Instruction File

You are operating an IterForge loop: an iterative, evidence-driven improvement process for a bounded workflow.

Your job is to improve the candidate implementation over several small experiments while preserving correctness, auditability, evaluator integrity, and generalization.

This file is intended to be read by Claude Code or another coding agent before starting an autonomous or semi-autonomous improvement run.

---

## 1. Core Principle

Any workflow with the following properties can be converted into an autonomous improvement loop:

- a bounded mutable surface;
- a trusted evaluator;
- cheap repeated trials;
- a clear score;
- explicit hard gates;
- an audit trail;
- a keep/revert decision rule.

The agent may propose and test improvements, but it must not compromise the evaluator, policy, logs, or trust boundary.

The objective is not to make large speculative changes. The objective is to run disciplined iterations where each change is tied to a hypothesis and measured by a trusted evaluator.

---

## 2. Objective

Improve the candidate implementation over multiple iterations.

The primary goal is to maximize the score reported by:

```bash
make run NOTE="<short hypothesis>"
```

Before each scored run, verify the project still passes:

```bash
make check
```

Only keep changes that pass all hard gates and improve the primary score.

If no score improvement is achieved, revert the change.

If the result is ambiguous, prefer reverting.

---

## 3. Editable Surface

You may edit only the mutable candidate surface.

Default editable files:

```text
candidates/*
logs/agent_journal.md
```

You may also edit candidate-owned tests if present:

```text
candidates/*_test.go
candidates/**/*.go
candidates/**/*.yaml
candidates/**/*.json
```

Do not edit any other files unless the human operator explicitly expands the editable surface.

When in doubt, do not edit the file.

---

## 4. Frozen Surface

Do not edit:

```text
evals/*
cmd/*
internal/*
policy.yaml
go.mod
go.sum
Makefile
logs/results.jsonl
```

These files define the trusted evaluator, experiment runner, policy, dependency boundary, and audit trail.

If improving the score appears to require editing a frozen file, stop and explain the issue instead of modifying it.

The evaluator must remain trustworthy. Never weaken, bypass, delete, or reconfigure it to improve the score.

---

## 5. Hard Gates

Reject a candidate if any hard gate fails.

A candidate must be rejected if:

- `make check` fails.
- `make run` fails.
- the evaluator reports `passed: false`.
- the candidate panics.
- the candidate introduces nondeterministic behavior.
- the candidate bypasses the evaluator.
- the candidate hardcodes visible examples from the golden set.
- the candidate special-cases individual test cases rather than generalizing.
- the candidate uses external network access.
- the candidate reads or modifies files outside the editable surface.
- the candidate changes frozen files.
- the candidate suppresses errors without addressing the root cause.
- the candidate improves the score by exploiting evaluator weaknesses.
- the candidate makes the implementation substantially less maintainable without a strong measured benefit.

All hard gates override the primary score.

---

## 6. Improvement Rule

After each experiment:

1. Compare the new score against the previous best score.
2. If the score improves and all hard gates pass, keep the change.
3. If the score does not improve, revert the change.
4. If the score improves but the change appears brittle, suspicious, or overfit, revert or flag for human review.
5. If the result is ambiguous, prefer reverting.
6. Record the decision in `logs/agent_journal.md`.

Use small, controlled changes.

Prefer one hypothesis per iteration.

Avoid bundling multiple unrelated ideas into one experiment.

---

## 7. Experiment Loop

Repeat the following loop for several iterations:

1. Inspect the current candidate implementation.
2. Inspect the latest result summary.
3. Identify the current best score.
4. Form one specific improvement hypothesis.
5. Modify only files in the editable surface.
6. Run:

```bash
make check
```

7. If checks pass, run:

```bash
make run NOTE="<hypothesis>"
```

8. Review the reported score and pass/fail status.
9. Decide whether to keep or revert the change.
10. Append a journal entry.
11. Continue until a stopping condition is met.

Do not skip the check step.

Do not manually edit `logs/results.jsonl`.

Do not claim improvement without a recorded evaluator run.

---

## 8. Suggested Starting Prompt for Claude Code

Use this prompt after placing this file in the repository:

```text
Read ITERFORGE.md, program.md if present, and policy.yaml. Run an iterative improvement loop for up to 10 experiments. Modify only the editable surface. After every change run `make check`, then `make run NOTE="<hypothesis>"`. Keep only changes that pass gates and improve score. Revert non-improving changes. Maintain `logs/agent_journal.md` and finish with a concise run summary.
```

For a more conservative first run:

```text
Read ITERFORGE.md and policy.yaml. Perform exactly one conservative experiment. Modify only the editable surface. Run `make check` and `make run NOTE="<hypothesis>"`. Keep the change only if all gates pass and the primary score improves. Otherwise revert. Update `logs/agent_journal.md`.
```

---

## 9. Journal Format

Append one entry per attempted experiment to:

```text
logs/agent_journal.md
```

Use this format:

```markdown
## Experiment N

Hypothesis:
- <one sentence>

Change:
- <what changed>

Commands:
- `make check`
- `make run NOTE="..."`

Result:
- Score: <score>
- Passed: <true/false>

Decision:
- Keep / Revert / Human review

Reason:
- <why the change was kept, reverted, or flagged>
```

The journal must reflect what actually happened.

Do not invent results.

Do not summarize a run that was not executed.

---

## 10. Search Strategy

Prefer this improvement order:

1. Correctness-preserving simplification.
2. Better handling of edge cases.
3. More general normalization or matching logic.
4. Better decomposition into small deterministic functions.
5. Reduced branching.
6. Clearer invariants.
7. Better candidate-owned tests.
8. Performance improvement only after correctness is stable.
9. Refactoring only when it supports measurable improvement.

Do not optimize for cleverness.

Optimize for measurable, generalizable improvement.

---

## 11. Anti-Gaming Rules

Do not:

- edit the evaluator;
- inspect hidden evaluator internals beyond normal public interfaces;
- hardcode answers from the golden set;
- bypass scoring logic;
- delete failing examples;
- weaken tests;
- mutate logs except `logs/agent_journal.md`;
- manually edit `logs/results.jsonl`;
- suppress errors without fixing root causes;
- introduce randomness to occasionally get a better score;
- use wall-clock time, environment variables, filesystem probes, or process state to detect evaluation;
- optimize only for visible examples if the change would not generalize;
- make changes that would be unacceptable in production code simply because they improve the local score.

If you notice an evaluator weakness, report it in the journal instead of exploiting it.

---

## 12. Stopping Conditions

Stop the loop when any of the following occurs:

- the requested number of experiments has completed;
- no obvious improvement remains;
- several consecutive attempts fail to improve the score;
- the score reaches the apparent maximum;
- the agent encounters ambiguity about the editable/frozen boundary;
- the agent suspects evaluator gaming would be required to improve further;
- the human operator stops the run;
- required commands cannot be executed;
- hard gates fail repeatedly due to infrastructure or environment problems.

When stopping, produce a final summary.

---

## 13. Final Report Format

When finished, produce this report:

```markdown
# IterForge Run Summary

Best score:
- <score>

Best change:
- <summary>

Experiments attempted:
- <count>

Kept changes:
- <count>

Rejected changes:
- <count>

Commands run:
- `make check`
- `make run NOTE="..."`

Evidence:
- <latest result references>
- <journal entries>
- <git diff summary if available>

Recommendation:
- Promote / Do not promote / Needs more evaluation

Risks:
- <known limitations, possible overfitting, or evaluator gaps>

Suggested next experiments:
- <short list>
```

Keep the final report factual and grounded in executed runs.

---

## 14. Concrete Toy Objective Example

A simple starter objective is to improve a deterministic Go function:

```go
func Transform(input string) string
```

The evaluator may score outputs against a golden set.

Example golden-set rows:

```jsonl
{"input":" Hello, WORLD! ","expected":"hello world"}
{"input":"foo_bar","expected":"foo bar"}
{"input":"multiple    spaces","expected":"multiple spaces"}
{"input":"Ross-Video","expected":"ross video"}
```

A good iterative loop could discover improvements such as:

1. trim leading and trailing whitespace;
2. lowercase text;
3. replace underscores with spaces;
4. replace hyphens with spaces;
5. collapse repeated whitespace;
6. preserve only expected character classes;
7. add candidate-owned table-driven tests.

This is intentionally simple. It provides a cheap, deterministic demonstration of the self-improvement loop before applying the method to more complex workflows such as RAG, prompt optimization, extraction, ranking, or production code.

---

## 15. Example Candidate Implementation Shape

The candidate surface might expose a function like this:

```go
package candidates

func Transform(input string) string {
    // Improve this implementation over multiple iterations.
    return input
}
```

The evaluator owns the scoring logic and should call this function as a black-box candidate.

The candidate should not know the evaluator internals.

---

## 16. Example Evaluation Categories

For real use cases, replace the toy evaluator with domain-specific metrics.

Examples:

| Workflow | Mutable Surface | Trusted Evaluator | Score |
|---|---|---|---|
| RAG optimization | chunking/retrieval config | offline query/evidence set | recall@k, MRR, citation precision |
| Prompt optimization | prompt template | golden answer set | correctness, faithfulness, cost |
| Extraction pipeline | extraction logic | labeled records | exact match, F1, schema validity |
| Code repair | implementation module | unit/mutation tests | pass rate, coverage, performance |
| Ranking/search | scoring function | relevance judgments | NDCG, MRR, recall |
| Data cleanup | normalization rules | labeled examples | accuracy, false-positive rate |

The invariant remains the same:

```text
bounded mutable surface
+ trusted evaluator
+ cheap trials
+ clear score
+ hard gates
+ audit trail
= autonomous improvement loop
```

---

## 17. Operating Discipline

Use git if available.

Before starting, inspect:

```bash
git status --short
```

After each kept change, inspect:

```bash
git diff
```

If a change is rejected, revert only the attempted candidate change.

Do not remove unrelated user work.

If the repository has uncommitted changes that were not made by the agent, avoid overwriting them.

If unsure whether a change is yours, stop and ask for human review.

---

## 18. Promotion Guidance

A candidate should be promoted only if:

- all hard gates pass;
- the primary score improves;
- secondary metrics do not regress materially;
- the change is understandable;
- the change appears generalizable;
- the journal explains the hypothesis and evidence;
- the diff is reviewable;
- the result is reproducible with the documented command.

A candidate should not be promoted merely because it improved one visible local metric.

Prefer evidence over explanation.

Prefer reproducibility over novelty.

Prefer small, reliable gains over speculative rewrites.

---

## 19. Default Make Targets

If a Makefile is present, prefer these commands:

```bash
make help
make check
make test
make run NOTE="<hypothesis>"
make summarize
make baseline
make loop N=10
make clean
```

Minimum required commands:

```bash
make check
make run NOTE="<hypothesis>"
```

The Make targets wrap a single binary; the equivalent direct commands are
`iterforge run -note "..."`, `iterforge summarize`, `iterforge compare`, and
`iterforge validate-policy` (see `docs/cli.md`).

If these commands are unavailable, inspect the repository documentation and identify the equivalent check and evaluation commands.

Do not invent successful runs.

---

## 20. Minimal Run Contract

Every experiment must have:

```text
hypothesis
candidate diff
check command
evaluation command
score
pass/fail result
keep/revert decision
journal entry
```

If any of these are missing, the experiment is incomplete.

---

## 21. Agent Behavior Requirements

Be conservative.

Be explicit.

Be reproducible.

Do not hide failures.

Do not inflate claims.

Do not weaken the evaluator.

Do not cross the mutable/frozen boundary.

Do not trade maintainability for marginal score gains unless the human operator explicitly requests aggressive optimization.

When in doubt, stop and report the ambiguity.

