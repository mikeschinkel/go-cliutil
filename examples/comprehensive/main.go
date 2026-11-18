// Command comprehensive demonstrates every go-cliutil feature. Each command's
// comment names what it shows; main_test.go runs them all through cliutiltest.
package main

import (
	"fmt"
	"os"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt/appinfo"
)

// Options embeds GlobalOptions, drops --dry-run with Omit, changes --timeout with
// Overrides, and adds the app's own global flags as typed fields.
type Options struct {
	*cliutil.GlobalOptions
	Input  string
	Region string
}

// NewOptions builds a fresh Options for each run.
func NewOptions() cliutil.Options {
	return &Options{
		GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{
			Omit: []string{cliutil.DryRunFlagName},
			Overrides: []cliutil.FlagDef{
				{Name: cliutil.TimeoutFlagName, DefaultInt: 30, Usage: "Seconds to wait for the server"},
			},
		}),
	}
}

// FlagDefs declares the app's own global flags, targeting this value's fields.
func (o *Options) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{
		{Name: "input", Shortcut: 'i', Usage: "File to read", String: &o.Input},
		{Name: "region", Usage: "Region to use", DefaultString: "us", String: &o.Region},
	}
}

// ApplyConfig copies config into typed fields, so the compiler checks each one.
func (o *Options) ApplyConfig(cfg cliutil.Config) error {
	c := cliutil.ConfigAs[*RootConfig](cfg)
	o.SetVerbosity(c.Verbosity())
	o.Input = c.Input()
	return nil
}

// RootConfig is the app's config. Typed methods feed ApplyConfig, and FlagValue
// shows the name-based route a library such as Viper would use.
type RootConfig struct {
	verbosity    int
	input        string
	exportFormat string
	region       string
}

// Config marks RootConfig as a cliutil.Config.
func (*RootConfig) Config() {}

// Verbosity returns the configured verbosity.
func (c *RootConfig) Verbosity() int { return c.verbosity }

// Input returns the configured input file.
func (c *RootConfig) Input() string { return c.input }

// ExportFormat returns the configured export format.
func (c *RootConfig) ExportFormat() string { return c.exportFormat }

// FlagValue supplies --region by name.
func (c *RootConfig) FlagValue(name string) (any, bool) {
	if name == "region" && c.region != "" {
		return c.region, true
	}
	return nil, false
}

// LoadConfig stands in for reading a config file.
func LoadConfig(cliutil.ConfigArgs) (cliutil.Config, error) {
	return &RootConfig{
		verbosity:    1,
		input:        "config-input.txt",
		exportFormat: "yaml",
		region:       "eu",
	}, nil
}

func main() {
	err := cliutil.Initialize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cliutil.ExitInitializationError)
	}
	runner := cliutil.NewCmdRunner(cliutil.CmdRunnerArgs{
		AppInfo: appinfo.New(appinfo.Args{
			Name:        "comprehensive",
			Description: "Every go-cliutil feature in one app",
			ExeName:     "comprehensive",
			InfoURL:     "https://github.com/mikeschinkel/go-cliutil",
		}),
		Config:  LoadConfig,
		Options: NewOptions,
	})
	err = runner.Run(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cliutil.ExitCode(err))
	}
}
