package main

import (
	"fmt"
	"os"

	"github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/render/png"
)

func main() {
	text := "HELLO WORLD"
	if len(os.Args) > 1 {
		text = os.Args[1]
	}

	code, err := qr.NewTextBuilder(text).
		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
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
