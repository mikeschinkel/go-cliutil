module modal_demo

go 1.25.3

require github.com/mikeschinkel/go-cliutil v0.0.0

require (
	github.com/mikeschinkel/go-dt v0.3.3 // indirect
	github.com/mikeschinkel/go-dt/appinfo v0.2.1 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
)

replace github.com/mikeschinkel/go-cliutil => ../..
replace github.com/mikeschinkel/go-dt =>  ../../../go-dt
replace github.com/mikeschinkel/go-dt/dtx =>  ../../../go-dt/dtx

