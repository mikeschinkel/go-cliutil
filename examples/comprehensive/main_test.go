package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
)

func TestCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		line     string
		want     string
		exitCode int
		wantErr  error
	}{
		{name: "default subcommand", line: "project", want: "list format=table tags= limit=10"},
		{name: "split flag", line: `project list --tag a,b --tag c\,d`, want: "tags=a|b|c,d"},
		{name: "nested with count", line: "project member add ann bob", want: "added ann,bob"},
		{name: "count too high", line: "project member add a b c d", exitCode: cliutil.ExitFlagsParseError, wantErr: cliutil.ErrTooManyOperands},
		{name: "count too low", line: "project member add", exitCode: cliutil.ExitFlagsParseError, wantErr: cliutil.ErrTooFewOperands},
		{name: "config beats default", line: "export", want: "export dir=. format=yaml source=config input=config-input.txt region=eu timeout=30s"},
		{name: "typed flag beats config", line: "export out --format csv --input typed.txt --region ap", want: "export dir=out format=csv source=command line input=typed.txt region=ap"},
		{name: "global flag before command", line: "--timeout 5 export", want: "timeout=5s"},
		{name: "flag routing", line: "--setup", want: "setup ran"},
		{name: "hidden command runs", line: "debug", want: "debug ran"},
		{name: "passthrough", line: "exec ls --all -v", want: "exec ls --all -v"},
		{name: "double dash", line: "echo -- --not-a-flag", want: "--not-a-flag"},
		{name: "operand default", line: "echo", want: "nothing"},
		{name: "runner", line: "report", want: "list format=table tags= limit=2"},
		{name: "main help", line: "help", want: "project"},
		{name: "command help", line: "export --help", want: "Export data to a directory"},
		{name: "omitted flag", line: "export --dry-run", exitCode: cliutil.ExitFlagsParseError, wantErr: cliutil.ErrUnknownFlag},
		{name: "extra operand", line: "setup extra", exitCode: cliutil.ExitFlagsParseError, wantErr: cliutil.ErrTooManyOperands},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cliutiltest.Run(t, cliutiltest.Args{
				CommandLine: strings.Fields(tt.line),
				Config:      LoadConfig,
				Options:     NewOptions,
			})
			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code: want %d, got %d (err: %v)", tt.exitCode, result.ExitCode, result.Err)
			}
			if tt.wantErr != nil && !errors.Is(result.Err, tt.wantErr) {
				t.Errorf("error: want %v, got %v", tt.wantErr, result.Err)
			}
			if !strings.Contains(result.Stdout, tt.want) {
				t.Errorf("stdout: want %q in %q", tt.want, result.Stdout)
			}
		})
	}
}

func TestHelpHidesHiddenCommand(t *testing.T) {
	t.Parallel()
	result := cliutiltest.Run(t, cliutiltest.Args{
		CommandLine: []string{"help"},
		Config:      LoadConfig,
		Options:     NewOptions,
	})
	if strings.Contains(result.Stdout, "Print diagnostics") {
		t.Errorf("help lists the hidden command:\n%s", result.Stdout)
	}
	if strings.Contains(result.Stdout, "--dry-run") {
		t.Errorf("help lists the omitted flag:\n%s", result.Stdout)
	}
}
