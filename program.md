# Autoresearch Program

You are operating an autonomous improvement loop for a bounded optimization task.

Your objective is to improve the candidate implementation while preserving evaluator integrity, auditability, and reproducibility.

## Objective

Improve the primary score reported by:

```bash
make run NOTE="<hypothesis>"
```

Equivalent raw command:

```bash
go run ./cmd/runexp -note "<hypothesis>"
```

The current toy objective is exact-match accuracy for text normalization. In a real workflow, this objective may be replaced by retrieval quality, prompt quality, extraction quality, ranking quality, benchmark performance, or another objective metric.

## Editable Surface

You may edit:

```text
candidates/*
logs/agent_journal.md
```

Treat `candidates/` as the mutable research surface.

## Frozen Surface

Do not edit these unless the human explicitly authorizes it:

```text
evals/*
cmd/*
internal/*
Makefile
policy.yaml
go.mod
go.sum
README.md
docs/*
logs/results.jsonl
```

The evaluator and golden set are the trusted measurement layer. Do not weaken, bypass, reinterpret, or modify them.

## Experiment Loop

For each attempt:

1. Inspect prior results and current candidate behavior.
2. Form one concrete hypothesis.
3. Make the smallest useful candidate change.
4. Run:

   ```bash
   make check
   make run NOTE="<hypothesis>"
   ```

5. Inspect the emitted result.
6. Keep the change only if:
   - all hard gates pass;
   - the primary score improves by at least the configured minimum delta;
   - the change does not appear to exploit the evaluator;
   - the implementation remains maintainable.
7. Otherwise revert the change.
8. Append a short note to `logs/agent_journal.md` with hypothesis, result, and decision.

## Anti-Gaming Rules

Do not:

- hard-code answers from the golden set;
- special-case input IDs or examples;
- read or parse `evals/golden_set.jsonl` from candidate code;
- change the evaluator, runner, policy, or golden set;
- reduce validation coverage;
- optimize one metric by violating a hard gate;
- add hidden network calls or uncontrolled dependencies;
- make the candidate nondeterministic unless the evaluator explicitly supports it.

## Promotion Criteria

Recommend promotion only when the candidate beats the incumbent by the policy threshold and all hard gates pass.

A promotion recommendation must include:

```text
hypothesis
diff summary
command run
primary score before/after
hard gate status
known risks
why the result should generalize
```

## Failure Handling

If an experiment fails to compile, times out, panics, or violates a gate, revert the candidate change and record the failure.

## Human Escalation

Ask for human approval before changing evaluator semantics, adding dependencies, changing the golden set, changing policy thresholds, or expanding the mutable surface.
