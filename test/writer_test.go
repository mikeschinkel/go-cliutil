package test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// TestNewWriter_WritesToStdIO proves that every writer NewWriter derives
// writes to the dt.StdIO given in WriterArgs.IO, not to the process streams.
func TestNewWriter_WritesToStdIO(t *testing.T) {
	var out, errOut bytes.Buffer
	w := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&out, &errOut),
		Verbosity: 3,
	})

	if w.Writer() != &out {
		t.Errorf("Writer() should be the StdIO's Output(), got %T", w.Writer())
	}
	if w.ErrWriter() != &errOut {
		t.Errorf("ErrWriter() should be the StdIO's ErrOut(), got %T", w.ErrWriter())
	}

	w.Printf("base\n")
	w.V2().Printf("v2\n")
	w.V3().Printf("v3\n")
	w.Loud().Printf("loud\n")
	w.V2().Loud().Printf("v2-loud\n")

	w.Errorf("err: %v\n", errors.New("line 1\nline 2"))
	w.V2().Errorf("v2-err\n")
	w.V3().Errorf("v3-err\n")
	w.Loud().Errorf("loud-err\n")
	w.V3().Loud().Errorf("v3-loud-err\n")

	wantOut := "base\nv2\nv3\nloud\nv2-loud\n"
	if got := out.String(); got != wantOut {
		t.Errorf("stdout: want %q, got %q", wantOut, got)
	}
	wantErr := "err: line 1; line 2\nv2-err\nv3-err\nloud-err\nv3-loud-err\n"
	if got := errOut.String(); got != wantErr {
		t.Errorf("stderr: want %q, got %q", wantErr, got)
	}
}

// TestNewWriter_VerbosityAgainstStdIO proves the verbosity gate still applies
// to the V2 and V3 writers when they write to a StdIO.
func TestNewWriter_VerbosityAgainstStdIO(t *testing.T) {
	var out, errOut bytes.Buffer
	w := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&out, &errOut),
		Verbosity: 2,
	})

	w.Printf("base\n")
	w.V2().Printf("v2\n")
	w.V3().Printf("v3\n")

	want := "base\nv2\n"
	if got := out.String(); got != want {
		t.Errorf("stdout at verbosity 2: want %q, got %q", want, got)
	}
}

// TestNewWriter_QuietAgainstStdIO proves Quiet silences Printf but not
// Errorf or Loud() when writing to a StdIO.
func TestNewWriter_QuietAgainstStdIO(t *testing.T) {
	var out, errOut bytes.Buffer
	w := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&out, &errOut),
		Quiet:     true,
		Verbosity: 1,
	})

	w.Printf("quiet\n")
	w.Loud().Printf("loud\n")
	w.Errorf("err\n")

	if got := out.String(); got != "loud\n" {
		t.Errorf("stdout when quiet: want %q, got %q", "loud\n", got)
	}
	if got := errOut.String(); got != "err\n" {
		t.Errorf("stderr when quiet: want %q, got %q", "err\n", got)
	}
}

// TestNewWriter_NilIODefaultsToProcessStreams proves a caller passing
// WriterArgs without IO keeps the process streams, and that no derived
// writer dereferences a nil StdIO. Printf("") writes zero bytes, which still
// calls Write on the underlying writer, so a nil one would panic here.
func TestNewWriter_NilIODefaultsToProcessStreams(t *testing.T) {
	w := cliutil.NewWriter(&cliutil.WriterArgs{Verbosity: 3})

	if w.Writer() == nil {
		t.Fatal("Writer() should not be nil when WriterArgs.IO is nil")
	}
	if w.ErrWriter() == nil {
		t.Fatal("ErrWriter() should not be nil when WriterArgs.IO is nil")
	}

	w.Printf("")
	w.V2().Printf("")
	w.V3().Printf("")
	w.Loud().Printf("")
	w.V2().Loud().Printf("")
	w.V3().Loud().Printf("")
	w.Loud().V2().Printf("")
	w.Errorf("")
	w.V2().Errorf("")
	w.V3().Errorf("")
	w.Loud().Errorf("")
}

// TestNewWriter_NilArgsDefaultsToProcessStreams proves NewWriter(nil) keeps
// working as it did before WriterArgs gained IO.
func TestNewWriter_NilArgsDefaultsToProcessStreams(t *testing.T) {
	w := cliutil.NewWriter(nil)

	if w.Writer() == nil {
		t.Fatal("Writer() should not be nil for NewWriter(nil)")
	}
	w.Printf("")
	w.V2().Printf("")
	w.V3().Printf("")
	w.Loud().Printf("")
	w.Errorf("")
}
