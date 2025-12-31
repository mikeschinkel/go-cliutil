package main

import (
	"fmt"
	"os"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/climenu"
)

var _ climenu.ModeState = (*DemoState)(nil)

// DemoState implements ModeState for this example
type DemoState struct {
	Counter int
}

func (d DemoState) ModeState() {}

func main() {
	w := cliutil.NewWriter(nil)
	// Create shared state
	state := &DemoState{Counter: 0}

	// Create mode manager
	mgr := climenu.NewModeManager(state, w, nil)

	// Create and register modes
	mainMode := createMainMode()
	mode1 := createMode1()
	mode2 := createMode2()

	err := mgr.RegisterMode(0, mainMode)
	if err != nil {
		w.Errorf("Error registering main mode: %v\n", err)
		os.Exit(1)
	}

	err = mgr.RegisterMode(1, mode1)
	if err != nil {
		w.Errorf("Error registering mode 1: %v\n", err)
		os.Exit(1)
	}

	err = mgr.RegisterMode(2, mode2)
	if err != nil {
		w.Errorf("Error registering mode 2: %v\n", err)
		os.Exit(1)
	}

	// Show modal menu
	fmt.Println("=== Modal Menu Demo ===")
	fmt.Println("Use F2 (Main), F3 (Counter), F4 (Actions) to switch between modes")
	fmt.Println("Press F1 for help, or use number keys for actions")
	fmt.Println()

	err = climenu.ShowMultiModeMenu(climenu.MultiModeMenuArgs{
		Manager: mgr,
		Writer:  cliutil.NewWriter(nil),
	})

	if err != nil {
		w.Errorf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nGoodbye!")
}

// createMainMode creates the Main mode (F2)
func createMainMode() climenu.MenuMode {
	return climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		ModeID:   0,
		ModeName: "Main",
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "Info",
				Description: "Show info about this demo",
				Handler: func(args *climenu.OptionHandlerArgs) error {
					args.Writer.Printf("This is a demo of the modal menu system.\n")
					args.Writer.Printf("Try switching modes with F3 (Counter) or F4 (Actions)!\n")
					return nil
				},
			},
		},
	})
}

// createMode1 creates Mode 1 (F3)
func createMode1() climenu.MenuMode {
	mode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		ModeID:   1,
		ModeName: "Counter",
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "Increment",
				Description: "Increment the counter",
				Handler: func(args *climenu.OptionHandlerArgs) error {
					state := args.Mode.(*customMode).state
					state.Counter++
					args.Writer.Printf("Counter: %d\n", state.Counter)
					return nil
				},
			},
			{
				Name:        "Show",
				Description: "Show current counter value",
				Handler: func(args *climenu.OptionHandlerArgs) error {
					state := args.Mode.(*customMode).state
					args.Writer.Printf("Counter: %d\n", state.Counter)
					return nil
				},
			},
		},
	})

	// Wrap in custom mode to hold state reference
	return &customMode{
		BaseMenuMode: mode,
		state:        nil, // Will be set in OnEnter
	}
}

// createMode2 creates Mode 2 (F4)
func createMode2() climenu.MenuMode {
	mode := climenu.NewBaseMenuMode(climenu.BaseMenuModeArgs{
		ModeID:   2,
		ModeName: "Actions",
		MenuOptions: []climenu.MenuOption{
			{
				Name:        "Reset",
				Description: "Reset the counter to zero",
				Handler: func(args *climenu.OptionHandlerArgs) error {
					state := args.Mode.(*customMode).state
					state.Counter = 0
					args.Writer.Printf("Counter reset to 0\n")
					return nil
				},
			},
			{
				Name:        "Double",
				Description: "Double the counter",
				Handler: func(args *climenu.OptionHandlerArgs) error {
					state := args.Mode.(*customMode).state
					state.Counter *= 2
					args.Writer.Printf("Counter doubled to %d\n", state.Counter)
					return nil
				},
			},
		},
	})

	// Wrap in custom mode to hold state reference
	return &customMode{
		BaseMenuMode: mode,
		state:        nil, // Will be set in OnEnter
	}
}

// customMode wraps BaseMenuMode to access shared state
type customMode struct {
	*climenu.BaseMenuMode
	state *DemoState
}

func (m *customMode) OnEnter(state climenu.ModeState) error {
	m.state = state.(*DemoState)
	fmt.Printf("Entered %s mode (Counter: %d)\n", m.ModeName(), m.state.Counter)
	return nil
}

func (m *customMode) OnExit(_ climenu.ModeState) error {
	fmt.Printf("Exiting %s mode\n", m.ModeName())
	return nil
}
