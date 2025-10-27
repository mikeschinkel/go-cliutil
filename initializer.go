package cliutil

import (
	"errors"
)

type InitializerArgs struct {
	Writer Writer
}

type InitializerFunc func(InitializerArgs) error

var initializerFuncs []InitializerFunc

func RegisterInitializerFunc(f InitializerFunc) {
	initializerFuncs = append(initializerFuncs, f)
}

func CallInitializerFuncs(args InitializerArgs) (err error) {
	var errs []error
	for _, f := range initializerFuncs {
		errs = append(errs, f(args))
	}
	return errors.Join(errs...)
}
