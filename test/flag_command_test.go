package test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-testutil"
)

// The command registry is package-level state with no reset, so each scenario
// runs in its own child process: the parent re-runs this test binary with
// flagCommandScenarioEnv set, and TestFlagCommandScenario runs that scenario.
const flagCommandScenarioEnv = "CLIUTIL_FLAG_COMMAND_SCENARIO"

// flagCommandScenarios each register commands and options the way an app's
// init() functions and main() would, then return the error from Initialize.
var flagCommandScenarios = map[string]func() error{
	"builtin-conflict": func() error {
		cliutil.RegisterCommand(newFlagCmd("quietly", "quiet"))
		return cliutil.Initialize(testutil.NewBufferedWriter())
	},
	"two-commands": func() error {
		cliutil.RegisterCommand(newFlagCmd("first", "go"))
		cliutil.RegisterCommand(newFlagCmd("second", "go"))
		return cliutil.Initialize(testutil.NewBufferedWriter())
	},
	"option-then-command": func() error {
		err := addAppOption("shared")
		if err != nil {
			return fmt.Errorf("AddCLIOption: %w", err)
		}
		cliutil.RegisterCommand(newFlagCmd("sharer", "shared"))
		return cliutil.Initialize(testutil.NewBufferedWriter())
	},
	"command-then-option": func() error {
		cliutil.RegisterCommand(newFlagCmd("sharer", "shared"))
		err := addAppOption("shared")
		if err != nil {
			return fmt.Errorf("AddCLIOption: %w", err)
		}
		return cliutil.Initialize(testutil.NewBufferedWriter())
	},
	"help-and-routing": func() error {
		cmd := newFlagCmd("setup", "setup")
		cliutil.RegisterCommand(cmd)
		if findGlobalFlag("setup") != nil {
			return errors.New("flag registered before Initialize")
		}
		err := cliutil.Initialize(testutil.NewBufferedWriter())
		if err != nil {
			return err
		}
		fd := findGlobalFlag("setup")
		if fd == nil {
			return errors.New("flag missing from global flags after Initialize")
		}
		if fd.Usage != "Run setup command" {
			return fmt.Errorf("unexpected usage %q", fd.Usage)
		}
		_, args, err := cliutil.ParseGlobalOptions([]string{"app", "--setup"})
		if err != nil {
			return err
		}
		if len(args) == 0 || args[0] != "setup" {
			return fmt.Errorf("--setup routed to %v, want [setup]", args)
		}
		if cliutil.GetExactCommand("setup") != cmd {
			return errors.New("setup command not found by path")
		}
		return nil
	},
}

func newFlagCmd(name, flagName string) *cliutil.CmdBase {
	return cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        name,
		Usage:       name,
		Description: "Run " + name,
		FlagName:    flagName,
	})
}

func addAppOption(name string) error {
	return cliutil.AddCLIOption(cliutil.FlagDef{
		Name:  name,
		Usage: "App option " + name,
		Bool:  new(bool),
	})
}

func findGlobalFlag(name string) *cliutil.FlagDef {
	for _, fd := range cliutil.GetGlobalFlagSet().FlagDefs {
		if fd.Name == name {
			return &fd
		}
	}
	return nil
}

// TestFlagCommandScenario runs one scenario when invoked as a child process and
// reports Initialize's error on stdout. It does nothing in a normal test run.
func TestFlagCommandScenario(t *testing.T) {
	name := os.Getenv(flagCommandScenarioEnv)
	if name == "" {
		t.Skip("only runs as a child of TestFlagCommands")
	}
	err := flagCommandScenarios[name]()
	if err == nil {
		fmt.Println("RESULT: ok")
		return
	}
	command, _ := cliutil.ErrValue[string](err, "command_name")
	flag, _ := cliutil.ErrValue[string](err, "flag_name")
	fmt.Printf("RESULT: registration_failed=%d duplicate=%d command=%s flag=%s\n",
		strings.Count(err.Error(), cliutil.ErrCommandRegistrationFailed.Error()),
		strings.Count(err.Error(), "duplicate flag"),
		command,
		flag,
	)
	fmt.Printf("ERROR: %v\n", err)
}

func TestFlagCommands(t *testing.T) {
	tests := []struct {
		scenario string
		want     string
	}{
		{"builtin-conflict", "registration_failed=1 duplicate=1 command=quietly flag=quiet"},
		{"two-commands", "registration_failed=1 duplicate=1 command=second flag=go"},
		{"option-then-command", "registration_failed=1 duplicate=1 command=sharer flag=shared"},
		{"command-then-option", "registration_failed=1 duplicate=1 command=sharer flag=shared"},
		{"help-and-routing", "ok"},
	}
	for _, tt := range tests {
		t.Run(tt.scenario, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFlagCommandScenario$", "-test.v")
			cmd.Env = append(os.Environ(), flagCommandScenarioEnv+"="+tt.scenario)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("child process failed: %v\n%s", err, out)
			}
			if !strings.Contains(string(out), "RESULT: "+tt.want+"\n") {
				t.Errorf("want RESULT: %s\ngot:\n%s", tt.want, out)
			}
		})
	}
}
