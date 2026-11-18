package cliutil

// ArgDef defines a command's positional argument. A command declares at most
// one ArgDef: a single value through String, or every remaining value through
// Strings.
type ArgDef struct {
	Name     string
	Usage    string
	Required bool // For a Strings target, Required means at least one value.

	String  *string   // Receives the single value.
	Strings *[]string // Receives every value; makes the argument variadic.

	MinCount int // Fewest values a Strings target accepts; 0 means no minimum.
	MaxCount int // Most values a Strings target accepts; 0 means no maximum.

	DefaultString  string   // Value for a String target when no word is given.
	DefaultStrings []string // Values for a Strings target when no word is given.

	Example string // OPTIONAL: sample value for example generation (e.g., "www")
}

// isVariadic reports whether the argument takes every remaining operand.
func (ad *ArgDef) isVariadic() bool {
	return ad.Strings != nil
}

// defaultText returns the default as help displays it, or "" when none is set.
func (ad *ArgDef) defaultText() string {
	if ad.isVariadic() {
		return joinDefaults(ad.DefaultStrings)
	}
	return ad.DefaultString
}

// minOperands returns the fewest operands the argument accepts.
func (ad *ArgDef) minOperands() int {
	if ad.Required && ad.MinCount == 0 {
		return 1
	}
	return ad.MinCount
}
