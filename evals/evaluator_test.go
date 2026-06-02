package evals

import "testing"

func TestEvaluate(t *testing.T) {
	res, err := Evaluate("golden_set.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if res.Total == 0 {
		t.Fatal("expected non-empty evaluation")
	}
	if res.Accuracy < 0 || res.Accuracy > 1 {
		t.Fatalf("invalid accuracy: %f", res.Accuracy)
	}
}
