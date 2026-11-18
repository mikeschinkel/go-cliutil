package test

import (
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
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

// FuzzNewGlobalOptions tests NewGlobalOptions and validation with random inputs
func FuzzNewGlobalOptions(f *testing.F) {
	f.Add("quiet", "timeout", 30, byte('T'), "usage")
	f.Add("", "", 0, byte(0), "")
	f.Add("nope", "nope", -1, byte('-'), "x")

	f.Fuzz(func(t *testing.T, omit, override string, def int, shortcut byte, usage string) {
		opts := cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
			Omit: []string{omit},
			Overrides: []cliutil.FlagDef{
				{Name: override, DefaultInt: def, Shortcut: shortcut, Usage: usage},
			},
		})
		// Just ensure neither panics
		_ = opts.FlagDefs()
		_ = cliutil.NewRegistry().Validate(opts)
	})
}

// runCommandLine parses newline-separated words against the test registry.
func runCommandLine(t *testing.T, input string) {
	t.Helper()
	var line []string
	if input != "" {
		line = strings.Split(input, "\n")
	}
	cliutiltest.Run(t, cliutiltest.Args{Registry: newRegistry(), CommandLine: line})
}

// FuzzCommandLine ensures no command line panics the parser.
func FuzzCommandLine(f *testing.F) {
	seeds := []string{
		"",
		"--",
		"-",
		"--=",
		"-=",
		"--help",
		"repo\nsetup\n--check\n.",
		"repo\n--timeout\n9\nsetup\n--tag\na\\,b,c",
		"task\n--all-fields\nshow\nE-1",
		"raw\n--help\n--\nx",
		"--route-me\nextra",
		"group\nmember\n--",
		"list\n-\n--\n-x",
		"repo\nsetup\n--format",
		"--timeout",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		runCommandLine(t, input)
	})
}
