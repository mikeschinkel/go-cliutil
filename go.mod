module github.com/mikeschinkel/go-cliutil

go 1.25.3

require (
	github.com/mikeschinkel/go-dt v0.3.3
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	github.com/mikeschinkel/go-testutil v0.2.1
	golang.org/x/term v0.38.0
)

require golang.org/x/sys v0.39.0 // indirect

replace github.com/mikeschinkel/go-dt/dtx => ../go-dt/dtx

replace github.com/mikeschinkel/go-dt => ../go-dt

replace github.com/mikeschinkel/go-testutil => ../go-testutil
