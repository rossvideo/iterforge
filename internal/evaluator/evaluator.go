// Package evaluator defines the standard evaluator output contract and decouples
// the runner from any specific evaluator implementation.
//
// An evaluator (in-process or an external command) reports a primary score, an
// optional set of named metrics, and an optional set of named hard gates. The
// runner enforces the gates in addition to the primary score, so a regression
// in safety/format/schema can fail a run even when the headline score looks
// fine.
package evaluator

import (
	"encoding/json"
	"fmt"
	"sort"
)

// Output is the normalized result of one evaluation.
type Output struct {
	Score         float64            `json:"score"`
	Passed        bool               `json:"passed"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
	Gates         map[string]bool    `json:"gates,omitempty"`
	FailureReason string             `json:"failure_reason,omitempty"`
}

// Parse reads standard evaluator JSON. A "score" field is required; metrics,
// gates, passed, and failure_reason are optional. Parse errors and the missing
// score are reported with actionable messages.
func Parse(data []byte) (Output, error) {
	// Decode into a shadow with a pointer score so we can detect its absence.
	var shadow struct {
		Score         *float64           `json:"score"`
		Passed        bool               `json:"passed"`
		Metrics       map[string]float64 `json:"metrics"`
		Gates         map[string]bool    `json:"gates"`
		FailureReason string             `json:"failure_reason"`
	}
	if err := json.Unmarshal(data, &shadow); err != nil {
		return Output{}, fmt.Errorf("invalid evaluator JSON: %w", err)
	}
	if shadow.Score == nil {
		return Output{}, fmt.Errorf("evaluator output missing required field %q", "score")
	}
	return Output{
		Score:         *shadow.Score,
		Passed:        shadow.Passed,
		Metrics:       shadow.Metrics,
		Gates:         shadow.Gates,
		FailureReason: shadow.FailureReason,
	}, nil
}

// GateFailures returns a sorted, human-readable message for each gate that is
// false. The empty slice means every reported gate passed.
func (o Output) GateFailures() []string {
	names := make([]string, 0, len(o.Gates))
	for name := range o.Gates {
		names = append(names, name)
	}
	sort.Strings(names)

	var failures []string
	for _, name := range names {
		if !o.Gates[name] {
			failures = append(failures, "gate failed: "+name)
		}
	}
	return failures
}
