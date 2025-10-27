package cliutil

import (
	"fmt"
	"regexp"
)

// ValidationFunc validates a flag value and returns an error if invalid
type ValidationFunc func(value any) error

// FlagDef defines a command flag declaratively
type FlagDef struct {
	Name           string
	Shortcut       byte
	Default        any
	Usage          string
	Required       bool
	Regex          *regexp.Regexp
	ValidationFunc ValidationFunc
	String         *string
	Bool           *bool
	Int64          *int64
	Int            *int
}

func (fd *FlagDef) Type() (ft FlagType) {
	switch {
	case fd.String != nil:
		return StringFlag
	case fd.Bool != nil:
		return BoolFlag
	case fd.Int != nil:
		return IntFlag
	case fd.Int64 != nil:
		return Int64Flag
	}
	return UnknownFlagType
}

// ValidateValue validates the flag value using the defined validation rules
func (fd *FlagDef) ValidateValue(value any) error {
	var err error
	var stringValue string
	var ok bool

	// Check required
	if fd.Required && (value == nil || value == "") {
		err = fmt.Errorf("flag --%s is required", fd.Name)
		goto end
	}

	// Skip further validation if value is empty and not required
	if value == nil || value == "" {
		goto end
	}

	// Regex validation (only for string values)
	if fd.Regex != nil {
		stringValue, ok = value.(string)
		if ok && !fd.Regex.MatchString(stringValue) {
			err = fmt.Errorf("flag --%s value '%s' does not match required pattern", fd.Name, stringValue)
			goto end
		}
	}

	// Custom validation function
	if fd.ValidationFunc != nil {
		err = fd.ValidationFunc(value)
		if err != nil {
			// Wrap the error with flag context
			err = fmt.Errorf("flag --%s validation failed: %w", fd.Name, err)
			goto end
		}
	}

end:
	return err
}

func (fd *FlagDef) SetValue(value any) {
	switch fd.Type() {
	case StringFlag:
		v := *value.(*string)
		if fd.String != nil {
			*fd.String = v
		}
	case BoolFlag:
		v := *value.(*bool)
		if fd.Bool != nil {
			*fd.Bool = v
		}
	case IntFlag:
		v := *value.(*int)
		if fd.Int != nil {
			*fd.Int = v
		}
	case Int64Flag:
		v := *value.(*int64)
		if fd.Int64 != nil {
			*fd.Int64 = v
		}
	case UnknownFlagType:
		// Just here to have all flag types in the switch
	}
}
