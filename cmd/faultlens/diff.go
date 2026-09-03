package main

import (
	"os"

	"github.com/faultlens/faultlens/internal/engine"
	"github.com/faultlens/faultlens/internal/output"
	"github.com/spf13/cobra"
)

// newDiffCmd builds the "faultlens diff" command comparing two JSON reports.
func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <before.json> <after.json>",
		Short: "Compare two FaultLens JSON reports",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := loadResultFile(args[0])
			if err != nil {
				return err
			}
			after, err := loadResultFile(args[1])
			if err != nil {
				return err
			}

			diff := output.ComputeDiff(before, after)
			if outputFlag == "markdown" {
				return output.RenderDiffMarkdown(cmd.OutOrStdout(), diff)
			}
			return output.RenderDiff(cmd.OutOrStdout(), diff)
		},
	}
}

// loadResultFile reads a FaultLens JSON report from disk.
func loadResultFile(path string) (*engine.Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return output.LoadResult(f)
}
