package cliutil

import (
	"fmt"
	"io"
	"os"

	"github.com/mikeschinkel/go-dt"
)

func Stdoutf(format string, args ...any) {
	Stdiof(os.Stdout, format, args...)
}
func Stderrf(format string, args ...any) {
	Stdiof(os.Stderr, format, args...)
}
func Stdiof(w io.Writer, format string, args ...any) {
	_, err := fmt.Fprintf(w, format, args...)
	dt.LogOnError(err)
}

func Stdout(args ...any) {
	Stdio(os.Stdout, args...)
}
func Stderr(args ...any) {
	Stdio(os.Stderr, args...)
}
func Stdio(w io.Writer, args ...any) {
	_, err := fmt.Fprint(w, args...)
	dt.LogOnError(err)
}
