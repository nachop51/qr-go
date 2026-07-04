package main

import (
	"fmt"
	"os"

	"nachop51/qr"
	"nachop51/qr/render/png"
)

func main() {
	text := "HELLO WORLD"
	if len(os.Args) > 1 {
		text = os.Args[1]
	}

	code, err := qr.NewTextQrBuilder(text).
		SetErrorCorrectionLevel(qr.QrCorrectionLevelHigh).
		SetRenderer(png.New()).
		Build()
	if err != nil {
		fmt.Fprintln(os.Stderr, "build:", err)
		os.Exit(1)
	}

	for _, s := range code.Segments {
		fmt.Printf("data: '%s', mode: '%v'\n", s.Data(), s.Mode())
	}

	if err := code.Render(); err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}

	fmt.Printf("version=%d mask=%d size=%d data=%q\n", code.Version, code.Mask, code.Size(), text)
}
