package climenu

import (
	"fmt"
	"strings"

	"github.com/mikeschinkel/go-cliutil"
)

// MultiModeMenuArgs contains arguments for displaying a modal menu
type MultiModeMenuArgs struct {
	Manager *ModeManager
	Writer  cliutil.Writer
}

// ShowMultiModeMenu displays a modal menu with mode switching support
// Users can switch between modes using function keys
//
// Behavior:
//   - [F1] - Help (shows vertical help with descriptions)
//   - [F2] - Switch to mode 0 (Main)
//   - [F3-F12] - Switch to modes 1-10
//   - [0] help - Shows vertical help with descriptions, then redisplays menu
//   - [9] quit - Returns with no error (user chose to exit)
//   - [1-8] - Calls mode.Handle(index) where index matches what user pressed
//   - Invalid keys - Silently ignored (no echo, no message)
func ShowMultiModeMenu(args MultiModeMenuArgs) (err error) {
	var mode MenuMode
	var key string
	var acceptableKeys []string

	// Call OnEnter on initial mode
	mode = args.Manager.CurrentMode()
	if mode != nil {
		err = mode.OnEnter(args.Manager.State())
		if err != nil {
			goto end
		}
	}

	for {
		// Get current mode
		mode = args.Manager.CurrentMode()
		if mode == nil {
			err = fmt.Errorf("no mode is active")
			goto end
		}

		// Display mode toggle bar and menu options
		displayModalMenu(args.Manager, args.Writer)

		// Build list of acceptable keys for current mode
		// Must be inside loop because different modes have different options
		acceptableKeys = buildAcceptableKeys(args.Manager)

		// Read keypress - loops internally until valid key, handles echo
		key, err = cliutil.ReadKeyPress(acceptableKeys, args.Writer)
		if err != nil {
			goto end
		}

		// Handle F-keys (mode switching)
		if strings.HasPrefix(key, "f") {
			// F1 = help, convert to "0" for menu handling
			if key == "f1" {
				key = "0"
			} else {
				// F2-F12 = mode switching
				err = handleFKey(key, args.Manager)
				if err != nil {
					goto end
				}
				continue // Redisplay menu after mode switch
			}
		}

		// Handle digit keys (menu options and quit)
		if key == "9" {
			// Quit
			goto end
		}

		// Handle menu option
		err = handleMenuKey(key, mode, args.Manager, args.Writer)
		if err != nil {
			goto end
		}

		// Check if manager or mode wants to exit
		if args.Manager.ShouldExit() || mode.ShouldExit() {
			goto end
		}
	}

end:
	return err
}

// displayModalMenu shows the mode toggle bar and current mode's options
func displayModalMenu(mgr *ModeManager, writer cliutil.Writer) {
	mode := mgr.CurrentMode()
	if mode == nil {
		return
	}

	currentModeID := mgr.CurrentModeID()

	// Build mode bar with current mode name + available modes
	// Format: "Main Menu: [F3] Explore [F4] Manage [F5] Compose — [F1] Help"
	var modeBar string
	modeBar += fmt.Sprintf("%s Menu: ", mode.ModeName())

	first := true
	// Show mode 0 first (F2) - unless it's the current mode
	if m := mgr.modes[0]; m != nil && currentModeID != 0 {
		modeBar += fmt.Sprintf("[f2] %s", m.ModeName())
		first = false
	}

	// Show modes 1-10 (F3-F12) - excluding current mode
	for i := 1; i <= 10; i++ {
		if m := mgr.modes[i]; m != nil && currentModeID != i {
			fkey := i + 2 // F3 for mode 1, F4 for mode 2, etc.
			if !first {
				modeBar += " "
			}
			modeBar += fmt.Sprintf("[f%d] %s", fkey, m.ModeName())
			first = false
		}
	}

	// Add F1 help at the end
	modeBar += " — [f1] Help"

	// Display mode bar
	writer.Printf("%s\n", modeBar)

	// Build current mode's options inline (skip help at index 0, show 1-8)
	var optStr string
	optStr += "Actions: "

	options := mode.MenuOptions()
	for i := 1; i < len(options); i++ {
		if i > 1 {
			optStr += " "
		}
		optStr += fmt.Sprintf("[%d] %s", i, options[i].Name)
	}
	optStr += " — [0] help [9] quit"

	writer.Printf("%s\n", optStr)
	writer.Printf("Choice: ")
}

// buildAcceptableKeys builds the list of acceptable keys for the current menu
// Only accepts keys that correspond to actual menu options
func buildAcceptableKeys(mgr *ModeManager) []string {
	var mode MenuMode
	var options []MenuOption
	var keys []string

	// Get current mode and its options
	mode = mgr.CurrentMode()
	if mode == nil {
		// No mode active - only accept F1 for help and 9 for quit
		return []string{"f1", "9"}
	}

	options = mode.MenuOptions()

	// Always accept F1 for help
	keys = []string{"f1"}

	// Accept digit keys only for valid option indices
	// Index 0 is always help, indices 1-N are actual menu options
	for i := 0; i < len(options) && i <= 8; i++ {
		keys = append(keys, fmt.Sprintf("%d", i))
	}

	// Always accept 9 for quit (unless menu has 10+ options, which is unlikely)
	if len(options) < 10 {
		keys = append(keys, "9")
	}

	// Add F-keys only for registered modes (F2=mode0, F3=mode1, etc.)
	for i := 0; i <= 10; i++ {
		if mgr.modes[i] != nil {
			keys = append(keys, fmt.Sprintf("f%d", i+2))
		}
	}

	return keys
}

// handleMenuKey handles digit keys (0-9) for menu options
// Returns error only for fatal errors; silently ignores invalid selections
func handleMenuKey(key string, mode MenuMode, mgr *ModeManager, writer cliutil.Writer) (err error) {
	var index int
	var options []MenuOption
	var handlerArgs *OptionHandlerArgs

	// Convert key to index
	if len(key) != 1 || key[0] < '0' || key[0] > '9' {
		// Not a digit - should not happen if called correctly
		goto end
	}

	index = int(key[0] - '0')

	// Handle quit (9)
	if index == 9 {
		goto end // Return nil to exit menu
	}

	// Validate index is within range
	options = mode.MenuOptions()
	if index >= len(options) {
		// Out of range - silently ignore (shouldn't happen with new API)
		goto end
	}

	// Create handler args and call mode's handler
	handlerArgs = &OptionHandlerArgs{
		Writer: writer,
		Mode:   mode,
		Index:  index,
	}

	err = mode.Handle(handlerArgs)

end:
	return err
}

// handleFKey handles F-key presses for mode switching
// F1 is treated as help (returns "0" to trigger help option)
// F2-F12 switch to modes 0-10
// Returns error only for fatal errors; silently ignores invalid mode switches
func handleFKey(key string, mgr *ModeManager) (err error) {
	var modeID int

	// F1 is handled by returning "0" which triggers help
	// This is handled in ShowMultiModeMenu by treating F1 as digit 0

	// Parse F-key number: f2 → 2, f12 → 12
	if len(key) == 2 && key[0] == 'f' {
		// f1-f9
		modeID = int(key[1]-'0') - 2 // F2→mode0, F3→mode1, etc.
	} else if len(key) == 3 && key[0] == 'f' {
		// f10, f11, f12
		modeID = ((int(key[1]-'0') * 10) + int(key[2]-'0')) - 2
	} else {
		// Invalid format - shouldn't happen
		goto end
	}

	// Attempt mode switch - silently ignore errors
	_ = mgr.SwitchMode(modeID)

end:
	return nil // Always return nil - mode switch errors are non-fatal
}
