package main

import (
	"fmt"
	"os"

	"github.com/faultlens/faultlens/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configFlag holds the --config flag shared by all commands.
var configFlag string

// newConfigCmd builds the config subcommand tree.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage FaultLens configuration",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigValidateCmd())
	return cmd
}

// newConfigInitCmd generates a default .faultlens.yaml in the current
// directory. It refuses to overwrite an existing file.
func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate a .faultlens.yaml with default settings",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := os.Stat(config.ProjectFileName); err == nil {
				return fmt.Errorf("%s already exists", config.ProjectFileName)
			}
			data, err := yaml.Marshal(config.Default())
			if err != nil {
				return err
			}
			if err := os.WriteFile(config.ProjectFileName, data, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", config.ProjectFileName)
			return nil
		},
	}
}

// newConfigShowCmd prints the fully merged effective configuration.
func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the effective configuration (defaults + project + user + --config)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configFlag)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		},
	}
}

// newConfigValidateCmd validates the effective configuration.
func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(configFlag)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "configuration is valid")
			return nil
		},
	}
}
