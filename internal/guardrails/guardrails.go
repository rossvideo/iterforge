// Package guardrails enforces policy boundaries that the score cannot override.
//
// The first guardrail detects changes to frozen paths: the evaluator, golden
// data, and harness infrastructure must not move during a candidate run, or the
// experiment is comparing against shifting ground.
package guardrails

import (
	"path"
	"sort"
	"strings"
)

// FrozenViolations returns the subset of changedFiles that match any frozen
// pattern. changedFiles are repo-relative paths (e.g. from git status).
//
// Supported pattern forms (a subset of gitignore-style globbing):
//   - "dir/" or "dir/**"  -> any path under dir
//   - "exact/file.go"     -> that file, or any path under it if it is a dir
//   - "*.go", "cmd/*.go"  -> path.Match against the full path and the basename
func FrozenViolations(changedFiles, frozenPatterns []string) []string {
	var out []string
	for _, f := range changedFiles {
		f = normalize(f)
		if f == "" {
			continue
		}
		for _, pat := range frozenPatterns {
			if matchPattern(pat, f) {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

func matchPattern(pat, file string) bool {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return false
	}
	if strings.HasSuffix(pat, "/**") {
		return underDir(strings.TrimSuffix(pat, "/**"), file)
	}
	if strings.HasSuffix(pat, "/") {
		return underDir(strings.TrimSuffix(pat, "/"), file)
	}
	if strings.ContainsAny(pat, "*?[") {
		if ok, _ := path.Match(pat, file); ok {
			return true
		}
		ok, _ := path.Match(pat, path.Base(file))
		return ok
	}
	// No glob, no trailing slash: exact file match, or treat as a directory
	// prefix so "policy.yaml" never matches "policy.yaml.bak" but "internal"
	// would match "internal/policy/policy.go".
	if file == pat {
		return true
	}
	return underDir(pat, file)
}

// minHardcodeLen ignores golden values too short to be meaningful evidence of
// memorization (e.g. "ai"), which would produce noisy false positives.
const minHardcodeLen = 4

// Finding is one hardcoded golden value detected in candidate source.
type Finding struct {
	File  string
	Value string
}

// GoldenHardcoding scans candidate source for golden-set expected values that
// appear as exact string literals. A candidate that embeds the answers is
// memorizing the eval rather than solving the task.
//
// sources maps file path -> file contents (candidate .go, excluding tests).
// expected is the list of golden expected outputs. Matching is conservative: a
// value must appear inside a double-quoted or backtick-quoted literal.
func GoldenHardcoding(sources map[string]string, expected []string) []Finding {
	// Stable file order for deterministic output.
	files := make([]string, 0, len(sources))
	for f := range sources {
		files = append(files, f)
	}
	sort.Strings(files)

	var findings []Finding
	for _, f := range files {
		src := sources[f]
		for _, v := range expected {
			if len(v) < minHardcodeLen {
				continue
			}
			if strings.Contains(src, `"`+v+`"`) || strings.Contains(src, "`"+v+"`") {
				findings = append(findings, Finding{File: f, Value: v})
			}
		}
	}
	return findings
}

func underDir(base, file string) bool {
	base = strings.TrimSuffix(normalize(base), "/")
	return file == base || strings.HasPrefix(file, base+"/")
}

func normalize(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "./")
	return p
}
