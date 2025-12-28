module test

go 1.25.3

require (
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-testutil v0.3.0
)

require (
	github.com/mikeschinkel/go-dt v0.3.3 // indirect
	github.com/mikeschinkel/go-dt/appinfo v0.2.1 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
)

replace github.com/mikeschinkel/go-cliutil => ..

replace github.com/mikeschinkel/go-testutil => ../../../go-pkgs/go-testutil
