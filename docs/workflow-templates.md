# Workflow Templates

`iterforge init -template <name>` scaffolds a project pre-wired for a specific
workflow. Each template defines its own mutable candidate, frozen evaluator,
golden-set shape, score, and hard gates. The shared runner/summarizer/log are
identical across templates.

List available templates:

```bash
go run ./cmd/iterforge init -h   # prints usage with the -template choices
```

## Available

### text-normalization (default)

| | |
|---|---|
| Mutable | `candidates.Transform(string) string` |
| Evaluator | exact-match accuracy vs `golden_set.jsonl` |
| Golden shape | `{"input": "...", "expected": "..."}` |
| Primary metric | `accuracy` |
| Hard gates | `min_accuracy`, `max_latency_ms`, `max_failures` |

Lowercase, strip punctuation, collapse whitespace.

### extraction

| | |
|---|---|
| Mutable | `candidates.Extract(string) Fields` (email, phone) |
| Evaluator | field-level exact-match accuracy |
| Golden shape | `{"input": "...", "email": "...", "phone": "..."}` |
| Primary metric | `field_accuracy` |
| Hard gates | `min_accuracy`, `max_latency_ms`, `max_failures` |

Watch for hallucinated fields, format drift (email case, phone punctuation), and
greedy-regex false positives.

### prompt-optimization

| | |
|---|---|
| Mutable | `candidates/prompt.txt` (the prompt — not code) |
| Evaluator | exact-match accuracy of model output vs `golden_set.jsonl` |
| Golden shape | `{"input": "...", "expected": "..."}` |
| Primary metric | `prompt_accuracy` |
| Hard gates | `min_accuracy`, `max_latency_ms`, `max_failures` |

The evaluator calls a model per example. By default a deterministic built-in
stub follows the directives named in the prompt, so the project runs with no API
key. Set `ITERFORGE_MODEL_CMD` to shell out to a real model: the command reads
the prompt from `IF_PROMPT` and the input from `IF_INPUT` and prints the result
to stdout. This is the pattern for any external-model / API-backed evaluator.

Editing the prompt changes the score (e.g. removing the `lowercase` directive
drops accuracy), so the optimization loop is real, not cosmetic.

### ranking

| | |
|---|---|
| Mutable | `candidates.Score(query, doc) float64` |
| Evaluator | orders docs by score; mean NDCG (primary) + MRR vs graded labels |
| Golden shape | `{"query": "...", "docs": [{"text": "...", "rel": 3}, ...]}` |
| Primary metric | `ndcg` |
| Hard gates | `min_accuracy`, `max_latency_ms`, `max_failures` |

Baseline scorer is token overlap. Watch for overfitting to visible queries,
gaming NDCG while MRR/tail relevance regresses, and nondeterministic tie-breaking.

## Acceptance

Every template is generated and verified to pass `make check`, `make baseline`,
and `make summarize` (see `internal/templates/templates_test.go` for the
per-template generation test).

## Not yet shipped

RAG retrieval/chunking and code-repair templates need extra infrastructure (a
retriever, a sandbox) and are deliberately omitted from the deterministic
starter set. The `prompt-optimization` template's `ITERFORGE_MODEL_CMD` hook is
the model for wiring them to external systems.
