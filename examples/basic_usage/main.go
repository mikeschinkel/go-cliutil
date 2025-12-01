package main

import (
	"fmt"
	"os"

	"github.com/mikeschinkel/go-cliutil"
)

// Example CLI application demonstrating go-cliutil basic usage
func main() {
	// Create CLI options with defaults
	opts, err := cliutil.NewCLIOptions(cliutil.CLIOptionsArgs{})
	if err != nil {
		cliutil.Stderrf("Error: %v\n", err)
		os.Exit(1)
	}

	// Parse command line arguments
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("Usage: basic_usage <name>")
		fmt.Println("\nExample:")
		fmt.Println("  basic_usage World")
		fmt.Println("\nOptions:")
		fmt.Println("  --quiet      Suppress output")
		fmt.Println("  --verbosity  Set verbosity level (0-3)")
		os.Exit(1)
	}

	// Simple example: greet the first argument
	name := args[0]

	// Show greeting unless quiet mode is enabled
	if !opts.Quiet() {
		greeting := fmt.Sprintf("Hello, %s!", name)
		fmt.Println(greeting)
	}

	// Show additional info if verbose
	if opts.Verbosity() > 1 {
		fmt.Printf("Verbosity level: %d\n", opts.Verbosity())
		fmt.Printf("Quiet mode: %v\n", opts.Quiet())
	}
}
