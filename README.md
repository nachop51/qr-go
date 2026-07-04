# qr-go

`qr-go` is a dependency-light QR code generator written in Go. It encodes text
or binary data into a QR matrix and renders it to a terminal, PNG, or SVG.

```go
import qr "github.com/nachop51/qr-go"
```

## Install

```sh
go get github.com/nachop51/qr-go
```

## Features

- Build QR codes from:
  - text via `NewTextBuilder`
  - binary data via `NewBinaryBuilder`
- Automatic segmentation (dynamic programming) across the four modes:
  numeric, alphanumeric, byte, and kanji
- Automatic version detection (1–40)
- Automatic mask selection
- Error correction levels: `CorrectionLevelLow`, `CorrectionLevelMedium`,
  `CorrectionLevelQuartile`, `CorrectionLevelHigh`
- Optional automatic ECI when text needs non-ASCII byte segments
  (disable with `SetTextECIPolicy(qr.TextECIPolicyDisabled)`)
- Pluggable renderers: terminal (default), PNG, SVG

## Quick start

`Build()` returns `(*qr.Code, error)`. The builder defaults to the terminal
renderer, so the code below prints the QR to stdout:

```go
package main

import (
	"log"

	qr "github.com/nachop51/qr-go"
)

func main() {
	code, err := qr.NewTextBuilder("HELLO WORLD").
		SetErrorCorrectionLevel(qr.CorrectionLevelMedium).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	if err := code.Render(); err != nil {
		log.Fatal(err)
	}
}
```

## Rendering

Renderers implement `render.Renderer`. Select one with `SetRenderer`, then call
`code.Render()`. Each renderer is an immutable value with fluent options.

### PNG

```go
import (
	"os"

	qr "github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/render/png"
)

f, _ := os.Create("qr.png")
defer f.Close()

code, err := qr.NewTextBuilder("https://example.com").
	SetRenderer(png.New().Writer(f).Width(512).Height(512).Quiet(4)).
	Build()
if err != nil {
	log.Fatal(err)
}
if err := code.Render(); err != nil {
	log.Fatal(err)
}
```

PNG options: `Writer(io.Writer)`, `Width(int)`, `Height(int)`, `Quiet(int)`,
`Dark(color.Color)`, `White(color.Color)`. If no writer is set, it writes to
`test.png` in the working directory.

### SVG

```go
import "github.com/nachop51/qr-go/render/svg"

builder.SetRenderer(svg.New().Writer(f).Module(8).Dark("#111111").Light("#ffffff"))
```

SVG options: `Writer(io.Writer)`, `Module(int)` (module size in px),
`Quiet(int)`, `Dark(string)`, `Light(string)`. Defaults to `os.Stdout`.

### Terminal (default)

```go
import "github.com/nachop51/qr-go/render/terminal"

builder.SetRenderer(terminal.New().Quiet(2))
```

Terminal options: `Writer(io.Writer)`, `Quiet(int)`, `Dark(string)`,
`Light(string)`. Defaults to `os.Stdout` with block characters.

## Binary data

```go
code, err := qr.NewBinaryBuilder([]byte{0x00, 0x01, 0x02, 0xff}).
	SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
	Build()
```

Binary input is always encoded as a single byte segment (no ECI).

## Text encoding and ECI

For text input the builder validates UTF-8. If the text needs non-ASCII byte
segments, ECI is enabled automatically. Disable it with:

```go
builder.SetTextECIPolicy(qr.TextECIPolicyDisabled)
```

## Inspecting the result

`qr.Code` exposes the encoding outcome and the module matrix:

```go
code.Version              // 1–40
code.Mask                 // 0–7
code.ErrorCorrectionLevel // the level used
code.IsECI                // whether ECI was applied
code.Segments             // the chosen segmentation
code.Size()               // modules per side
code.IsDark(x, y)         // module colour at (x, y)
```

## Command-line demo

A small playground lives in `cmd/`:

```sh
go run ./cmd "HELLO WORLD"
```

## Project layout

- package root (`*.go`) — QR encoding library and public API
- `internal/spec` — QR spec tables (capacity, ECC, alignment, format info)
- `internal/coding` — bit stream and Reed–Solomon error correction
- `internal/matrix` — the module grid
- `render/` — the renderer contract and the terminal / PNG / SVG renderers
- `cmd/` — command-line example

## License

[MIT](LICENSE)
