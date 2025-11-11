package cliutil

import (
	"fmt"
	"io"
	"os"

	"github.com/mikeschinkel/go-dt/dtx"
)

func Stdoutf(format string, args ...any) {
	Stdiof(os.Stdout, format, args...)
}
func Stderrf(format string, args ...any) {
	Stdiof(os.Stderr, format, args...)
}
func Stdiof(w io.Writer, format string, args ...any) {
	_, err := fmt.Fprintf(w, format, args...)
	dtx.LogOnError(err)
}
