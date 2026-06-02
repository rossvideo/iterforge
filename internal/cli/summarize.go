package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"time"

	"iterforge/internal/resultlog"
)

// summary is the machine-readable form emitted by -json.
type summary struct {
	Records        int            `json:"records"`
	Passed         int            `json:"passed"`
	Failed         int            `json:"failed"`
	ScoreDirection string         `json:"score_direction"`
	Latest         brief          `json:"latest"`
	Best           brief          `json:"best"`
	Decisions      map[string]int `json:"decisions,omitempty"`
}

// brief is the JSON projection of one record.
type brief struct {
	ID           string             `json:"id,omitempty"`
	TimestampUTC string             `json:"timestamp_utc,omitempty"`
	Note         string             `json:"note"`
	Score        float64            `json:"score"`
	GatesPassed  bool               `json:"gates_passed"`
	Decision     string             `json:"decision,omitempty"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
}

func toBrief(r resultlog.Record) brief {
	return brief{
		ID:           r.ID,
		TimestampUTC: r.TimestampUTC,
		Note:         r.Note,
		Score:        r.Score,
		GatesPassed:  r.HardGates.Passed,
		Decision:     r.Decision,
		Metrics:      r.Metrics,
	}
}

// Summarize reports campaign-level statistics over the results log.
func Summarize(args []string) int {
	fs := flag.NewFlagSet("summarize", flag.ExitOnError)
	path := fs.String("results", "logs/results.jsonl", "path to results.jsonl")
	last := fs.Int("last", 0, "only summarize the most recent N runs (0 = all)")
	failedOnly := fs.Bool("failed-only", false, "only summarize runs that did not pass gates")
	since := fs.String("since", "", "only summarize runs at or after this RFC3339 time")
	jsonOut := fs.Bool("json", false, "emit the summary as JSON")
	_ = fs.Parse(args)

	var sinceTime time.Time
	if *since != "" {
		t, err := time.Parse(time.RFC3339, *since)
		if err != nil {
			return errExit(fmt.Errorf("invalid -since %q: want RFC3339 (e.g. 2026-06-02T00:00:00Z)", *since))
		}
		sinceTime = t
	}

	records, err := resultlog.Read(*path)
	if err != nil {
		return errExit(err)
	}
	if len(records) == 0 {
		fmt.Println("No experiment records found.")
		return 0
	}

	// Filter by time first, then window to the most recent N, then failures.
	if !sinceTime.IsZero() {
		kept := make([]resultlog.Record, 0, len(records))
		for _, r := range records {
			ts, err := time.Parse(time.RFC3339, r.TimestampUTC)
			if err == nil && !ts.Before(sinceTime) {
				kept = append(kept, r)
			}
		}
		records = kept
	}
	if *last > 0 && len(records) > *last {
		records = records[len(records)-*last:]
	}
	if *failedOnly {
		kept := make([]resultlog.Record, 0, len(records))
		for _, r := range records {
			if !r.HardGates.Passed {
				kept = append(kept, r)
			}
		}
		records = kept
	}
	if len(records) == 0 {
		if *jsonOut {
			fmt.Println(`{"records":0}`)
		} else {
			fmt.Println("No matching records (filters excluded every run).")
		}
		return 0
	}

	// Score direction comes from the records themselves; fall back to higher.
	direction := "higher_is_better"
	for i := len(records) - 1; i >= 0; i-- {
		if records[i].ScoreDirection != "" {
			direction = records[i].ScoreDirection
			break
		}
	}

	latest := records[len(records)-1]
	best, _ := resultlog.Best(records, direction)

	passed, failed := 0, 0
	decisions := map[string]int{}
	for _, r := range records {
		if r.HardGates.Passed {
			passed++
		} else {
			failed++
		}
		if r.Decision != "" {
			decisions[r.Decision]++
		}
	}

	if *jsonOut {
		b, err := json.MarshalIndent(summary{
			Records:        len(records),
			Passed:         passed,
			Failed:         failed,
			ScoreDirection: direction,
			Latest:         toBrief(latest),
			Best:           toBrief(best),
			Decisions:      decisions,
		}, "", "  ")
		if err != nil {
			return errExit(err)
		}
		fmt.Println(string(b))
		return 0
	}

	fmt.Println("# Experiment Summary")
	fmt.Printf("\nRecords: %d\n", len(records))
	fmt.Printf("Score direction: %s\n", direction)
	fmt.Printf("Gates passed/failed: %d/%d\n", passed, failed)
	fmt.Printf("\nLatest score: %.4f (note=%q, gates=%t)%s\n", latest.Score, latest.Note, latest.HardGates.Passed, formatMetrics(latest.Metrics))
	fmt.Printf("Best score:   %.4f (note=%q, gates=%t, ts=%s)%s\n", best.Score, best.Note, best.HardGates.Passed, best.TimestampUTC, formatMetrics(best.Metrics))

	if len(decisions) > 0 {
		fmt.Println("\n## Decisions")
		names := make([]string, 0, len(decisions))
		for d := range decisions {
			names = append(names, d)
		}
		// Order by count descending, then name for stable ties.
		sort.Slice(names, func(i, j int) bool {
			if decisions[names[i]] != decisions[names[j]] {
				return decisions[names[i]] > decisions[names[j]]
			}
			return names[i] < names[j]
		})
		for _, d := range names {
			fmt.Printf("%4d  %s\n", decisions[d], d)
		}
	}

	// Top runs ordered by the configured direction.
	ordered := make([]resultlog.Record, len(records))
	copy(ordered, records)
	sort.SliceStable(ordered, func(i, j int) bool {
		if direction == "lower_is_better" {
			return ordered[i].Score < ordered[j].Score
		}
		return ordered[i].Score > ordered[j].Score
	})

	fmt.Println("\n## Top Runs")
	limit := 5
	if len(ordered) < limit {
		limit = len(ordered)
	}
	for i := 0; i < limit; i++ {
		r := ordered[i]
		ts := r.TimestampUTC
		if parsed, err := time.Parse(time.RFC3339, r.TimestampUTC); err == nil {
			ts = parsed.Format("2006-01-02 15:04:05Z")
		}
		fmt.Printf("%d. score=%.4f gates=%t latency_ms=%.3f failed=%d ts=%s note=%q\n",
			i+1, r.Score, r.HardGates.Passed, r.Eval.LatencyMS, r.Eval.Failed, ts, r.Note)
	}

	if failed > 0 {
		fmt.Println("\n## Failure Reasons")
		for _, r := range records {
			if !r.HardGates.Passed && r.FailureReason != "" {
				fmt.Printf("- %s: %s\n", r.Note, r.FailureReason)
			}
		}
	}
	return 0
}

// formatMetrics renders a record's metrics map deterministically, e.g.
// "  [accuracy=1.0000]". Empty maps render as nothing.
func formatMetrics(m map[string]float64) string {
	if len(m) == 0 {
		return ""
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	out := "  ["
	for i, name := range names {
		if i > 0 {
			out += " "
		}
		out += fmt.Sprintf("%s=%.4f", name, m[name])
	}
	return out + "]"
}
