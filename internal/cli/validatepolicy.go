package cli

import (
	"flag"
	"fmt"

	"iterforge/internal/policy"
)

// ValidatePolicy loads and validates a policy file.
func ValidatePolicy(args []string) int {
	fs := flag.NewFlagSet("validate-policy", flag.ContinueOnError)
	policyPath := fs.String("policy", "policy.yaml", "path to policy.yaml")
	if code, ok := parseFlags(fs, args); !ok {
		return code
	}

	p, err := policy.Load(*policyPath)
	if err != nil {
		return errExit(fmt.Errorf("could not load %s: %w", *policyPath, err))
	}
	if err := p.Validate(); err != nil {
		return errExit(err)
	}
	fmt.Printf("ok: %s is valid (primary_metric=%s, score_direction=%s)\n",
		*policyPath, p.PrimaryMetric, p.ScoreDirection)
	return 0
}
