package test

import (
	"errors"
	"strings"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-cliutil/cliutiltest"
)

// typedConfig reaches flags through ApplyConfig.
type typedConfig struct {
	input  string
	format string
	err    error
}

func (*typedConfig) Config() {}

// namedConfig reaches flags through FlagValuer, as a Viper adapter would.
type namedConfig map[string]any

func (namedConfig) Config() {}

func (c namedConfig) FlagValue(name string) (any, bool) {
	value, ok := c[name]
	return value, ok
}

// configOptions applies typed config to a global flag.
type configOptions struct {
	*cliutil.GlobalOptions
	input string
}

func newConfigOptions() cliutil.Options {
	return &configOptions{GlobalOptions: cliutil.NewGlobalOptions(cliutil.GlobalOptionsArgs{})}
}

func (o *configOptions) FlagDefs() []cliutil.FlagDef {
	return []cliutil.FlagDef{{Name: "input", DefaultString: "default-input", String: &o.input}}
}

func (o *configOptions) ApplyConfig(cfg cliutil.Config) error {
	c, ok := cfg.(*typedConfig)
	if !ok {
		return nil
	}
	if c.err != nil {
		return c.err
	}
	o.input = c.input
	return nil
}

// exportCmd applies typed config to a command flag.
type exportCmd struct {
	*cliutil.CmdBase
	format  string
	count   int
	tags    []string
	needed  string
	capture *cliutil.HandleArgs
}

func (c *exportCmd) ApplyConfig(cfg cliutil.Config) error {
	typed, ok := cfg.(*typedConfig)
	if ok && typed.format != "" {
		c.format = typed.format
	}
	return nil
}

func (c *exportCmd) Handle(args cliutil.HandleArgs) error {
	*c.capture = args
	return nil
}

func configRegistry(capture *cliutil.HandleArgs, required bool) *cliutil.Registry {
	r := cliutil.NewRegistry()
	r.Register(func() cliutil.Command {
		c := &exportCmd{capture: capture}
		c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
			Name: "export",
			FlagSets: []*cliutil.FlagSet{{Name: "export", FlagDefs: []cliutil.FlagDef{
				{Name: "format", DefaultString: "json", String: &c.format},
				{Name: "count", DefaultInt: 1, Int: &c.count},
				{Name: "tag", Strings: &c.tags},
				{Name: "needed", Required: required, String: &c.needed},
			}}},
		})
		return c
	})
	return r
}

func staticConfig(cfg cliutil.Config) cliutil.ConfigFunc {
	return func(cliutil.ConfigArgs) (cliutil.Config, error) { return cfg, nil }
}

func TestConfigPrecedence(t *testing.T) {
	t.Parallel()
	typed := &typedConfig{input: "config-input", format: "yaml"}
	tests := []struct {
		name       string
		config     cliutil.Config
		line       string
		flag       string
		want       any
		wantSource cliutil.FlagSource
	}{
		{"default without config", nil, "export", "format", "json", cliutil.DefaultSource},
		{"ApplyConfig on command", typed, "export", "format", "yaml", cliutil.ConfigSource},
		{"ApplyConfig on options", typed, "export", "input", "config-input", cliutil.ConfigSource},
		{"typed beats ApplyConfig on command", typed, "export --format csv", "format", "csv", cliutil.CommandLineSource},
		{"typed beats ApplyConfig on options", typed, "export --input typed", "input", "typed", cliutil.CommandLineSource},
		{"untouched flag stays default", typed, "export", "count", 1, cliutil.DefaultSource},
		{"FlagValuer string", namedConfig{"format": "toml"}, "export", "format", "toml", cliutil.ConfigSource},
		{"FlagValuer float to int", namedConfig{"count": float64(4)}, "export", "count", 4, cliutil.ConfigSource},
		{"FlagValuer equal to default", namedConfig{"format": "json"}, "export", "format", "json", cliutil.ConfigSource},
		{"typed beats FlagValuer", namedConfig{"format": "toml"}, "export --format csv", "format", "csv", cliutil.CommandLineSource},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got cliutil.HandleArgs
			result := cliutiltest.Run(t, cliutiltest.Args{
				Registry:    configRegistry(&got, false),
				Config:      staticConfig(tt.config),
				Options:     newConfigOptions,
				CommandLine: strings.Fields(tt.line),
			})
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			flag, ok := got.Flags.Get(tt.flag)
			if !ok {
				t.Fatalf("flag %q missing", tt.flag)
			}
			if flag.Value != tt.want {
				t.Errorf("value: want %v, got %v", tt.want, flag.Value)
			}
			if flag.Source != tt.wantSource {
				t.Errorf("source: want %v, got %v", tt.wantSource, flag.Source)
			}
		})
	}
}

func TestFlagValuerList(t *testing.T) {
	t.Parallel()
	var got cliutil.HandleArgs
	result := cliutiltest.Run(t, cliutiltest.Args{
		Registry:    configRegistry(&got, false),
		Config:      staticConfig(namedConfig{"tag": []any{"a", "b"}}),
		CommandLine: []string{"export"},
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if tags := got.Flag[[]string]("tag"); strings.Join(tags, ",") != "a,b" {
		t.Errorf("tags: %q", tags)
	}
}

func TestRequiredFlag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  cliutil.Config
		line    string
		wantErr bool
	}{
		{"missing", nil, "export", true},
		{"typed", nil, "export --needed x", false},
		{"from config", namedConfig{"needed": "x"}, "export", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got cliutil.HandleArgs
			result := cliutiltest.Run(t, cliutiltest.Args{
				Registry:    configRegistry(&got, true),
				Config:      staticConfig(tt.config),
				CommandLine: strings.Fields(tt.line),
			})
			if tt.wantErr != (result.Err != nil) {
				t.Fatalf("error: %v", result.Err)
			}
			if tt.wantErr && !strings.Contains(result.Err.Error(), "--needed") {
				t.Errorf("error does not name the flag: %v", result.Err)
			}
		})
	}
}

func TestConfigFuncReceivesParsedFlags(t *testing.T) {
	t.Parallel()
	var seen string
	var got cliutil.HandleArgs
	result := cliutiltest.Run(t, cliutiltest.Args{
		Registry: configRegistry(&got, false),
		Config: func(args cliutil.ConfigArgs) (cliutil.Config, error) {
			flag, _ := args.Flags.Get("format")
			seen, _ = flag.Value.(string)
			if args.AppInfo == nil || args.IO == nil {
				t.Error("ConfigArgs lacks AppInfo or IO")
			}
			return nil, nil
		},
		CommandLine: []string{"export", "--format", "csv"},
	})
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if seen != "csv" {
		t.Errorf("ConfigFunc saw format %q", seen)
	}
}

func TestConfigErrors(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	tests := []struct {
		name     string
		config   cliutil.ConfigFunc
		wantErr  error
		exitCode int
	}{
		{"load fails", func(cliutil.ConfigArgs) (cliutil.Config, error) { return nil, boom }, cliutil.ErrConfigLoadFailed, cliutil.ExitConfigLoadError},
		{"apply fails", staticConfig(&typedConfig{err: boom}), cliutil.ErrConfigApplyFailed, cliutil.ExitConfigParseError},
		{"value cannot convert", staticConfig(namedConfig{"count": "many"}), cliutil.ErrConfigApplyFailed, cliutil.ExitConfigParseError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got cliutil.HandleArgs
			result := cliutiltest.Run(t, cliutiltest.Args{
				Registry:    configRegistry(&got, false),
				Config:      tt.config,
				Options:     newConfigOptions,
				CommandLine: []string{"export"},
			})
			if !errors.Is(result.Err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, result.Err)
			}
			if result.ExitCode != tt.exitCode {
				t.Errorf("exit code: want %d, got %d", tt.exitCode, result.ExitCode)
			}
		})
	}
}

func TestConfigAs(t *testing.T) {
	t.Parallel()
	cfg := &typedConfig{input: "x"}
	if cliutil.ConfigAs[*typedConfig](cfg) != cfg {
		t.Error("ConfigAs returned another value")
	}
	message := mustPanic(t, func() { cliutil.ConfigAs[namedConfig](cfg) })
	for _, want := range []string{"*test.typedConfig", "test.namedConfig"} {
		if !strings.Contains(message, want) {
			t.Errorf("panic %q does not name %q", message, want)
		}
	}
}

// operandConfigCmd writes config into its operand's target.
type operandConfigCmd struct {
	*cliutil.CmdBase
	dir string
}

func (c *operandConfigCmd) ApplyConfig(cliutil.Config) error {
	c.dir = "config-dir"
	return nil
}

func (c *operandConfigCmd) Handle(args cliutil.HandleArgs) error {
	args.Writer.Printf("dir=%s\n", c.dir)
	return nil
}

func TestTypedOperandBeatsApplyConfig(t *testing.T) {
	t.Parallel()
	r := cliutil.NewRegistry()
	r.Register(func() cliutil.Command {
		c := &operandConfigCmd{}
		c.CmdBase = cliutil.NewCmdBase(cliutil.CmdArgs{
			Name:    "bare",
			ArgDefs: []*cliutil.ArgDef{{Name: "dir", DefaultString: "default-dir", String: &c.dir}},
		})
		return c
	})
	tests := []struct{ name, line, want string }{
		{"typed operand wins", "bare typed-dir", "dir=typed-dir"},
		{"config beats operand default", "bare", "dir=config-dir"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cliutiltest.Run(t, cliutiltest.Args{
				Registry:    r,
				Config:      staticConfig(&typedConfig{}),
				CommandLine: strings.Fields(tt.line),
			})
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			if !strings.Contains(result.Stdout, tt.want) {
				t.Errorf("want %q in %q", tt.want, result.Stdout)
			}
		})
	}
}

func TestHandlerErrorExitCode(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	r := cliutil.NewRegistry()
	r.Register(func() cliutil.Command {
		return newCmd[tagBare](cliutil.CmdArgs{Name: "bare"}, func(cliutil.HandleArgs) error { return boom })
	})
	result := cliutiltest.Run(t, cliutiltest.Args{Registry: r, CommandLine: []string{"bare"}})
	if !errors.Is(result.Err, boom) || result.ExitCode != cliutil.ExitKnownRuntimeError {
		t.Errorf("err=%v exit=%d", result.Err, result.ExitCode)
	}
}
