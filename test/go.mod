module test

go 1.25.3

replace (
	github.com/mikeschinkel/go-cliutil => ..
	github.com/mikeschinkel/go-testutil => ../../go-testutil
)

require (
	github.com/mikeschinkel/go-cliutil v0.2.0
	github.com/mikeschinkel/go-testutil v0.2.0
)

require (
	github.com/mikeschinkel/go-dt v0.2.5 // indirect
	github.com/mikeschinkel/go-dt/appinfo v0.2.1 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
)
