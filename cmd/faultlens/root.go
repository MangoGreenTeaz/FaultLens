package main

import (
	"github.com/spf13/cobra"
)

// newRootCmd builds the root command for FaultLens.
//
// The command tree is intentionally thin: it collects input, invokes the
// analysis engine, and renders output. Subcommands (errors, timeline,
// incident) are registered here as they are implemented in later phases.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "faultlens [file]",
		Short: "FaultLens — See beyond the error.",
		Long: `FaultLens is a local-first, offline-first log incident diagnosis CLI.

It identifies error patterns, detects anomalous time points, analyzes
relationships between errors, and infers the most likely root cause from
transparent, explainable rules.

Run "faultlens <file>" for a full analysis report, or pipe logs in:

    cat app.log | faultlens`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCmd())

	return cmd
}
