package cliutil

type Flags interface {
	Flags()
}

type ArgsParser = interface {
	Flags
	Parse([]string) error
	SetGlobalFlags(*GlobalFlags)
}

func ParseOptions(flags ArgsParser, opts Options) (_ Options, args []string, err error) {
	var gFlags *GlobalFlags
	var errs []error

	// Start by getting the global flags which could return an error
	// ParseGlobalFlags() will return args from os.Args with global flags removed
	gFlags, args, err = ParseGlobalFlags()
	errs = AppendErr(errs, err)

	// Next get the application-specific flags with global flags merged in
	flags.SetGlobalFlags(gFlags)
	errs = AppendErr(errs, flags.Parse(args))
	errs = AppendErr(errs, opts.Parse(flags))
	return opts, args, CombineErrs(errs)
}
