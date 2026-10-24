package cliutil

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unicode/utf8"

	"github.com/mikeschinkel/go-dt"
	"golang.org/x/term"
)

// IsTerminalError checks if an error is related to terminal/input operations
// These errors should abort the entire operation rather than continue
func IsTerminalError(err error) (isTermErr bool) {
	var pathErr *os.PathError
	var errno syscall.Errno
	var ok bool

	if err == nil {
		goto end
	}

	switch {
	case errors.As(err, &pathErr):
		// Check for syscall errors related to terminal operations
		errno, ok = pathErr.Err.(syscall.Errno)
		if ok && errno == syscall.ENOTTY { // ENOTTY = "inappropriate ioctl for device"
			isTermErr = true
			goto end
		}
	case errors.As(err, &errno):
		// Check for direct syscall errors
		if errno == syscall.ENOTTY { // ENOTTY = "inappropriate ioctl for device"
			isTermErr = true
			goto end
		}
	}

end:
	return isTermErr
}

// IsInteractive returns true if stdin is a terminal
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// EnterRawMode puts terminal into raw mode and returns a cleanup function
func EnterRawMode(fd uintptr) (restore func(), err error) {
	var oldState *term.State

	oldState, err = term.MakeRaw(int(fd))
	if err != nil {
		goto end
	}

	restore = func() {
		dt.LogOnErrFunc(func() error {
			return term.Restore(int(fd), oldState)
		})
	}

end:
	return restore, err
}

// KeyPress represents a key press with modifier information
type KeyPress struct {
	Rune rune // The character pressed
	Ctrl bool // Ctrl key was held
	Alt  bool // Alt key was held (future)
	FKey int  // 1-12 for F1-F12, 0 if not an F-key
}

// String converts a KeyPress to a human-readable string
// Returns format: "f1"-"f12" for F-keys, "0"-"9" for digits,
// "ctrl+c" for Ctrl combinations, "a"-"z" for letters
func (kp KeyPress) String() (keyStr string) {
	// F-keys take priority
	if kp.FKey > 0 && kp.FKey <= 12 {
		keyStr = fmt.Sprintf("f%d", kp.FKey)
		goto end
	}

	// Ctrl+key combinations
	if kp.Ctrl && kp.Rune != 0 {
		keyStr = fmt.Sprintf("ctrl+%c", kp.Rune)
		goto end
	}

	// Plain character (digit, letter, etc.)
	if kp.Rune != 0 {
		keyStr = string(kp.Rune)
		goto end
	}

	// Unknown/empty
	keyStr = ""

end:
	return keyStr
}

// MatchesAny checks if this KeyPress matches any acceptable key
// Returns the matched key string and true if found, empty string and false otherwise
func (kp KeyPress) MatchesAny(acceptable []string) (matched string, ok bool) {
	keyStr := kp.String()
	if keyStr == "" {
		goto end
	}

	// Check if keyStr matches any acceptable key
	for _, acceptableKey := range acceptable {
		if keyStr == acceptableKey {
			matched = keyStr
			ok = true
			goto end
		}
	}

end:
	return matched, ok
}

// ReadSingleKey reads one keypress without requiring Enter
func ReadSingleKey() (key rune, err error) {
	var restore func()
	var buf []byte
	var n int

	if !IsInteractive() {
		err = fmt.Errorf("stdin is not a terminal")
		goto end
	}

	// Enter raw mode
	restore, err = EnterRawMode(os.Stdin.Fd())
	if err != nil {
		goto end
	}
	defer restore()

	// Read single keypress
	buf = make([]byte, 3) // UTF-8 may be multi-byte
	n, err = os.Stdin.Read(buf)
	if err != nil {
		goto end
	}

	key, _ = utf8.DecodeRune(buf[:n])

end:
	return key, err
}

// ReadKeyPress reads keyboard input and loops until an acceptable key is pressed
// It handles echo internally - only echoes valid keys, silent for invalid keys
//
// Parameters:
//
//	acceptableKeys: Slice of acceptable key strings (e.g., []string{"f1", "f2", "0", "9", "ctrl+c"})
//	writer: Writer for echoing valid keys
//
// Returns:
//
//	key: The matched key string from acceptableKeys (e.g., "f2", "0")
//	err: Terminal errors only (not validation errors)
//
// Behavior:
//   - Loops internally until a valid key is pressed
//   - Invalid keys: completely silent (no echo, no output)
//   - Valid keys: echoed in display format ("F2\n\n" for F-keys, "0\n\n" for digits, "^C\n\n" for Ctrl)
//
// Key Format Specification:
//   - F-keys: "f1", "f2", ..., "f12" (lowercase in acceptableKeys)
//   - Digits: "0", "1", ..., "9"
//   - Ctrl: "ctrl+c", "ctrl+d", etc.
//   - Letters: "a", "b", ..., "z"
func ReadKeyPress(acceptableKeys []string, writer Writer) (key string, err error) {
	var kp KeyPress
	var matched string
	var ok bool

	// Validate parameters
	if len(acceptableKeys) == 0 {
		err = fmt.Errorf("acceptableKeys cannot be empty")
		goto end
	}
	if writer == nil {
		err = fmt.Errorf("writer cannot be nil")
		goto end
	}

	// Loop until valid key (NO goto label needed)
	for {
		kp, err = readKeyPress()
		if err != nil {
			goto end // Terminal error - exit immediately
		}

		matched, ok = kp.MatchesAny(acceptableKeys)
		if !ok {
			continue // Invalid - silent loop, NO echo
		}

		// Valid key - echo and return
		EchoKey(matched, writer)
		key = matched
		goto end
	}

end:
	return key, err
}

// readKeyPress reads one keypress and detects Ctrl modifier and F-keys
// Ctrl+letter combinations are sent as ASCII control codes (0x01-0x1A)
// F-keys are sent as escape sequences: ESC [ N ~ or ESC O X
// This is an internal helper - use the public ReadKeyPress(acceptableKeys, writer) instead
func readKeyPress() (kp KeyPress, err error) {
	kp, err = readKeyPressRaw()
	return kp, err
}

// readKeyPressRaw reads one keypress and detects Ctrl modifier and F-keys
// Ctrl+letter combinations are sent as ASCII control codes (0x01-0x1A)
// F-keys are sent as escape sequences: ESC [ N ~ or ESC O X
// This is an internal helper - use the public ReadKeyPress(acceptableKeys, writer) instead
func readKeyPressRaw() (kp KeyPress, err error) {
	var restore func()
	var buf []byte
	var n int
	var b byte

	if !IsInteractive() {
		err = fmt.Errorf("stdin is not a terminal")
		goto end
	}

	// Enter raw mode
	restore, err = EnterRawMode(os.Stdin.Fd())
	if err != nil {
		goto end
	}
	defer restore()

	// Read single keypress (may be multi-byte for escape sequences)
	buf = make([]byte, 8) // Enough for escape sequences
	n, err = os.Stdin.Read(buf)
	if err != nil {
		goto end
	}

	// Check first byte
	b = buf[0]

	// ESC starts an escape sequence (0x1B)
	if b == 0x1B && n > 1 {
		// Parse escape sequences for F-keys (including Ctrl+F-keys as testing hack)
		kp.FKey = ParseFKey(buf[:n])
		if kp.FKey > 0 {
			goto end // Successfully parsed F-key
		}
		// Other escape sequences (future: Alt key, arrow keys, etc.)
		goto end
	}

	// Control characters: 0x01-0x1A (Ctrl+A through Ctrl+Z)
	// Also: 0x00 (Ctrl+Space/Ctrl+2), 0x1C-0x1F (Ctrl+4 through Ctrl+7)
	if b < 0x20 {
		kp.Ctrl = true

		// Map control codes to their corresponding keys
		// Ctrl+A=0x01 ... Ctrl+Z=0x1A
		if b >= 0x01 && b <= 0x1A {
			kp.Rune = rune('a' + (b - 0x01)) // Map to lowercase letter
		} else if b == 0x00 {
			kp.Rune = '0' // Ctrl+0 or Ctrl+Space (varies by terminal)
		} else if //goland:noinspection GoDfaConstantCondition
		b >= 0x1C && b <= 0x1F {
			// Ctrl+4 through Ctrl+7 (terminal-dependent)
			kp.Rune = rune('4' + (b - 0x1C))
		} else {
			// Other control characters
			kp.Rune = rune(b)
		}
		goto end
	}

	// Normal character (no Ctrl)
	kp.Rune, _ = utf8.DecodeRune(buf[:n])
	kp.Ctrl = false

end:
	return kp, err
}

// ParseFKey parses escape sequences for F-keys
// Returns 1-12 for F1-F12, or 0 if not an F-key
// TESTING HACK: Also recognizes Ctrl+F-keys (e.g., ESC[15;5~) and returns same number as plain F-key
func ParseFKey(buf []byte) (fKey int) {
	var numStr string
	var semicolonPos int
	var lastChar byte

	if len(buf) < 3 {
		goto end
	}
	if buf[0] != 0x1B {
		goto end
	}

	if buf[1] == 'O' {
		// ESC O sequences (alternate format for F1-F4)
		// ESC O P = F1, ESC O Q = F2, ESC O R = F3, ESC O S = F4
		switch buf[2] {
		case 'P':
			fKey = 1
		case 'Q':
			fKey = 2
		case 'R':
			fKey = 3
		case 'S':
			fKey = 4
		}
		goto end
	}

	if buf[1] != '[' {
		goto end
	}

	// ESC [ sequences
	lastChar = buf[len(buf)-1]

	// Check for F1-F4: ESC[P/Q/R/S (plain) or ESC[1;5P/Q/R/S (Ctrl)
	// Plain format: ESC [ P/Q/R/S (3 bytes)
	if len(buf) == 3 {
		switch lastChar {
		case 'P':
			fKey = 1
			goto end
		case 'Q':
			fKey = 2
			goto end
		case 'R':
			fKey = 3
			goto end
		case 'S':
			fKey = 4
			goto end
		}
	}

	// Modified format: ESC [ 1 ; 5 P/Q/R/S (6 bytes, Ctrl+F1-F4)
	if len(buf) == 6 && buf[2] == '1' && buf[3] == ';' && buf[4] == '5' {
		// TESTING HACK: Strip ;5 modifier, treat Ctrl+Fn same as Fn
		switch lastChar {
		case 'P':
			fKey = 1
			goto end
		case 'Q':
			fKey = 2
			goto end
		case 'R':
			fKey = 3
			goto end
		case 'S':
			fKey = 4
			goto end
		}
	}

	if len(buf) < 4 {
		goto end
	}

	// Standard ESC[ sequences must end with ~
	if buf[len(buf)-1] != '~' {
		goto end
	}

	// TESTING HACK: Strip out ;5 modifier (Ctrl) if present
	// ESC[15;5~ becomes ESC[15~ for parsing
	// Find semicolon position
	for i := 2; i < len(buf); i++ {
		if buf[i] == ';' {
			semicolonPos = i
			break
		}
	}

	// Extract the number part (between [ and ; or ~)
	if semicolonPos > 0 {
		numStr = string(buf[2:semicolonPos])
	} else {
		numStr = string(buf[2 : len(buf)-1])
	}

	// Parse single digit: ESC [ N ~
	if len(numStr) == 1 {
		switch numStr[0] {
		case '1', '2', '3', '4', '5':
			fKey = int(numStr[0] - '0') // F1-F5 (rare format)
		}
		goto end
	}

	// Parse double digit: ESC [ N N ~ (or ESC [ N N ; 5 ~)
	switch numStr {
	case "11", "12", "13", "14", "15":
		fKey = int(numStr[1] - '0') // F1-F5
	case "17", "18", "19":
		fKey = int(numStr[1] - '1') // F6-F8
	case "20", "21":
		fKey = int(numStr[1] - '0' + 9) // F9-F10
	case "23", "24":
		fKey = int(numStr[1] - '1' + 9) // F11-F12
	}

end:
	return fKey
}

// EchoKey echoes a key string to the writer in display format
// F-keys: "F2\n\n" (uppercase), Digits: "0\n\n", Ctrl: "^C\n\n"
// This is primarily an internal helper but exported for testing
func EchoKey(keyStr string, writer Writer) {
	var output string

	// F-keys: f1 → F1
	if len(keyStr) >= 2 && keyStr[0] == 'f' {
		output = fmt.Sprintf("F%s\n\n", keyStr[1:])
		goto echo
	}

	// Ctrl combinations: ctrl+c → ^C
	if len(keyStr) > 5 && keyStr[:5] == "ctrl+" {
		output = fmt.Sprintf("^%c\n\n", keyStr[5]-32) // Convert to uppercase
		goto echo
	}

	// Plain character: 0 → 0, a → a, etc.
	output = fmt.Sprintf("%s\n\n", keyStr)

echo:
	writer.Printf(output)
}
