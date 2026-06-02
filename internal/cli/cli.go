// Package cli implements the iterforge subcommands. Each exported function
// takes the subcommand's argument slice and returns a process exit code, so the
// single iterforge binary and any wrapper can invoke them uniformly.
package cli

import (
	"fmt"
	"os"
)

// errExit prints an error to stderr and returns exit code 1.
func errExit(err error) int {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	return 1
}
