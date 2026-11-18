package test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
)

// appOptions is an app's Options type embedding GlobalOptions.
type appOptions struct {
	*cliutil.GlobalOptions
	input string
	level int
}

func newAppOptions() cliutil.Options {
	return &appOptions{GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
		Omit: []string{cliutil.DryRunFlagName},
		Overrides: []cliutil.FlagDef{
			{Name: cliutil.TimeoutFlagName, DefaultInt: 30, Shortcut: 'T', Usage: "Custom timeout"},
		},
	})}
}

func (o *appOptions) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{
		{Name: "input", Shortcut: 'i', DefaultString: "in.txt", String: &o.input},
		{Name: "level", Int: &o.level},
	}
}

// optionsRun runs one command and returns the HandleArgs it received.
func optionsRun(t *testing.T, opts cliutil.OptionsFunc, line ...string) (cliutiltest.Result, cliutil.HandleArgs) {
	t.Helper()
	var got cliutil.HandleArgs
	r := cliutil.NewRegistry()
	r.Register(func() cliutil.Command {
		return newCmd[tagBare](cliutil.CmdArgs{Name: "bare"}, func(args cliutil.HandleArgs) error {
			got = args
			return nil
		})
	})
	result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, Options: opts, CommandLine: append([]string{"bare"}, line...)})
	return result, got
}

func TestAppOptions(t *testing.T) {
	t.Parallel()
	result, got := optionsRun(t, newAppOptions, "--input", "x.txt", "-T", "5", "-q")
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	opts := got.OptionsAs[*appOptions]()
	if opts.input != "x.txt" || opts.Timeout() != 5*time.Second || !opts.Quiet() {
		t.Errorf("options: input=%q timeout=%v quiet=%v", opts.input, opts.Timeout(), opts.Quiet())
	}
}

func TestOverrideDefault(t *testing.T) {
	t.Parallel()
	result, got := optionsRun(t, newAppOptions)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	opts := got.OptionsAs[*appOptions]()
	if opts.Timeout() != 30*time.Second || opts.input != "in.txt" {
		t.Errorf("defaults: timeout=%v input=%q", opts.Timeout(), opts.input)
	}
}

func TestOmittedFlag(t *testing.T) {
	t.Parallel()
	result, _ := optionsRun(t, newAppOptions, "--dry-run")
	if !errors.Is(result.Err, cliutil.ErrUnknownFlag) {
		t.Errorf("omitted flag parsed: %v", result.Err)
	}
	opts := newAppOptions().(*appOptions)
	message := mustPanic(t, func() { opts.DryRun() })
	if !strings.Contains(message, cliutil.DryRunFlagName) {
		t.Errorf("panic %q does not name the flag", message)
	}
	mustPanic(t, func() { opts.SetDryRun(true) })
}

func TestOmittedFlagNotInHelp(t *testing.T) {
	t.Parallel()
	r := cliutil.NewRegistry()
	r.Register(func() cliutil.Command {
		return newCmd[tagHelp](cliutil.CmdArgs{Name: "help"}, func(args cliutil.HandleArgs) error {
			return cliutil.ShowMainHelp(args.UsageArgs())
		})
	})
	result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, Options: newAppOptions})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if strings.Contains(result.Stdout, "--dry-run") {
		t.Errorf("help shows the omitted flag:\n%s", result.Stdout)
	}
	for _, want := range []string{"--input", "Custom timeout", "(default: 30)"} {
		if !strings.Contains(result.Stdout, want) {
			t.Errorf("help lacks %q:\n%s", want, result.Stdout)
		}
	}
}

func TestInvalidGlobalOptionsArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args cliutil.GlobalOptionsArgs
		want string
	}{
		{"omit unknown", cliutil.GlobalOptionsArgs{Omit: []string{"nope"}}, "Omit"},
		{"override unknown", cliutil.GlobalOptionsArgs{Overrides: []cliutil.FlagDef{{Name: "nope"}}}, "Overrides"},
		{"override target", cliutil.GlobalOptionsArgs{Overrides: []cliutil.FlagDef{{Name: "quiet", Bool: new(bool)}}}, "target"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := cliutil.NewRegistry().Validate(cliutil.NewGlobalOptions(tt.args))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("want error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestInvalidVerbosityFlag(t *testing.T) {
	t.Parallel()
	result, _ := optionsRun(t, nil, "--verbosity", "9")
	if !errors.Is(result.Err, cliutil.ErrInvalidateVerbosity) {
		t.Errorf("got %v", result.Err)
	}
}
