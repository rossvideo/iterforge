package evals

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"iterforge/candidates"
)

type Example struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

type CaseResult struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

type Result struct {
	Total       int                `json:"total"`
	Passed      int                `json:"passed"`
	Failed      int                `json:"failed"`
	Accuracy    float64            `json:"accuracy"`
	LatencyMS   float64            `json:"latency_ms"`
	Metrics     map[string]float64 `json:"metrics,omitempty"`
	Gates       map[string]bool    `json:"gates,omitempty"`
	CaseResults []CaseResult       `json:"case_results"`
}

func LoadGoldenSet(path string) ([]Example, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var examples []Example
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ex Example
		if err := json.Unmarshal(line, &ex); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, lineNo, err)
		}
		examples = append(examples, ex)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(examples) == 0 {
		return nil, fmt.Errorf("golden set is empty: %s", path)
	}
	return examples, nil
}

func Evaluate(goldenSetPath string) (Result, error) {
	examples, err := LoadGoldenSet(goldenSetPath)
	if err != nil {
		return Result{}, err
	}

	start := time.Now()
	caseResults := make([]CaseResult, 0, len(examples))
	passed := 0
	nonEmpty := 0

	for _, ex := range examples {
		actual := candidates.Transform(ex.Input)
		ok := actual == ex.Expected
		if ok {
			passed++
		}
		if actual != "" {
			nonEmpty++
		}
		caseResults = append(caseResults, CaseResult{
			Input:    ex.Input,
			Expected: ex.Expected,
			Actual:   actual,
			Passed:   ok,
		})
	}

	latencyMS := float64(time.Since(start).Microseconds()) / 1000.0
	total := len(examples)
	failed := total - passed
	accuracy := float64(passed) / float64(total)
	return Result{
		Total:     total,
		Passed:    passed,
		Failed:    failed,
		Accuracy:  accuracy,
		LatencyMS: latencyMS,
		Metrics:   map[string]float64{"accuracy": accuracy},
		// Named hard gate: the candidate must never collapse an input to the
		// empty string. Enforced by the runner regardless of the score.
		Gates:       map[string]bool{"no_empty_output": nonEmpty == total},
		CaseResults: caseResults,
	}, nil
}
