package guardrails

import (
	"reflect"
	"testing"
)

func TestGoldenHardcoding(t *testing.T) {
	expected := []string{"hello world", "ross video research labs", "ai"}
	tests := []struct {
		name    string
		sources map[string]string
		want    []Finding
	}{
		{
			name:    "clean candidate",
			sources: map[string]string{"candidates/candidate.go": "return strings.ToLower(in)"},
			want:    nil,
		},
		{
			name:    "double-quoted literal flagged",
			sources: map[string]string{"candidates/candidate.go": `if in == x { return "hello world" }`},
			want:    []Finding{{File: "candidates/candidate.go", Value: "hello world"}},
		},
		{
			name:    "backtick literal flagged",
			sources: map[string]string{"candidates/candidate.go": "return `ross video research labs`"},
			want:    []Finding{{File: "candidates/candidate.go", Value: "ross video research labs"}},
		},
		{
			name:    "short value ignored",
			sources: map[string]string{"candidates/candidate.go": `return "ai"`},
			want:    nil,
		},
		{
			name:    "substring without quotes not flagged",
			sources: map[string]string{"candidates/candidate.go": "// hello world is a comment"},
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GoldenHardcoding(tt.sources, expected)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GoldenHardcoding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrozenViolations(t *testing.T) {
	frozen := []string{
		"evals/",
		"cmd/**",
		"internal/",
		"policy.yaml",
		"go.sum",
		"testdata/*.json",
	}
	tests := []struct {
		name    string
		changed []string
		want    []string
	}{
		{
			name:    "candidate change is allowed",
			changed: []string{"candidates/candidate.go", "logs/agent_journal.md"},
			want:    nil,
		},
		{
			name:    "evaluator change is frozen",
			changed: []string{"evals/evaluator.go"},
			want:    []string{"evals/evaluator.go"},
		},
		{
			name:    "double-star dir match",
			changed: []string{"cmd/runexp/main.go"},
			want:    []string{"cmd/runexp/main.go"},
		},
		{
			name:    "exact file match",
			changed: []string{"policy.yaml"},
			want:    []string{"policy.yaml"},
		},
		{
			name:    "exact file does not match sibling prefix",
			changed: []string{"policy.yaml.bak"},
			want:    nil,
		},
		{
			name:    "dir prefix without trailing slash",
			changed: []string{"internal/policy/policy.go"},
			want:    []string{"internal/policy/policy.go"},
		},
		{
			name:    "glob match",
			changed: []string{"testdata/golden.json"},
			want:    []string{"testdata/golden.json"},
		},
		{
			name:    "leading ./ is normalized",
			changed: []string{"./evals/evaluator.go"},
			want:    []string{"evals/evaluator.go"},
		},
		{
			name:    "mixed allowed and frozen",
			changed: []string{"candidates/x.go", "go.sum", "evals/golden_set.jsonl"},
			want:    []string{"go.sum", "evals/golden_set.jsonl"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FrozenViolations(tt.changed, frozen)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FrozenViolations() = %v, want %v", got, tt.want)
			}
		})
	}
}
