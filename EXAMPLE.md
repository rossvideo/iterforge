# IterForge Basic Example

This file shows a complete minimal example of an IterForge-style improvement loop.

The goal is to demonstrate the pattern:

```text
bounded mutable surface
+ trusted evaluator
+ cheap trials
+ clear score
+ hard gates
+ audit trail
= autonomous improvement loop
```

This example uses a toy Go text-normalization function.

Claude Code or another coding agent is allowed to improve only the candidate implementation. The evaluator, golden set, runner, policy, and Makefile are frozen.

> **Orientation.** This is a conceptual walkthrough of the loop discipline. To
> get a real, runnable project with exactly this shape (its own `cmd/runexp`,
> `cmd/summarize`, evaluator, golden set, and Makefile), scaffold one:
>
> ```bash
> go run ./cmd/iterforge init -template text-normalization my-loop
> ```
>
> The IterForge **tool** itself exposes one binary, `iterforge`
> (run/summarize/validate-policy/compare/init — see `docs/cli.md`). The
> simplified evaluator below shows the minimum contract; real evaluators also
> report a `metrics` map and named `gates` (see `docs/evaluator-contract.md`).

---

## 1. Scenario

We have a Go function:

```go
func Transform(input string) string
```

The function should normalize messy input strings into canonical lowercase text.

Examples:

```text
" Hello, WORLD! "       -> "hello world"
"foo_bar"              -> "foo bar"
"multiple    spaces"   -> "multiple spaces"
"Ross-Video"           -> "ross video"
```

The agent's job is to improve `Transform` over several iterations.

The agent must:

- make one small change at a time;
- run checks after each change;
- run the evaluator after each passing check;
- keep only score-improving changes;
- revert non-improving changes;
- record every attempt in `logs/agent_journal.md`.

---

## 2. Repository Layout

```text
iterforge-example/
├── ITERFORGE.md
├── EXAMPLE.md
├── Makefile
├── go.mod
├── candidates/
│   ├── candidate.go
│   └── candidate_test.go
├── evals/
│   ├── evaluator.go
│   └── golden_set.jsonl
├── cmd/
│   ├── runexp/
│   │   └── main.go
│   └── summarize/
│       └── main.go
├── logs/
│   ├── agent_journal.md
│   └── results.jsonl
└── policy.yaml
```

---

## 3. Mutable and Frozen Surfaces

### Editable Surface

Claude Code may edit:

```text
candidates/candidate.go
candidates/candidate_test.go
logs/agent_journal.md
```

### Frozen Surface

Claude Code must not edit:

```text
evals/*
cmd/*
policy.yaml
Makefile
go.mod
go.sum
logs/results.jsonl
```

The frozen surface defines the trusted evaluator and audit trail.

If the score can only be improved by editing frozen files, the agent must stop and report that the task/evaluator contract is insufficient.

---

## 4. Candidate Implementation: Initial Version

File:

```text
candidates/candidate.go
```

Initial implementation:

```go
package candidates

func Transform(input string) string {
	return input
}
```

This implementation is intentionally bad. It simply returns the input unchanged.

The agent should improve it over several iterations.

---

## 5. Candidate-Owned Test File

File:

```text
candidates/candidate_test.go
```

Candidate-owned tests are allowed because they help guide local development. They are not the trusted evaluator.

```go
package candidates

import "testing"

func TestTransformBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims and lowercases",
			input:    " Hello, WORLD! ",
			expected: "hello world",
		},
		{
			name:     "underscore becomes space",
			input:    "foo_bar",
			expected: "foo bar",
		},
		{
			name:     "hyphen becomes space",
			input:    "Ross-Video",
			expected: "ross video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Transform(tt.input)
			if got != tt.expected {
				t.Fatalf("Transform(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}
```

---

## 6. Golden Set

File:

```text
evals/golden_set.jsonl
```

The golden set is frozen. The agent must not edit it.

```jsonl
{"input":" Hello, WORLD! ","expected":"hello world"}
{"input":"foo_bar","expected":"foo bar"}
{"input":"multiple    spaces","expected":"multiple spaces"}
{"input":"Ross-Video","expected":"ross video"}
{"input":"Tabs\tand\nlines","expected":"tabs and lines"}
{"input":"Customer-ID: ABC_123","expected":"customer id abc 123"}
{"input":"  Mixed---Separators___Here  ","expected":"mixed separators here"}
{"input":"Keep 42 Numbers","expected":"keep 42 numbers"}
```

---

## 7. Trusted Evaluator

File:

```text
evals/evaluator.go
```

The evaluator is frozen. The agent must not edit it.

The evaluator should:

1. read the golden set;
2. call `candidates.Transform(input)` for each example;
3. count exact matches;
4. report a score from `0.0` to `1.0`.

Simplified evaluator shape:

```go
package evals

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"iterforge-example/candidates"
)

type Example struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

type Result struct {
	Score   float64 `json:"score"`
	Passed  bool    `json:"passed"`
	Correct int     `json:"correct"`
	Total   int     `json:"total"`
}

func Evaluate(path string) (Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	total := 0
	correct := 0

	for scanner.Scan() {
		var ex Example
		if err := json.Unmarshal(scanner.Bytes(), &ex); err != nil {
			return Result{}, err
		}

		total++
		got := candidates.Transform(ex.Input)
		if got == ex.Expected {
			correct++
		}
	}

	if err := scanner.Err(); err != nil {
		return Result{}, err
	}

	if total == 0 {
		return Result{}, fmt.Errorf("golden set is empty")
	}

	score := float64(correct) / float64(total)

	return Result{
		Score:   score,
		Passed:  score >= 0.0,
		Correct: correct,
		Total:   total,
	}, nil
}
```

The real project may include stronger gates, richer metrics, latency checks, or cost limits.

---

## 8. Makefile Commands

The agent should use these commands.

```makefile
.PHONY: help test check run summarize fmt vet baseline clean

NOTE ?= manual

help:
	@echo "Available targets:"
	@echo "  make check                 Run formatting, vetting, and tests"
	@echo "  make test                  Run Go tests"
	@echo "  make run NOTE='hypothesis' Run one scored experiment"
	@echo "  make summarize             Summarize experiment results"
	@echo "  make baseline              Run baseline experiment"
	@echo "  make fmt                   Format Go code"
	@echo "  make vet                   Run go vet"
	@echo "  make clean                 Remove transient files"

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

check: fmt vet test

run:
	go run ./cmd/runexp -note "$(NOTE)"

baseline:
	go run ./cmd/runexp -note "baseline"

summarize:
	go run ./cmd/summarize

clean:
	rm -f ./tmp/*
```

Minimum required loop:

```bash
make check
make run NOTE="<hypothesis>"
```

---

## 9. Claude Code Starting Prompt

Use this prompt to start the loop:

```text
Read ITERFORGE.md, EXAMPLE.md, and policy.yaml. Run up to 6 IterForge experiments on the toy Transform function. Modify only candidates/candidate.go, candidates/candidate_test.go, and logs/agent_journal.md. After every candidate change, run `make check`, then `make run NOTE="<hypothesis>"`. Keep only changes that pass all gates and improve the score. Revert non-improving changes. Do not edit evals/*, cmd/*, policy.yaml, Makefile, go.mod, or logs/results.jsonl. Finish with a concise run summary.
```

For a single conservative iteration:

```text
Read ITERFORGE.md and EXAMPLE.md. Perform exactly one conservative improvement to candidates/candidate.go. Run `make check`, then `make run NOTE="<hypothesis>"`. Keep the change only if the score improves and all gates pass. Otherwise revert. Update logs/agent_journal.md.
```

---

## 10. Example Iteration 0: Baseline

Initial candidate:

```go
package candidates

func Transform(input string) string {
	return input
}
```

Command:

```bash
make check
make run NOTE="baseline unchanged transform"
```

Possible result:

```json
{
  "score": 0.0,
  "passed": true,
  "correct": 0,
  "total": 8
}
```

Journal entry:

```markdown
## Experiment 0

Hypothesis:
- Baseline unchanged implementation establishes the starting score.

Change:
- No code change.

Commands:
- `make check`
- `make run NOTE="baseline unchanged transform"`

Result:
- Score: 0.0000
- Passed: true
- Correct: 0 / 8

Decision:
- Keep

Reason:
- Baseline measurement only.
```

---

## 11. Example Iteration 1: Trim and Lowercase

Hypothesis:

```text
Trimming leading/trailing whitespace and lowercasing should improve examples with casing and surrounding spaces.
```

Candidate change:

```go
package candidates

import "strings"

func Transform(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}
```

Command:

```bash
make check
make run NOTE="trim whitespace and lowercase"
```

Possible result:

```json
{
  "score": 0.25,
  "passed": true,
  "correct": 2,
  "total": 8
}
```

Decision:

```text
Keep
```

Reason:

```text
Score improved from 0.0000 to 0.2500 and all hard gates passed.
```

Journal entry:

```markdown
## Experiment 1

Hypothesis:
- Trimming leading/trailing whitespace and lowercasing should improve examples with casing and surrounding spaces.

Change:
- Added strings.TrimSpace and strings.ToLower in candidates.Transform.

Commands:
- `make check`
- `make run NOTE="trim whitespace and lowercase"`

Result:
- Score: 0.2500
- Passed: true
- Correct: 2 / 8

Decision:
- Keep

Reason:
- Score improved from 0.0000 to 0.2500 and all hard gates passed.
```

---

## 12. Example Iteration 2: Replace Separators

Hypothesis:

```text
Replacing underscores and hyphens with spaces should improve examples containing common word separators.
```

Candidate change:

```go
package candidates

import "strings"

func Transform(input string) string {
	s := strings.TrimSpace(input)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	return s
}
```

Command:

```bash
make check
make run NOTE="replace underscores and hyphens with spaces"
```

Possible result:

```json
{
  "score": 0.50,
  "passed": true,
  "correct": 4,
  "total": 8
}
```

Decision:

```text
Keep
```

Journal entry:

```markdown
## Experiment 2

Hypothesis:
- Replacing underscores and hyphens with spaces should improve examples containing common word separators.

Change:
- Replaced `_` and `-` with spaces after trim/lowercase normalization.

Commands:
- `make check`
- `make run NOTE="replace underscores and hyphens with spaces"`

Result:
- Score: 0.5000
- Passed: true
- Correct: 4 / 8

Decision:
- Keep

Reason:
- Score improved from 0.2500 to 0.5000 and all hard gates passed.
```

---

## 13. Example Iteration 3: Normalize All Non-Alphanumeric Runs

Hypothesis:

```text
Converting any run of non-alphanumeric characters into a single space should handle punctuation, tabs, newlines, repeated hyphens, underscores, and colons more generally.
```

Candidate change:

```go
package candidates

import (
	"strings"
	"unicode"
)

func Transform(input string) string {
	var b strings.Builder
	lastWasSpace := true

	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastWasSpace = false
			continue
		}

		if !lastWasSpace {
			b.WriteByte(' ')
			lastWasSpace = true
		}
	}

	return strings.TrimSpace(b.String())
}
```

Command:

```bash
make check
make run NOTE="normalize non-alphanumeric runs to single spaces"
```

Possible result:

```json
{
  "score": 1.0,
  "passed": true,
  "correct": 8,
  "total": 8
}
```

Decision:

```text
Keep
```

Journal entry:

```markdown
## Experiment 3

Hypothesis:
- Converting any run of non-alphanumeric characters into a single space should handle punctuation, tabs, newlines, repeated hyphens, underscores, and colons more generally.

Change:
- Rewrote Transform to lowercase input, preserve letters/digits, and collapse non-alphanumeric runs into single spaces.

Commands:
- `make check`
- `make run NOTE="normalize non-alphanumeric runs to single spaces"`

Result:
- Score: 1.0000
- Passed: true
- Correct: 8 / 8

Decision:
- Keep

Reason:
- Score improved from 0.5000 to 1.0000 and all hard gates passed. The change is general rather than hardcoded to individual examples.
```

---

## 14. Example Iteration 4: Rejected Overfit Attempt

Hypothesis:

```text
Special-casing the exact input "Customer-ID: ABC_123" could improve one remaining example.
```

Candidate change:

```go
func Transform(input string) string {
	if input == "Customer-ID: ABC_123" {
		return "customer id abc 123"
	}

	// previous implementation...
}
```

This is not allowed.

Even if it improved the score, it violates the anti-gaming rule against hardcoding visible golden-set examples.

Decision:

```text
Revert
```

Journal entry:

```markdown
## Experiment 4

Hypothesis:
- Special-casing the exact input "Customer-ID: ABC_123" could improve one remaining example.

Change:
- Attempted to add an exact input special case.

Commands:
- Not run.

Result:
- Score: unchanged
- Passed: false by policy

Decision:
- Revert

Reason:
- Rejected before evaluation because the change hardcoded a visible golden-set example and would not generalize.
```

---

## 15. Example Iteration 5: Rejected Frozen File Modification

Hypothesis:

```text
Relaxing the evaluator to use fuzzy string matching would improve the score.
```

Attempted change:

```text
evals/evaluator.go
```

This is not allowed.

The evaluator is frozen.

Decision:

```text
Stop and report
```

Journal entry:

```markdown
## Experiment 5

Hypothesis:
- Relaxing the evaluator to use fuzzy string matching would improve the score.

Change:
- No change kept. The proposed change would modify `evals/evaluator.go`.

Commands:
- Not run.

Result:
- Score: unchanged
- Passed: false by policy

Decision:
- Revert / Human review

Reason:
- The proposed improvement crossed the frozen evaluator boundary. If fuzzy matching is desired, the human operator must update the evaluation contract.
```

---

## 16. Final Candidate After Successful Iterations

A good final candidate for this toy example:

```go
package candidates

import (
	"strings"
	"unicode"
)

func Transform(input string) string {
	var b strings.Builder
	lastWasSpace := true

	for _, r := range strings.ToLower(input) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastWasSpace = false
			continue
		}

		if !lastWasSpace {
			b.WriteByte(' ')
			lastWasSpace = true
		}
	}

	return strings.TrimSpace(b.String())
}
```

This implementation is:

- deterministic;
- simple;
- general;
- not hardcoded to the golden set;
- easy to review;
- cheap to evaluate;
- suitable for promotion in the toy setting.

---

## 17. Final Run Summary Example

```markdown
# IterForge Run Summary

Best score:
- 1.0000

Best change:
- Rewrote `Transform` to lowercase input, preserve letters/digits, and collapse runs of non-alphanumeric characters into a single space.

Experiments attempted:
- 5

Kept changes:
- 3

Rejected changes:
- 2

Commands run:
- `make check`
- `make run NOTE="baseline unchanged transform"`
- `make run NOTE="trim whitespace and lowercase"`
- `make run NOTE="replace underscores and hyphens with spaces"`
- `make run NOTE="normalize non-alphanumeric runs to single spaces"`

Evidence:
- Latest evaluator result: 8 / 8 correct, score 1.0000.
- Journal entries: Experiments 0-5.

Recommendation:
- Promote for the toy evaluator.

Risks:
- The evaluator is small. A larger hidden holdout set should be used before assuming the transformation generalizes broadly.
- Unicode handling may need stricter product requirements.

Suggested next experiments:
- Add a private holdout set.
- Add examples with apostrophes, accented characters, slashes, and emoji.
- Add latency checks if the function will process large volumes of text.
```

---

## 18. Mapping This Toy Example to Real Workflows

The same structure applies to more useful workflows.

### RAG Optimization

```text
Mutable surface:
- chunking strategy
- retrieval parameters
- reranking configuration
- prompt template

Frozen evaluator:
- query set
- expected evidence
- answer-quality rubric
- latency/cost gates

Score:
- recall@k
- MRR
- citation precision
- groundedness
- answer correctness
```

### Prompt Optimization

```text
Mutable surface:
- prompt template
- examples
- output schema instructions

Frozen evaluator:
- golden test cases
- expected outputs
- LLM-as-judge rubric
- deterministic schema validator

Score:
- correctness
- faithfulness
- format validity
- cost
- latency
```

### Extraction Pipeline

```text
Mutable surface:
- extraction logic
- normalization rules
- validation rules

Frozen evaluator:
- labeled documents
- expected extracted fields
- schema validator

Score:
- exact match
- field-level F1
- schema validity
```

### Code Repair

```text
Mutable surface:
- implementation files
- candidate-owned tests

Frozen evaluator:
- hidden tests
- public tests
- static analysis
- benchmark thresholds

Score:
- test pass rate
- mutation score
- performance
```

---

## 19. Minimal Adaptation Checklist

To adapt this example to a new workflow, define:

```text
1. What is the mutable surface?
2. What is the frozen evaluator?
3. What command runs checks?
4. What command runs one scored experiment?
5. What is the primary score?
6. What hard gates override the score?
7. What counts as overfitting or evaluator gaming?
8. Where are results logged?
9. What is the keep/revert rule?
10. What stopping condition should the agent use?
```

If any answer is unclear, the workflow is not ready for autonomous iteration.

---

## 20. Key Lesson

The candidate can be simple.

The evaluator must be trusted.

The loop must be disciplined.

The score must be cheap and repeatable.

The agent must only improve inside the allowed mutable surface.

That is the essence of IterForge.

