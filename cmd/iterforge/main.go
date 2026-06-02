// Command iterforge is the single entry point for the IterForge harness.
//
//	iterforge init <name>        scaffold a new project
//	iterforge validate-policy     validate policy.yaml
//	iterforge run --note "..."   run one scored experiment
//	iterforge summarize           summarize results
//	iterforge compare             compare two runs
package main

import (
	"fmt"
	"os"

	"iterforge/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	sub, args := os.Args[1], os.Args[2:]

	var code int
	switch sub {
	case "init":
		code = cli.InitProject(args)
	case "validate-policy":
		code = cli.ValidatePolicy(args)
	case "run":
		code = cli.Run(args)
	case "summarize":
		code = cli.Summarize(args)
	case "compare":
		code = cli.Compare(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "iterforge: unknown command %q\n\n", sub)
		usage()
		code = 1
	}
	os.Exit(code)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: iterforge <command> [flags]

commands:
  init             scaffold a new project (-template, -dir)
  validate-policy  validate policy.yaml (-policy)
  run              run one scored experiment (-note, -policy)
  summarize        summarize results (-last, -since, -failed-only, -json)
  compare          compare two runs (-baseline, -candidate)`)
}
