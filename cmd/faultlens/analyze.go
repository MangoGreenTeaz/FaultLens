package main

import (
	"fmt"
	"os"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/config"
	"github.com/MangoGreenTeaz/FaultLens/internal/engine"
	"github.com/MangoGreenTeaz/FaultLens/internal/output"
	"github.com/spf13/cobra"
)

// Analysis flags shared by the root command and its subcommands.
var (
	formatFlag     string
	outputFlag     string
	outputFileFlag string
	fromFlag       string
	toFlag         string
	excludeFlag    string
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
	cmd.PersistentFlags().StringVar(&configFlag, "config", "",
		"path to a configuration file (overrides project and user config)")
	cmd.PersistentFlags().StringVarP(&outputFileFlag, "output-file", "o", "",
		"write the report to a file instead of stdout")
	cmd.PersistentFlags().StringVar(&excludeFlag, "exclude", "",
		"exclude files matching this glob pattern (e.g. '*.debug.log') when expanding directories")
}

// runAnalysis is the shared body of the root, errors, timeline and incident
// commands. kind selects the report view.
//
// Input resolution:
//   - no arguments        → stdin
//   - one or more paths   → files, globs and directories are expanded and
//     merged into a single analysis
func runAnalysis(cmd *cobra.Command, args []string, kind string) error {
	from, err := parseTimeFlag(fromFlag)
	if err != nil {
		return err
	}
	to, err := parseTimeFlag(toFlag)
	if err != nil {
		return err
	}
	cfg, err := config.Load(configFlag)
	if err != nil {
		return err
	}

	var res *engine.Result
	if len(args) == 0 {
		res, err = engine.Run(cmd.InOrStdin(), engine.Options{
			Format: formatFlag,
			From:   from,
			To:     to,
			Source: "stdin",
			Config: cfg,
		})
		if err != nil {
			return err
		}
	} else {
		files, err := expandPaths(args, excludeFlag)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			return fmt.Errorf("no log files matched the given paths")
		}
		res, err = engine.RunFiles(files, engine.Options{
			Format: formatFlag,
			From:   from,
			To:     to,
			Source: fmt.Sprintf("%d files", len(files)),
			Config: cfg,
		})
		if err != nil {
			return err
		}
	}

	// Output destination: stdout, or a file when -o is given.
	out := cmd.OutOrStdout()
	if outputFileFlag != "" {
		f, err := os.Create(outputFileFlag)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	switch outputFlag {
	case "json":
		return output.RenderJSON(out, res)
	case "markdown":
		return output.RenderMarkdown(out, res, kind)
	case "html":
		return output.RenderHTML(out, res)
	default:
		return output.RenderTerminal(out, res, kind)
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
