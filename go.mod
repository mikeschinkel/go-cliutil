module github.com/mikeschinkel/go-cliutil

go 1.25.3

replace (
	github.com/mikeschinkel/go-dt => ../go-dt
	github.com/mikeschinkel/go-dt/appinfo => ../go-dt/appinfo
	github.com/mikeschinkel/go-dt/de => ../go-dt/de
)

require (
	github.com/mikeschinkel/go-dt v0.0.0-20251105233453-a7985f775567
	github.com/mikeschinkel/go-dt/appinfo v0.0.0-20251106125543-42540c8e051a
)

require github.com/mikeschinkel/go-dt/de v0.0.0-20251105233453-a7985f775567 // indirect
