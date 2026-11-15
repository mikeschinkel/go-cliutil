// Package preinit tests cliWriter.Errorf in a test binary that never calls
// cliutil.Initialize() or dt.EnsureUserHomeDir(), so the home directory is
// unknown to both go-cliutil and go-dt.
package preinit

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
)

func TestErrorf_BeforeInitializePrintsFullPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}
	var errOut bytes.Buffer
	w := cliutil.NewWriter(&cliutil.WriterArgs{
		IO:        dt.NewStdIO(&bytes.Buffer{}, &errOut),
		Verbosity: 1,
	})
	path := home + "/a/b"
	w.Errorf("Error: %v\n", errors.New("open "+path+"\nretry"))
	want := "Error: open " + path + "; retry\n"
	if got := errOut.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
