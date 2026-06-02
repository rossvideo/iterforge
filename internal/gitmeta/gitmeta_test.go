package gitmeta

import "testing"

// Capture must never panic or block, in or out of a git repo. Values vary by
// environment, so we only assert it returns and that dirty is false when there
// is no SHA.
func TestCaptureDoesNotPanic(t *testing.T) {
	m := Capture()
	if m.SHA == "" && m.Dirty {
		t.Errorf("dirty=true with empty SHA is inconsistent: %+v", m)
	}
}
