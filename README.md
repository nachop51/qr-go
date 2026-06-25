# qr-go

`qr-go` is a Go QR code generator that builds PNG QR images from text or binary data.

The reusable package lives in `qr/` and is imported as:

```go
import qr "nachop51/qr/qr"
```

![Generated QR example](image.png)

## Features

- Build QR codes from:
  - text via `NewTextQrBuilder`
  - binary data via `NewBinaryQrBuilder`
- Automatic segmentation across:
  - numeric
  - alphanumeric
  - byte
  - kanji
- Automatic version detection
- Automatic mask selection
- Supported error correction levels:
  - `QrCorrectionLevelLow`
  - `QrCorrectionLevelMedium`
  - `QrCorrectionLevelQuartile`
  - `QrCorrectionLevelHigh`
- Optional automatic ECI for text that needs non-ASCII byte segments
  - disable it with `SetDisableECI(true)`
- PNG output via `Draw()` + `Save()`
- Configurable:
  - width
  - height
  - filename
  - black color
  - white color

## Project layout

- `main.go` — small local example/playground
- `qr/` — QR encoding and rendering library
- `image.png` — sample generated output

## Usage

### Text example

`Build()` returns `(*QrObject, error)`. The resulting `QrObject` exposes fields such as `Version`, `Mask`, `Segments`, `ErrorCorrectionLevel`, and `Filename`.

```go
package main

import (
	"fmt"
	"log"

	qr "nachop51/qr/qr"
)

func main() {
	code, err := qr.NewTextQrBuilder("ABC日本123").
		SetFilename("text.png").
		SetErrorCorrectionLevel(qr.QrCorrectionLevelMedium).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("version=%d mask=%d segments=%d\n", code.Version, code.Mask, len(code.Segments))

	code.Draw()
	if err := code.Save(); err != nil {
		log.Fatal(err)
	}
}
```

For text input, the builder validates UTF-8. If the text requires non-ASCII byte segments, ECI is enabled automatically unless you call:

```go
builder.SetDisableECI(true)
```

### Binary example

Use `NewBinaryQrBuilder` when you want to encode raw bytes directly.

```go
package main

import (
	"log"

	qr "nachop51/qr/qr"
)

func main() {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x01, 0x02}

	code, err := qr.NewBinaryQrBuilder(payload).
		SetFilename("binary.png").
		SetErrorCorrectionLevel(qr.QrCorrectionLevelHigh).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	code.Draw()
	if err := code.Save(); err != nil {
		log.Fatal(err)
	}
}
```

## Builder API

Current exported builder methods:

- `SetWidth(int)`
- `SetHeight(int)`
- `SetFilename(string)`
- `SetBlackColor(color.Color)`
- `SetWhiteColor(color.Color)`
- `SetErrorCorrectionLevel(QrCorrectionLevel)`
- `SetDisableECI(bool)`
- `Build() (*QrObject, error)`

### Default builder values

New builders currently default to:

- width: `400`
- height: `400`
- filename: `image.png`
- error correction level: `QrCorrectionLevelMedium`
- black color: `color.Black`
- white color: `color.White`

## Built object

After a successful build, `QrObject` provides:

### Fields

- `Version`
- `Mask`
- `Segments`
- `ErrorCorrectionLevel`
- `Filename`

### Methods

- `Draw()`
- `Save() error`
- `Debug()`

`Draw()` renders the QR code into the internal image buffer, and `Save()` writes that image as a PNG file.

## Running the local example

The repository includes a small `main.go` playground. From the project root:

```bash
go run .
```

That will generate an output image using the current example in `main.go`.

## Notes and limitations

- This project is still under active development.
- There are currently **no automated tests** in the repository.
- PNG is the current output format exposed by the library.
- Text input must be valid UTF-8.
- For arbitrary raw bytes, use `NewBinaryQrBuilder`.
- `Build()` can fail for:
  - invalid dimensions
  - invalid UTF-8 text
  - data that does not fit the selected error correction level

## Roadmap

Likely next improvements:

- add automated tests
- expand API documentation and examples
- provide a small CLI for generating QR codes from the command line
