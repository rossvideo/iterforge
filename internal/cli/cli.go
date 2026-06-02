// Package cli implements the iterforge subcommands. Each exported function
// takes the subcommand's argument slice and returns a process exit code, so the
// single iterforge binary and any wrapper can invoke them uniformly.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// errExit prints an error to stderr and returns exit code 1.
func errExit(err error) int {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	return 1
}

// parseFlags parses args with a ContinueOnError flag set so a bad flag returns
// an exit code instead of terminating the process (which keeps the commands
// testable). It returns (code, ok): on a parse error ok is false and code is the
// exit code to return — 0 for -h/-help, 2 for a usage error.
func parseFlags(fs *flag.FlagSet, args []string) (int, bool) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, false
		}
		return 2, false
	}
	return 0, true
}
