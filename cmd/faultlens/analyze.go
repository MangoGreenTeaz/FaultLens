package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/faultlens/faultlens/internal/engine"
	"github.com/faultlens/faultlens/internal/output"
	"github.com/spf13/cobra"
)

// Analysis flags shared by the root command and its subcommands.
var (
	formatFlag string
	outputFlag string
	fromFlag   string
	toFlag     string
)

// addAnalyzeFlags registers the persistent analysis flags.
func addAnalyzeFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&formatFlag, "format", "auto",
		"log format: auto, plain, json, java, nginx")
	cmd.PersistentFlags().StringVar(&outputFlag, "output", "terminal",
		"output format: terminal, json, markdown")
	cmd.PersistentFlags().StringVar(&fromFlag, "from", "",
		"only analyze events at or after this time (RFC3339, e.g. 2026-08-31T14:00:00Z)")
	cmd.PersistentFlags().StringVar(&toFlag, "to", "",
		"only analyze events at or before this time (RFC3339)")
}

// runAnalysis is the shared body of the root, errors, timeline and incident
// commands. kind selects the report view.
func runAnalysis(cmd *cobra.Command, args []string, kind string) error {
	// Input: a file argument, or stdin when none is given.
	var src io.Reader
	var source string
	if len(args) >= 1 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		src = f
		source = args[0]
	} else {
		src = cmd.InOrStdin()
		source = "stdin"
	}

	from, err := parseTimeFlag(fromFlag)
	if err != nil {
		return err
	}
	to, err := parseTimeFlag(toFlag)
	if err != nil {
		return err
	}

	res, err := engine.Run(src, engine.Options{
		Format: formatFlag,
		From:   from,
		To:     to,
		Source: source,
	})
	if err != nil {
		return err
	}

	switch outputFlag {
	case "json":
		return output.RenderJSON(cmd.OutOrStdout(), res)
	case "markdown":
		return output.RenderMarkdown(cmd.OutOrStdout(), res, kind)
	default:
		return output.RenderTerminal(cmd.OutOrStdout(), res, kind)
	}
}

// parseTimeFlag parses an RFC3339 time flag; empty means no bound.
func parseTimeFlag(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time %q (use RFC3339): %w", s, err)
	}
	return t, nil
}
