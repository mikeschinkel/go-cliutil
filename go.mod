module github.com/mikeschinkel/go-cliutil

go 1.25.3

require (
	github.com/mikeschinkel/go-dt v0.6.0
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	github.com/mikeschinkel/go-testutil v0.2.1
	golang.org/x/term v0.40.0
)

require golang.org/x/sys v0.41.0 // indirect

replace github.com/mikeschinkel/go-dt/dtx => ../go-dt/dtx

replace github.com/mikeschinkel/go-dt => ../go-dt

replace github.com/mikeschinkel/go-testutil => ../go-testutil
