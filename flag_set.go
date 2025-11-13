package cliutil

import (
	"errors"
	"flag"
	"fmt"
	"slices"
	"strings"
)

// FlagSet combines a FlagSet with automatic config binding
type FlagSet struct {
	Name     string
	FlagSet  *flag.FlagSet
	FlagDefs []FlagDef
	Values   map[string]any
}

// Parse extracts flags and returns remaining args
func (fs *FlagSet) Parse(args []string) (remainingArgs []string, err error) {
	var fsFlagNames, fsArgs, nonFSArgs []string

	if fs == nil {
		err = fmt.Errorf("FlagSet is nil")
		goto end
	}

	err = fs.Build()
	if err != nil {
		goto end
	}

	// Parse only the flags, collect non-flag arguments
	fsFlagNames = fs.FlagNames()
	fsArgs, nonFSArgs = fs.classifyFlagArgs(args, fsFlagNames)

	if len(fsArgs) == 0 {
		goto end
	}

	// Parse the global flags we found
	err = fs.FlagSet.Parse(fsArgs)
	if err != nil {
		goto end
	}

	err = fs.Validate()
	if err != nil {
		goto end
	}

	err = fs.Assign()

end:
	return nonFSArgs, err
}

func (fs *FlagSet) Build() (err error) {
	var errs []error

	if fs.Name == "" {
		err = fmt.Errorf("name cannot be empty for FlagSet with flags %v", fs.FlagNames())
	}

	fs.FlagSet = flag.NewFlagSet(fs.Name, flag.ContinueOnError)
	fs.Values = make(map[string]any)

	// Add all defined flags to the flag set
	for _, flagDef := range fs.FlagDefs {
		switch flagDef.Type() {
		case StringFlag:
			defaultVal := ""
			if flagDef.Default != nil {
				defaultVal = flagDef.Default.(string)
				*flagDef.String = defaultVal
			}
			fs.Values[flagDef.Name] = fs.FlagSet.String(flagDef.Name, defaultVal, flagDef.Usage)
		case BoolFlag:
			defaultVal := false
			if flagDef.Default != nil {
				defaultVal = flagDef.Default.(bool)
				*flagDef.Bool = defaultVal
			}
			fs.Values[flagDef.Name] = fs.FlagSet.Bool(flagDef.Name, defaultVal, flagDef.Usage)
		case Int64Flag:
			defaultVal := int64(0)
			if flagDef.Default != nil {
				defaultVal = flagDef.Default.(int64)
				*flagDef.Int64 = defaultVal
			}
			fs.Values[flagDef.Name] = fs.FlagSet.Int64(flagDef.Name, defaultVal, flagDef.Usage)
		case IntFlag:
			defaultVal := 0
			if flagDef.Default != nil {
				defaultVal = flagDef.Default.(int)
				*flagDef.Int = defaultVal
			}
			fs.Values[flagDef.Name] = fs.FlagSet.Int(flagDef.Name, defaultVal, flagDef.Usage)
		default:
			errs = append(errs, fmt.Errorf("unknown flag type for %s", flagDef.Name))
		}
	}
	if len(errs) > 0 {
		err = errors.Join(errs...)
	}
	return err
}

func (fs *FlagSet) FlagNames() (names []string) {
	names = make([]string, len(fs.FlagDefs))
	for i, fd := range fs.FlagDefs {
		names[i] = fd.Name
	}
	return names
}

// Validate validates all flag values using their defined validation rules
func (fs *FlagSet) Validate() (err error) {
	var errs []error
	var value any

	for _, flagDef := range fs.FlagDefs {
		switch flagDef.Type() {
		case StringFlag:
			stringPtr := fs.Values[flagDef.Name].(*string)
			value = *stringPtr
		case BoolFlag:
			boolPtr := fs.Values[flagDef.Name].(*bool)
			value = *boolPtr
		case Int64Flag:
			int64Ptr := fs.Values[flagDef.Name].(*int64)
			value = *int64Ptr
		case IntFlag:
			intPtr := fs.Values[flagDef.Name].(*int)
			value = *intPtr
		default:
			errs = append(errs, fmt.Errorf("unknown flag type for %s", flagDef.Name))
			continue
		}

		// Validate the value
		errs = append(errs, flagDef.ValidateValue(value))
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
	}
	return err
}

// classifyFlagArgs separates arguments into flag args and non-flag args
func (fs *FlagSet) classifyFlagArgs(args []string, fsFlagNames []string) (fsArgs []string, nonFSArgs []string) {
	var i int

	for i < len(args) {
		arg := args[i]

		// Non-flag argument
		if !strings.HasPrefix(arg, "-") {
			nonFSArgs = append(nonFSArgs, arg)
			i++
			continue
		}

		// Extract flag name (handle both -flag and --flag)
		flagName := strings.TrimPrefix(arg, "-")
		flagName = strings.TrimPrefix(flagName, "-")

		// Check for flag=value format
		if equalPos := strings.Index(flagName, "="); equalPos != -1 {
			flagName = flagName[:equalPos]
		}

		// Check if this flag belongs to this FlagSet
		if !slices.Contains(fsFlagNames, flagName) {
			// This flag doesn't belong to us, skip it and its value
			nonFSArgs = append(nonFSArgs, arg)
			i++
			continue
		}

		// This is our flag
		fsArgs = append(fsArgs, arg)

		// If flag=value format, we're done with this argument
		if strings.Contains(arg, "=") {
			i++
			continue
		}

		// Check if next argument is the flag value (not another flag)
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			fsArgs = append(fsArgs, args[i+1])
			i += 2 // Skip both flag and value
		} else {
			i++ // Just the flag (boolean flag)
		}
	}

	return fsArgs, nonFSArgs
}

func (fs *FlagSet) Assign() (err error) {
	var errs []error
	for _, flagDef := range fs.FlagDefs {
		switch flagDef.Type() {
		case StringFlag:
			value := fs.Values[flagDef.Name].(*string)
			*flagDef.String = *value
		case BoolFlag:
			value := fs.Values[flagDef.Name].(*bool)
			*flagDef.Bool = *value
		case Int64Flag:
			value := fs.Values[flagDef.Name].(*int64)
			*flagDef.Int64 = *value
		case IntFlag:
			value := fs.Values[flagDef.Name].(*int)
			*flagDef.Int = *value
		default:
			errs = append(errs, fmt.Errorf("unknown flag type for %s", flagDef.Name))
		}
	}
	if len(errs) > 0 {
		err = errors.Join(errs...)
	}
	return err
}
