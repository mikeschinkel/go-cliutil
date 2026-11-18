package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
)

// cmdOf gives each tag type its own command type, since a Registry identifies
// commands by type.
type cmdOf[Tag any] struct {
	*cliutil.CmdBase
	run func(cliutil.HandleArgs) error
}

func (c *cmdOf[Tag]) Handle(args cliutil.HandleArgs) error {
	if c.run == nil {
		args.Writer.Printf("ran=%s\n", strings.Join(args.CommandPath, "."))
		return nil
	}
	return c.run(args)
}

func newCmd[Tag any](args cliutil.CmdArgs, run func(cliutil.HandleArgs) error) cliutil.Command {
	return &cmdOf[Tag]{CmdBase: cliutil.NewCmdBase(args), run: run}
}

type (
	tagHelp   struct{}
	tagRepo   struct{}
	tagSetup  struct{}
	tagTask   struct{}
	tagShow   struct{}
	tagList   struct{}
	tagRaw    struct{}
	tagGroup  struct{}
	tagMember struct{}
	tagRouted struct{}
	tagBare   struct{}
	tagBad    struct{}
	tagBad2   struct{}
)

func newHelpCmd() cliutil.Command {
	var topic []string
	return newCmd[tagHelp](cliutil.CmdArgs{
		Name:    "help",
		ArgDefs: []*cliutil.ArgDef{{Name: "command", Strings: &topic}},
	}, func(args cliutil.HandleArgs) error {
		args.Writer.Printf("help topic=%q path=%q\n", topic, args.CommandPath)
		return nil
	})
}

// newSetupCmd mirrors the "repo setup" command that exposed the original bugs.
func newSetupCmd() cliutil.Command {
	var dir, format string
	var check bool
	var count int
	var size int64
	var tags, names []string
	return newCmd[tagSetup](cliutil.CmdArgs{
		Name:    "setup",
		ArgDefs: []*cliutil.ArgDef{{Name: "dir", DefaultString: "default-dir", String: &dir}},
		FlagSets: []*cliutil.FlagSet{{Name: "setup", FlagDefs: []cliutil.FlagDef{
			{Name: "check", Shortcut: 'c', Bool: &check},
			{Name: "format", Shortcut: 'F', DefaultString: "text", String: &format},
			{Name: "count", DefaultInt: 7, Int: &count},
			{Name: "size", DefaultInt64: 9, Int64: &size},
			{Name: "tag", Split: true, Strings: &tags},
			{Name: "name", DefaultStrings: []string{"anon"}, Strings: &names},
		}}},
	}, func(args cliutil.HandleArgs) error {
		args.Writer.Printf("check=%v format=%s count=%d size=%d dir=%s tags=%q names=%q operands=%q params=%q\n",
			check, format, count, size, dir, tags, names, args.Operands.Strings(), args.Parameters)
		return nil
	})
}

func newShowCmd() cliutil.Command {
	var id string
	var allFields, pretty bool
	return newCmd[tagShow](cliutil.CmdArgs{
		Name:    "show",
		ArgDefs: []*cliutil.ArgDef{{Name: "id", Required: true, String: &id}},
		FlagSets: []*cliutil.FlagSet{{Name: "show", FlagDefs: []cliutil.FlagDef{
			{Name: "all-fields", Bool: &allFields},
			{Name: "pretty", Shortcut: 'p', Bool: &pretty},
		}}},
	}, func(args cliutil.HandleArgs) error {
		args.Writer.Printf("id=%s all=%v pretty=%v\n", id, allFields, pretty)
		return nil
	})
}

func newListCmd() cliutil.Command {
	var items []string
	return newCmd[tagList](cliutil.CmdArgs{
		Name:    "list",
		ArgDefs: []*cliutil.ArgDef{{Name: "item", MinCount: 1, MaxCount: 2, Strings: &items}},
	}, func(args cliutil.HandleArgs) error {
		args.Writer.Printf("items=%q\n", items)
		return nil
	})
}

func newRawCmd() cliutil.Command {
	return newCmd[tagRaw](cliutil.CmdArgs{Name: "raw", Passthrough: true}, func(args cliutil.HandleArgs) error {
		args.Writer.Printf("params=%q operands=%q\n", args.Parameters, args.Operands.Strings())
		return nil
	})
}

// newRegistry returns the registry most tests run against.
func newRegistry() *cliutil.Registry {
	r := cliutil.NewRegistry()
	r.Register(newHelpCmd)
	r.Register(func() cliutil.Command { return newCmd[tagRepo](cliutil.CmdArgs{Name: "repo"}, nil) })
	r.Register(newSetupCmd, &cmdOf[tagRepo]{})
	r.Register(func() cliutil.Command { return newCmd[tagTask](cliutil.CmdArgs{Name: "task"}, nil) })
	r.Register(newShowCmd, &cmdOf[tagTask]{})
	r.Register(newListCmd)
	r.Register(newRawCmd)
	r.Register(func() cliutil.Command {
		return newCmd[tagGroup](cliutil.CmdArgs{Name: "group", DelegateTo: &cmdOf[tagMember]{}}, nil)
	})
	r.Register(func() cliutil.Command { return newCmd[tagMember](cliutil.CmdArgs{Name: "member"}, nil) }, &cmdOf[tagGroup]{})
	r.Register(func() cliutil.Command {
		return newCmd[tagRouted](cliutil.CmdArgs{Name: "routed", FlagName: "route-me"}, nil)
	})
	r.Register(func() cliutil.Command { return newCmd[tagBare](cliutil.CmdArgs{Name: "bare"}, nil) })
	return r
}

func run(t *testing.T, line string) cliutiltest.Result {
	t.Helper()
	return cliutiltest.Run(t, cliutiltest.Args{Registry: newRegistry(), CommandLine: strings.Fields(line)})
}

// mustPanic returns the message of the panic fn raises.
func mustPanic(t *testing.T, fn func()) (message string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected a panic")
		}
		message = fmt.Sprint(recovered)
	}()
	fn()
	return message
}
