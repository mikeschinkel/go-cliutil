package cliutil

import (
	"reflect"
	"slices"
	"sync"
)

// CommandFunc constructs a command. Each run calls it, so flag and operand
// targets are fields of a fresh value and nothing leaks between runs.
type CommandFunc func() Command

type registration struct {
	fn      CommandFunc
	parents []reflect.Type
}

// Registry holds the commands an app can run.
type Registry struct {
	mu            sync.RWMutex
	registrations []registration
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

var defaultRegistry = NewRegistry()

// DefaultRegistry returns the Registry RegisterCommand fills.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// RegisterCommand registers a command in the DefaultRegistry, normally from the
// init() of the command's own file. Parents are prototypes identified by type:
//
//	cliutil.RegisterCommand(NewJobRunCmd, &JobCmd{})
func RegisterCommand(fn CommandFunc, parents ...Command) {
	defaultRegistry.Register(fn, parents...)
}

// Register records a command. It only records, so it is safe to call from
// init(); definition errors surface from Validate and when a run starts.
func (r *Registry) Register(fn CommandFunc, parents ...Command) {
	parentTypes := make([]reflect.Type, len(parents))
	for i, parent := range parents {
		parentTypes[i] = commandType(parent)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registrations = append(r.registrations, registration{fn: fn, parents: parentTypes})
}

// Validate builds the registered commands and reports every definition error,
// checking command flags against the global flags opts declares. A nil opts
// means the built-in global flags.
func (r *Registry) Validate(opts Options) (err error) {
	var defs []FlagDef

	if opts == nil {
		opts = NewGlobalOptions(GlobalOptionsArgs{})
	}
	defs, err = globalFlagDefs(opts)
	if err != nil {
		err = WithErr(err, ErrCommandRegistrationFailed)
		goto end
	}
	_, err = newCommandSet(r.snapshot(), defs)

end:
	return err
}

func (r *Registry) snapshot() []registration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.registrations)
}
