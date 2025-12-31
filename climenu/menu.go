package climenu

import (
	"fmt"
	"log/slog"

	"github.com/mikeschinkel/go-cliutil"
)

// OptionHandlerArgs provides arguments to menu option handlers
type OptionHandlerArgs struct {
	Writer cliutil.Writer
	Mode   MenuMode
	Index  int // The index of the selected option (0-8)
}

// MenuOption represents a single menu choice
type MenuOption struct {
	Name        string
	Description string
	Handler     func(args *OptionHandlerArgs) error // Function to execute when option is selected
}

// MenuMode represents a group of related options (e.g., "Analysis", "Commit Drafting")
// MenuOptions() should return all options WITH help at index 0
type MenuMode interface {
	// Core menu methods

	MenuOptions() []MenuOption            // Returns options with help at index 0
	Handle(args *OptionHandlerArgs) error // Handle selection
	ShouldExit() bool                     // Returns true if mode wants to exit the menu loop
	RequestExit()                         // Signal that the menu loop should exit
	SetWriter(cliutil.Writer)             // Sets the CLI writer to use
	SetLogger(*slog.Logger)               // Sets the Logger to use

	// Modal menu methods (for multi-mode support)

	ModeID() int                   // Returns 0-9 for mode registry, -1 if not modal
	ModeName() string              // Returns display name ("Main", "Explore", etc.)
	OnEnter(state ModeState) error // Called when switching TO this mode
	OnExit(state ModeState) error  // Called when switching FROM this mode
}

// MenuArgs contains arguments for displaying a menu
type MenuArgs struct {
	Mode   MenuMode
	Writer cliutil.Writer
}

// ShowMenu displays the mode's options inline with [0] help and [9] quit
// Format: [1] option1 [2] option2 — [0] help [9] quit
//
// Behavior:
//   - [0] help - Shows vertical help with descriptions, then redisplays menu
//   - [9] quit - Returns with no error (user chose to exit)
//   - [1-8] - Calls mode.Handle(index) where index matches what user pressed
//   - Invalid - Shows error message and loops
func ShowMenu(args MenuArgs) (err error) {
	options := args.Mode.MenuOptions()

	for {
		var choice rune

		// Build inline display string (skip help at index 0, show 1-8)
		var optStr string
		for i := 1; i < len(options); i++ {
			if i > 1 {
				optStr += " "
			}
			optStr += fmt.Sprintf("[%d] %s", i, options[i].Name)
		}
		optStr += " — [0] help [9] quit"

		args.Writer.Printf("\n%s\n", optStr)
		args.Writer.Printf("Choice: ")

		choice, err = cliutil.ReadSingleKey()
		if err != nil {
			goto end
		}

		// Echo the choice
		args.Writer.Printf("%c\n\n", choice)

		// Handle selection
		switch choice {
		case '9':
			// Quit
			goto end

		case '0', '1', '2', '3', '4', '5', '6', '7', '8':
			// Convert to index
			index := int(choice - '0')

			// Validate index is within range
			if index >= len(options) {
				args.Writer.Printf("Invalid option.\n")
				continue
			}

			// Create handler args and call mode's handler
			handlerArgs := &OptionHandlerArgs{
				Writer: args.Writer,
				Mode:   args.Mode,
				Index:  index,
			}

			err = args.Mode.Handle(handlerArgs)
			if err != nil {
				goto end
			}

			// Check if mode wants to exit
			if args.Mode.ShouldExit() {
				goto end
			}
			continue

		default:
			args.Writer.Printf("Invalid option.\n")
			continue
		}
	}

end:
	return err
}

// showHelp displays all menu options vertically with descriptions
func showHelp(options []MenuOption, writer cliutil.Writer) {
	writer.Printf("\nMenu Options:\n\n")

	// Show options 1-8
	for i := 1; i < len(options); i++ {
		if options[i].Description != "" {
			writer.Printf("[%d] %s — %s\n", i, options[i].Name, options[i].Description)
		} else {
			writer.Printf("[%d] %s\n", i, options[i].Name)
		}
	}

	// Show help and quit
	writer.Printf("[0] %s — %s\n", options[0].Name, options[0].Description)
	writer.Printf("[9] quit — Exit this menu\n")
	writer.Printf("\n")
}

// BaseMenuMode provides a convenient base implementation that prepends help
var _ MenuMode = (*BaseMenuMode)(nil)

type BaseMenuMode struct {
	menuOptions   []MenuOption
	exitRequested bool
	modeID        int    // -1 for non-modal, 0-9 for modal
	modeName      string // Display name for modal menus
	Writer        cliutil.Writer
	Logger        *slog.Logger
}

func (b *BaseMenuMode) SetLogger(lgr *slog.Logger) {
	b.Logger = lgr
}

// NewBaseMenuMode creates a BaseMenuMode with help prepended at index 0
func NewBaseMenuMode(args BaseMenuModeArgs) *BaseMenuMode {
	mode := &BaseMenuMode{
		modeID:   args.ModeID,
		modeName: args.ModeName,
		Writer:   args.Writer,
	}

	// Default modeID to -1 if not specified (non-modal)
	if mode.modeID == 0 && args.ModeName == "" {
		mode.modeID = -1
	}

	// Create help handler as a closure that accesses mode's options
	helpHandler := func(handlerArgs *OptionHandlerArgs) error {
		showHelp(mode.MenuOptions(), handlerArgs.Writer)
		return nil
	}

	// Prepend help option at index 0
	mode.menuOptions = append(
		[]MenuOption{{
			Name:        "help",
			Description: "Show this help message",
			Handler:     helpHandler,
		}},
		args.MenuOptions...,
	)

	return mode
}

func (b *BaseMenuMode) SetWriter(w cliutil.Writer) {
	b.Writer = w
}

func (b *BaseMenuMode) MenuOptions() []MenuOption {
	return b.menuOptions
}

func (b *BaseMenuMode) Handle(args *OptionHandlerArgs) (err error) {
	var options []MenuOption
	var option MenuOption

	options = b.MenuOptions()

	// Validate index
	if args.Index < 0 || args.Index >= len(options) {
		goto end
	}

	// Get the option
	option = options[args.Index]

	// Handler is required
	if option.Handler == nil {
		panic(fmt.Sprintf("menu option %q at index %d has no handler", option.Name, args.Index))
	}

	// Invoke the handler
	err = option.Handler(args)

end:
	return err
}

func (b *BaseMenuMode) ShouldExit() bool {
	return b.exitRequested
}

// RequestExit signals that the menu loop should exit after this operation
func (b *BaseMenuMode) RequestExit() {
	b.exitRequested = true
}

// ModeID returns the mode ID (0-9 for modal, -1 for non-modal)
func (b *BaseMenuMode) ModeID() int {
	return b.modeID
}

// ModeName returns the display name of this mode
func (b *BaseMenuMode) ModeName() string {
	return b.modeName
}

// OnEnter is called when switching to this mode
// Default implementation does nothing
func (b *BaseMenuMode) OnEnter(_ ModeState) error {
	return nil
}

// OnExit is called when switching from this mode
// Default implementation does nothing
func (b *BaseMenuMode) OnExit(_ ModeState) error {
	return nil
}

type BaseMenuModeArgs struct {
	MenuOptions []MenuOption // User's options (without help)
	ModeID      int          // Optional: 0-9 for modal menus, -1 for non-modal (default)
	ModeName    string       // Optional: Display name for modal menus
	Writer      cliutil.Writer
}
