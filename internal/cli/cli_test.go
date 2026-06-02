package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"iterforge/internal/resultlog"
)

// capture runs f with os.Stdout redirected and returns what it printed plus the
// exit code.
func capture(t *testing.T, f func() int) (string, int) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := f()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), code
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

const validPolicy = `primary_metric: accuracy
score_direction: higher_is_better
experiment:
  results_path: logs/results.jsonl
  golden_set_path: evals/golden_set.jsonl
  mutable_paths:
    - candidates/
  frozen_paths:
    - evals/
`

func TestValidatePolicy(t *testing.T) {
	dir := t.TempDir()
	good := writeFile(t, dir, "good.yaml", validPolicy)
	bad := writeFile(t, dir, "bad.yaml", "primary_metric: accuracy\nscore_direction: sideways\n")

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"valid", []string{"-policy", good}, 0},
		{"invalid direction", []string{"-policy", bad}, 1},
		{"missing file", []string{"-policy", filepath.Join(dir, "nope.yaml")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePolicy(tt.args); got != tt.want {
				t.Errorf("ValidatePolicy(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestSummarizeEmpty(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "none.jsonl")
	out, code := capture(t, func() int { return Summarize([]string{"-results", missing}) })
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if want := "No experiment records found."; !strings.Contains(out, want) {
		t.Errorf("output %q missing %q", out, want)
	}
}

func TestSummarizeJSONAndFailedOnly(t *testing.T) {
	dir := t.TempDir()
	log := writeFile(t, dir, "r.jsonl", ""+
		`{"id":"a","score":1,"hard_gates":{"passed":true},"decision":"candidate_pending_review","metrics":{"accuracy":1}}`+"\n"+
		`{"id":"b","score":0,"hard_gates":{"passed":false},"decision":"rejected_hard_gate"}`+"\n"+
		`{"id":"c","score":0,"hard_gates":{"passed":false},"decision":"rejected_evaluator_timeout"}`+"\n")

	// -json over all records.
	out, code := capture(t, func() int { return Summarize([]string{"-results", log, "-json"}) })
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var s struct {
		Records   int            `json:"records"`
		Passed    int            `json:"passed"`
		Failed    int            `json:"failed"`
		Decisions map[string]int `json:"decisions"`
	}
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if s.Records != 3 || s.Passed != 1 || s.Failed != 2 {
		t.Errorf("got records=%d passed=%d failed=%d, want 3/1/2", s.Records, s.Passed, s.Failed)
	}
	if s.Decisions["rejected_hard_gate"] != 1 {
		t.Errorf("decisions = %v", s.Decisions)
	}

	// -failed-only + -json keeps only the two failures.
	out, _ = capture(t, func() int { return Summarize([]string{"-results", log, "-failed-only", "-json"}) })
	_ = json.Unmarshal([]byte(out), &s)
	if s.Records != 2 || s.Passed != 0 {
		t.Errorf("failed-only got records=%d passed=%d, want 2/0", s.Records, s.Passed)
	}
}

func TestCompareExitCodes(t *testing.T) {
	dir := t.TempDir()
	log := writeFile(t, dir, "r.jsonl", ""+
		`{"id":"base","score":0.8,"score_direction":"higher_is_better","hard_gates":{"passed":true},"decision":"candidate_pending_review"}`+"\n"+
		`{"id":"better","score":0.95,"score_direction":"higher_is_better","hard_gates":{"passed":true},"decision":"candidate_pending_review"}`+"\n"+
		`{"id":"worse","score":0.70,"score_direction":"higher_is_better","hard_gates":{"passed":true},"decision":"candidate_pending_review"}`+"\n")
	noPolicy := filepath.Join(dir, "nope.yaml") // forces minDelta=0, direction from records

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"promote", []string{"-results", log, "-policy", noPolicy, "-baseline", "base", "-candidate", "better"}, 0},
		{"regressed", []string{"-results", log, "-policy", noPolicy, "-baseline", "base", "-candidate", "worse"}, 3},
		{"unknown id", []string{"-results", log, "-policy", noPolicy, "-baseline", "base", "-candidate", "ghost"}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, code := capture(t, func() int { return Compare(tt.args) })
			if code != tt.want {
				t.Errorf("Compare(%v) = %d, want %d", tt.args, code, tt.want)
			}
		})
	}
}

// runPolicy writes a policy pointing at temp paths and returns its path plus
// the results-log path. The in-process evaluator scores the main repo's
// candidates.Transform against the golden set, so expected values must be the
// already-normalized form.
func runPolicy(t *testing.T, dir, golden string) (policyPath, resultsPath string) {
	t.Helper()
	resultsPath = filepath.Join(dir, "results.jsonl")
	content := fmt.Sprintf(`primary_metric: accuracy
score_direction: higher_is_better
hard_gates:
  min_accuracy: 0.80
experiment:
  results_path: %s
  golden_set_path: %s
  mutable_paths:
    - %s
  frozen_paths:
    - %s
`, resultsPath, golden, filepath.Join(dir, "candidates"), filepath.Join(dir, "evals"))
	policyPath = writeFile(t, dir, "policy.yaml", content)
	return policyPath, resultsPath
}

func TestRunRecordsResult(t *testing.T) {
	t.Run("passing candidate", func(t *testing.T) {
		dir := t.TempDir()
		golden := writeFile(t, dir, "golden.jsonl", ""+
			`{"input":"  Hello,   WORLD!  ","expected":"hello world"}`+"\n"+
			`{"input":"AI--augmented","expected":"ai augmented"}`+"\n")
		pol, results := runPolicy(t, dir, golden)

		_, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "pass"}) })
		if code != 0 {
			t.Fatalf("Run exit = %d, want 0", code)
		}
		recs, err := resultlog.Read(results)
		if err != nil || len(recs) != 1 {
			t.Fatalf("read %d records (err=%v), want 1", len(recs), err)
		}
		r := recs[0]
		if r.Decision != "candidate_pending_review" {
			t.Errorf("decision = %q, want candidate_pending_review", r.Decision)
		}
		if r.Score != 1 || !r.HardGates.Passed {
			t.Errorf("score=%v passed=%v, want 1/true", r.Score, r.HardGates.Passed)
		}
		if r.Note != "pass" {
			t.Errorf("note = %q, want pass", r.Note)
		}
	})

	t.Run("failing gate", func(t *testing.T) {
		dir := t.TempDir()
		// Expected value the candidate cannot produce -> accuracy 0 < min.
		golden := writeFile(t, dir, "golden.jsonl", `{"input":"abc","expected":"NOT-THE-OUTPUT"}`+"\n")
		pol, results := runPolicy(t, dir, golden)

		_, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "fail"}) })
		if code != 2 {
			t.Fatalf("Run exit = %d, want 2 (gate fail)", code)
		}
		recs, _ := resultlog.Read(results)
		if len(recs) != 1 {
			t.Fatalf("want 1 record, got %d", len(recs))
		}
		if recs[0].Decision != "rejected_hard_gate" {
			t.Errorf("decision = %q, want rejected_hard_gate", recs[0].Decision)
		}
	})
}

func TestRunDetectsGoldenHardcoding(t *testing.T) {
	dir := t.TempDir()
	golden := writeFile(t, dir, "golden.jsonl",
		`{"input":"  Hello,   WORLD!  ","expected":"hello world"}`+"\n")
	pol, results := runPolicy(t, dir, golden)

	// A candidate source under the mutable path that embeds the golden answer.
	if err := os.MkdirAll(filepath.Join(dir, "candidates"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "candidates"), "cheat.go",
		"package candidates\n\nvar memorized = \"hello world\"\n")

	_, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "cheater"}) })
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2 (guardrail)", code)
	}
	recs, _ := resultlog.Read(results)
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	if recs[0].Decision != "rejected_golden_hardcoding" {
		t.Errorf("decision = %q, want rejected_golden_hardcoding", recs[0].Decision)
	}
}

func TestRunDetectsResultsTamper(t *testing.T) {
	dir := t.TempDir()
	golden := writeFile(t, dir, "golden.jsonl",
		`{"input":"  Hello,   WORLD!  ","expected":"hello world"}`+"\n")
	pol, results := runPolicy(t, dir, golden)

	// First run establishes the log and its fingerprint sidecar.
	if _, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "first"}) }); code != 0 {
		t.Fatalf("first run exit = %d, want 0", code)
	}

	// Edit the append-only log out of band.
	f, err := os.OpenFile(results, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"id":"injected","score":99}` + "\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	// Second run must detect the out-of-band edit.
	if _, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "second"}) }); code != 2 {
		t.Fatalf("second run exit = %d, want 2 (tamper)", code)
	}
	recs, _ := resultlog.Read(results)
	last := recs[len(recs)-1]
	if last.Decision != "rejected_results_tampered" {
		t.Errorf("decision = %q, want rejected_results_tampered", last.Decision)
	}
}

// evalPolicy writes a policy that drives an external evaluator command.
func evalPolicy(t *testing.T, dir, evalCmd string, timeoutSecs int) (policyPath, resultsPath string) {
	t.Helper()
	resultsPath = filepath.Join(dir, "results.jsonl")
	content := fmt.Sprintf(`primary_metric: score
score_direction: higher_is_better
hard_gates:
  min_accuracy: 0.80
commands:
  evaluate: %s
experiment:
  timeout_seconds: %d
  results_path: %s
  golden_set_path: %s
  mutable_paths:
    - %s
  frozen_paths:
    - %s
`, evalCmd, timeoutSecs, resultsPath, filepath.Join(dir, "golden.jsonl"),
		filepath.Join(dir, "candidates"), filepath.Join(dir, "evals"))
	policyPath = writeFile(t, dir, "policy.yaml", content)
	return policyPath, resultsPath
}

func TestRunExternalEvaluator(t *testing.T) {
	dir := t.TempDir()
	cmd := `printf '{"score":0.9,"metrics":{"recall":0.9},"gates":{"schema_valid":true}}'`
	pol, results := evalPolicy(t, dir, cmd, 30)

	if _, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "ext"}) }); code != 0 {
		t.Fatalf("Run exit = %d, want 0", code)
	}
	recs, err := resultlog.Read(results)
	if err != nil || len(recs) != 1 {
		t.Fatalf("read %d records (err=%v), want 1", len(recs), err)
	}
	r := recs[0]
	if r.Score != 0.9 {
		t.Errorf("score = %v, want 0.9 (from external evaluator)", r.Score)
	}
	if r.Metrics["recall"] != 0.9 {
		t.Errorf("metrics = %v, want recall=0.9", r.Metrics)
	}
	if !r.Gates["schema_valid"] {
		t.Errorf("gates = %v, want schema_valid=true", r.Gates)
	}
	if r.Decision != "candidate_pending_review" {
		t.Errorf("decision = %q, want candidate_pending_review", r.Decision)
	}
}

func TestRunExternalEvaluatorTimeout(t *testing.T) {
	dir := t.TempDir()
	pol, results := evalPolicy(t, dir, `sleep 2; printf '{"score":1}'`, 1)

	if _, code := capture(t, func() int { return Run([]string{"-policy", pol, "-note", "slow"}) }); code != 2 {
		t.Fatalf("Run exit = %d, want 2 (timeout)", code)
	}
	recs, _ := resultlog.Read(results)
	if len(recs) != 1 || recs[0].Decision != "rejected_evaluator_timeout" {
		t.Fatalf("got %d records, decision=%q; want 1 rejected_evaluator_timeout", len(recs), func() string {
			if len(recs) > 0 {
				return recs[0].Decision
			}
			return ""
		}())
	}
}

func TestRunDetectsFrozenChange(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	for _, args := range [][]string{{"init"}, {"config", "user.email", "t@t"}, {"config", "user.name", "t"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Project with relative paths (so they match git's relative output).
	writeFile(t, dir, "golden.jsonl", `{"input":"  Hello,   WORLD!  ","expected":"hello world"}`+"\n")
	if err := os.MkdirAll(filepath.Join(dir, "candidates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "evals"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A change under the frozen path (untracked file shows in git status).
	writeFile(t, filepath.Join(dir, "evals"), "marker.txt", "touched\n")
	writeFile(t, dir, "policy.yaml", `primary_metric: accuracy
score_direction: higher_is_better
hard_gates:
  min_accuracy: 0.80
experiment:
  results_path: results.jsonl
  golden_set_path: golden.jsonl
  mutable_paths:
    - candidates/
  frozen_paths:
    - evals/
`)

	// gitmeta runs in the working directory.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	_, code := capture(t, func() int { return Run([]string{"-policy", "policy.yaml", "-note", "frozen"}) })
	if code != 2 {
		t.Fatalf("Run exit = %d, want 2 (frozen change)", code)
	}
	recs, _ := resultlog.Read("results.jsonl")
	if len(recs) != 1 || recs[0].Decision != "rejected_frozen_change" {
		t.Fatalf("decision = %q (records=%d), want rejected_frozen_change",
			func() string {
				if len(recs) > 0 {
					return recs[0].Decision
				}
				return ""
			}(), len(recs))
	}
}

func TestBadFlagReturnsCodeNotExit(t *testing.T) {
	// Each command must return an exit code for an unknown flag rather than
	// calling os.Exit (which would kill the test process).
	cmds := map[string]func([]string) int{
		"validate-policy": ValidatePolicy,
		"run":             Run,
		"summarize":       Summarize,
		"compare":         Compare,
		"init":            InitProject,
	}
	for name, fn := range cmds {
		t.Run(name, func(t *testing.T) {
			if code := fn([]string{"-no-such-flag"}); code != 2 {
				t.Errorf("%s(-no-such-flag) = %d, want 2", name, code)
			}
		})
	}
}

func TestHelpFlagExitsZero(t *testing.T) {
	if code := Summarize([]string{"-h"}); code != 0 {
		t.Errorf("summarize -h = %d, want 0", code)
	}
}

func TestCompareTooFewRecords(t *testing.T) {
	dir := t.TempDir()
	log := writeFile(t, dir, "r.jsonl", `{"id":"only","score":1,"hard_gates":{"passed":true}}`+"\n")
	_, code := capture(t, func() int { return Compare([]string{"-results", log}) })
	if code != 1 {
		t.Errorf("exit = %d, want 1 (need >= 2 records)", code)
	}
}
