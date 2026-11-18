package cliutil

// FlagSource says where a flag's value came from.
type FlagSource int

const (
	DefaultSource     FlagSource = iota // The flag's typed default.
	ConfigSource                        // The app's config.
	CommandLineSource                   // A word the user typed.
)

// String returns the source's name.
func (s FlagSource) String() string {
	switch s {
	case ConfigSource:
		return "config"
	case CommandLineSource:
		return "command line"
	case DefaultSource:
		return "default"
	}
	return "unknown"
}

// Flag describes one flag that applies to a run.
type Flag struct {
	Name   string     // Long name, without dashes.
	Value  any        // Resolved value: string, bool, int, int64 or []string.
	Source FlagSource // Where Value came from.
	Global bool       // True for a global flag, false for a command's own flag.
	Words  []string   // The words as typed, such as ["--format","json"]; nil when not typed.
	Def    *FlagDef   // The flag's definition.
}

// Flags holds every flag that applies to a run, global and command.
type Flags struct {
	all   []Flag
	typed []string // Names of the flags the user typed, in typed order.
}

// Get returns the named flag and whether the run defines it.
func (f Flags) Get(name string) (flag Flag, ok bool) {
	for _, flag = range f.all {
		if flag.Name == name {
			return flag, true
		}
	}
	return Flag{}, false
}

// All returns every flag in definition order, global flags first.
func (f Flags) All() []Flag {
	return append([]Flag(nil), f.all...)
}

// Passed returns the flags the user typed, in the order first typed.
func (f Flags) Passed() (flags []Flag) {
	for _, name := range f.typed {
		flag, ok := f.Get(name)
		if ok {
			flags = append(flags, flag)
		}
	}
	return flags
}
