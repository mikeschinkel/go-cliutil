package cliutil_test

import (
	"fmt"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-testutil"
)

// TestParseFKey tests the ParseFKey function with various escape sequences
func TestParseFKey(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int
	}{
		// ESC [ N N ~ format (most common)
		{"F1 - ESC[11~", []byte{0x1B, '[', '1', '1', '~'}, 1},
		{"F2 - ESC[12~", []byte{0x1B, '[', '1', '2', '~'}, 2},
		{"F3 - ESC[13~", []byte{0x1B, '[', '1', '3', '~'}, 3},
		{"F4 - ESC[14~", []byte{0x1B, '[', '1', '4', '~'}, 4},
		{"F5 - ESC[15~", []byte{0x1B, '[', '1', '5', '~'}, 5},
		{"F6 - ESC[17~", []byte{0x1B, '[', '1', '7', '~'}, 6}, // Note: 17, not 16!
		{"F7 - ESC[18~", []byte{0x1B, '[', '1', '8', '~'}, 7},
		{"F8 - ESC[19~", []byte{0x1B, '[', '1', '9', '~'}, 8},
		{"F9 - ESC[20~", []byte{0x1B, '[', '2', '0', '~'}, 9},
		{"F10 - ESC[21~", []byte{0x1B, '[', '2', '1', '~'}, 10},
		{"F11 - ESC[23~", []byte{0x1B, '[', '2', '3', '~'}, 11}, // Note: 23, not 22!
		{"F12 - ESC[24~", []byte{0x1B, '[', '2', '4', '~'}, 12},

		// ESC [ N ~ format (rare, but valid for F1-F5)
		{"F1 - ESC[1~ (rare)", []byte{0x1B, '[', '1', '~'}, 1},
		{"F2 - ESC[2~ (rare)", []byte{0x1B, '[', '2', '~'}, 2},
		{"F3 - ESC[3~ (rare)", []byte{0x1B, '[', '3', '~'}, 3},
		{"F4 - ESC[4~ (rare)", []byte{0x1B, '[', '4', '~'}, 4},
		{"F5 - ESC[5~ (rare)", []byte{0x1B, '[', '5', '~'}, 5},

		// ESC O X format (alternate for F1-F4)
		{"F1 - ESC O P", []byte{0x1B, 'O', 'P'}, 1},
		{"F2 - ESC O Q", []byte{0x1B, 'O', 'Q'}, 2},
		{"F3 - ESC O R", []byte{0x1B, 'O', 'R'}, 3},
		{"F4 - ESC O S", []byte{0x1B, 'O', 'S'}, 4},

		// Invalid sequences - should return 0
		{"Too short - 2 bytes", []byte{0x1B, '['}, 0},
		{"Too short - 1 byte", []byte{0x1B}, 0},
		{"Empty buffer", []byte{}, 0},
		{"Not ESC sequence", []byte{'F', '1'}, 0},
		{"ESC [ without tilde", []byte{0x1B, '[', '1', '1'}, 0},
		{"ESC [ wrong terminator", []byte{0x1B, '[', '1', '1', 'X'}, 0},
		{"ESC [ invalid number", []byte{0x1B, '[', '9', '9', '~'}, 0},
		{"ESC [ 16~ (gap before F6)", []byte{0x1B, '[', '1', '6', '~'}, 0},
		{"ESC [ 22~ (gap before F11)", []byte{0x1B, '[', '2', '2', '~'}, 0},
		{"ESC [ 25~ (beyond F12)", []byte{0x1B, '[', '2', '5', '~'}, 0},
		{"ESC [ 10~ (not defined)", []byte{0x1B, '[', '1', '0', '~'}, 0},
		{"ESC O invalid letter", []byte{0x1B, 'O', 'Z'}, 0},
		{"ESC O too short", []byte{0x1B, 'O'}, 0},
		{"ESC with wrong second byte", []byte{0x1B, 'X', 'P'}, 0},

		// Edge cases with extra data (should still parse correctly)
		{"F5 ESC O P with trailing", []byte{0x1B, 'O', 'P', 'x'}, 1},

		// Single digit sequences (valid for F1-F5 only)
		{"ESC[6~ (not a function key)", []byte{0x1B, '[', '6', '~'}, 0},
		{"ESC[7~ (not a function key)", []byte{0x1B, '[', '7', '~'}, 0},
		{"ESC[8~ (not a function key)", []byte{0x1B, '[', '8', '~'}, 0},
		{"ESC[9~ (not a function key)", []byte{0x1B, '[', '9', '~'}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cliutil.ParseFKey(tt.input)
			if result != tt.expected {
				t.Errorf("ParseFKey(%v) = %d, want %d",
					tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseFKey_AllValidKeys ensures all F1-F12 are supported
func TestParseFKey_AllValidKeys(t *testing.T) {
	// Test that we can parse all F1-F12 in their most common format
	expectedKeys := map[string]struct {
		sequence []byte
		fkey     int
	}{
		"F1":  {[]byte{0x1B, '[', '1', '1', '~'}, 1},
		"F2":  {[]byte{0x1B, '[', '1', '2', '~'}, 2},
		"F3":  {[]byte{0x1B, '[', '1', '3', '~'}, 3},
		"F4":  {[]byte{0x1B, '[', '1', '4', '~'}, 4},
		"F5":  {[]byte{0x1B, '[', '1', '5', '~'}, 5},
		"F6":  {[]byte{0x1B, '[', '1', '7', '~'}, 6},
		"F7":  {[]byte{0x1B, '[', '1', '8', '~'}, 7},
		"F8":  {[]byte{0x1B, '[', '1', '9', '~'}, 8},
		"F9":  {[]byte{0x1B, '[', '2', '0', '~'}, 9},
		"F10": {[]byte{0x1B, '[', '2', '1', '~'}, 10},
		"F11": {[]byte{0x1B, '[', '2', '3', '~'}, 11},
		"F12": {[]byte{0x1B, '[', '2', '4', '~'}, 12},
	}

	for name, test := range expectedKeys {
		t.Run(name, func(t *testing.T) {
			result := cliutil.ParseFKey(test.sequence)
			if result != test.fkey {
				t.Errorf("%s: ParseFKey(%v) = %d, want %d",
					name, test.sequence, result, test.fkey)
			}
		})
	}
}

// TestParseFKey_GapValidation tests that the gaps in escape sequences are handled
func TestParseFKey_GapValidation(t *testing.T) {
	// These sequences have gaps - 16 and 22 are not valid F-key codes
	gapTests := []struct {
		name     string
		sequence []byte
		note     string
	}{
		{
			"ESC[16~ is not F6",
			[]byte{0x1B, '[', '1', '6', '~'},
			"F6 uses 17, not 16",
		},
		{
			"ESC[22~ is not F11",
			[]byte{0x1B, '[', '2', '2', '~'},
			"F11 uses 23, not 22",
		},
	}

	for _, tt := range gapTests {
		t.Run(tt.name, func(t *testing.T) {
			result := cliutil.ParseFKey(tt.sequence)
			if result != 0 {
				t.Errorf("%s: ParseFKey(%v) = %d, want 0 (note: %s)",
					tt.name, tt.sequence, result, tt.note)
			}
		})
	}

	// Verify the correct sequences work
	correctTests := []struct {
		name     string
		sequence []byte
		expected int
	}{
		{"F6 uses ESC[17~", []byte{0x1B, '[', '1', '7', '~'}, 6},
		{"F11 uses ESC[23~", []byte{0x1B, '[', '2', '3', '~'}, 11},
	}

	for _, tt := range correctTests {
		t.Run(tt.name, func(t *testing.T) {
			result := cliutil.ParseFKey(tt.sequence)
			if result != tt.expected {
				t.Errorf("%s: ParseFKey(%v) = %d, want %d",
					tt.name, tt.sequence, result, tt.expected)
			}
		})
	}
}

// TestParseFKey_BothFormats verifies F1-F4 work in both ESC[NN~ and ESC O X formats
func TestParseFKey_BothFormats(t *testing.T) {
	dualFormatKeys := []struct {
		fkey       int
		escBracket []byte
		escO       []byte
	}{
		{1, []byte{0x1B, '[', '1', '1', '~'}, []byte{0x1B, 'O', 'P'}},
		{2, []byte{0x1B, '[', '1', '2', '~'}, []byte{0x1B, 'O', 'Q'}},
		{3, []byte{0x1B, '[', '1', '3', '~'}, []byte{0x1B, 'O', 'R'}},
		{4, []byte{0x1B, '[', '1', '4', '~'}, []byte{0x1B, 'O', 'S'}},
	}

	for _, tt := range dualFormatKeys {
		t.Run(fmt.Sprintf("F%d ESC[ format", tt.fkey), func(t *testing.T) {
			result := cliutil.ParseFKey(tt.escBracket)
			if result != tt.fkey {
				t.Errorf("F%d ESC[ format: ParseFKey(%v) = %d, want %d",
					tt.fkey, tt.escBracket, result, tt.fkey)
			}
		})

		t.Run(fmt.Sprintf("F%d ESC O format", tt.fkey), func(t *testing.T) {
			result := cliutil.ParseFKey(tt.escO)
			if result != tt.fkey {
				t.Errorf("F%d ESC O format: ParseFKey(%v) = %d, want %d",
					tt.fkey, tt.escO, result, tt.fkey)
			}
		})
	}
}

// TestKeyPress_String tests the String() method on KeyPress
func TestKeyPress_String(t *testing.T) {
	tests := []struct {
		name     string
		kp       cliutil.KeyPress
		expected string
	}{
		// F-keys
		{"F1", cliutil.KeyPress{FKey: 1}, "f1"},
		{"F2", cliutil.KeyPress{FKey: 2}, "f2"},
		{"F12", cliutil.KeyPress{FKey: 12}, "f12"},

		// Digits
		{"Digit 0", cliutil.KeyPress{Rune: '0'}, "0"},
		{"Digit 9", cliutil.KeyPress{Rune: '9'}, "9"},

		// Letters
		{"Letter a", cliutil.KeyPress{Rune: 'a'}, "a"},
		{"Letter z", cliutil.KeyPress{Rune: 'z'}, "z"},

		// Ctrl combinations
		{"Ctrl+C", cliutil.KeyPress{Rune: 'c', Ctrl: true}, "ctrl+c"},
		{"Ctrl+D", cliutil.KeyPress{Rune: 'd', Ctrl: true}, "ctrl+d"},

		// Edge cases
		{"Empty KeyPress", cliutil.KeyPress{}, ""},
		{"F-key takes priority over rune", cliutil.KeyPress{FKey: 5, Rune: 'a'}, "f5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.kp.String()
			if result != tt.expected {
				t.Errorf("KeyPress.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestKeyPress_MatchesAny tests the MatchesAny() method on KeyPress
func TestKeyPress_MatchesAny(t *testing.T) {
	tests := []struct {
		name          string
		kp            cliutil.KeyPress
		acceptable    []string
		expectedMatch string
		expectedOk    bool
	}{
		// Matching cases
		{"F1 matches in list", cliutil.KeyPress{FKey: 1}, []string{"f1", "f2"}, "f1", true},
		{"Digit 0 matches", cliutil.KeyPress{Rune: '0'}, []string{"0", "1"}, "0", true},
		{"Ctrl+C matches", cliutil.KeyPress{Rune: 'c', Ctrl: true}, []string{"ctrl+c"}, "ctrl+c", true},

		// Non-matching cases
		{"F3 not in list", cliutil.KeyPress{FKey: 3}, []string{"f1", "f2"}, "", false},
		{"Digit 5 not in list", cliutil.KeyPress{Rune: '5'}, []string{"0", "1"}, "", false},
		{"Empty acceptable list", cliutil.KeyPress{FKey: 1}, []string{}, "", false},
		{"Empty KeyPress", cliutil.KeyPress{}, []string{"f1", "0"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, ok := tt.kp.MatchesAny(tt.acceptable)
			if matched != tt.expectedMatch || ok != tt.expectedOk {
				t.Errorf("KeyPress.MatchesAny() = (%q, %v), want (%q, %v)",
					matched, ok, tt.expectedMatch, tt.expectedOk)
			}
		})
	}
}

// TestEchoKey tests the EchoKey helper function
func TestEchoKey(t *testing.T) {
	tests := []struct {
		name     string
		keyStr   string
		expected string
	}{
		// F-keys (lowercase input → uppercase output)
		{"F1", "f1", "F1\n\n"},
		{"F2", "f2", "F2\n\n"},
		{"F12", "f12", "F12\n\n"},

		// Digits
		{"Digit 0", "0", "0\n\n"},
		{"Digit 9", "9", "9\n\n"},

		// Ctrl combinations (ctrl+c → ^C)
		{"Ctrl+C", "ctrl+c", "^C\n\n"},
		{"Ctrl+D", "ctrl+d", "^D\n\n"},

		// Plain characters
		{"Letter a", "a", "a\n\n"},
		{"Letter z", "z", "z\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := testutil.NewBufferedWriter()
			cliutil.EchoKey(tt.keyStr, writer)
			output := writer.GetPrintOutput()
			if output != tt.expected {
				t.Errorf("EchoKey(%q) output = %q, want %q", tt.keyStr, output, tt.expected)
			}
		})
	}
}
