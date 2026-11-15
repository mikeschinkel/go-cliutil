// Package homedir tests that cliWriter.Errorf shortens the user's home
// directory to ~ once cliutil.Initialize() has run. It is its own package, and
// so its own test binary, because Initialize records the home directory in
// package-level state that nothing resets.
package homedir

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

// initWithHome points HOME at home, runs Initialize with a writer whose stderr
// is a buffer, and returns that buffer.
func initWithHome(t *testing.T, home string) (errOut *bytes.Buffer) {
	t.Helper()
	t.Setenv("HOME", home)
	errOut = &bytes.Buffer{}
	w := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&bytes.Buffer{}, errOut),
		Verbosity: 1,
	})
	err := cliutil.Initialize(w)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	return errOut
}

func TestInitialize_EnsuresUserHomeDir(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a")
	initWithHome(t, home)
	got := dt.GetUserHomeDir()
	if string(got) != home {
		t.Errorf("dt.GetUserHomeDir(): want %q, got %q", home, got)
	}
}

func TestErrorf_ShortensHomeDir(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a")
	errOut := initWithHome(t, home)
	cliutil.GetWriter().Errorf("Error: %v\n", errors.New("open "+home+"/a/b: denied; also "+home+"/c"))
	want := "Error: open ~/a/b: denied; also ~/c\n"
	if got := errOut.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestErrorf_LeavesSiblingOfHomeDir(t *testing.T) {
	tmp := t.TempDir()
	errOut := initWithHome(t, filepath.Join(tmp, "a"))
	sibling := filepath.Join(tmp, "ab", "x")
	cliutil.GetWriter().Errorf("Error: %v\n", errors.New("open "+sibling))
	want := "Error: open " + sibling + "\n"
	if got := errOut.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestErrorf_MultiLineErrorWithHomeDir(t *testing.T) {
	home := filepath.Join(t.TempDir(), "a")
	errOut := initWithHome(t, home)
	cliutil.GetWriter().Errorf("Error: %v\n", errors.New("bad config\nfile="+home+"/cfg.json"))
	want := "Error: bad config; file=~/cfg.json\n"
	if got := errOut.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
