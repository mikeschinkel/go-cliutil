package cliutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
)

var _ Writer = (*BufferedWriter)(nil)

// BufferedWriter implements Writer and captures all output in buffers for testing
type BufferedWriter struct {
	stdBuf     *bytes.Buffer
	errBuf     *bytes.Buffer
	mu         sync.RWMutex
	quiet      bool
	verbosity  int
	useLevel   int
	loudWriter Writer
	v2Writer   Writer
	v3Writer   Writer
}

func (w *BufferedWriter) Writer() io.Writer {
	return w.stdBuf
}

func (w *BufferedWriter) ErrWriter() io.Writer {
	return w.errBuf
}

// Verify BufferedWriter implements Writer interface
var _ Writer = (*BufferedWriter)(nil)

// NewBufferedWriter creates a new BufferedWriter with default settings
func NewBufferedWriter() *BufferedWriter {
	return &BufferedWriter{
		stdBuf:    &bytes.Buffer{},
		errBuf:    &bytes.Buffer{},
		quiet:     false,
		verbosity: 3, // Default to max verbosity for testing
		useLevel:  1, // Default level
	}
}

// NewBufferedWriterWithVerbosity creates a BufferedWriter with specified verbosity level
func NewBufferedWriterWithVerbosity(verbosity int) *BufferedWriter {
	if verbosity < 1 || verbosity > 3 {
		panic(fmt.Sprintf("Invalid verbosity for BufferedWriter; must be between 1-3; got %d", verbosity))
	}
	return &BufferedWriter{
		stdBuf:    &bytes.Buffer{},
		errBuf:    &bytes.Buffer{},
		quiet:     false,
		verbosity: verbosity,
		useLevel:  1,
	}
}

// Printf writes formatted output to stdBuf buffer
func (w *BufferedWriter) Printf(format string, args ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.quiet {
		return
	}
	if w.verbosity < w.useLevel {
		return
	}

	formatted := fmt.Sprintf(format, args...)
	w.stdBuf.WriteString(formatted)
}

// Errorf writes formatted error output to doterr buffer
func (w *BufferedWriter) Errorf(format string, args ...any) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Process error arguments to flatten newlines (same as cliWriter)
	processedArgs := make([]any, len(args))
	for i, arg := range args {
		if err, ok := arg.(error); ok {
			processedArgs[i] = strings.Replace(err.Error(), "\n", "; ", -1)
		} else {
			processedArgs[i] = arg
		}
	}

	formatted := fmt.Sprintf(format, processedArgs...)
	w.errBuf.WriteString(formatted)
}

// Loud returns a Writer that ignores the quiet setting
func (w *BufferedWriter) Loud() Writer {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.loudWriter != nil {
		return w.loudWriter
	}

	w.loudWriter = &BufferedWriter{
		stdBuf:    w.stdBuf, // Share the same buffers
		errBuf:    w.errBuf,
		quiet:     false, // Always loud
		verbosity: w.verbosity,
		useLevel:  w.useLevel,
	}
	return w.loudWriter
}

// V2 returns a Writer for verbosity level 2
func (w *BufferedWriter) V2() Writer {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.v2Writer != nil {
		return w.v2Writer
	}

	w.v2Writer = &BufferedWriter{
		stdBuf:    w.stdBuf, // Share the same buffers
		errBuf:    w.errBuf,
		quiet:     w.quiet,
		verbosity: w.verbosity,
		useLevel:  2, // Level 2
	}
	return w.v2Writer
}

// V3 returns a Writer for verbosity level 3
func (w *BufferedWriter) V3() Writer {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.v3Writer != nil {
		return w.v3Writer
	}

	w.v3Writer = &BufferedWriter{
		stdBuf:    w.stdBuf, // Share the same buffers
		errBuf:    w.errBuf,
		quiet:     w.quiet,
		verbosity: w.verbosity,
		useLevel:  3, // Level 3
	}
	return w.v3Writer
}

// Testing helper methods

// GetStdout returns the current stdBuf buffer contents
func (w *BufferedWriter) GetStdout() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stdBuf.String()
}

// GetStderr returns the current doterr buffer contents
func (w *BufferedWriter) GetStderr() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.errBuf.String()
}

// GetAllOutput returns both stdBuf and doterr combined
func (w *BufferedWriter) GetAllOutput() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stdBuf.String() + w.errBuf.String()
}

// ContainsStdout returns true if stdBuf buffer contains the specified substring
func (w *BufferedWriter) ContainsStdout(s string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return strings.Contains(w.stdBuf.String(), s)
}

// ContainsStderr returns true if doterr buffer contains the specified substring
func (w *BufferedWriter) ContainsStderr(s string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return strings.Contains(w.errBuf.String(), s)
}

// ContainsOutput returns true if either buffer contains the specified substring
func (w *BufferedWriter) ContainsOutput(s string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return strings.Contains(w.stdBuf.String(), s) || strings.Contains(w.errBuf.String(), s)
}

// Reset clears both stdBuf and doterr buffers
func (w *BufferedWriter) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.stdBuf.Reset()
	w.errBuf.Reset()
}

// SetQuiet sets the quiet mode (suppresses all Printf output)
func (w *BufferedWriter) SetQuiet(quiet bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.quiet = quiet
}

// SetVerbosity sets the verbosity level (1-3)
func (w *BufferedWriter) SetVerbosity(verbosity int) {
	if verbosity < 1 || verbosity > 3 {
		panic(fmt.Sprintf("Invalid verbosity for BufferedWriter; must be between 1-3; got %d", verbosity))
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.verbosity = verbosity
}

// GetStdoutLines returns stdBuf content split into lines (excluding empty lines)
func (w *BufferedWriter) GetStdoutLines() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	content := w.stdBuf.String()
	if content == "" {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// GetStderrLines returns doterr content split into lines (excluding empty lines)
func (w *BufferedWriter) GetStderrLines() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	content := w.errBuf.String()
	if content == "" {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return result
}

// CountStdoutLines returns the number of non-empty lines in stdBuf
func (w *BufferedWriter) CountStdoutLines() int {
	return len(w.GetStdoutLines())
}

// CountStderrLines returns the number of non-empty lines in doterr
func (w *BufferedWriter) CountStderrLines() int {
	return len(w.GetStderrLines())
}
