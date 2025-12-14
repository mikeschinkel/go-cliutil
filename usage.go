package cliutil

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
)

type TopCmdRow struct {
	Display string // e.g. "serve [sub]" padded in template
	Desc    string
	Order   int // Display order (0=last, 1+=ordered)
}

type Usage struct {
	appinfo.AppInfo
	CLIWriter   Writer
	TopCmdRows  []TopCmdRow
	GlobalFlags []FlagRow
	Examples    []Example
}
type UsageArgs struct {
	appinfo.AppInfo
	Writer Writer
}

// BuildUsage Build the data for the template (auto + optional custom examples)
func BuildUsage(args UsageArgs) Usage {
	var rows []TopCmdRow
	var cmd Command
	var sub []Command
	var display string
	var globalFlags []FlagRow
	var globalFS *FlagSet
	var fd FlagDef
	var shortcut string

	// COMMANDS rows
	for _, cmd = range GetTopLevelCmds() {
		// Skip hidden commands
		if cmd.IsHidden() {
			continue
		}

		sub = GetSubCmds(cmd.Name())
		display = cmd.Name()
		if len(sub) > 0 {
			display += " [" + sub[0].Name() + "]"
		}
		rows = append(rows, TopCmdRow{
			Display: display,
			Desc:    cmd.Description(),
			Order:   cmd.Order(),
		})
	}
	// Sort by Order first (1-N), then by name alphabetically within each order
	// Commands with Order=0 (unspecified) appear last
	slices.SortFunc(rows, func(a, b TopCmdRow) int {
		// If orders are different
		if a.Order != b.Order {
			// Order 0 (unspecified) should come last
			if a.Order == 0 {
				return 1 // a comes after b
			}
			if b.Order == 0 {
				return -1 // a comes before b
			}
			// Both have explicit orders, sort ascending
			return a.Order - b.Order
		}
		// Same order, sort alphabetically by display name
		return strings.Compare(a.Display, b.Display)
	})

	// GLOBAL FLAGS rows
	globalFS = GetGlobalFlagSet()
	if globalFS != nil {
		for _, fd = range globalFS.FlagDefs {
			shortcut = ""
			if fd.Shortcut != 0 {
				shortcut = string(fd.Shortcut)
			}

			globalFlags = append(globalFlags, FlagRow{
				Name:     fd.Name,
				Shortcut: shortcut,
				Descr:    fd.Usage,
				Usage:    fd.Usage,
				Default:  fmt.Sprintf("%v", fd.Default),
				Required: fd.Required,
			})
		}
	}

	// EXAMPLES rows
	examples := collectExamples(args.ExeName())

	return Usage{
		AppInfo: appinfo.New(appinfo.Args{
			Name:        args.Name(),
			Description: args.Description(),
			Version:     args.Version(),
			ExeName:     args.ExeName(),
			InfoURL:     args.InfoURL(),
		}),
		CLIWriter:   args.Writer,
		TopCmdRows:  rows,
		GlobalFlags: globalFlags,
		Examples:    examples,
	}
}

// --- Example generation ----

func collectExamples(exe dt.Filename) []Example {
	// Start with universal help patterns:
	all := []Example{
		{Descr: "Show help for a specific command", Cmd: fmt.Sprintf("%s help <command>", exe)},
		{Descr: "Show help for a subcommand", Cmd: fmt.Sprintf("%s help <command> <subcommand>", exe)},
	}

	// Merge per-command contributions.
	// If a command implements ExampleProvider, use its Examples()
	// (and append autos depending on IncludeAutoExamples()).
	// Otherwise, auto-generate for that command.
	for _, cmd := range GetTopLevelCmds() {
		// Skip hidden commands
		if cmd.IsHidden() {
			continue
		}
		if cmd.NoExamples() {
			continue
		}
		custom := cmd.Examples()
		switch {
		case len(custom) == 0:
			// No custom examples returned => fall back to autos
			all = append(all, autoExamplesForCommand(exe, cmd)...)
		case cmd.AutoExamples():
			// Use auto-generated examples AND there are custom examples
			all = append(all, custom...)
			all = append(all, autoExamplesForCommand(exe, cmd)...)
		default:
			// Only use custom examples
			all = append(all, custom...)
		}
	}

	// You could de-dupe if multiple commands happen to produce identical examples
	all = dedupeExamples(all)
	return all
}

func autoExamplesForCommand(exe dt.Filename, cmd Command) []Example {
	var out []Example

	// 1) A canonical "help" example for the command itself
	out = append(out, Example{
		Descr: fmt.Sprintf("Help for %s", cmd.Name()),
		Cmd:   fmt.Sprintf("%s help %s", exe, cmd.Name()),
	})

	// 2) If it has subcommands, show help for the first subcommand
	sub := GetSubCmds(cmd.Name())
	if len(sub) > 0 {
		out = append(out, Example{
			Descr: fmt.Sprintf("Help for %s %s", cmd.Name(), sub[0].Name()),
			Cmd:   fmt.Sprintf("%s help %s %s", exe, cmd.Name(), sub[0].Name()),
		})
	}

	// 3) A runnable usage example built from Usage() + best-guess flags/args
	usage := strings.TrimSpace(cmd.Usage())
	if usage == "" {
		usage = cmd.Name()
	}

	// If Usage() *already* includes the command name (your example does), we don't duplicate it.
	var cmdline string
	if strings.HasPrefix(usage, cmd.Name()) {
		cmdline = fmt.Sprintf("%s %s", exe, usage)
	} else {
		cmdline = fmt.Sprintf("%s %s %s", exe, cmd.Name(), usage)
	}

	// Append sample flags and args, using Example if present; else Default; else omit.
	flags := sampleFlags(cmd)
	args := sampleArgs(cmd)

	suffix := strings.TrimSpace(strings.Join(append(flags, args...), " "))
	if suffix != "" {
		cmdline = strings.TrimSpace(cmdline + " " + suffix)
	}

	out = append(out, Example{
		Descr: fmt.Sprintf("Example: %s", cmd.Name()),
		Cmd:   normalizeSpaces(cmdline),
	})

	return out
}

func sampleFlags(cmd Command) []string {
	var parts []string
	for _, fs := range cmd.FlagSets() {
		for _, fd := range fs.FlagDefs {
			val := fd.Example
			if val == "" && fd.Default != nil {
				val = fmt.Sprintf("%v", fd.Default)
			}
			// Only include flags when we have a decent sample; skip booleans set to false, etc.
			if val != "" {
				// Use GNU long form: --name=value
				parts = append(parts, fmt.Sprintf("--%s=%s", fd.Name, quoteIfNeeded(val)))
			}
		}
	}
	return parts
}

func sampleArgs(cmd Command) (parts []string) {
	// We can only derive arg defs if your Command exposes them.
	// If ArgDefs are only embedded in your CmdBase, expose them via an optional interface:
	for _, ad := range cmd.ArgDefs() {
		val := ad.Example
		if val == "" && ad.Default != nil {
			val = fmt.Sprintf("%v", ad.Default)
		}
		// For required args with no example/default, put a placeholder to signal requiredness.
		if val == "" && ad.Required {
			val = "<" + ad.Name + ">"
		}
		if val != "" {
			parts = append(parts, quoteIfNeeded(val))
		}
	}
	return
}

func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t\"'") {
		s = fmt.Sprintf("%q", s)
	}
	return s
}

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func dedupeExamples(in []Example) []Example {
	seen := map[string]struct{}{}
	var out []Example
	for _, e := range in {
		key := e.Descr + "||" + e.Cmd
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, e)
	}
	return out
}

// --- Command-specific help ---

type FlagRow struct {
	Flag     string
	Descr    string
	Name     string
	Shortcut string
	Usage    string
	Default  string
	Required bool
}

type SubCmdRow struct {
	Name  string
	Descr string
	Cmd   CmdUsage
}

type ArgRow struct {
	Arg      string
	Descr    string
	Name     string
	Usage    string
	Required bool
	Default  string
	Example  string
}

type CmdUsage struct {
	CLIName     string
	CmdName     string
	Usage       string
	Description string
	Width       int
	ArgRows     []ArgRow
	FlagRows    []FlagRow
	SubCmdRows  []SubCmdRow
	Examples    []Example
}

// BuildCmdUsage builds the data structure for command-specific help
func BuildCmdUsage(cmd Command) CmdUsage {
	var args, usage strings.Builder
	var argRows []ArgRow
	var flagRows []FlagRow
	var subCmdRows []SubCmdRow
	var subCmd Command
	var maxSize int
	var hasOptArgs, hasFlags bool

	argDefs := cmd.ArgDefs()
	// Collect arguments
	for i, ad := range argDefs {
		arg := fmt.Sprintf("<%s>", ad.Name)
		if !ad.Required {
			hasOptArgs = true
			args.WriteString("[")
		}
		args.WriteString(arg)
		if i < len(argDefs)-1 {
			args.WriteString(" ")
		}

		descr := ad.Usage
		def := fmt.Sprintf("%v", ad.Default)
		if def != "" {
			descr = fmt.Sprintf("%s (default=%s)", descr, def)
		}
		if ad.Required {
			descr = fmt.Sprintf("%s [required]", descr)
		}
		argRow := ArgRow{
			Arg:      arg,
			Descr:    appendCompulsion(descr, ad.Required),
			Name:     ad.Name,
			Usage:    ad.Usage,
			Required: ad.Required,
			Default:  fmt.Sprintf("%v", ad.Default),
			Example:  ad.Example,
		}
		argRows = append(argRows, argRow)
		maxSize = max(len(argRow.Arg), maxSize)
	}
	if hasOptArgs {
		args.WriteString("]")
	}

	// Collect flags from command's FlagSets
	for _, fs := range cmd.FlagSets() {
		for _, fd := range fs.FlagDefs {
			hasFlags = true
			flag := "--" + fd.Name
			if fd.Shortcut != 0 {
				flag = fmt.Sprintf("-%c, %s", fd.Shortcut, flag)
			}
			descr := fd.Usage
			def := fmt.Sprintf("%v", fd.Default)
			if def != "" {
				descr = fmt.Sprintf("%s [default=%s]", descr, def)
			}
			if fd.Required {
				hasOptArgs = true
			}
			flagRows = append(flagRows, FlagRow{
				Flag:     flag,
				Descr:    appendCompulsion(descr, fd.Required),
				Name:     fd.Name,
				Shortcut: string(fd.Shortcut),
				Usage:    fd.Usage,
				Default:  fmt.Sprintf("%v", fd.Default),
				Required: fd.Required,
			})
			maxSize = max(len(flag)+2, maxSize)
		}
	}

	// Collect subcommands
	for _, subCmd = range GetSubCmds(cmd.Name()) {
		if subCmd.IsHidden() {
			continue
		}
		subCmdRows = append(subCmdRows, SubCmdRow{
			Name:  subCmd.Name(),
			Descr: subCmd.Description(),
			Cmd: CmdUsage{
				CmdName:     subCmd.Name(),
				Usage:       subCmd.Usage(),
				Description: subCmd.Description(),
			},
		})
		maxSize = max(len(subCmd.Name()), maxSize)
	}
	maxSize++

	// Get examples
	examples := cmd.Examples()
	//if len(examples) == 0 && cmd.AutoExamples() {
	//	// TODO: Generate auto examples for this command
	//}

	switch {
	case cmd.Usage() != "":
		usage.WriteString(cmd.Usage())
	default:
		names := cmd.FullNames()
		// TODOL Test this for subcommands
		usage.WriteString(names[0])
		if hasOptArgs {
			usage.WriteString(" ")
			usage.WriteString(args.String())
		}
		if hasFlags {
			usage.WriteString(" [flags]")
		}
	}

	return CmdUsage{
		CLIName:     cmd.CLIName(),
		CmdName:     cmd.Name(),
		Usage:       usage.String(),
		Description: cmd.Description(),
		ArgRows:     argRows,
		FlagRows:    flagRows,
		SubCmdRows:  subCmdRows,
		Examples:    examples,
		Width:       maxSize,
	}
}

func appendCompulsion(s string, required bool) string {
	var c string
	switch required {
	case true:
		c = "required"
	case false:
		c = "optional"
	}
	return fmt.Sprintf("%s [%s]", s, c)
}
