package report

import (
	"testing"

	"iterforge/internal/resultlog"
)

func passing(id string, score float64) resultlog.Record {
	return resultlog.Record{
		ID:        id,
		Score:     score,
		Decision:  "candidate_pending_review",
		HardGates: resultlog.GateStatus{Passed: true},
	}
}

func TestFindByID(t *testing.T) {
	records := []resultlog.Record{passing("a", 1), passing("b", 2)}
	if r, ok := FindByID(records, "b"); !ok || r.Score != 2 {
		t.Errorf("FindByID(b) = %+v, ok=%v", r, ok)
	}
	if _, ok := FindByID(records, "missing"); ok {
		t.Error("FindByID(missing) should be false")
	}
}

func TestCompareRecommendation(t *testing.T) {
	tests := []struct {
		name      string
		baseline  resultlog.Record
		candidate resultlog.Record
		direction string
		minDelta  float64
		want      string
	}{
		{
			name:      "higher improved",
			baseline:  passing("b", 0.80),
			candidate: passing("c", 0.90),
			direction: "higher_is_better",
			minDelta:  0.01,
			want:      "promote candidate",
		},
		{
			name:      "higher regressed",
			baseline:  passing("b", 0.90),
			candidate: passing("c", 0.80),
			direction: "higher_is_better",
			minDelta:  0.01,
			want:      "keep baseline: candidate regressed",
		},
		{
			name:      "within delta",
			baseline:  passing("b", 0.900),
			candidate: passing("c", 0.9005),
			direction: "higher_is_better",
			minDelta:  0.01,
			want:      "no change: within minimum_delta, keep baseline",
		},
		{
			name:      "lower is better improved",
			baseline:  passing("b", 0.50),
			candidate: passing("c", 0.20),
			direction: "lower_is_better",
			minDelta:  0.01,
			want:      "promote candidate",
		},
		{
			name:      "rejected candidate never promoted",
			baseline:  passing("b", 0.80),
			candidate: resultlog.Record{ID: "c", Score: 0.99, Decision: "rejected_frozen_change", HardGates: resultlog.GateStatus{Passed: false}},
			direction: "higher_is_better",
			minDelta:  0.01,
			want:      "reject candidate: rejected_frozen_change",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compare(tt.baseline, tt.candidate, tt.direction, tt.minDelta)
			if got.Recommendation != tt.want {
				t.Errorf("Recommendation = %q, want %q", got.Recommendation, tt.want)
			}
		})
	}
}
