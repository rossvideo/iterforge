package cli

import (
	"flag"
	"fmt"

	"iterforge/internal/policy"
	"iterforge/internal/report"
	"iterforge/internal/resultlog"
)

// Compare diffs two runs and recommends keep vs promote. It returns 0 only when
// the recommendation is to promote, else 3 (or 1 on error), so scripts can gate.
func Compare(args []string) int {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	resultsPath := fs.String("results", "logs/results.jsonl", "path to results.jsonl")
	policyPath := fs.String("policy", "policy.yaml", "path to policy.yaml")
	baselineID := fs.String("baseline", "", "baseline record id (default: best run)")
	candidateID := fs.String("candidate", "", "candidate record id (default: latest run)")
	_ = fs.Parse(args)

	records, err := resultlog.Read(*resultsPath)
	if err != nil {
		return errExit(err)
	}
	if len(records) < 2 {
		return errExit(fmt.Errorf("need at least 2 records to compare, have %d", len(records)))
	}

	// Policy is authoritative for direction and minimum delta; fall back to the
	// log's own direction if the policy cannot be read.
	direction := "higher_is_better"
	minDelta := 0.0
	if p, perr := policy.Load(*policyPath); perr == nil {
		direction = p.ScoreDirection
		minDelta = p.MinimumDelta
	} else {
		for i := len(records) - 1; i >= 0; i-- {
			if records[i].ScoreDirection != "" {
				direction = records[i].ScoreDirection
				break
			}
		}
	}

	baseline, err := resolve(records, *baselineID, direction, true)
	if err != nil {
		return errExit(err)
	}
	candidate, err := resolve(records, *candidateID, direction, false)
	if err != nil {
		return errExit(err)
	}

	c := report.Compare(baseline, candidate, direction, minDelta)
	printComparison(c)

	if c.Recommendation != "promote candidate" {
		return 3
	}
	return 0
}

// resolve picks a record by id, or defaults to best (baseline) / latest (candidate).
func resolve(records []resultlog.Record, id, direction string, isBaseline bool) (resultlog.Record, error) {
	if id != "" {
		r, ok := report.FindByID(records, id)
		if !ok {
			return resultlog.Record{}, fmt.Errorf("no record with id %q", id)
		}
		return r, nil
	}
	if isBaseline {
		best, _ := resultlog.Best(records, direction)
		return best, nil
	}
	return records[len(records)-1], nil
}

func printComparison(c report.Comparison) {
	fmt.Println("# Comparison")
	fmt.Printf("\nDirection: %s\n", c.Direction)
	fmt.Printf("Baseline:  %s  score=%.4f  note=%q  gates=%t\n",
		c.Baseline.ID, c.Baseline.Score, c.Baseline.Note, c.Baseline.HardGates.Passed)
	fmt.Printf("Candidate: %s  score=%.4f  note=%q  gates=%t  decision=%s\n",
		c.Candidate.ID, c.Candidate.Score, c.Candidate.Note, c.Candidate.HardGates.Passed, c.Candidate.Decision)
	fmt.Println("\n## Deltas (candidate - baseline)")
	fmt.Printf("score:       %+.4f\n", c.ScoreDelta)
	fmt.Printf("accuracy:    %+.4f\n", c.AccuracyDelta)
	fmt.Printf("latency_ms:  %+.3f\n", c.LatencyDeltaMS)
	fmt.Printf("failed:      %+d\n", c.FailedDelta)
	fmt.Printf("duration_ms: %+d\n", c.DurationDeltaMS)
	if c.Candidate.FailureReason != "" {
		fmt.Printf("\nCandidate failures: %s\n", c.Candidate.FailureReason)
	}
	fmt.Printf("\n## Recommendation\n%s\n", c.Recommendation)
}
