package candidates

import "testing"

func TestTransformBasic(t *testing.T) {
	got := Transform("  Hello,   WORLD!  ")
	want := "hello world"
	if got != want {
		t.Fatalf("Transform mismatch: got %q want %q", got, want)
	}
}
