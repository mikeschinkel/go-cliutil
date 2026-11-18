package test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
)

func TestParser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		line    string
		want    []string
		wantErr error
		errText string
	}{
		// Flag syntax
		{name: "long", line: "repo setup --format json", want: []string{"format=json"}},
		{name: "long equals", line: "repo setup --format=json", want: []string{"format=json"}},
		{name: "short", line: "repo setup -F json", want: []string{"format=json"}},
		{name: "short equals", line: "repo setup -F=json", want: []string{"format=json"}},
		{name: "short bool", line: "repo setup -c", want: []string{"check=true"}},
		{name: "multi-letter after one dash", line: "repo setup -format json", wantErr: cliutil.ErrUnknownFlag, errText: "-format"},
		{name: "bundled shortcuts refused", line: "repo setup -cF", wantErr: cliutil.ErrUnknownFlag, errText: "-cF"},
		{name: "shortcut with two dashes", line: "repo setup --c", wantErr: cliutil.ErrUnknownFlag, errText: "--c"},
		{name: "value may start with a dash", line: "repo setup --format -v", want: []string{"format=-v"}},
		{name: "missing value", line: "repo setup --format", wantErr: cliutil.ErrFlagValueMissing, errText: "--format"},
		{name: "invalid int", line: "repo setup --count x", wantErr: cliutil.ErrInvalidFlagValue, errText: "--count"},
		{name: "last scalar wins", line: "repo setup --format a --format b", want: []string{"format=b"}},
		{name: "unknown flag", line: "repo setup --nope", wantErr: cliutil.ErrUnknownFlag, errText: "--nope"},

		// Global flag placement
		{name: "global before", line: "--timeout 9 repo setup", want: []string{"check=false"}},
		{name: "global between", line: "repo --timeout 9 setup", want: []string{"check=false"}},
		{name: "global after", line: "repo setup --timeout 9", want: []string{"check=false"}},
		{name: "global bool between", line: "repo -q setup", want: nil},

		// Command flag placement, using the lines that motivated the rule
		{name: "flags after operand", line: "task show E-1 --all-fields -p", want: []string{"id=E-1 all=true pretty=true"}},
		{name: "flags before operand", line: "task show --all-fields -p E-1", want: []string{"id=E-1 all=true pretty=true"}},
		{name: "flag before all commands", line: "--all-fields task show E-1", wantErr: cliutil.ErrFlagBeforeCommand, errText: "--all-fields"},
		{name: "flag between commands", line: "task --all-fields show E-1", wantErr: cliutil.ErrFlagBeforeCommand, errText: "--all-fields"},

		// Booleans
		{name: "bool keeps next word", line: "repo setup --check .", want: []string{"check=true", "dir=."}},
		{name: "bool false", line: "repo setup --check=false .", want: []string{"check=false", "dir=."}},
		{name: "bool bad value", line: "repo setup --check=maybe", wantErr: cliutil.ErrInvalidFlagValue},

		// Double dash
		{name: "double dash", line: "repo setup -- --check", want: []string{"check=false", "dir=--check"}},
		{name: "double dash keeps later flags", line: "list -- -a --b", want: []string{`items=["-a" "--b"]`}},

		// Passthrough
		{name: "passthrough", line: "raw --check -q x -- y", want: []string{`params=["--check" "-q" "x" "--" "y"]`, `operands=["--check" "-q" "x" "--" "y"]`}},
		{name: "passthrough keeps help", line: "raw --help", want: []string{`params=["--help"]`}},
		{name: "global before passthrough", line: "-q raw --x", want: nil},

		// Strings flags
		{name: "repeat", line: "repo setup --name a --name b", want: []string{`names=["a" "b"]`}},
		{name: "no split without Split", line: "repo setup --name a,b", want: []string{`names=["a,b"]`}},
		{name: "split", line: "repo setup --tag a,b --tag c", want: []string{`tags=["a" "b" "c"]`}},
		{name: "escaped comma", line: `repo setup --tag a\,b,c`, want: []string{`tags=["a,b" "c"]`}},

		// Defaults
		{name: "typed defaults", line: "repo setup", want: []string{"format=text", "count=7", "size=9", "dir=default-dir", `names=["anon"]`}},

		// Operands
		{name: "extra operand", line: "repo setup a b", wantErr: cliutil.ErrTooManyOperands, errText: "b"},
		{name: "operand on command without ArgDef", line: "bare a", wantErr: cliutil.ErrTooManyOperands, errText: "a"},
		{name: "required operand missing", line: "task show", wantErr: cliutil.ErrTooFewOperands, errText: "id"},
		{name: "variadic below minimum", line: "list", wantErr: cliutil.ErrTooFewOperands},
		{name: "variadic above maximum", line: "list a b c", wantErr: cliutil.ErrTooManyOperands},
		{name: "variadic in range", line: "list a b", want: []string{`items=["a" "b"]`}},

		// Routing
		{name: "no words shows help", line: "", want: []string{"help topic="}},
		{name: "only global flags shows help", line: "-q", want: nil},
		{name: "help flag", line: "repo setup --help", want: []string{`topic=["repo" "setup"]`, `path=["help"]`}},
		{name: "help flag first", line: "--help repo setup", want: []string{`topic=["repo" "setup"]`}},
		{name: "help command topic", line: "help repo setup", want: []string{`topic=["repo" "setup"]`}},
		{name: "flag routed", line: "--route-me", want: []string{"ran=routed"}},
		{name: "default subcommand", line: "group", want: []string{"ran=group.member"}},
		{name: "unknown command", line: "nope", wantErr: cliutil.ErrUnknownCommand, errText: "nope"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := run(t, tt.line)
			if tt.wantErr != nil {
				if !errors.Is(result.Err, tt.wantErr) {
					t.Fatalf("error: want %v, got %v", tt.wantErr, result.Err)
				}
				if !strings.Contains(result.Err.Error(), tt.errText) {
					t.Errorf("error %q does not name %q", result.Err, tt.errText)
				}
				if result.ExitCode != cliutil.ExitFlagsParseError {
					t.Errorf("exit code: want %d, got %d", cliutil.ExitFlagsParseError, result.ExitCode)
				}
				return
			}
			if result.Err != nil {
				t.Fatalf("unexpected error: %v", result.Err)
			}
			for _, want := range tt.want {
				if !strings.Contains(result.Stdout, want) {
					t.Errorf("want %q in %q", want, result.Stdout)
				}
			}
		})
	}
}

func TestExtraOperandShowsCommandUsage(t *testing.T) {
	t.Parallel()
	result := run(t, "repo setup a b")
	if !strings.Contains(result.Stderr, "USAGE:") {
		t.Errorf("stderr has no usage: %q", result.Stderr)
	}
}

func TestParametersAndOperandPositions(t *testing.T) {
	t.Parallel()
	r := cliutil.NewRegistry()
	var got cliutil.HandleArgs
	r.Register(func() cliutil.Command {
		var dir string
		return newCmd[tagBare](cliutil.CmdArgs{
			Name:    "bare",
			ArgDefs: []*cliutil.ArgDef{{Name: "dir", String: &dir}},
			FlagSets: []*cliutil.FlagSet{{Name: "bare", FlagDefs: []cliutil.FlagDef{
				{Name: "format", String: new(string)},
				{Name: "check", Bool: new(bool)},
			}}},
		}, func(args cliutil.HandleArgs) error {
			got = args
			return nil
		})
	})
	line := []string{"-q", "bare", "--format", "json", ".", "--check"}
	result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, CommandLine: line})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if strings.Join(got.CommandLine, " ") != strings.Join(line, " ") {
		t.Errorf("CommandLine: %q", got.CommandLine)
	}
	if strings.Join(got.CommandPath, " ") != "bare" {
		t.Errorf("CommandPath: %q", got.CommandPath)
	}
	if strings.Join(got.Parameters, " ") != "--format json . --check" {
		t.Errorf("Parameters: %q", got.Parameters)
	}
	if len(got.Operands) != 1 || got.Operands[0].Value != "." || got.Operands[0].Position != 2 || got.Operands[0].Def == nil {
		t.Errorf("Operands: %+v", got.Operands)
	}

	var passed []string
	for _, flag := range got.Flags.Passed() {
		passed = append(passed, flag.Name)
	}
	if strings.Join(passed, " ") != "quiet format check" {
		t.Errorf("Passed: %q", passed)
	}
	format, ok := got.Flags.Get("format")
	if !ok || format.Source != cliutil.CommandLineSource || format.Global || strings.Join(format.Words, " ") != "--format json" || format.Def == nil {
		t.Errorf("format flag: %+v", format)
	}
	quiet, _ := got.Flags.Get("quiet")
	if !quiet.Global {
		t.Errorf("quiet flag is not global: %+v", quiet)
	}
	timeout, _ := got.Flags.Get("timeout")
	if timeout.Source != cliutil.DefaultSource || timeout.Words != nil || timeout.Value != cliutil.DefaultTimeout {
		t.Errorf("timeout flag: %+v", timeout)
	}
	if len(got.Flags.All()) != 7 {
		t.Errorf("All: %d flags", len(got.Flags.All()))
	}
}

func TestFlagAccessorPanics(t *testing.T) {
	t.Parallel()
	r := cliutil.NewRegistry()
	var got cliutil.HandleArgs
	r.Register(func() cliutil.Command {
		return newCmd[tagBare](cliutil.CmdArgs{Name: "bare"}, func(args cliutil.HandleArgs) error {
			got = args
			return nil
		})
	})
	result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, CommandLine: []string{"bare"}})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if got.Flag[int]("timeout") != cliutil.DefaultTimeout {
		t.Error("Flag[int] returned the wrong value")
	}
	tests := []struct {
		name  string
		call  func()
		wants []string
	}{
		{"undefined flag", func() { got.Flag[string]("nope") }, []string{`"nope"`, `"bare"`}},
		{"wrong type", func() { got.Flag[string]("timeout") }, []string{`"timeout"`, `"bare"`, "int", "string"}},
		{"wrong options type", func() { got.OptionsAs[*appOptions]() }, []string{"*cliutil.GlobalOptions", "*test.appOptions"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := mustPanic(t, tt.call)
			for _, want := range tt.wants {
				if !strings.Contains(message, want) {
					t.Errorf("panic %q does not name %q", message, want)
				}
			}
		})
	}
}

// TestRunsShareNothing proves a value set by one run never reaches another.
func TestRunsShareNothing(t *testing.T) {
	t.Parallel()
	r := newRegistry()
	withFlag := []string{"repo", "setup", "--check", "--tag", "a", "first"}
	without := []string{"repo", "setup", "second"}

	check := func(t *testing.T, line []string, want string) {
		t.Helper()
		result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, CommandLine: line})
		if result.Err != nil {
			t.Error(result.Err)
		}
		if !strings.Contains(result.Stdout, want) {
			t.Errorf("want %q in %q", want, result.Stdout)
		}
	}
	const withWant, withoutWant = "check=true", "check=false format=text count=7 size=9 dir=second tags=[]"

	t.Run("with then without", func(t *testing.T) {
		check(t, withFlag, withWant)
		check(t, without, withoutWant)
	})
	t.Run("without then with", func(t *testing.T) {
		check(t, without, withoutWant)
		check(t, withFlag, withWant)
	})
	t.Run("parallel", func(t *testing.T) {
		var wg sync.WaitGroup
		for range 50 {
			wg.Go(func() { check(t, withFlag, withWant) })
			wg.Go(func() { check(t, without, withoutWant) })
		}
		wg.Wait()
	})
}

func TestFlagsOnlyMode(t *testing.T) {
	t.Parallel()
	single := cliutil.NewRegistry()
	single.Register(newSetupCmd)
	result := cliutiltest.Run(t, cliutiltest.Args{
		Registry:    single,
		Mode:        cliutil.FlagsOnlyMode,
		CommandLine: []string{"--check", "here"},
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if !strings.Contains(result.Stdout, "check=true") || !strings.Contains(result.Stdout, "dir=here") {
		t.Errorf("stdout: %q", result.Stdout)
	}

	result = cliutiltest.Run(t, cliutiltest.Args{Registry: newRegistry(), Mode: cliutil.FlagsOnlyMode})
	if !errors.Is(result.Err, cliutil.ErrInvalidRegistration) {
		t.Errorf("several commands: got %v", result.Err)
	}
}
