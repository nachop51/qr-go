module github.com/nachop51/qr-go

go 1.25.0

require (
	github.com/makiuchi-d/gozxing v0.1.1
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.9
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	golang.org/x/image v0.43.0
	golang.org/x/text v0.38.0 // pinned: overrides vulnerable x/text in the oksvg/x/net graph
	rsc.io/qr v0.2.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
