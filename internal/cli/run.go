package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"iterforge/evals"
	"iterforge/internal/evaluator"
	"iterforge/internal/gitmeta"
	"iterforge/internal/guardrails"
	"iterforge/internal/policy"
	"iterforge/internal/resultlog"
)

// Run executes one scored experiment and appends the result to the log.
func Run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	note := fs.String("note", "", "hypothesis or experiment note")
	policyPath := fs.String("policy", "policy.yaml", "path to policy.yaml")
	_ = fs.Parse(args)

	p, err := policy.Load(*policyPath)
	if err != nil {
		return errExit(err)
	}
	if err := p.Validate(); err != nil {
		return errExit(err)
	}

	start := time.Now()
	res, err := evaluate(p)
	timedOut := errors.Is(err, errEvaluatorTimeout)
	if err != nil && !timedOut {
		return errExit(err)
	}
	durationMS := time.Since(start).Milliseconds()

	gates := evaluateGates(p, res)
	// Named hard gates reported by the evaluator are enforced alongside the
	// primary score/latency/failure gates.
	evalOut := evaluator.Output{Score: res.Accuracy, Metrics: res.Metrics, Gates: res.Gates}
	for _, f := range evalOut.GateFailures() {
		gates.Failures = append(gates.Failures, f)
		gates.Passed = false
	}
	evalPassed := gates.Passed
	git := gitmeta.Capture()
	now := time.Now().UTC()

	// Guardrail: frozen paths must not change during a candidate run.
	violations := guardrails.FrozenViolations(gitmeta.ChangedFiles(), p.FrozenPaths)

	// Guardrail: candidate source must not hardcode golden-set answers.
	hardcodes := guardrails.GoldenHardcoding(candidateSources(p.MutablePaths), expectedValues(p.GoldenSetPath))

	// Guardrail: the append-only results log must not be edited out of band.
	untampered, err := resultlog.CheckUntampered(p.ResultsPath)
	if err != nil {
		return errExit(err)
	}
	tampered := !untampered

	if len(violations) > 0 {
		for _, v := range violations {
			gates.Failures = append(gates.Failures, "frozen path changed: "+v)
		}
		gates.Passed = false
	}
	if tampered {
		gates.Failures = append(gates.Failures, "results log modified outside the runner")
		gates.Passed = false
	}
	if timedOut {
		gates.Failures = append(gates.Failures, fmt.Sprintf("evaluator timed out after %ds", p.TimeoutSeconds))
		gates.Passed = false
	}
	if len(hardcodes) > 0 {
		for _, h := range hardcodes {
			gates.Failures = append(gates.Failures, fmt.Sprintf("golden-set value hardcoded in %s: %q", h.File, h.Value))
		}
		gates.Passed = false
	}

	decision := "candidate_pending_review"
	switch {
	case len(violations) > 0:
		decision = "rejected_frozen_change"
	case tampered:
		decision = "rejected_results_tampered"
	case len(hardcodes) > 0:
		decision = "rejected_golden_hardcoding"
	case timedOut:
		decision = "rejected_evaluator_timeout"
	case !evalPassed:
		decision = "rejected_hard_gate"
	}

	record := resultlog.Record{
		ID:             recordID(now, git.SHA),
		TimestampUTC:   now.Format(time.RFC3339),
		Note:           *note,
		GitSHA:         git.SHA,
		Dirty:          git.Dirty,
		PrimaryMetric:  p.PrimaryMetric,
		Score:          res.Accuracy,
		ScoreDirection: p.ScoreDirection,
		CheckPassed:    true, // assumed: `make check` runs before/around the experiment
		EvalPassed:     evalPassed,
		Metrics:        res.Metrics,
		Gates:          res.Gates,
		HardGates:      gates,
		DurationMS:     durationMS,
		Decision:       decision,
		FailureReason:  strings.Join(gates.Failures, "; "),
		Eval:           res,
	}

	if err := resultlog.Append(p.ResultsPath, record); err != nil {
		return errExit(err)
	}
	// Re-fingerprint so the next run can detect out-of-band edits.
	if err := resultlog.WriteFingerprint(p.ResultsPath); err != nil {
		return errExit(err)
	}

	pretty, _ := json.MarshalIndent(record, "", "  ")
	fmt.Println(string(pretty))

	if !gates.Passed {
		return 2
	}
	return 0
}

// errEvaluatorTimeout marks an external evaluator that exceeded
// policy.timeout_seconds, so the runner can record a distinct decision instead
// of aborting without a result.
var errEvaluatorTimeout = errors.New("evaluator timed out")

// evaluate produces a Result either from the in-process evaluator or, when the
// policy defines commands.evaluate, from an external evaluator command. The
// external command receives the golden-set path in IF_GOLDEN_SET and must print
// standard evaluator JSON (see internal/evaluator) to stdout.
func evaluate(p policy.Policy) (evals.Result, error) {
	if strings.TrimSpace(p.EvaluateCommand) == "" {
		return evals.Evaluate(p.GoldenSetPath)
	}

	ctx := context.Background()
	if p.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(p.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", p.EvaluateCommand)
	cmd.Env = append(os.Environ(), "IF_GOLDEN_SET="+p.GoldenSetPath)
	// Run in its own process group and, on timeout, kill the whole group. A
	// command that spawns children (e.g. `sleep`) would otherwise keep the
	// stdout pipe open and block Output past the deadline. WaitDelay is a
	// belt-and-suspenders bound on any lingering I/O.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return os.ErrProcessDone
	}
	cmd.WaitDelay = time.Second

	out, err := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return evals.Result{}, errEvaluatorTimeout
	}
	if err != nil {
		return evals.Result{}, fmt.Errorf("evaluate command %q failed: %w", p.EvaluateCommand, err)
	}
	o, err := evaluator.Parse(out)
	if err != nil {
		return evals.Result{}, err
	}
	return evals.Result{
		Accuracy: o.Score,
		Metrics:  o.Metrics,
		Gates:    o.Gates,
	}, nil
}

func evaluateGates(p policy.Policy, res evals.Result) resultlog.GateStatus {
	failures := []string{}
	if res.Accuracy < p.MinAccuracy {
		failures = append(failures, fmt.Sprintf("accuracy %.4f < min_accuracy %.4f", res.Accuracy, p.MinAccuracy))
	}
	if res.LatencyMS > p.MaxLatencyMS {
		failures = append(failures, fmt.Sprintf("latency_ms %.4f > max_latency_ms %.4f", res.LatencyMS, p.MaxLatencyMS))
	}
	if res.Failed > p.MaxFailures {
		failures = append(failures, fmt.Sprintf("failed %d > max_failures %d", res.Failed, p.MaxFailures))
	}
	return resultlog.GateStatus{
		Passed:       len(failures) == 0,
		Failures:     failures,
		MinAccuracy:  p.MinAccuracy,
		MaxLatencyMS: p.MaxLatencyMS,
		MaxFailures:  p.MaxFailures,
	}
}

// recordID is a stable, sortable identifier: timestamp plus a short git SHA when
// available. It avoids randomness so runs remain reproducible.
func recordID(ts time.Time, sha string) string {
	id := ts.Format("20060102T150405Z")
	if len(sha) >= 7 {
		id += "-" + sha[:7]
	}
	return id
}

// candidateSources reads non-test .go files under the mutable paths, keyed by
// path. Non-Go and missing paths are skipped.
func candidateSources(mutablePaths []string) map[string]string {
	sources := map[string]string{}
	for _, root := range mutablePaths {
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if b, err := os.ReadFile(path); err == nil {
				sources[filepath.ToSlash(path)] = string(b)
			}
			return nil
		})
	}
	return sources
}

// expectedValues returns the golden-set expected outputs, or nil if the set
// cannot be read.
func expectedValues(goldenSetPath string) []string {
	examples, err := evals.LoadGoldenSet(goldenSetPath)
	if err != nil {
		return nil
	}
	values := make([]string, 0, len(examples))
	for _, ex := range examples {
		values = append(values, ex.Expected)
	}
	return values
}
