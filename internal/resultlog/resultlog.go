// Package resultlog defines the append-only experiment result contract.
//
// It is the single source of truth for the JSONL record schema shared by the
// runner (which appends) and the summarizer (which reads). Keeping one struct
// here prevents the two commands from drifting into incompatible formats.
package resultlog

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"iterforge/evals"
)

// GateStatus is the hard-gate outcome for one experiment.
type GateStatus struct {
	Passed       bool     `json:"passed"`
	Failures     []string `json:"failures"`
	MinAccuracy  float64  `json:"min_accuracy,omitempty"`
	MaxLatencyMS float64  `json:"max_latency_ms,omitempty"`
	MaxFailures  int      `json:"max_failures,omitempty"`
}

// Record is one experiment result. JSON field names are stable; new fields
// must be additive so old logs remain readable.
type Record struct {
	ID             string             `json:"id"`
	TimestampUTC   string             `json:"timestamp_utc"`
	Note           string             `json:"note"`
	GitSHA         string             `json:"git_sha,omitempty"`
	Dirty          bool               `json:"dirty"`
	PrimaryMetric  string             `json:"primary_metric"`
	Score          float64            `json:"score"`
	ScoreDirection string             `json:"score_direction"`
	CheckPassed    bool               `json:"check_passed"`
	EvalPassed     bool               `json:"eval_passed"`
	HardGates      GateStatus         `json:"hard_gates"`
	Metrics        map[string]float64 `json:"metrics,omitempty"`
	Gates          map[string]bool    `json:"gates,omitempty"`
	DurationMS     int64              `json:"duration_ms"`
	Decision       string             `json:"decision"`
	FailureReason  string             `json:"failure_reason"`
	Eval           evals.Result       `json:"eval"`
}

// Append writes one record as a single JSONL line, creating parent dirs and
// the file if needed. It never rewrites existing content.
func Append(path string, r Record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	encoded, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = file.Write(append(encoded, '\n'))
	return err
}

// Read parses every record from a JSONL log. A missing file yields no records
// and no error. Blank lines are skipped. A malformed line is reported with its
// line number so the operator can find it.
func Read(path string) ([]Record, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []Record
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("malformed result on line %d of %s: %w", lineNo, path, err)
		}
		records = append(records, r)
	}
	return records, scanner.Err()
}

// sidecarPath is where the log's fingerprint is stored.
func sidecarPath(path string) string { return path + ".sha256" }

// Fingerprint returns the hex SHA-256 of the file's contents. A missing file
// fingerprints as the empty string (no error), so first runs are handled.
func Fingerprint(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// CheckUntampered reports whether the log still matches its stored fingerprint.
// The runner is the only sanctioned writer, so a mismatch means the append-only
// log was edited out of band. With no sidecar yet (first run), it returns true.
func CheckUntampered(path string) (bool, error) {
	stored, err := os.ReadFile(sidecarPath(path))
	if os.IsNotExist(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	current, err := Fingerprint(path)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(stored)) == current, nil
}

// WriteFingerprint records the log's current fingerprint to the sidecar. Call
// it immediately after Append so the next run can detect out-of-band edits.
func WriteFingerprint(path string) error {
	fp, err := Fingerprint(path)
	if err != nil {
		return err
	}
	return os.WriteFile(sidecarPath(path), []byte(fp+"\n"), 0o644)
}

// Best returns the highest- or lowest-scoring record per direction. The bool is
// false when records is empty. An unknown direction is treated as higher.
func Best(records []Record, direction string) (Record, bool) {
	if len(records) == 0 {
		return Record{}, false
	}
	best := records[0]
	for _, r := range records[1:] {
		if direction == "lower_is_better" {
			if r.Score < best.Score {
				best = r
			}
		} else if r.Score > best.Score {
			best = r
		}
	}
	return best, true
}
