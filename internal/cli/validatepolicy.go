package cli

import (
	"flag"
	"fmt"

	"iterforge/internal/policy"
)

// ValidatePolicy loads and validates a policy file.
func ValidatePolicy(args []string) int {
	fs := flag.NewFlagSet("validate-policy", flag.ExitOnError)
	policyPath := fs.String("policy", "policy.yaml", "path to policy.yaml")
	_ = fs.Parse(args)

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
