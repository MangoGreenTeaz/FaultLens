package main

import (
	"github.com/spf13/cobra"
)

// newRootCmd builds the root command for FaultLens.
//
// The command tree is intentionally thin: it collects input, invokes the
// analysis engine, and renders output. All analysis logic lives in internal/.
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
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(cmd, args, "report")
		},
	}

	addAnalyzeFlags(cmd)
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newErrorsCmd())
	cmd.AddCommand(newTimelineCmd())
	cmd.AddCommand(newIncidentCmd())

	return cmd
}

// newErrorsCmd runs error clustering.
func newErrorsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "errors [file]",
		Short: "Analyze error clustering",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(cmd, args, "errors")
		},
	}
}

// newTimelineCmd runs timeline analysis.
func newTimelineCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "timeline [file]",
		Short: "Analyze the per-minute timeline and anomalies",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(cmd, args, "timeline")
		},
	}
}

// newIncidentCmd runs incident diagnosis.
func newIncidentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "incident [file]",
		Short: "Diagnose the most likely root cause",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(cmd, args, "incident")
		},
	}
}
