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

Upgrading from v0.1? See [RELEASE_NOTES.md](RELEASE_NOTES.md) for the v0.2
breaking changes and migration table.

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
- Pluggable renderers: terminal (default, compact half-block), PNG, SVG
- Centered logo overlay (PNG and SVG output) from any image format: PNG, JPEG,
  GIF, WebP, or SVG
- Content helpers for Wi-Fi, contacts, calendar events, geo, tel, SMS, and email

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

PNG options: `Writer(io.Writer)`, `Filename(string)`, `Width(int)`,
`Height(int)`, `Quiet(int)`, `Dark(color.Color)`, `White(color.Color)`,
`Logo(image.Image)`, `LogoModules(int)`. If no writer is set, it writes to the
file named by `Filename` (default `image.png`).

#### Logo overlay

Embed a centered logo with `Logo`. Its default span is a conservative
recommendation derived from the EC level (see `Code.MaxLogoModules`) and can
be lowered with `LogoModules`:

```go
builder.SetErrorCorrectionLevel(qr.CorrectionLevelHigh). // recommended with a logo
	SetRenderer(png.New().Logo(myLogo))               // conservative default span
// or a smaller span:
png.New().Logo(myLogo).LogoModules(5)
```

The logo hides modules, so error correction must recover them. Recommended
spans are `High` → size/4, `Quartile` → size/5, `Medium` → size/6, `Low` →
size/7. Explicit spans are capped to that recommendation and reported through
the renderer-local `WarningHandler`. These are conservative guidelines, not a
scanability guarantee; test the finished code with the scanners and print
conditions you support.

The SVG renderer supports the same `Logo` / `LogoModules` API, embedding the
image as a base64 data URI:

```go
png.New().Logo(myLogo).LogoModules(11)  // raster output
svg.New().Logo(myLogo)                  // vector output, default span
```

Both renderers take the logo as an `image.Image`. To load one from a file of any
supported formats: PNG, JPEG, GIF, WebP, or **SVG**. Use the `logo` package; it
detects the format and rasterizes SVG for you:

```go
import "github.com/nachop51/qr-go/logo"

f, _ := os.Open("brand.svg") // or .png / .jpg / .webp / .gif
myLogo, err := logo.Decode(f)
if err != nil {
	log.Fatal(err)
}
png.New().Logo(myLogo)
```

### SVG

```go
import "github.com/nachop51/qr-go/render/svg"

builder.SetRenderer(svg.New().Writer(f).Module(8).Dark("#111111").Light("#ffffff"))
```

SVG options: `Writer(io.Writer)`, `Module(int)` (module size in px),
`Quiet(int)`, `Dark(string)`, `Light(string)`, `Logo(image.Image)`,
`LogoModules(int)`. Defaults to `os.Stdout`.

SVG color options use one strict grammar shared with PNG: CSS named colors,
`transparent`, 3/4/6/8-digit hex, and comma-form `rgb`/`rgba`/`hsl`/`hsla`.
CSS variables, URLs, and paint servers are rejected. Colors are canonicalized
before XML emission.

### Terminal (default)

```go
import "github.com/nachop51/qr-go/render/terminal"

builder.SetRenderer(terminal.New().Invert())
```

The terminal renderer defaults to a compact **half-block** style (Unicode
`▀ ▄ █`), which packs two module rows into each character cell so every module
is roughly square. Options:

- `Writer(io.Writer)`, `Quiet(int)`: output sink and quiet-zone size.
- `Invert()`: swap dark/light. Use this on a dark-background terminal so the
  code renders with the correct contrast for scanning.
- `Block()`: the classic full-width block style (two cells per module).
- `Dark(string)`, `Light(string)`: custom fill strings; these imply `Block()`.

Defaults to `os.Stdout`.

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

## Content helpers

The `content` package builds the specially formatted payloads that scanners
turn into actions: connect to Wi-Fi, save a contact, add a calendar event.
Each helper returns a plain string for `NewTextBuilder`:

```go
import (
	qr "github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/content"
)

// Join a Wi-Fi network on scan.
code, _ := qr.NewTextBuilder(
	content.WiFi{SSID: "CoffeeShop", Pass: "latte123"}.String(),
).Build()

// A contact card.
content.VCard{First: "Jane", Last: "Doe", Email: "jane@acme.test"}.String()

// A calendar event.
content.Event{Summary: "Launch", Start: start, End: end}.String()

// One-liners.
content.Tel("+15551234567")
content.SMS("+15551234567", "call me")
content.Geo(48.8584, 2.2945)
content.Email("a@b.test", "Subject", "Body")
content.URL("https://example.com")
```

`WiFi`, `VCard`, and `Event` implement `fmt.Stringer`; the rest are functions.
Reserved characters are escaped for you.

## Inspecting the result

`qr.Code` exposes an immutable view of the encoding outcome:

```go
code.Version()         // 1–40
code.Mask()            // 0–7
code.CorrectionLevel() // the level used
code.UsesECI()         // whether ECI was applied
code.Segments()        // a copy of the chosen segmentation
code.Size()            // modules per side
code.IsDark(x, y)      // false outside the matrix
```

## Command-line tool

`qrgo` exposes the whole library from the terminal. Install it, or run it from
the repo:

```sh
go install github.com/nachop51/qr-go/cmd/qrgo@latest
# or, in a checkout:
go run ./cmd/qrgo "HELLO WORLD"
```

With no subcommand the arguments are encoded as plain text and printed to the
terminal. Structured content types are subcommands. Most take their value as a
positional argument, while multi-field types (`wifi`, `vcard`, `event`) use
flags:

```sh
qrgo "HELLO WORLD"                                   # terminal (default)
qrgo url https://example.com -o code.png             # PNG (format from extension)
qrgo tel +15551234567 -f svg > call.svg              # SVG to stdout
qrgo geo 48.8584 2.2945                              # two positionals
qrgo wifi --ssid CoffeeShop --pass latte123 --ecc H -o wifi.svg
qrgo vcard --name "Jane Doe" --email jane@acme.test -o card.png
qrgo event --summary Launch --start "2026-07-05 14:30" -f svg > event.svg
echo "https://example.com" | qrgo -o code.png        # content from stdin
```

The output format is inferred from `-o`'s extension (`.png`/`.svg`) or set with
`-f/--format`; with no `-o` the code goes to stdout.

Render/output flags apply to every command: `-e/--ecc {L,M,Q,H}`,
`-q/--quiet <modules>`, `--version {0,1..40}`, `--mask {-1,0..7}`,
`--dark`/`--light` validated CSS colors, `--size`/`--width`/`--height` (PNG px) and `--scale` (SVG module px),
`--logo <file>` + `--logo-modules` for a centered logo (PNG/SVG),
`--invert`/`--block` for the terminal, `--no-eci`, and `-i/--info` to print the
encoding outcome (version, mask, segments) to stderr. File output is rendered
to a same-directory temporary file and atomically replaced only after success.
Run `qrgo --help` for the
overview, `qrgo help <type>` (e.g. `qrgo help wifi`) for a type's own options,
and `qrgo completion <shell>` for shell completion.
Use `qrgo build-version` to print the installed program release; `--version`
is reserved for forcing the QR symbol version in v0.2.

Renderer limits are: quiet zone 0–256 modules, PNG edges up to 8192 and 64
megapixels, SVG module size 1–1024, and logos up to 16 MiB encoded, 4096 pixels
per source edge, and 16 megapixels. The browser caps PNG export at 4096².
Zone-less CLI event date-times use the machine local timezone; explicit
RFC3339 offsets remain authoritative. vCard and iCalendar output uses CRLF and
75-octet UTF-8-safe line folding.

## Project layout

- package root (`*.go`): QR encoding library and public API
- `content/`: payload helpers (Wi-Fi, vCard, calendar, geo, tel, SMS, email)
- `logo/`: decode a logo from any image format (PNG/JPEG/GIF/WebP/SVG)
- `internal/spec`: QR spec tables (capacity, ECC, alignment, format info)
- `internal/coding`: bit stream and Reed–Solomon error correction
- `internal/matrix`: the module grid
- `render/`: the renderer contract and the terminal / PNG / SVG renderers
- `cmd/qrgo/`: the `qrgo` command-line tool

## License

[MIT](LICENSE)

## Trademark notice

QR Code is a registered trademark of DENSO WAVE INCORPORATED. This project is
not affiliated with or endorsed by DENSO WAVE; the term is used descriptively
to refer to the symbology defined in ISO/IEC 18004.
