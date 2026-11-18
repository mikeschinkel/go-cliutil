// Package cliutil provides output management and synchronized writing for CLI applications.
package cliutil

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/mikeschinkel/go-dt"
)

// Writer defines the interface for user-facing writer
type Writer interface {
	Printf(string, ...any)
	Errorf(string, ...any)
	Loud() Writer
	V2() Writer
	V3() Writer
	Writer() io.Writer
	ErrWriter() io.Writer
}

var _ Writer = (*cliWriter)(nil)

// outputWriter writes to stdout/doterr for normal CLI usage
type cliWriter struct {
	io        dt.StdIO
	writer    io.Writer
	errWriter io.Writer
	quiet     bool
	loud      Writer
	v2        Writer
	v3        Writer
	useLevel  int
	verbosity Verbosity
}

func (w *cliWriter) Writer() io.Writer {
	return w.writer
}

func (w *cliWriter) ErrWriter() io.Writer {
	return w.errWriter
}

func (w *cliWriter) V2() Writer {
	if w.v2 != nil {
		goto end
	}
	w.v2 = &cliWriter{
		io:        w.io,
		writer:    w.io.Output(),
		errWriter: w.io.ErrOut(),
		verbosity: w.verbosity,
		useLevel:  2,
	}
end:
	return w.v2
}

func (w *cliWriter) V3() Writer {
	if w.v3 != nil {
		goto end
	}
	w.v3 = &cliWriter{
		io:        w.io,
		writer:    w.io.Output(),
		errWriter: w.io.ErrOut(),
		verbosity: w.verbosity,
		useLevel:  3,
	}
end:
	return w.v3
}

func (w *cliWriter) Loud() Writer {
	if w.loud != nil {
		goto end
	}
	w.loud = &cliWriter{
		io:        w.io,
		writer:    w.io.Output(),
		errWriter: w.io.ErrOut(),
		quiet:     false,
	}
end:
	return w.loud
}

// WriterArgs configures the Writer NewWriter returns.
type WriterArgs struct {
	// IO supplies the streams the Writer and every Writer it derives (V2, V3,
	// Loud) write to. Nil means the process streams, dt.NewDefaultStdIO().
	IO        dt.StdIO
	Quiet     bool
	Verbosity Verbosity
}

// NewWriter creates a console writer writer
//
//goland:noinspection GoUnusedExportedFunction
func NewWriter(args *WriterArgs) Writer {
	if args == nil {
		args = &WriterArgs{Verbosity: 1}
	}
	if args.IO == nil {
		args.IO = dt.NewDefaultStdIO()
	}
	if args.Verbosity < 1 || 3 < args.Verbosity {
		panic(fmt.Sprintf("Invalid verbosity for cliutil.Writer.SetVerbosity(); must be between 1-3; got %d", args.Verbosity))
	}
	return &cliWriter{
		io:        args.IO,
		writer:    args.IO.Output(),
		errWriter: args.IO.ErrOut(),
		quiet:     args.Quiet,
		verbosity: args.Verbosity,
	}
}

// Printf writes formatted writer to stdout
func (w *cliWriter) Printf(format string, args ...any) {
	if w.quiet {
		goto end
	}
	if int(w.verbosity) < w.useLevel {
		goto end
	}
	_, _ = fmt.Fprintf(w.writer, format, args...)
end:
	return
}

// Errorf writes formatted error writer to doterr
func (w *cliWriter) Errorf(format string, args ...any) {
	for i, arg := range args {
		err, ok := arg.(error)
		if !ok {
			continue
		}
		// Replace newlines in errors with semicolons
		args[i] = shortenHomeDir(strings.ReplaceAll(err.Error(), "\n", "; "))
	}
	_, _ = fmt.Fprintf(w.errWriter, format, args...)
}

// homeDir is the user's home directory as Initialize found it. Errorf reads
// this copy rather than dt.GetUserHomeDir(), which panics before
// dt.EnsureUserHomeDir() runs, so printing an error can never panic. Before
// Initialize runs it is empty and Errorf prints paths in full.
var (
	homeDir   dt.DirPath
	homeDirMu sync.RWMutex
)

func setHomeDir(dir dt.DirPath) {
	homeDirMu.Lock()
	defer homeDirMu.Unlock()
	homeDir = dir
}

// shortenHomeDir replaces every "<home>/" in s with "~/". Matching the
// trailing slash keeps a sibling such as "/home/ab" from being shortened when
// the home directory is "/home/a".
func shortenHomeDir(s string) string {
	homeDirMu.RLock()
	dir := homeDir
	homeDirMu.RUnlock()
	if dir != "" {
		s = strings.ReplaceAll(s, string(dir)+"/", "~/")
	}
	return s
}
