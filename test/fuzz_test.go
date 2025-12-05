package test

import (
	"testing"

	"github.com/mikeschinkel/go-cliutil"
)

// FuzzParseVerbosity tests ParseVerbosity with random inputs to ensure it doesn't panic
func FuzzParseVerbosity(f *testing.F) {
	// Seed corpus with valid verbosity values
	seeds := []int{
		0,  // Quiet
		1,  // Normal
		2,  // Verbose
		3,  // Debug
		-1, // Invalid
		10, // Invalid
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, verbosity int) {
		// Just ensure it doesn't panic
		_, _ = cliutil.ParseVerbosity(verbosity)
	})
}

// FuzzNewGlobalOptions tests NewGlobalOptions with random inputs
func FuzzNewGlobalOptions(f *testing.F) {
	// Seed corpus with various option combinations
	type seed struct {
		quiet     bool
		verbosity int
		timeout   int
		dryRun    bool
		force     bool
	}

	seeds := []seed{
		{false, 0, 0, false, false},
		{true, 1, 30, false, false},
		{false, 2, 60, true, false},
		{false, 3, 120, false, true},
		{true, -1, -1, true, true},
	}

	for _, s := range seeds {
		f.Add(s.quiet, s.verbosity, s.timeout, s.dryRun, s.force)
	}

	f.Fuzz(func(t *testing.T, quiet bool, verbosity, timeout int, dryRun, force bool) {
		// Just ensure it doesn't panic
		_, _ = cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
			Quiet:     &quiet,
			Verbosity: &verbosity,
			Timeout:   &timeout,
			DryRun:    &dryRun,
			Force:     &force,
		})
	})
}
