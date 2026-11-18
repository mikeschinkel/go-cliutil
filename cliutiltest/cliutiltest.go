// Package cliutiltest runs a command line in-process, so a consumer's tests hold
// the table and not the harness. Only test code imports it, so it is never
// compiled into an app binary.
package cliutiltest

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// Args describes one in-process run.
type Args struct {
	Registry    *cliutil.Registry   // Nil means cliutil.DefaultRegistry().
	CommandLine []string            // The words after the program name.
	Config      cliutil.ConfigFunc  // Nil means the run has no config.
	Options     cliutil.OptionsFunc // Nil means the built-in global flags.
	Stdin       io.Reader           // Nil means empty input.
	AppInfo     appinfo.AppInfo     // Nil means a placeholder named "app".
	Mode        cliutil.CLIUtilMode // Zero means command mode.
	Context     context.Context     // Nil means context.Background().
}

// Result is what one run produced.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// Run runs a command line and returns its output and exit code. Each call uses
// its own streams and command values, so parallel subtests can share a Registry.
// A panic, such as a handler asking for an undefined flag or the wrong type,
// fails the test with the panic's message.
func Run(t testing.TB, args Args) (result Result) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	stdin := args.Stdin
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	info := args.AppInfo
	if info == nil {
		info = appinfo.New(appinfo.Args{Name: "app", Description: "Test app", ExeName: "app"})
	}

	runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		AppInfo:  info,
		Context:  args.Context,
		Registry: args.Registry,
		Config:   args.Config,
		Options:  args.Options,
		IO:       dt.NewStdIO(&stdout, &stderr, stdin),
		Mode:     args.Mode,
	})

	defer func() {
		recovered := recover()
		if recovered != nil {
			t.Fatalf("cliutiltest.Run(%q) panicked: %v", args.CommandLine, recovered)
		}
	}()

	result.Err = runner.Run(args.CommandLine)
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.ExitCode = cliutil.ExitCode(result.Err)
	return result
}
