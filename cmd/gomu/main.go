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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gomu version 0.1.0")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is .gomu.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
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

	if verbose {
		cfg.Verbose = true
	}

	engine, err := gomu.NewEngine(cfg)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	return engine.Run(path)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}