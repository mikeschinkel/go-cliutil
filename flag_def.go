package cliutil

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mikeschinkel/go-dt"
)

// ValidationFunc validates a flag value and returns an error if invalid
type ValidationFunc func(value any) error

// FlagDef defines a flag declaratively. Exactly one target (String, Bool, Int,
// Int64 or Strings) receives the value, and only the Default field matching
// that target may be set.
type FlagDef struct {
	Name           string
	Shortcut       byte
	Usage          string
	Required       bool // Satisfied by the command line or by config.
	Regex          *regexp.Regexp
	ValidationFunc ValidationFunc

	String  *string
	Bool    *bool
	Int64   *int64
	Int     *int
	Strings *[]string // Filled by repeating the flag.

	// Split also splits each value of a Strings flag on commas. A comma written
	// as \, stays literal; quotes cannot do this because the shell removes them.
	Split bool

	DefaultString  string
	DefaultStrings []string
	DefaultBool    bool
	DefaultInt     int
	DefaultInt64   int64

	Example string // OPTIONAL: sample value for example generation (e.g., "www")
}

// Type returns the flag's type, derived from its target.
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
	case fd.Strings != nil:
		return StringsFlag
	}
	return UnknownFlagType
}

// ValidateValue validates the flag value using the defined validation rules
func (fd *FlagDef) ValidateValue(value any) error {
	var err error
	var values []string

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
		switch v := value.(type) {
		case string:
			values = []string{v}
		case []string:
			values = v
		}
		for _, s := range values {
			if !fd.Regex.MatchString(s) {
				err = NewErr(dt.ErrInvalidFlagName, "flag_value", s)
				goto end
			}
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

// takesValue reports whether the flag needs a value; a boolean flag never does.
func (fd *FlagDef) takesValue() bool {
	return fd.Type() != BoolFlag
}

// defaultValue returns the typed default for the flag's target.
func (fd *FlagDef) defaultValue() (value any) {
	switch fd.Type() {
	case StringFlag:
		value = fd.DefaultString
	case BoolFlag:
		value = fd.DefaultBool
	case IntFlag:
		value = fd.DefaultInt
	case Int64Flag:
		value = fd.DefaultInt64
	case StringsFlag:
		value = slices.Clone(fd.DefaultStrings)
	case UnknownFlagType:
		// Registration rejects a flag with no target.
	}
	return value
}

// defaultText returns the default as help displays it, or "" when the default
// is the target type's zero value.
func (fd *FlagDef) defaultText() (text string) {
	switch fd.Type() {
	case StringFlag:
		text = fd.DefaultString
	case BoolFlag:
		if fd.DefaultBool {
			text = "true"
		}
	case IntFlag:
		if fd.DefaultInt != 0 {
			text = strconv.Itoa(fd.DefaultInt)
		}
	case Int64Flag:
		if fd.DefaultInt64 != 0 {
			text = strconv.FormatInt(fd.DefaultInt64, 10)
		}
	case StringsFlag:
		text = joinDefaults(fd.DefaultStrings)
	case UnknownFlagType:
		// Registration rejects a flag with no target.
	}
	return text
}

// assign writes value, which must have the flag's type, into the target.
func (fd *FlagDef) assign(value any) {
	switch fd.Type() {
	case StringFlag:
		*fd.String = value.(string)
	case BoolFlag:
		*fd.Bool = value.(bool)
	case IntFlag:
		*fd.Int = value.(int)
	case Int64Flag:
		*fd.Int64 = value.(int64)
	case StringsFlag:
		*fd.Strings = slices.Clone(value.([]string))
	case UnknownFlagType:
		// Registration rejects a flag with no target.
	}
}

// current reads the value now held by the target.
func (fd *FlagDef) current() (value any) {
	switch fd.Type() {
	case StringFlag:
		value = *fd.String
	case BoolFlag:
		value = *fd.Bool
	case IntFlag:
		value = *fd.Int
	case Int64Flag:
		value = *fd.Int64
	case StringsFlag:
		value = slices.Clone(*fd.Strings)
	case UnknownFlagType:
		// Registration rejects a flag with no target.
	}
	return value
}

// differsFromDefault reports whether the target no longer holds the default.
func (fd *FlagDef) differsFromDefault() bool {
	if fd.Type() == StringsFlag {
		return !slices.Equal(*fd.Strings, fd.DefaultStrings)
	}
	return !reflect.DeepEqual(fd.current(), fd.defaultValue())
}

// sharesTarget reports whether both definitions write to the same variable.
func (fd *FlagDef) sharesTarget(other *FlagDef) bool {
	switch fd.Type() {
	case StringFlag:
		return fd.String == other.String
	case BoolFlag:
		return fd.Bool == other.Bool
	case IntFlag:
		return fd.Int == other.Int
	case Int64Flag:
		return fd.Int64 == other.Int64
	case StringsFlag:
		return fd.Strings == other.Strings
	case UnknownFlagType:
		// A flag with no target shares nothing.
	}
	return false
}

// parseWords converts the raw values typed for the flag into its type. A
// scalar flag typed more than once keeps its last value.
func (fd *FlagDef) parseWords(raws []string) (value any, err error) {
	var last string

	if fd.Type() == StringsFlag {
		values := make([]string, 0, len(raws))
		for _, raw := range raws {
			if fd.Split {
				values = append(values, splitOnCommas(raw)...)
				continue
			}
			values = append(values, raw)
		}
		return values, nil
	}

	last = raws[len(raws)-1]
	switch fd.Type() {
	case StringFlag:
		value = last
	case BoolFlag:
		value, err = strconv.ParseBool(last)
	case IntFlag:
		value, err = strconv.Atoi(last)
	case Int64Flag:
		value, err = strconv.ParseInt(last, 10, 64)
	case StringsFlag, UnknownFlagType:
		// StringsFlag returned above; registration rejects a flag with no target.
	}
	if err != nil {
		err = NewErr(ErrInvalidFlagValue, "flag_name", fd.Name, "flag_value", last,
			fmt.Errorf("invalid value %q for flag --%s", last, fd.Name))
	}
	return value, err
}

// coerce converts a value supplied by config into the flag's type.
func (fd *FlagDef) coerce(value any) (out any, err error) {
	var raw string

	switch v := value.(type) {
	case string:
		if fd.Type() == StringsFlag {
			return fd.parseWords([]string{v})
		}
		raw = v
	case []string:
		if fd.Type() == StringsFlag {
			return slices.Clone(v), nil
		}
		err = fmt.Errorf("config value for flag --%s is a list", fd.Name)
		goto end
	case []any:
		if fd.Type() != StringsFlag {
			err = fmt.Errorf("config value for flag --%s is a list", fd.Name)
			goto end
		}
		values := make([]string, len(v))
		for i, item := range v {
			values[i] = fmt.Sprint(item)
		}
		return values, nil
	case float64:
		// Decoded JSON and YAML numbers arrive as float64.
		raw = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		raw = fmt.Sprint(v)
	}
	out, err = fd.parseWords([]string{raw})

end:
	if err != nil {
		err = NewErr(ErrInvalidFlagValue, "flag_name", fd.Name, err)
	}
	return out, err
}

// validateDef reports definition errors in the flag.
func (fd *FlagDef) validateDef() (err error) {
	var errs []error
	var targets int
	var defaults []FlagType

	if !flagNameRegex.MatchString(fd.Name) {
		errs = append(errs, fmt.Errorf("flag '%s': name may contain only lowercase letters, numbers, and dashes", fd.Name))
	}
	if fd.Shortcut != 0 && !isShortcutByte(fd.Shortcut) {
		errs = append(errs, fmt.Errorf("flag '%s': shortcut must be a single ASCII letter or digit", fd.Name))
	}

	for _, set := range []bool{fd.String != nil, fd.Bool != nil, fd.Int != nil, fd.Int64 != nil, fd.Strings != nil} {
		if set {
			targets++
		}
	}
	if targets != 1 {
		errs = append(errs, NewErr(ErrFlagTypeNotDiscoverable, "flag_name", fd.Name,
			fmt.Errorf("flag '%s': exactly one of String, Bool, Int, Int64 or Strings must be set", fd.Name)))
	}

	if fd.DefaultString != "" {
		defaults = append(defaults, StringFlag)
	}
	if fd.DefaultBool {
		defaults = append(defaults, BoolFlag)
	}
	if fd.DefaultInt != 0 {
		defaults = append(defaults, IntFlag)
	}
	if fd.DefaultInt64 != 0 {
		defaults = append(defaults, Int64Flag)
	}
	if fd.DefaultStrings != nil {
		defaults = append(defaults, StringsFlag)
	}
	for _, ft := range defaults {
		if ft != fd.Type() {
			errs = append(errs, fmt.Errorf("flag '%s': default does not match the flag's target", fd.Name))
			break
		}
	}

	if fd.Split && fd.Strings == nil {
		errs = append(errs, fmt.Errorf("flag '%s': Split requires a Strings target", fd.Name))
	}
	return CombineErrs(errs)
}

// isShortcutByte reports whether b can follow a single dash as a shortcut.
func isShortcutByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// splitOnCommas splits s on commas, keeping each comma written as \, literal.
func splitOnCommas(s string) (parts []string) {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == ',':
			sb.WriteByte(',')
			i++
		case s[i] == ',':
			parts = append(parts, sb.String())
			sb.Reset()
		default:
			sb.WriteByte(s[i])
		}
	}
	return append(parts, sb.String())
}

// joinDefaults formats a list default for help.
func joinDefaults(values []string) string {
	return strings.Join(values, ",")
}
