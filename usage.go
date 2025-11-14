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
}

type Usage struct {
	appinfo.AppInfo
	CLIWriter  Writer
	TopCmdRows []TopCmdRow
	Examples   []Example
}
type UsageArgs struct {
	appinfo.AppInfo
	Writer Writer
}

// BuildUsage Build the data for the template (auto + optional custom examples)
func BuildUsage(args UsageArgs) Usage {
	// COMMANDS rows
	var rows []TopCmdRow
	for _, cmd := range GetTopLevelCmds() {
		sub := GetSubCmds(cmd.Name())
		display := cmd.Name()
		if len(sub) > 0 {
			display += " [" + sub[0].Name() + "]"
		}
		rows = append(rows, TopCmdRow{
			Display: display,
			Desc:    cmd.Description(),
		})
	}
	// Keep a stable, nice ordering if your registry isn't already
	slices.SortFunc(rows, func(a, b TopCmdRow) int {
		return strings.Compare(a.Display, b.Display)
	})

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
		CLIWriter:  args.Writer,
		TopCmdRows: rows,
		Examples:   examples,
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
