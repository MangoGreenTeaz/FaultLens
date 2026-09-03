package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build-time overridable version metadata.
//
// Override with:
//
//	go build -ldflags "-X main.version=v1.0.0 -X main.commit=<sha> -X main.date=<iso>"
var (
	version = "0.2.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the FaultLens version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "faultlens version %s (commit: %s, built: %s)\n",
				version, commit, date)
			return nil
		},
	}
}
