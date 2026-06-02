package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAvailableIncludesBuiltinTemplates(t *testing.T) {
	got := Available()
	for _, want := range []string{"extraction", "prompt-optimization", "ranking", "text-normalization"} {
		found := false
		for _, n := range got {
			if n == want {
				found = true
			}
		}
		if !found {
			t.Errorf("Available() = %v, missing %q", got, want)
		}
	}
	for _, n := range got {
		if n == sharedLayer {
			t.Errorf("Available() must not include the shared layer %q", sharedLayer)
		}
	}
}

func TestInitGeneratesProject(t *testing.T) {
	for _, tmpl := range Available() {
		t.Run(tmpl, func(t *testing.T) {
			parent := t.TempDir()
			if err := Init("demo", parent, tmpl); err != nil {
				t.Fatalf("Init: %v", err)
			}
			root := filepath.Join(parent, "demo")

			// Files every template must produce (shared layer + universal
			// per-template files). Candidate files differ per template, so they
			// are not asserted here.
			want := []string{
				"go.mod", "policy.yaml", "Makefile", "README.md", "ITERFORGE.md",
				"evals/evaluator.go", "evals/evaluator_test.go", "evals/golden_set.jsonl",
				"cmd/runexp/main.go", "cmd/summarize/main.go",
				"logs/agent_journal.md", "logs/results.jsonl",
			}
			for _, rel := range want {
				if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
					t.Errorf("missing %s: %v", rel, err)
				}
			}

			gomod := read(t, root, "go.mod")
			if !strings.Contains(gomod, "module demo") {
				t.Errorf("go.mod missing module name:\n%s", gomod)
			}

			// No .tmpl files and no unsubstituted placeholders may leak.
			_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if strings.HasSuffix(p, ".tmpl") {
					t.Errorf("generated project contains a .tmpl file: %s", p)
				}
				if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), modulePlaceholder) {
					t.Errorf("%s still contains %s placeholder", p, modulePlaceholder)
				}
				return nil
			})
		})
	}
}

func TestInitDefaultsTemplate(t *testing.T) {
	parent := t.TempDir()
	if err := Init("demo", parent, ""); err != nil {
		t.Fatalf("Init with empty template: %v", err)
	}
	readme := read(t, filepath.Join(parent, "demo"), "README.md")
	if !strings.Contains(readme, "demo") {
		t.Error("README not generated from default template")
	}
}

func TestInitRejectsUnknownTemplate(t *testing.T) {
	if err := Init("demo", t.TempDir(), "does-not-exist"); err == nil {
		t.Fatal("expected error for unknown template")
	}
}

func TestInitRefusesNonEmptyDestination(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "demo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init("demo", parent, ""); err == nil {
		t.Fatal("expected error for non-empty destination")
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}
