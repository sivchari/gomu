// Package main provides the CLI interface for gomu mutation testing tool.
package main

import (
	"fmt"
	"os"

	"github.com/sivchari/gomu/internal/config"
	"github.com/sivchari/gomu/pkg/gomu"
	"github.com/spf13/cobra"
)

var (
	configFile string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "gomu",
	Short: "A high-performance mutation testing tool for Go",
	Long: `gomu is a mutation testing tool that helps validate the quality of your Go test suite.
It introduces controlled changes (mutations) to your code and checks if your tests catch them.

Features:
- Incremental analysis for fast reruns
- Git integration for change detection  
- Parallel execution with goroutines
- Go-specific mutations (generics, error handling, etc.)`,
	RunE: runMutationTesting,
}

var runCmd = &cobra.Command{
	Use:   "run [path]",
	Short: "Run mutation testing on the specified path",
	Long:  "Run mutation testing on the specified path or current directory",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMutationTesting,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("gomu version 0.1.0")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gomu configuration",
	Long:  "Commands for managing gomu configuration files",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new gomu configuration file",
	RunE: func(cmd *cobra.Command, _ []string) error {
		force, _ := cmd.Flags().GetBool("force")

		filename := ".gomu.yaml"

		// Check if file already exists
		if _, err := os.Stat(filename); err == nil && !force {
			return fmt.Errorf("configuration file %s already exists (use --force to overwrite)", filename)
		}

		cfg := config.Default()
		if err := cfg.Save(filename); err != nil {
			return fmt.Errorf("failed to save configuration file: %w", err)
		}

		fmt.Printf("✅ Created %s\n", filename)
		fmt.Printf("💡 Edit the file to customize your mutation testing settings\n")

		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	RunE: func(_ *cobra.Command, args []string) error {
		configFile := ""
		if len(args) > 0 {
			configFile = args[0]
		}

		_, err := config.Load(configFile)
		if err != nil {
			fmt.Printf("❌ Configuration validation failed: %v\n", err)

			return fmt.Errorf("invalid configuration file: %w", err)
		}

		fmt.Printf("✅ Configuration is valid\n")

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .gomu.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	// Run command flags
	runCmd.Flags().Bool("ci-mode", false, "enable CI mode with quality gates and reporting")
	runCmd.Flags().Float64("threshold", 80.0, "minimum mutation score threshold")
	runCmd.Flags().String("output", "json", "output format (json, html, console)")
	runCmd.Flags().Bool("fail-on-gate", true, "fail build when quality gate is not met")
	runCmd.Flags().Int("workers", 4, "number of parallel workers")
	runCmd.Flags().Int("timeout", 30, "test timeout in seconds")
	runCmd.Flags().Bool("incremental", true, "enable incremental analysis")
	runCmd.Flags().String("base-branch", "main", "base branch for incremental analysis")

	// Config subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)

	// Config init flags
	configInitCmd.Flags().Bool("force", false, "overwrite existing config file")
}

func runMutationTesting(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if CI mode is enabled
	ciMode, _ := cmd.Flags().GetBool("ci-mode")
	
	// TODO: Handle CI-specific flags via environment variables or action.yaml

	engine, err := gomu.NewEngineWithCIMode(cfg, ciMode)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	if err := engine.RunWithContext(cmd.Context(), path); err != nil {
		return fmt.Errorf("mutation testing failed: %w", err)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
