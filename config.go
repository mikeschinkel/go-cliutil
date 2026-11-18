package cliutil

import (
	"fmt"
	"reflect"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// Config represents any config object that can be passed to commands
type Config interface {
	Config()
}

// ConfigArgs is the input to a ConfigFunc.
type ConfigArgs struct {
	AppInfo appinfo.AppInfo
	Flags   Flags // Parsed already, so a --config flag can choose the file.
	IO      dt.StdIO
}

// ConfigFunc loads the app's config for one run. go-cliutil imports no config
// library; the app decides how config is stored and read.
type ConfigFunc func(ConfigArgs) (Config, error)

// ConfigApplier copies config values into typed fields, so the compiler checks
// every mapping. Each run calls ApplyConfig on the Options value and then on the
// matched command, after typed defaults and before command-line values, so a
// flag the user typed beats config.
type ConfigApplier interface {
	ApplyConfig(Config) error
}

// FlagValuer is the name-based route for a Config such as Viper: it supplies a
// value for a flag the user did not type.
type FlagValuer interface {
	FlagValue(name string) (value any, ok bool)
}

// ConfigAs returns c as T. It panics naming both types when c is not a T, since
// that is a programming error.
func ConfigAs[T Config](c Config) T {
	value, ok := any(c).(T)
	if !ok {
		panic(fmt.Sprintf("cliutil.ConfigAs: config is %T, not %s", c, reflect.TypeFor[T]()))
	}
	return value
}
