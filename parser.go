package cliutil

import (
	"fmt"
	"slices"
	"strings"
)

const helpCommandName = "help"

// typedFlag collects what the user typed for one flag.
type typedFlag struct {
	def    *FlagDef
	global bool
	raws   []string
	words  []string
	value  any
}

// parsedLine is the result of parsing one command line.
type parsedLine struct {
	cmd        Command
	path       []string
	parameters []string
	typed      []*typedFlag
	operands   Operands
}

// flagWord is a word split into its flag parts.
type flagWord struct {
	name     string // Long name or one-letter shortcut, without dashes.
	short    bool   // Written with a single dash.
	value    string
	hasValue bool
}

// isFlagWord reports whether a word has the shape of a flag.
func isFlagWord(word string) bool {
	return len(word) > 1 && word[0] == '-' && word != "--"
}

// splitFlagWord splits --name, --name=value, -x and -x=value.
func splitFlagWord(word string) (fw flagWord) {
	body, isLong := strings.CutPrefix(word, "--")
	if !isLong {
		body = word[1:]
		fw.short = true
	}
	fw.name, fw.value, fw.hasValue = strings.Cut(body, "=")
	return fw
}

// flagLookup finds flag definitions by long name or shortcut.
type flagLookup []*FlagDef

func newFlagLookup(sets ...[]FlagDef) (lookup flagLookup) {
	for _, defs := range sets {
		for i := range defs {
			lookup = append(lookup, &defs[i])
		}
	}
	return lookup
}

func commandFlagLookup(cmd Command) (lookup flagLookup) {
	if cmd == nil {
		return nil
	}
	for _, fs := range cmd.FlagSets() {
		lookup = append(lookup, newFlagLookup(fs.FlagDefs)...)
	}
	return lookup
}

// find applies GNU rules: a long name takes two dashes and a shortcut takes one.
// A multi-letter name after a single dash matches nothing.
func (l flagLookup) find(fw flagWord) *FlagDef {
	for _, fd := range l {
		if fw.short && len(fw.name) == 1 && fd.Shortcut == fw.name[0] {
			return fd
		}
		if !fw.short && fd.Name == fw.name {
			return fd
		}
	}
	return nil
}

// lineParser parses one command line against one run's commands.
type lineParser struct {
	set     *CommandSet
	globals flagLookup
	mode    CLIUtilMode
}

// commandScan is what the first pass learns about the command words.
type commandScan struct {
	cmd         Command
	path        string   // Dot path of the words matched so far.
	typedWords  []string // The command words as typed.
	wordIndexes []int    // Index of each command word.
	globalWords []string // Global flag words passed on the way.
	stoppedAt   int      // Index of the word that ended the scan.
}

// parse resolves the command and sorts every word into flags and operands.
func (p *lineParser) parse(words []string) (line *parsedLine, err error) {
	var scan commandScan

	if p.mode == FlagsOnlyMode {
		return p.parseFlagsOnly(words)
	}

	helpAt := findHelpFlag(words)
	if helpAt >= 0 {
		rest := slices.Delete(slices.Clone(words), helpAt, helpAt+1)
		scan = p.scanCommand(rest)
		if !isPassthroughBefore(scan, helpAt) {
			topic := append([]string{helpCommandName}, scan.typedWords...)
			words = append(topic, scan.globalWords...)
		}
	}

	scan = p.scanCommand(words)
	if scan.cmd == nil && scan.stoppedAt >= len(words) {
		// Nothing but global flags was typed, so show help.
		words = append([]string{helpCommandName}, words...)
		scan = p.scanCommand(words)
	}
	if scan.cmd != nil {
		scan.cmd, scan.path = p.set.GetDefaultCommand(scan.path)
	}
	return p.sortWords(words, scan)
}

// parseFlagsOnly runs the app's only command with no command word.
func (p *lineParser) parseFlagsOnly(words []string) (*parsedLine, error) {
	commands := p.set.All()
	if len(commands) != 1 {
		return nil, NewErr(ErrInvalidRegistration, "command_count", len(commands),
			fmt.Errorf("FlagsOnlyMode requires exactly one registered command; found %d", len(commands)))
	}
	return p.sortWords(words, commandScan{cmd: commands[0], path: commands[0].Name()})
}

// findHelpFlag returns the index of --help before any --, or -1.
func findHelpFlag(words []string) int {
	for i, word := range words {
		switch word {
		case "--":
			return -1
		case "--help":
			return i
		}
	}
	return -1
}

// isPassthroughBefore reports whether a Passthrough command's words precede the
// --help at index helpAt, which makes --help that command's own word.
func isPassthroughBefore(scan commandScan, helpAt int) bool {
	if scan.cmd == nil || !scan.cmd.Passthrough() {
		return false
	}
	return helpAt > scan.wordIndexes[len(scan.wordIndexes)-1]
}

// scanCommand walks the leading words, skipping global flags, and matches the
// longest run of command words. It stops at the first other word.
func (p *lineParser) scanCommand(words []string) (scan commandScan) {
	i := 0
	for i < len(words) {
		word := words[i]
		if word == "--" {
			break
		}
		if isFlagWord(word) {
			fw := splitFlagWord(word)
			fd := p.globals.find(fw)
			if fd != nil {
				width := 1
				if fd.takesValue() && !fw.hasValue {
					width = 2
				}
				end := min(i+width, len(words))
				scan.globalWords = append(scan.globalWords, words[i:end]...)
				i += width
				continue
			}
			routed := p.routedCommand(scan, fw)
			if routed == nil {
				break
			}
			scan.match(routed, routed.Name(), i)
			i++
			continue
		}
		next := word
		if scan.path != "" {
			next = scan.path + "." + word
		}
		cmd := p.set.GetExactCommand(next)
		if cmd == nil {
			break
		}
		scan.match(cmd, next, i)
		scan.typedWords[len(scan.typedWords)-1] = word
		i++
		if cmd.Passthrough() {
			break
		}
	}
	scan.stoppedAt = min(i, len(words))
	return scan
}

// routedCommand returns the command a leading --name flag routes to.
func (p *lineParser) routedCommand(scan commandScan, fw flagWord) Command {
	if scan.cmd != nil || fw.short || fw.hasValue {
		return nil
	}
	return p.set.flagRouted(fw.name)
}

func (s *commandScan) match(cmd Command, path string, index int) {
	s.cmd = cmd
	s.path = path
	s.typedWords = append(s.typedWords, cmd.Name())
	s.wordIndexes = append(s.wordIndexes, index)
}

// sortWords makes the second pass, sorting each word into a flag or an operand.
func (p *lineParser) sortWords(words []string, scan commandScan) (line *parsedLine, err error) {
	var cmdFlags flagLookup
	var argDef *ArgDef

	line = &parsedLine{cmd: scan.cmd}
	if scan.cmd != nil {
		line.path = strings.Split(scan.path, ".")
		cmdFlags = commandFlagLookup(scan.cmd)
		if defs := scan.cmd.ArgDefs(); len(defs) > 0 {
			argDef = defs[0]
		}
	}

	lastCommandWord := -1
	if len(scan.wordIndexes) > 0 {
		lastCommandWord = scan.wordIndexes[len(scan.wordIndexes)-1]
	}
	paramStart := lastCommandWord + 1
	line.parameters = slices.Clone(words[paramStart:])

	passthrough := scan.cmd != nil && scan.cmd.Passthrough()
	end := len(words)
	if passthrough {
		end = paramStart
		for i := paramStart; i < len(words); i++ {
			line.addOperand(words[i], i-paramStart, nil)
		}
	}

	typedByDef := make(map[*FlagDef]*typedFlag)
	for i := 0; i < end; i++ {
		word := words[i]
		if slices.Contains(scan.wordIndexes, i) {
			continue
		}
		if word == "--" {
			for j := i + 1; j < len(words); j++ {
				line.addOperand(words[j], j-paramStart, argDef)
			}
			break
		}
		if !isFlagWord(word) {
			if scan.cmd == nil {
				return nil, NewErr(ErrUnknownCommand, "command", word,
					fmt.Errorf("unknown command: %s", word))
			}
			line.addOperand(word, i-paramStart, argDef)
			continue
		}

		fw := splitFlagWord(word)
		fd := p.globals.find(fw)
		global := fd != nil
		if fd == nil && i > lastCommandWord {
			fd = cmdFlags.find(fw)
		}
		if fd == nil {
			return nil, p.unknownFlagErr(words, i, scan, fw)
		}

		raw := fw.value
		flagWords := []string{word}
		switch {
		case fw.hasValue:
		case !fd.takesValue():
			raw = "true"
		case i+1 < len(words):
			i++
			raw = words[i]
			flagWords = append(flagWords, raw)
		default:
			return nil, NewErr(ErrFlagValueMissing, "flag", word,
				fmt.Errorf("flag %s needs a value", word))
		}

		typed, seen := typedByDef[fd]
		if !seen {
			typed = &typedFlag{def: fd, global: global}
			typedByDef[fd] = typed
			line.typed = append(line.typed, typed)
		}
		typed.raws = append(typed.raws, raw)
		typed.words = append(typed.words, flagWords...)
	}

	if scan.cmd == nil {
		return nil, NewErr(ErrUnknownCommand, "command_line", strings.Join(words, " "),
			fmt.Errorf("unknown command: %s", strings.Join(words, " ")))
	}

	for _, typed := range line.typed {
		typed.value, err = typed.def.parseWords(typed.raws)
		if err != nil {
			return nil, err
		}
	}
	return line, nil
}

func (l *parsedLine) addOperand(value string, position int, def *ArgDef) {
	l.operands = append(l.operands, Operand{Value: value, Position: position, Def: def})
}

// unknownFlagErr names the flag, and says so when the flag belongs to a command
// whose words come later on the line.
func (p *lineParser) unknownFlagErr(words []string, at int, scan commandScan, fw flagWord) error {
	path := scan.path
	for _, word := range words[at+1:] {
		if isFlagWord(word) {
			continue
		}
		next := word
		if path != "" {
			next = path + "." + word
		}
		cmd := p.set.GetExactCommand(next)
		if cmd == nil {
			break
		}
		path = next
		if commandFlagLookup(cmd).find(fw) != nil {
			command := strings.ReplaceAll(path, ".", " ")
			return NewErr(ErrFlagBeforeCommand, "flag", words[at], "command", command,
				fmt.Errorf("flag %s must follow the command words '%s'", words[at], command))
		}
	}
	return NewErr(ErrUnknownFlag, "flag", words[at], fmt.Errorf("unknown flag: %s", words[at]))
}

// assignOperands checks the operand count and writes operands to the target.
func assignOperands(cmd Command, operands Operands) (err error) {
	var argDef *ArgDef

	if cmd.Passthrough() {
		goto end
	}
	if defs := cmd.ArgDefs(); len(defs) > 0 {
		argDef = defs[0]
	}

	switch {
	case argDef == nil:
		if len(operands) > 0 {
			err = tooManyOperandsErr(operands[0].Value)
		}
	case argDef.isVariadic():
		err = assignVariadic(argDef, operands)
	case len(operands) > 1:
		err = tooManyOperandsErr(operands[1].Value)
	case len(operands) == 1:
		*argDef.String = operands[0].Value
	case argDef.Required:
		err = NewErr(ErrTooFewOperands, "argument", argDef.Name,
			fmt.Errorf("required argument '%s' missing", argDef.Name))
	default:
		*argDef.String = argDef.DefaultString
	}

end:
	return err
}

func assignVariadic(argDef *ArgDef, operands Operands) (err error) {
	count := len(operands)
	minimum := argDef.minOperands()

	switch {
	case count < minimum:
		err = NewErr(ErrTooFewOperands, "argument", argDef.Name, "count", count,
			fmt.Errorf("argument '%s' needs at least %d value(s); got %d", argDef.Name, minimum, count))
	case argDef.MaxCount > 0 && count > argDef.MaxCount:
		err = NewErr(ErrTooManyOperands, "argument", argDef.Name, "count", count,
			fmt.Errorf("argument '%s' takes at most %d value(s); got %d", argDef.Name, argDef.MaxCount, count))
	case count == 0:
		*argDef.Strings = slices.Clone(argDef.DefaultStrings)
	default:
		*argDef.Strings = operands.Strings()
	}
	return err
}

func tooManyOperandsErr(word string) error {
	return NewErr(ErrTooManyOperands, "operand", word, fmt.Errorf("unexpected argument: %s", word))
}
