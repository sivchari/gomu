// Package main provides the CLI interface for gomu mutation testing tool.
package main

import (
	"fmt"
	"os"

	"github.com/sivchari/gomu/internal/ci"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		filename := ".gomu.yaml"

		// Check if file already exists
		if _, err := os.Stat(filename); err == nil && !force {
			return fmt.Errorf("configuration file %s already exists (use --force to overwrite)", filename)
		}

		cfg := config.DefaultYAML()
		if err := cfg.SaveYAML(filename); err != nil {
			return err
		}

		fmt.Printf("✅ Created %s\n", filename)
		fmt.Printf("💡 Edit the file to customize your mutation testing settings\n")
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := ""
		if len(args) > 0 {
			configFile = args[0]
		}

		_, err := config.LoadYAML(configFile)
		if err != nil {
			fmt.Printf("❌ Configuration validation failed: %v\n", err)
			return err
		}

		fmt.Printf("✅ Configuration is valid\n")
		return nil
	},
}

var ciCmd = &cobra.Command{
	Use:   "ci [path]",
	Short: "Run mutation testing in CI/CD environment",
	Long: `Run mutation testing optimized for CI/CD environments.
This command includes:
- Quality gates with configurable thresholds
- GitHub/GitLab integration for PR comments
- Incremental analysis based on changed files
- HTML and JSON report generation
- Automatic failure on quality gate violations`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCIMutationTesting,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .gomu.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(ciCmd)
	rootCmd.AddCommand(configCmd)

	// Config subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)

	// Config init flags
	configInitCmd.Flags().Bool("force", false, "overwrite existing config file")

	// CI-specific flags
	ciCmd.Flags().String("ci-config", "", "CI-specific configuration file (defaults to main config)")
	ciCmd.Flags().Float64("threshold", 80.0, "minimum mutation score threshold for quality gate")
	ciCmd.Flags().String("format", "json", "output format (json, html, console)")
	ciCmd.Flags().Bool("fail-on-gate", true, "fail build when quality gate is not met")
}

func runMutationTesting(_ *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if verbose {
		cfg.Verbose = true
	}

	engine, err := gomu.NewEngine(cfg)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	if err := engine.Run(path); err != nil {
		return fmt.Errorf("mutation testing failed: %w", err)
	}

	return nil
}

func runCIMutationTesting(cmd *cobra.Command, args []string) error {
	workDir := "."
	if len(args) > 0 {
		workDir = args[0]
	}

	// Get CI-specific config file
	ciConfigFile, _ := cmd.Flags().GetString("ci-config")
	if ciConfigFile == "" {
		ciConfigFile = configFile
	}

	fmt.Println("🧬 Starting CI Mutation Testing...")
	fmt.Printf("📁 Working directory: %s\n", workDir)
	fmt.Printf("⚙️  Configuration: %s\n", ciConfigFile)

	// Create CI engine
	engine, err := ci.NewCIEngine(ciConfigFile, workDir)
	if err != nil {
		return fmt.Errorf("failed to create CI engine: %w", err)
	}

	// Run CI mutation testing
	if err := engine.Run(); err != nil {
		return fmt.Errorf("CI mutation testing failed: %w", err)
	}

	fmt.Println("✅ CI mutation testing completed successfully")
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
