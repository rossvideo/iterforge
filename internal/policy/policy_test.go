package policy

import (
	"os"
	"path/filepath"
	"testing"
)

// validPolicy returns a Policy that passes Validate; tests mutate one field.
func validPolicy() Policy {
	p := Default()
	p.MutablePaths = []string{"candidates/"}
	p.FrozenPaths = []string{"evals/"}
	return p
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Policy)
		wantErr bool
	}{
		{"valid", func(*Policy) {}, false},
		{"valid lower_is_better", func(p *Policy) { p.ScoreDirection = LowerIsBetter }, false},
		{"empty primary_metric", func(p *Policy) { p.PrimaryMetric = "" }, true},
		{"whitespace primary_metric", func(p *Policy) { p.PrimaryMetric = "  " }, true},
		{"bad score_direction", func(p *Policy) { p.ScoreDirection = "bigger" }, true},
		{"empty score_direction", func(p *Policy) { p.ScoreDirection = "" }, true},
		{"negative minimum_delta", func(p *Policy) { p.MinimumDelta = -0.1 }, true},
		{"empty results_path", func(p *Policy) { p.ResultsPath = "" }, true},
		{"empty golden_set_path", func(p *Policy) { p.GoldenSetPath = "" }, true},
		{"no mutable paths", func(p *Policy) { p.MutablePaths = nil }, true},
		{"no frozen paths", func(p *Policy) { p.FrozenPaths = nil }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validPolicy()
			tt.mutate(&p)
			err := p.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestLoadParsesPathsAndDirection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := `primary_metric: accuracy
score_direction: lower_is_better
minimum_delta: 0.002
hard_gates:
  min_accuracy: 0.9
  max_failures: 1
experiment:
  results_path: logs/results.jsonl
  golden_set_path: evals/golden_set.jsonl
  mutable_paths:
    - candidates/
    - logs/agent_journal.md
  frozen_paths:
    - evals/
    - policy.yaml
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.ScoreDirection != LowerIsBetter {
		t.Errorf("ScoreDirection = %q, want %q", p.ScoreDirection, LowerIsBetter)
	}
	if p.MinimumDelta != 0.002 {
		t.Errorf("MinimumDelta = %g, want 0.002", p.MinimumDelta)
	}
	if len(p.MutablePaths) != 2 {
		t.Errorf("MutablePaths = %v, want 2 items", p.MutablePaths)
	}
	if len(p.FrozenPaths) != 2 {
		t.Errorf("FrozenPaths = %v, want 2 items", p.FrozenPaths)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("loaded policy should be valid, got %v", err)
	}
}

func TestLoadParsesEvaluateCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "primary_metric: accuracy\ncommands:\n  evaluate: go run ./cmd/myeval\nexperiment:\n  results_path: logs/results.jsonl\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if p.EvaluateCommand != "go run ./cmd/myeval" {
		t.Errorf("EvaluateCommand = %q, want %q", p.EvaluateCommand, "go run ./cmd/myeval")
	}
}

func TestLoadRejectsBadNumber(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	content := "primary_metric: accuracy\nminimum_delta: notanumber\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for non-numeric minimum_delta, got nil")
	}
}
