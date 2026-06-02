// Package gitmeta captures lightweight git provenance for an experiment.
//
// It degrades gracefully: outside a git repository, or when git is absent, it
// returns an empty SHA and dirty=false rather than failing the experiment.
package gitmeta

import (
	"os/exec"
	"strings"
)

// Meta is the git context at experiment time.
type Meta struct {
	SHA   string
	Dirty bool
}

// Capture reads the current commit SHA and whether the working tree is dirty.
// All errors are swallowed by design; provenance is best-effort metadata.
func Capture() Meta {
	sha, err := run("rev-parse", "HEAD")
	if err != nil {
		return Meta{}
	}
	status, err := run("status", "--porcelain")
	return Meta{SHA: sha, Dirty: err == nil && status != ""}
}

// ChangedFiles lists repo-relative paths that differ from HEAD: modified,
// staged, and untracked. Outside a git repo it returns nil. For a rename it
// reports the destination path.
func ChangedFiles() []string {
	out, err := run("status", "--porcelain")
	if err != nil || out == "" {
		return nil
	}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		p := strings.TrimSpace(line[3:])
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		if p != "" {
			files = append(files, p)
		}
	}
	return files
}

func run(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
