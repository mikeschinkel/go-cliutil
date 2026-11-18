package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

func flagsCmd(defs ...cliutil.FlagDef) cliutil.CommandFunc {
	return func() cliutil.Command {
		return newCmd[tagBad](cliutil.CmdArgs{
			Name:     "bad",
			FlagSets: []*cliutil.FlagSet{{Name: "bad", FlagDefs: defs}},
		}, nil)
	}
}

func argsCmd(defs ...*cliutil.ArgDef) cliutil.CommandFunc {
	return func() cliutil.Command {
		return newCmd[tagBad](cliutil.CmdArgs{Name: "bad", ArgDefs: defs}, nil)
	}
}

func TestRegistryValidation(t *testing.T) {
	t.Parallel()
	s, ss := new(string), new([]string)
	tests := []struct {
		name string
		reg  func(*cliutil.Registry)
		want string
	}{
		{"second ArgDef", func(r *cliutil.Registry) {
			r.Register(argsCmd(&cliutil.ArgDef{Name: "a", String: s}, &cliutil.ArgDef{Name: "b", String: s}))
		}, "at most one ArgDef"},
		{"count on single value", func(r *cliutil.Registry) {
			r.Register(argsCmd(&cliutil.ArgDef{Name: "a", String: s, MinCount: 1}))
		}, "MinCount or MaxCount"},
		{"max below min", func(r *cliutil.Registry) {
			r.Register(argsCmd(&cliutil.ArgDef{Name: "a", Strings: ss, MinCount: 3, MaxCount: 2}))
		}, "invalid MinCount"},
		{"ArgDef without target", func(r *cliutil.Registry) {
			r.Register(argsCmd(&cliutil.ArgDef{Name: "a"}))
		}, "exactly one of String or Strings"},
		{"ArgDef default mismatch", func(r *cliutil.Registry) {
			r.Register(argsCmd(&cliutil.ArgDef{Name: "a", String: s, DefaultStrings: []string{"x"}}))
		}, "default does not match"},
		{"flag default mismatch", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", String: s, DefaultInt: 1}))
		}, "default does not match"},
		{"flag without target", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a"}))
		}, "exactly one of"},
		{"flag with two targets", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", String: s, Bool: new(bool)}))
		}, "exactly one of"},
		{"split without Strings", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", String: s, Split: true}))
		}, "Split requires"},
		{"invalid flag name", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "Bad_Name", String: s}))
		}, "lowercase"},
		{"invalid shortcut", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", Shortcut: '-', String: s}))
		}, "shortcut"},
		{"duplicate flag", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", String: s}, cliutil.FlagDef{Name: "a", String: s}))
		}, "duplicate flag 'a'"},
		{"duplicate shortcut", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", Shortcut: 'x', String: s}, cliutil.FlagDef{Name: "b", Shortcut: 'x', String: s}))
		}, "duplicate flag shortcut"},
		{"flag named like global", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "quiet", Bool: new(bool)}))
		}, "named like a global flag"},
		{"shortcut used by global", func(r *cliutil.Registry) {
			r.Register(flagsCmd(cliutil.FlagDef{Name: "a", Shortcut: 'q', String: s}))
		}, "used by a global flag"},
		{"passthrough with flags", func(r *cliutil.Registry) {
			r.Register(func() cliutil.Command {
				return newCmd[tagBad](cliutil.CmdArgs{Name: "bad", Passthrough: true, FlagSets: []*cliutil.FlagSet{{Name: "x"}}}, nil)
			})
		}, "Passthrough command cannot declare flags"},
		{"passthrough with operand", func(r *cliutil.Registry) {
			r.Register(func() cliutil.Command {
				return newCmd[tagBad](cliutil.CmdArgs{Name: "bad", Passthrough: true, ArgDefs: []*cliutil.ArgDef{{Name: "a", String: s}}}, nil)
			})
		}, "Passthrough command cannot declare an ArgDef"},
		{"missing parent", func(r *cliutil.Registry) {
			r.Register(flagsCmd(), &cmdOf[tagRepo]{})
		}, "parent command type"},
		{"same type twice", func(r *cliutil.Registry) {
			r.Register(flagsCmd())
			r.Register(flagsCmd())
		}, "registered twice"},
		{"same path twice", func(r *cliutil.Registry) {
			r.Register(flagsCmd())
			r.Register(func() cliutil.Command { return newCmd[tagBad2](cliutil.CmdArgs{Name: "bad"}, nil) })
		}, "share the path"},
		{"nil command", func(r *cliutil.Registry) {
			r.Register(func() cliutil.Command { return nil })
		}, "returned no command"},
		{"subcommand with FlagName", func(r *cliutil.Registry) {
			r.Register(func() cliutil.Command { return newCmd[tagRepo](cliutil.CmdArgs{Name: "repo"}, nil) })
			r.Register(func() cliutil.Command {
				return newCmd[tagBad](cliutil.CmdArgs{Name: "bad", FlagName: "bad"}, nil)
			}, &cmdOf[tagRepo]{})
		}, "subcommands cannot have FlagName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := cliutil.NewRegistry()
			tt.reg(r)
			err := r.Validate(nil)
			if !errors.Is(err, cliutil.ErrCommandRegistrationFailed) {
				t.Fatalf("want registration failure, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}

func TestFlagNameConflicts(t *testing.T) {
	t.Parallel()
	routed := func(name string) cliutil.CommandFunc {
		return func() cliutil.Command { return newCmd[tagBad](cliutil.CmdArgs{Name: "first", FlagName: name}, nil) }
	}
	tests := []struct {
		name string
		reg  func(*cliutil.Registry)
		opts cliutil.Options
		flag string
	}{
		{"conflicts with built-in", func(r *cliutil.Registry) { r.Register(routed("quiet")) }, nil, "quiet"},
		{"conflicts with app global", func(r *cliutil.Registry) { r.Register(routed("input")) }, newAppOptions(), "input"},
		{"two commands", func(r *cliutil.Registry) {
			r.Register(routed("go"))
			r.Register(func() cliutil.Command { return newCmd[tagBad2](cliutil.CmdArgs{Name: "second", FlagName: "go"}, nil) })
		}, nil, "go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := cliutil.NewRegistry()
			tt.reg(r)
			err := r.Validate(tt.opts)
			if !errors.Is(err, dt.ErrInvalidDuplicateFlag) {
				t.Fatalf("want duplicate flag error, got %v", err)
			}
			flag, _ := cliutil.ErrValue[string](err, "flag_name")
			if flag != tt.flag {
				t.Errorf("flag_name: want %q, got %q", tt.flag, flag)
			}
		})
	}
}

func TestValidRegistryValidates(t *testing.T) {
	t.Parallel()
	err := newRegistry().Validate(nil)
	if err != nil {
		t.Fatal(err)
	}
}
