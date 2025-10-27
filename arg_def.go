package cliutil

// ArgDef defines a positional command argument
type ArgDef struct {
	Name     string
	Usage    string
	Required bool
	Default  any
	String   *string // Where to assign the argument value
	Example  string  // OPTIONAL: sample value for example generation (e.g., "www")
}
