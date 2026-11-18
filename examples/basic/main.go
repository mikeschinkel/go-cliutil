// Command basic is the smallest useful go-cliutil app: two commands and help.
//
//	go run . greet World
//	go run . greet World --shout
//	go run . add 2 3 5
//	go run . help greet
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
)

func init() {
	cliutil.RegisterCommand(NewGreetCmd)
	cliutil.RegisterCommand(NewAddCmd)
	cliutil.RegisterCommand(NewHelpCmd)
}

// GreetCmd prints a greeting.
type GreetCmd struct {
	*cliutil.CmdBase
	name  string
	shout bool
}

// NewGreetCmd constructs the greet command.
func NewGreetCmd() cliutil.Command {
	c := &GreetCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "greet",
		Description: "Greet someone by name",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "name", Usage: "Who to greet", DefaultString: "World", String: &c.name},
		},
		FlagSets: []*cliutil.FlagSet{{
			Name: "greet",
			FlagDefs: []cliutil.FlagDef{
				{Name: "shout", Shortcut: 's', Usage: "Greet loudly", Bool: &c.shout},
			},
		}},
	})
	return c
}

// Handle prints the greeting.
func (c *GreetCmd) Handle(args cliutil.HandleArgs) error {
	greeting := fmt.Sprintf("Hello, %s!", c.name)
	if c.shout {
		greeting = strings.ToUpper(greeting)
	}
	args.Writer.Printf("%s\n", greeting)
	return nil
}

// AddCmd adds whole numbers.
type AddCmd struct {
	*cliutil.CmdBase
	numbers []string
}

// NewAddCmd constructs the add command.
func NewAddCmd() cliutil.Command {
	c := &AddCmd{}
	c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
		Name:        "add",
		Description: "Add whole numbers",
		ArgDefs: []*cliutil.ArgDef{
			{Name: "number", Usage: "Numbers to add", MinCount: 2, Strings: &c.numbers},
		},
	})
	return c
}

// Handle prints the sum.
func (c *AddCmd) Handle(args cliutil.HandleArgs) error {
	sum := 0
	for _, word := range c.numbers {
		number, err := strconv.Atoi(word)
		if err != nil {
			return fmt.Errorf("%q is not a whole number", word)
		}
		sum += number
	}
	args.Writer.Printf("%d\n", sum)
	return nil
}

// HelpCmd shows help for the app or for one command.
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
		ArgDefs: []*cliutil.ArgDef{
			{Name: "command", Usage: "Command to explain", Strings: &c.topic},
		},
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

func main() {
	runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		AppInfo: appinfo.New(appinfo.Args{
			Name:        "basic",
			Description: "A minimal go-cliutil app",
			ExeName:     "basic",
			InfoURL:     "https://github.com/mikeschinkel/go-cliutil",
		}),
	})
	err := runner.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cliutil.ExitCode(err))
	}
}
