package resultlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppendThenRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.jsonl")
	want := []Record{
		{ID: "a", Note: "first", Score: 0.5, ScoreDirection: "higher_is_better"},
		{ID: "b", Note: "second", Score: 0.9, ScoreDirection: "higher_is_better"},
	}
	for _, r := range want {
		if err := Append(path, r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("read %d records, want %d", len(got), len(want))
	}
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("order/ids wrong: %+v", got)
	}
}

func TestReadMissingFile(t *testing.T) {
	got, err := Read(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil {
		t.Fatalf("missing file should not error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil records, got %v", got)
	}
}

func TestReadSkipsBlankLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.jsonl")
	content := "{\"id\":\"a\",\"score\":1}\n\n{\"id\":\"b\",\"score\":2}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
}

func TestReadMalformedReportsLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.jsonl")
	content := "{\"id\":\"a\"}\nnot json\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path)
	if err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
}

func TestTamperDetection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "results.jsonl")

	// No sidecar yet -> untampered.
	if ok, err := CheckUntampered(path); err != nil || !ok {
		t.Fatalf("first run should be untampered: ok=%v err=%v", ok, err)
	}

	if err := Append(path, Record{ID: "a", Score: 1}); err != nil {
		t.Fatal(err)
	}
	if err := WriteFingerprint(path); err != nil {
		t.Fatal(err)
	}

	// Unchanged after fingerprint -> untampered.
	if ok, err := CheckUntampered(path); err != nil || !ok {
		t.Fatalf("unchanged log should be untampered: ok=%v err=%v", ok, err)
	}

	// Out-of-band edit -> tampered.
	if err := os.WriteFile(path, []byte("{\"id\":\"hacked\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := CheckUntampered(path); err != nil || ok {
		t.Fatalf("edited log should be tampered: ok=%v err=%v", ok, err)
	}
}

func TestBest(t *testing.T) {
	records := []Record{
		{ID: "a", Score: 0.5},
		{ID: "b", Score: 0.9},
		{ID: "c", Score: 0.2},
	}
	tests := []struct {
		direction string
		wantID    string
	}{
		{"higher_is_better", "b"},
		{"lower_is_better", "c"},
		{"", "b"}, // unknown defaults to higher
	}
	for _, tt := range tests {
		best, ok := Best(records, tt.direction)
		if !ok {
			t.Fatalf("%s: expected ok", tt.direction)
		}
		if best.ID != tt.wantID {
			t.Errorf("%s: best=%s, want %s", tt.direction, best.ID, tt.wantID)
		}
	}
	if _, ok := Best(nil, "higher_is_better"); ok {
		t.Error("Best(nil) should return ok=false")
	}
}
