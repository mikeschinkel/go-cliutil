package cliutil

// FlagSet is a named group of flag definitions.
type FlagSet struct {
	Name     string
	FlagDefs []FlagDef
}

// FlagNames returns each flag's long name and, when defined, its shortcut.
func (fs *FlagSet) FlagNames() (names []string) {
	for _, fd := range fs.FlagDefs {
		names = append(names, fd.Name)
		// Include shortcut if defined
		if fd.Shortcut != 0 {
			names = append(names, string(fd.Shortcut))
		}
	}
	return names
}
