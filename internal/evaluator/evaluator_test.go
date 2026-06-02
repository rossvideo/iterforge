package evaluator

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    Output
	}{
		{
			name:  "full output",
			input: `{"score":0.9,"passed":true,"metrics":{"accuracy":0.9},"gates":{"schema_valid":true},"failure_reason":""}`,
			want: Output{
				Score:   0.9,
				Passed:  true,
				Metrics: map[string]float64{"accuracy": 0.9},
				Gates:   map[string]bool{"schema_valid": true},
			},
		},
		{
			name:  "score only",
			input: `{"score":0}`,
			want:  Output{Score: 0},
		},
		{
			name:    "missing score",
			input:   `{"passed":true}`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			input:   `{not json`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGateFailures(t *testing.T) {
	out := Output{Gates: map[string]bool{
		"schema_valid":  true,
		"format_valid":  false,
		"within_budget": false,
	}}
	got := out.GateFailures()
	want := []string{"gate failed: format_valid", "gate failed: within_budget"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GateFailures() = %v, want %v (sorted)", got, want)
	}

	if f := (Output{}).GateFailures(); len(f) != 0 {
		t.Errorf("no gates should yield no failures, got %v", f)
	}
}
