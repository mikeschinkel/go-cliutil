package cliutil

// Operand is one positional word given to a command.
type Operand struct {
	Value    string  // The word as typed.
	Position int     // Index of the word within HandleArgs.Parameters.
	Def      *ArgDef // The definition it fills; nil under Passthrough.
}

// Operands holds the positional words given to a command, in typed order.
type Operands []Operand

// Strings returns each operand's value.
func (o Operands) Strings() []string {
	values := make([]string, len(o))
	for i, operand := range o {
		values[i] = operand.Value
	}
	return values
}
