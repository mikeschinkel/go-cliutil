package cliutil

import (
	"fmt"
	"log/slog"
)

// ModeState is an interface for user-defined shared state across modes
// Applications implement this to hold their specific state
// Example: SquireModeState in squiresvc
type ModeState interface {
	ModeState() // Marker method
}

// ModeManager coordinates multiple MenuModes and handles mode switching
type ModeManager struct {
	modes       map[int]MenuMode // Mode registry (0-9)
	currentMode int              // Currently active mode ID
	state       ModeState        // User-defined shared state
	exitLoop    bool             // Exit signal
	Writer      Writer
	Logger      *slog.Logger
}

// NewModeManager creates a new ModeManager with the given shared state
func NewModeManager(state ModeState, w Writer, lgr *slog.Logger) *ModeManager {
	if w == nil {
		w = NewWriter(nil)
	}
	if lgr == nil {
		lgr = slog.Default()
	}
	return &ModeManager{
		modes:       make(map[int]MenuMode),
		currentMode: -1, // No mode active yet
		state:       state,
		exitLoop:    false,
		Writer:      w,
		Logger:      lgr,
	}
}

// RegisterMode registers a mode at the given ID (0-9)
// Returns error if ID is out of range or already registered
func (m *ModeManager) RegisterMode(id int, mode MenuMode) (err error) {
	// Validate ID range
	if id < 0 || id > 9 {
		err = fmt.Errorf("mode ID must be 0-9, got %d", id)
		goto end
	}

	// Check for duplicate registration
	if _, exists := m.modes[id]; exists {
		err = fmt.Errorf("mode ID %d already registered", id)
		goto end
	}

	if m.Writer == nil {
		m.Writer = NewWriter(nil)
	}
	mode.SetWriter(m.Writer)

	if m.Logger == nil {
		m.Logger = slog.Default()
	}
	mode.SetLogger(m.Logger)

	// Register the mode
	m.modes[id] = mode

	// If this is the first mode, set it as current
	if m.currentMode == -1 {
		m.currentMode = id
	}

end:
	return err
}

// SwitchMode switches to the mode with the given ID
// Calls OnExit on current mode, then OnEnter on new mode
// Returns error if mode ID is not registered or lifecycle methods fail
func (m *ModeManager) SwitchMode(id int) (err error) {
	var newMode MenuMode
	var exists bool

	// Validate mode exists
	newMode, exists = m.modes[id]
	if !exists {
		err = fmt.Errorf("mode ID %d not registered", id)
		goto end
	}

	// Don't switch if already in this mode
	if id == m.currentMode {
		goto end
	}

	// Call OnExit on current mode (if any)
	if m.currentMode != -1 {
		err = m.modes[m.currentMode].OnExit(m.state)
		if err != nil {
			goto end
		}
	}

	// Switch to new mode
	m.currentMode = id

	// Call OnEnter on new mode
	err = newMode.OnEnter(m.state)
	if err != nil {
		goto end
	}

end:
	return err
}

// CurrentMode returns the currently active mode
// Returns nil if no mode is active
func (m *ModeManager) CurrentMode() MenuMode {
	if m.currentMode == -1 {
		return nil
	}
	return m.modes[m.currentMode]
}

// CurrentModeID returns the ID of the currently active mode
// Returns -1 if no mode is active
func (m *ModeManager) CurrentModeID() int {
	return m.currentMode
}

// State returns the shared state object
func (m *ModeManager) State() ModeState {
	return m.state
}

// RequestExit signals that the modal menu loop should exit
func (m *ModeManager) RequestExit() {
	m.exitLoop = true
}

// ShouldExit returns true if exit has been requested
func (m *ModeManager) ShouldExit() bool {
	return m.exitLoop
}
