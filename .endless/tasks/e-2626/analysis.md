Replacing $HOME with ~ is a display concern (Mike's decision while planning go-dt E-2623): it belongs in go-cliutil's error printing, not in doterr or in go-dt's ErrKV().

cliWriter.Errorf is the place because it already picks out each error argument and rewrites it. Stderr and Stderrf exist for when a writer fails; an app that wants ~ paths there adds them itself.

Initialize() calls dt.EnsureUserHomeDir() so that the library depending on the home directory sets it up, rather than requiring every app to. Nothing in go-cfgstore, go-logutil or go-cliutil needs dt's cached home directory before cliutil.Initialize() runs: go-cfgstore reads it with dt.UserHomeDir(), which does not use the cache.

Errorf reads a private copy captured in Initialize() rather than calling dt.GetUserHomeDir() itself, so printing an error can never panic: before Initialize() the copy is empty and paths print in full. The home directory does not change over an app's life; dt's cache changes only if something calls EnsureUserHomeDir() again after HOME changes.

Replacing "${HOME}/" (with the trailing slash) rather than "${HOME}" keeps a sibling directory whose name merely starts with the home directory's name (home "/home/a", sibling "/home/ab") from being shortened.
