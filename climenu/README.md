# climenu

Interactive CLI menu framework for Go applications with support for both single-mode and multi-mode modal interfaces.

## Features

- **Single-key navigation** - No need to press Enter, just press a number key
- **Single-mode menus** - Simple menus with 0-8 numbered options plus help and quit
- **Multi-mode menus** - Modal menus that support switching between different modes using function keys (F1-F12)
- **Built-in help system** - Automatic help generation from menu option descriptions
- **Mode lifecycle hooks** - `OnEnter` and `OnExit` callbacks for mode transitions
- **Shared state management** - Pass state between modes using the `ModeState` interface
- **Extensible base implementation** - `BaseMenuMode` provides common functionality

## Installation

```bash
go get github.com/mikeschinkel/go-cliutil/climenu
```

## Quick Start

### Single-Mode Menu

```go
package main

import (
    "fmt"
    "github.com/mikeschinkel/go-cliutil"
    "github.com/mikeschinkel/go-cliutil/climenu"
)

func main() {
    writer := cliutil.NewWriter(nil)

    // Define menu options
    options := []climenu.MenuOption{
        {
            Name:        "create",
            Description: "Create a new item",
            Handler: func(args *climenu.OptionHandlerArgs) error {
                args.Writer.Printf("Creating new item...\n")
                return nil
            },
        },
        {
            Name:        "list",
            Description: "List all items",
            Handler: func(args *climenu.OptionHandlerArgs) error {
                args.Writer.Printf("Listing items...\n")
                return nil
            },
        },
    }

    // Create menu mode
    mode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
        MenuOptions: options,
        Writer:      writer,
    })

    // Show menu
    err := climenu.ShowMenu(climenu.MenuArgs{
        Mode:   mode,
        Writer: writer,
    })
    if err != nil {
        panic(err)
    }
}
```

### Multi-Mode Menu

```go
package main

import (
    "github.com/mikeschinkel/go-cliutil"
    "github.com/mikeschinkel/go-cliutil/climenu"
)

// Define shared state
type AppState struct{}
func (AppState) ModeState() {} // Marker method

func main() {
    writer := cliutil.NewWriter(nil)
    state := &AppState{}

    // Create mode manager
    mgr := climenu.NewModeManager(state, writer, nil)

    // Create "Main" mode (mode 0, accessible via F2)
    mainMode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
        ModeID:   0,
        ModeName: "Main",
        MenuOptions: []climenu.MenuOption{
            {
                Name:        "start",
                Description: "Start the application",
                Handler: func(args *climenu.OptionHandlerArgs) error {
                    args.Writer.Printf("Starting...\n")
                    return nil
                },
            },
        },
        Writer: writer,
    })

    // Create "Settings" mode (mode 1, accessible via F3)
    settingsMode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
        ModeID:   1,
        ModeName: "Settings",
        MenuOptions: []climenu.MenuOption{
            {
                Name:        "config",
                Description: "Configure settings",
                Handler: func(args *climenu.OptionHandlerArgs) error {
                    args.Writer.Printf("Configuring...\n")
                    return nil
                },
            },
        },
        Writer: writer,
    })

    // Register modes
    mgr.RegisterMode(0, mainMode)
    mgr.RegisterMode(1, settingsMode)

    // Show multi-mode menu
    err := climenu.ShowMultiModeMenu(climenu.MultiModeMenuArgs{
        Manager: mgr,
        Writer:  writer,
    })
    if err != nil {
        panic(err)
    }
}
```

## Core Concepts

### MenuMode Interface

The `MenuMode` interface defines the contract for all menu modes:

```go
type MenuMode interface {
    // Core menu methods
    MenuOptions() []MenuOption            // Returns options with help at index 0
    Handle(args *OptionHandlerArgs) error // Handle user selection
    ShouldExit() bool                     // Returns true if mode wants to exit
    RequestExit()                         // Signal that the menu loop should exit
    SetWriter(cliutil.Writer)             // Sets the CLI writer to use
    SetLogger(*slog.Logger)               // Sets the Logger to use

    // Modal menu methods (for multi-mode support)
    ModeID() int                   // Returns 0-9 for mode registry, -1 if not modal
    ModeName() string              // Returns display name ("Main", "Settings", etc.)
    OnEnter(state ModeState) error // Called when switching TO this mode
    OnExit(state ModeState) error  // Called when switching FROM this mode
}
```

### MenuOption

Each menu option consists of:

```go
type MenuOption struct {
    Name        string                                  // Display name (e.g., "create", "list")
    Description string                                  // Help text shown in help menu
    Handler     func(args *OptionHandlerArgs) error    // Function to execute when selected
}
```

### BaseMenuMode

The `BaseMenuMode` type provides a convenient base implementation that:

- Automatically prepends a help option at index 0
- Implements all `MenuMode` interface methods
- Handles option dispatching
- Manages exit state

You can use it directly with `NewBaseMenuMode()` or embed it in your own types.

### Single-Mode Menus

Single-mode menus display options inline with numbered keys:

```
[1] create [2] list [3] delete — [0] help [9] quit
Choice: _
```

Key bindings:
- `0` - Show help (vertical list with descriptions)
- `1-8` - Select menu options
- `9` - Quit the menu

### Multi-Mode Menus

Multi-mode menus support switching between different modes using function keys:

```
Main Menu: [f3] Settings [f4] Admin — [f1] Help
Actions: [1] start [2] stop [3] restart — [0] help [9] quit
Choice: _
```

Key bindings:
- `F1` - Show help for current mode
- `F2-F12` - Switch to modes 0-10 (F2=mode 0, F3=mode 1, etc.)
- `0` - Show help (same as F1)
- `1-8` - Select menu options in current mode
- `9` - Quit the entire menu system

### ModeManager

The `ModeManager` coordinates multiple modes and handles:

- Mode registration (0-9 mode IDs)
- Mode switching with lifecycle hooks
- Shared state management
- Exit coordination

### ModeState

Applications define their own state struct that implements the `ModeState` interface:

```go
type AppState struct {
    // Your application's shared state fields
    currentProject string
    settings       map[string]string
}

// Marker method required by ModeState interface
func (AppState) ModeState() {}
```

This state is passed to `OnEnter` and `OnExit` hooks during mode transitions.

## API Reference

### Functions

#### ShowMenu

```go
func ShowMenu(args MenuArgs) error
```

Displays a single-mode menu with inline options. Loops until user quits or an error occurs.

#### ShowMultiModeMenu

```go
func ShowMultiModeMenu(args MultiModeMenuArgs) error
```

Displays a multi-mode menu with mode switching support. Loops until user quits or an error occurs.

#### NewBaseMenuMode

```go
func NewBaseMenuMode(args BaseMenuModeArgs) *BaseMenuMode
```

Creates a new `BaseMenuMode` with help automatically prepended at index 0.

#### NewModeManager

```go
func NewModeManager(state ModeState, w cliutil.Writer, lgr *slog.Logger) *ModeManager
```

Creates a new `ModeManager` with the given shared state, writer, and logger.

### Types

#### MenuArgs

```go
type MenuArgs struct {
    Mode   MenuMode
    Writer cliutil.Writer
}
```

Arguments for `ShowMenu`.

#### MultiModeMenuArgs

```go
type MultiModeMenuArgs struct {
    Manager *ModeManager
    Writer  cliutil.Writer
}
```

Arguments for `ShowMultiModeMenu`.

#### BaseMenuModeArgs

```go
type BaseMenuModeArgs struct {
    MenuOptions []MenuOption    // User's options (without help)
    ModeID      int             // Optional: 0-9 for modal menus, -1 for non-modal
    ModeName    string          // Optional: Display name for modal menus
    Writer      cliutil.Writer
}
```

Arguments for `NewBaseMenuMode`.

#### OptionHandlerArgs

```go
type OptionHandlerArgs struct {
    Writer cliutil.Writer
    Mode   MenuMode
    Index  int  // The index of the selected option (0-8)
}
```

Arguments passed to menu option handlers.

### ModeManager Methods

#### RegisterMode

```go
func (m *ModeManager) RegisterMode(id int, mode MenuMode) error
```

Registers a mode at the given ID (0-9). Returns error if ID is out of range or already registered.

#### SwitchMode

```go
func (m *ModeManager) SwitchMode(id int) error
```

Switches to the mode with the given ID. Calls `OnExit` on current mode, then `OnEnter` on new mode.

#### CurrentMode

```go
func (m *ModeManager) CurrentMode() MenuMode
```

Returns the currently active mode, or `nil` if no mode is active.

#### CurrentModeID

```go
func (m *ModeManager) CurrentModeID() int
```

Returns the ID of the currently active mode, or `-1` if no mode is active.

#### State

```go
func (m *ModeManager) State() ModeState
```

Returns the shared state object.

#### RequestExit

```go
func (m *ModeManager) RequestExit()
```

Signals that the modal menu loop should exit.

#### ShouldExit

```go
func (m *ModeManager) ShouldExit() bool
```

Returns `true` if exit has been requested.

## Design Patterns

### Exit Handling

There are multiple ways to exit a menu:

1. **User presses 9** - Normal quit, returns `nil`
2. **Handler returns error** - Abnormal exit, returns the error
3. **Mode calls `RequestExit()`** - Programmatic exit after completing an action
4. **ModeManager calls `RequestExit()`** - Exit the entire multi-mode menu system

### Custom MenuMode Implementation

You can implement `MenuMode` directly or embed `BaseMenuMode`:

```go
type CustomMode struct {
    *climenu.BaseMenuMode
    // Your custom fields
}

func NewCustomMode() *CustomMode {
    mode := &CustomMode{}
    mode.BaseMenuMode = climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
        ModeID:   0,
        ModeName: "Custom",
        MenuOptions: []climenu.MenuOption{
            // Your options
        },
    })
    return mode
}

// Override lifecycle hooks if needed
func (m *CustomMode) OnEnter(state climenu.ModeState) error {
    // Custom enter logic
    return m.BaseMenuMode.OnEnter(state)
}

func (m *CustomMode) OnExit(state climenu.ModeState) error {
    // Custom exit logic
    return m.BaseMenuMode.OnExit(state)
}
```

## License

MIT License - Copyright (c) Mike Schinkel
