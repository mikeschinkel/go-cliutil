package main

import (
	"strings"

	"github.com/mikeschinkel/go-cliutil"
)

func init() {
	cliutil.RegisterCommand(NewHelpCmd)
	cliutil.RegisterCommand(NewProjectCmd)
	cliutil.RegisterCommand(NewProjectListCmd, &ProjectCmd{})
	cliutil.RegisterCommand(NewProjectMemberCmd, &ProjectCmd{})
	cliutil.RegisterCommand(NewProjectMemberAddCmd, &ProjectMemberCmd{})
	cliutil.RegisterCommand(NewExportCmd)
	cliutil.RegisterCommand(NewSetupCmd)
	cliutil.RegisterCommand(NewDebugCmd)
	cliutil.RegisterCommand(NewExecCmd)
	cliutil.RegisterCommand(NewEchoCmd)
	cliutil.RegisterCommand(NewReportCmd)
}

// HelpCmd shows a variadic operand holding the help topic.
type HelpCmd struct {
	*cliutil.CmdBase
	topic []string
}

// NewHelpCmd constructs the help command.
func NewHelpCmd() cliutil.Command {
	c := &HelpCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "help",
		Usage:       "help [command]",
		Description: "Show help information",
		Order:       99,
		ArgDefs:     []*cliutil.ArgDef{{Name: "command", Usage: "Command to explain", Strings: &c.topic}},
	})
	return c
}

// Handle shows the help the topic asks for.
func (c *HelpCmd) Handle(args cliutil.HandleArgs) error {
	if len(c.topic) == 0 {
		return cliutil.ShowMainHelp(args.UsageArgs())
	}
	return cliutil.ShowCmdHelp(c.topic, args.UsageArgs())
}

// ProjectCmd shows a command group with a default subcommand.
type ProjectCmd struct{ *cliutil.CmdBase }

// NewProjectCmd constructs the project group.
func NewProjectCmd() cliutil.Command {
	return &ProjectCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "project",
		Description: "Work with projects",
		DelegateTo:  &ProjectListCmd{},
	})}
}

// ProjectListCmd shows a split Strings flag, typed flag defaults and Flag[T].
type ProjectListCmd struct {
	*cliutil.CmdBase
	format string
	tags   []string
	limit  int
}

// NewProjectListCmd constructs the project list command.
func NewProjectListCmd() cliutil.Command {
	c := &ProjectListCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "list",
		Description: "List projects",
		FlagSets: []*cliutil.FlagSet{{
			Name: "list",
			FlagDefs: []cliutil.FlagDef{
				{Name: "format", Usage: "Output format", DefaultString: "table", String: &c.format},
				{Name: "tag", Usage: "Tags to match; repeat or separate with commas", Split: true, Strings: &c.tags},
				{Name: "limit", Usage: "Most projects to list", DefaultInt: 10, Int: &c.limit},
			},
		}},
	})
	return c
}

// Handle prints the listing's settings.
func (c *ProjectListCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("list format=%s tags=%s limit=%d\n",
		c.format, strings.Join(c.tags, "|"), args.Flag[int]("limit"))
	return nil
}

// ProjectMemberCmd shows a nested command group.
type ProjectMemberCmd struct{ *cliutil.CmdBase }

// NewProjectMemberCmd constructs the project member group.
func NewProjectMemberCmd() cliutil.Command {
	return &ProjectMemberCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "member",
		Description: "Work with project members",
	})}
}

// ProjectMemberAddCmd shows a variadic operand with count limits.
type ProjectMemberAddCmd struct {
	*cliutil.CmdBase
	names []string
}

// NewProjectMemberAddCmd constructs the project member add command.
func NewProjectMemberAddCmd() cliutil.Command {
	c := &ProjectMemberAddCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "add",
		Description: "Add one to three members",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "name", Usage: "Members to add", MinCount: 1, MaxCount: 3, Strings: &c.names},
		},
	})
	return c
}

// Handle prints the members added.
func (c *ProjectMemberAddCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("added %s\n", strings.Join(c.names, ","))
	return nil
}

// ExportCmd shows a typed ArgDef default, ApplyConfig on a command, and OptionsAs.
type ExportCmd struct {
	*cliutil.CmdBase
	dir    string
	format string
}

// NewExportCmd constructs the export command.
func NewExportCmd() cliutil.Command {
	c := &ExportCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "export",
		Description: "Export data to a directory",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "dir", Usage: "Directory to export to", DefaultString: ".", String: &c.dir},
		},
		FlagSets: []*cliutil.FlagSet{{
			Name: "export",
			FlagDefs: []cliutil.FlagDef{
				{Name: "format", Usage: "Export format", DefaultString: "json", String: &c.format},
			},
		}},
	})
	return c
}

// ApplyConfig sets the export format from config; a typed --format still wins.
func (c *ExportCmd) ApplyConfig(cfg cliutil.Config) error {
	c.format = cliutil.ConfigAs[*RootConfig](cfg).ExportFormat()
	return nil
}

// Handle prints where the export would go and where each value came from.
func (c *ExportCmd) Handle(args cliutil.HandleArgs) error {
	opts := args.OptionsAs[*Options]()
	format, _ := args.Flags.Get("format")
	args.Writer.Printf("export dir=%s format=%s source=%s input=%s region=%s timeout=%s\n",
		c.dir, c.format, format.Source, opts.Input, opts.Region, opts.Timeout())
	return nil
}

// SetupCmd shows flag routing: --setup runs this command.
type SetupCmd struct{ *cliutil.CmdBase }

// NewSetupCmd constructs the setup command.
func NewSetupCmd() cliutil.Command {
	return &SetupCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "setup",
		Description: "Set the app up",
		FlagName:    "setup",
	})}
}

// Handle reports that setup ran.
func (c *SetupCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("setup ran\n")
	return nil
}

// DebugCmd shows a hidden command: it runs but help does not list it.
type DebugCmd struct{ *cliutil.CmdBase }

// NewDebugCmd constructs the debug command.
func NewDebugCmd() cliutil.Command {
	return &DebugCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "debug",
		Description: "Print diagnostics",
		Hide:        true,
	})}
}

// Handle reports that debug ran.
func (c *DebugCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("debug ran\n")
	return nil
}

// ExecCmd shows Passthrough: every later word arrives unparsed.
type ExecCmd struct{ *cliutil.CmdBase }

// NewExecCmd constructs the exec command.
func NewExecCmd() cliutil.Command {
	return &ExecCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "exec",
		Description: "Pass words to another program",
		Passthrough: true,
	})}
}

// Handle prints the words it received.
func (c *ExecCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("exec %s\n", strings.Join(args.Parameters, " "))
	return nil
}

// EchoCmd shows -- ending flags, so operands can start with a dash.
type EchoCmd struct {
	*cliutil.CmdBase
	words []string
}

// NewEchoCmd constructs the echo command.
func NewEchoCmd() cliutil.Command {
	c := &EchoCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "echo",
		Description: "Print words; use -- before words that start with a dash",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "word", Usage: "Words to print", DefaultStrings: []string{"nothing"}, Strings: &c.words},
		},
	})
	return c
}

// Handle prints the words.
func (c *EchoCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("%s\n", strings.Join(c.words, " "))
	return nil
}

// ReportCmd shows one command running another through Runner.
type ReportCmd struct{ *cliutil.CmdBase }

// NewReportCmd constructs the report command.
func NewReportCmd() cliutil.Command {
	return &ReportCmd{CmdBase: cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "report",
		Description: "Print a report built from other commands",
	})}
}

// Handle runs project list as part of the report.
func (c *ReportCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("report start\n")
	return args.Runner.Run([]string{"project", "list", "--limit", "2"})
}
