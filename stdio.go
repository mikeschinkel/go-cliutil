package cliutil

import (
	"fmt"
	"os"

	"github.com/mikeschinkel/go-dt/dtx"
)

func Stdoutf(format string, args ...any) {
	_, err := fmt.Fprintf(os.Stdout, format, args...)
	dtx.LogOnError(err)
}
func Stderrf(format string, args ...any) {
	_, err := fmt.Fprintf(os.Stderr, format, args...)
	dtx.LogOnError(err)
}
