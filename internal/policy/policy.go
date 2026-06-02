package policy

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Valid score directions.
const (
	HigherIsBetter = "higher_is_better"
	LowerIsBetter  = "lower_is_better"
)

type Policy struct {
	PrimaryMetric   string
	ScoreDirection  string
	MinimumDelta    float64
	MinAccuracy     float64
	MaxLatencyMS    float64
	MaxFailures     int
	ResultsPath     string
	GoldenSetPath   string
	TimeoutSeconds  int
	MutablePaths    []string
	FrozenPaths     []string
	EvaluateCommand string
}

func Default() Policy {
	return Policy{
		PrimaryMetric:  "accuracy",
		ScoreDirection: HigherIsBetter,
		MinimumDelta:   0.001,
		MinAccuracy:    0.80,
		MaxLatencyMS:   100,
		MaxFailures:    2,
		ResultsPath:    "logs/results.jsonl",
		GoldenSetPath:  "evals/golden_set.jsonl",
		TimeoutSeconds: 30,
	}
}

// Load reads the small policy.yaml subset used by this starter kit.
// It intentionally avoids external dependencies to keep the harness portable.
//
// Parse failures (e.g. a non-numeric minimum_delta) return an actionable
// error rather than panicking, so callers and the validator can report them.
func Load(path string) (Policy, error) {
	p := Default()
	file, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer file.Close()

	var parseErr error
	float := func(key, val string) float64 {
		f, err := strconv.ParseFloat(val, 64)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("invalid number for %q: %q", key, val)
		}
		return f
	}
	integer := func(key, val string) int {
		i, err := strconv.Atoi(val)
		if err != nil && parseErr == nil {
			parseErr = fmt.Errorf("invalid integer for %q: %q", key, val)
		}
		return i
	}

	scanner := bufio.NewScanner(file)
	section := ""
	var list *[]string // active list being populated (mutable/frozen paths)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") || line == "-" {
			if list != nil {
				item := strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "-")), "\"")
				if item != "" {
					*list = append(*list, item)
				}
			}
			continue
		}
		// Any non-list line ends the current list block.
		list = nil
		if strings.HasSuffix(line, ":") {
			name := strings.TrimSuffix(line, ":")
			switch name {
			case "mutable_paths", "editable_paths":
				list = &p.MutablePaths
			case "frozen_paths":
				list = &p.FrozenPaths
			default:
				section = name
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		qualified := key
		if section != "" {
			qualified = section + "." + key
		}
		switch qualified {
		case "primary_metric":
			p.PrimaryMetric = val
		case "score_direction":
			p.ScoreDirection = val
		case "minimum_delta":
			p.MinimumDelta = float(key, val)
		case "hard_gates.min_accuracy":
			p.MinAccuracy = float(key, val)
		case "hard_gates.max_latency_ms":
			p.MaxLatencyMS = float(key, val)
		case "hard_gates.max_failures":
			p.MaxFailures = integer(key, val)
		case "experiment.timeout_seconds":
			p.TimeoutSeconds = integer(key, val)
		case "experiment.results_path":
			p.ResultsPath = val
		case "experiment.golden_set_path":
			p.GoldenSetPath = val
		case "commands.evaluate":
			p.EvaluateCommand = val
		}
	}
	if err := scanner.Err(); err != nil {
		return p, err
	}
	if parseErr != nil {
		return p, parseErr
	}
	return p, nil
}

// Validate checks that the policy is complete and internally consistent.
// It returns a single error describing every problem found so the operator
// can fix them in one pass.
func (p Policy) Validate() error {
	var problems []string
	if strings.TrimSpace(p.PrimaryMetric) == "" {
		problems = append(problems, "primary_metric must not be empty")
	}
	switch p.ScoreDirection {
	case HigherIsBetter, LowerIsBetter:
	default:
		problems = append(problems, fmt.Sprintf(
			"score_direction must be %q or %q, got %q",
			HigherIsBetter, LowerIsBetter, p.ScoreDirection))
	}
	if p.MinimumDelta < 0 {
		problems = append(problems, fmt.Sprintf("minimum_delta must be >= 0, got %g", p.MinimumDelta))
	}
	if strings.TrimSpace(p.ResultsPath) == "" {
		problems = append(problems, "experiment.results_path must not be empty")
	}
	if strings.TrimSpace(p.GoldenSetPath) == "" {
		problems = append(problems, "experiment.golden_set_path must not be empty")
	}
	if len(p.MutablePaths) == 0 {
		problems = append(problems, "at least one mutable path is required (experiment.mutable_paths)")
	}
	if len(p.FrozenPaths) == 0 {
		problems = append(problems, "at least one frozen path is required (experiment.frozen_paths)")
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid policy:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}
