# Adaptation Guide

To adapt IterForge to a real workflow, preserve the control-plane separation:

```text
mutable candidate (code / config / prompt)
        ↓
frozen evaluator  →  score + metrics{} + gates{}
        ↓
guardrails (frozen-change, tamper, golden-hardcoding, timeout)
        ↓
append-only result log
        ↓
keep / revert / promotion decision
```

## Fastest start: scaffold

Generate a working project for a known shape, then edit it:

```bash
go run ./cmd/iterforge init -template extraction my-extractor
go run ./cmd/iterforge init -template prompt-optimization my-prompt
```

Templates: `text-normalization`, `extraction`, `prompt-optimization`
(see [workflow-templates.md](workflow-templates.md)). Each scaffolds a project
that passes `make check && make baseline && make summarize` out of the box.

## The three things you change

1. **Mutable surface** — the thing being optimized. Code (`candidates/*.go`),
   config, or a prompt file (`candidates/prompt.txt`). List it under
   `experiment.mutable_paths` in `policy.yaml`.
2. **Frozen evaluator** — scores the candidate and reports gates. Either the
   in-process `evals.Evaluate`, or an external command via
   `commands.evaluate` (see below). Listed under `frozen_paths`.
3. **Golden / eval data** — `evals/golden_set.jsonl`. Never read by the
   candidate (the golden-hardcoding guardrail enforces this).

## Evaluator output: score, metrics, gates

The evaluator reports a primary `score`, a `metrics` map (surfaced by
`iterforge summarize`), and a `gates` map of named hard gates the runner
enforces *in addition to* the score. A `false` gate fails the run even at a
perfect score. See [evaluator-contract.md](evaluator-contract.md).

Use gates for what must hold regardless of score: schema validity, required
fields present, no unsafe output, latency/cost budgets.

## Plugging in a real model or evaluator

Set `commands.evaluate` in `policy.yaml` to shell out to any language:

```yaml
commands:
  evaluate: python eval.py
experiment:
  timeout_seconds: 60
```

The command receives the golden-set path in `IF_GOLDEN_SET` and must print
standard evaluator JSON (`{"score":..,"metrics":{..},"gates":{..}}`) to stdout.
The prompt-optimization template's stub uses `ITERFORGE_MODEL_CMD` the same way
for the *model* call. `timeout_seconds` bounds the command (process-group kill);
overruns are recorded as `rejected_evaluator_timeout`.

## Per-workflow shapes

### RAG retrieval / chunking
- Mutable: `candidates/{retriever,chunker,ranker}.go`, config.
- Metrics: `recall_at_k`, `mrr`, `citation_precision`, `groundedness`, `latency_ms_p95`, `cost_usd`.
- Gates: `no_uncited_claims`, `latency_within_budget`, `cost_within_budget`.

### Prompt optimization
- Mutable: `candidates/prompt.txt` (or templates).
- Metrics: rubric/exact-match score, `faithfulness`, `refusal_accuracy`, cost, latency.
- Gates: `schema_valid`, `no_unsafe_output`, `no_format_regression`.
- Use an LLM judge only with calibration examples and deterministic rubric output.

### Data extraction
- Mutable: `candidates/extractor.go`.
- Metrics: field-level precision/recall/F1, `exact_record_match`.
- Gates: `schema_valid`, `required_fields_present`, `no_hallucinated_fields`.

### Code repair / performance tuning
- Mutable: `candidates/target_package/**`.
- Gates: `tests_pass` (`go test ./...`), `no_benchmark_regression`, `no_frozen_tests_changed`.

## Private holdout

Keep a holdout dataset outside the agent-visible workspace. Run it manually
before promotion — the golden set the agent optimizes against is not the final
judge. Use `iterforge compare` to vet a candidate against a baseline, and only
promote on a `promote candidate` recommendation.

## Operating the loop

```bash
make check                       # fmt + vet + tests + policy validation
iterforge run -note "hypothesis" # one scored, gated, audited experiment
iterforge summarize -last 20     # recent trend + decision histogram
iterforge compare                # best vs latest, keep/promote recommendation
```

## Recommended Claude Code prompt

```text
Read program.md and policy.yaml. Improve the mutable candidate only; never edit
frozen files or read the golden set. For each attempt: form a hypothesis, make
one small change, run `make check`, then `iterforge run -note "<hypothesis>"`,
inspect the recorded decision and metrics, and keep or revert. Record concise
notes in logs/agent_journal.md. Stop after a clear improvement or 5 attempts.
```
