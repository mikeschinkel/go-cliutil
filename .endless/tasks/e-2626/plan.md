commands.go, `Initialize(w Writer)`: before its existing steps, call `dt.EnsureUserHomeDir()` and return its error; then store `dt.GetUserHomeDir()` in a private package variable (for example `homeDir dt.DirPath`).

writer.go, `(*cliWriter).Errorf`: for each `error` argument, after replacing newlines with "; ", replace every occurrence of `string(homeDir) + "/"` with `"~/"` when `homeDir` is not empty.

Tests in the test/ module (writer_test.go or a new file), with HOME set through t.Setenv to a temp dir: an error holding `<home>/a/b` prints `~/a/b` after Initialize(); `<home>schinkel/x` is unchanged; a multi-line error still prints with "; "; and an Errorf before Initialize() prints the full path without panicking.
