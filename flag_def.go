package cliutil

import (
	"regexp"

	"github.com/mikeschinkel/go-dt"
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
	Example        string // OPTIONAL: sample value for example generation (e.g., "www")
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
		err = NewErr(dt.ErrFlagIsRequired)
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
			err = NewErr(dt.ErrInvalidFlagName, "flag_value", stringValue)
			goto end
		}
	}

	// Custom validation function
	if fd.ValidationFunc != nil {
		err = fd.ValidationFunc(value)
		if err != nil {
			goto end
		}
	}

end:
	if err != nil {
		err = WithErr(err, dt.ErrFlagValidationFailed, "flag_name", fd.Name)
	}
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
