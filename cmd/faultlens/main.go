// Package main is the entry point of the FaultLens CLI.
//
// It only wires the Cobra command tree together; all analysis logic
// lives in the internal/ packages so it can be reused by future
// GitHub Actions or API integrations.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
