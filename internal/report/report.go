// Package report compares experiment results to support keep/revert decisions.
package report

import "iterforge/internal/resultlog"

// Comparison is a baseline-vs-candidate diff with a recommendation.
type Comparison struct {
	Baseline        resultlog.Record
	Candidate       resultlog.Record
	Direction       string
	ScoreDelta      float64
	AccuracyDelta   float64
	LatencyDeltaMS  float64
	FailedDelta     int
	DurationDeltaMS int64
	Improved        bool
	Regressed       bool
	Recommendation  string
}

// FindByID returns the record with the given id.
func FindByID(records []resultlog.Record, id string) (resultlog.Record, bool) {
	for _, r := range records {
		if r.ID == id {
			return r, true
		}
	}
	return resultlog.Record{}, false
}

// Compare diffs candidate against baseline. minDelta is the smallest score
// change treated as meaningful (from policy). The recommendation favors safety:
// a candidate that failed gates is never promoted regardless of score.
func Compare(baseline, candidate resultlog.Record, direction string, minDelta float64) Comparison {
	scoreDelta := candidate.Score - baseline.Score

	improved := scoreDelta >= minDelta
	regressed := scoreDelta <= -minDelta
	if direction == "lower_is_better" {
		improved = scoreDelta <= -minDelta
		regressed = scoreDelta >= minDelta
	}

	rec := ""
	switch {
	case candidate.Decision != "" && candidate.Decision != "candidate_pending_review":
		rec = "reject candidate: " + candidate.Decision
	case !candidate.HardGates.Passed:
		rec = "reject candidate: hard gates failed"
	case improved:
		rec = "promote candidate"
	case regressed:
		rec = "keep baseline: candidate regressed"
	default:
		rec = "no change: within minimum_delta, keep baseline"
	}

	return Comparison{
		Baseline:        baseline,
		Candidate:       candidate,
		Direction:       direction,
		ScoreDelta:      scoreDelta,
		AccuracyDelta:   candidate.Eval.Accuracy - baseline.Eval.Accuracy,
		LatencyDeltaMS:  candidate.Eval.LatencyMS - baseline.Eval.LatencyMS,
		FailedDelta:     candidate.Eval.Failed - baseline.Eval.Failed,
		DurationDeltaMS: candidate.DurationMS - baseline.DurationMS,
		Improved:        improved,
		Regressed:       regressed,
		Recommendation:  rec,
	}
}
