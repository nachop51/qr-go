package main

import (
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	qr "github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/logo"
	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/png"
	"github.com/nachop51/qr-go/render/svg"
	"github.com/nachop51/qr-go/render/terminal"
)

// resolveFormat picks the output format. An explicit -f/--format wins; otherwise
// it is inferred from the -o extension, defaulting to terminal.
func resolveFormat(flag, output string) (string, error) {
	if flag != "" {
		switch strings.ToLower(strings.TrimSpace(flag)) {
		case "terminal", "term", "ansi", "stdout":
			return "terminal", nil
		case "png":
			return "png", nil
		case "svg":
			return "svg", nil
		default:
			return "", fmt.Errorf("invalid format %q (want terminal, png, or svg)", flag)
		}
	}
	switch strings.ToLower(filepath.Ext(output)) {
	case ".png":
		return "png", nil
	case ".svg":
		return "svg", nil
	default:
		return "terminal", nil
	}
}

// buildRenderer constructs the configured renderer for the given format, writing
// to w.
func buildRenderer(format string, o *options, w io.Writer) (render.Renderer, error) {
	switch format {
	case "terminal":
		if o.logo != "" {
			return nil, fmt.Errorf("the terminal renderer has no logo support; use -f png or -f svg")
		}
		t := terminal.New().Writer(w)
		if o.invert {
			t = t.Invert()
		}
		if o.block {
			t = t.Block()
		}
		if o.quiet >= 0 {
			t = t.Quiet(o.quiet)
		}
		return t, nil

	case "png":
		dark, err := parseHexColor(o.dark)
		if err != nil {
			return nil, fmt.Errorf("--dark: %w", err)
		}
		light, err := parseHexColor(o.light)
		if err != nil {
			return nil, fmt.Errorf("--light: %w", err)
		}
		width, height := o.width, o.height
		if o.size > 0 {
			width, height = o.size, o.size
		}
		p := png.New().Writer(w).Dark(dark).White(light).Width(width).Height(height)
		if o.quiet >= 0 {
			p = p.Quiet(o.quiet)
		}
		if o.logo != "" {
			img, err := loadLogo(o.logo)
			if err != nil {
				return nil, err
			}
			p = p.Logo(img).LogoModules(o.logoModules)
		}
		return p, nil

	case "svg":
		s := svg.New().Writer(w).Dark(o.dark).Light(o.light).Module(o.scale)
		if o.quiet >= 0 {
			s = s.Quiet(o.quiet)
		}
		if o.logo != "" {
			img, err := loadLogo(o.logo)
			if err != nil {
				return nil, err
			}
			s = s.Logo(img).LogoModules(o.logoModules)
		}
		return s, nil

	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

// loadLogo decodes a logo image file (PNG, JPEG, GIF, WebP, or SVG).
func loadLogo(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("--logo: %w", err)
	}
	defer f.Close()
	img, err := logo.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("--logo: %w", err)
	}
	return img, nil
}

// parseECC maps a level letter (or full name) to the library's CorrectionLevel.
func parseECC(s string) (qr.CorrectionLevel, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "L", "LOW":
		return qr.CorrectionLevelLow, nil
	case "", "M", "MEDIUM":
		return qr.CorrectionLevelMedium, nil
	case "Q", "QUARTILE":
		return qr.CorrectionLevelQuartile, nil
	case "H", "HIGH":
		return qr.CorrectionLevelHigh, nil
	default:
		return qr.CorrectionLevel{}, fmt.Errorf("invalid error-correction level %q (want L, M, Q, or H)", s)
	}
}

// eccName is the inverse of parseECC, for the --info dump.
func eccName(l qr.CorrectionLevel) string {
	switch l {
	case qr.CorrectionLevelLow:
		return "L"
	case qr.CorrectionLevelMedium:
		return "M"
	case qr.CorrectionLevelQuartile:
		return "Q"
	case qr.CorrectionLevelHigh:
		return "H"
	default:
		return "?"
	}
}

// parseHexColor parses #rgb, #rrggbb, or #rrggbbaa (the leading # is optional).
func parseHexColor(s string) (color.Color, error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	component := func(sub string) (uint8, error) {
		v, err := strconv.ParseUint(sub, 16, 8)
		return uint8(v), err
	}

	c := color.RGBA{A: 0xff}
	var err error
	switch len(h) {
	case 3: // #rgb -> #rrggbb
		if c.R, err = component(string([]byte{h[0], h[0]})); err == nil {
			if c.G, err = component(string([]byte{h[1], h[1]})); err == nil {
				c.B, err = component(string([]byte{h[2], h[2]}))
			}
		}
	case 6:
		if c.R, err = component(h[0:2]); err == nil {
			if c.G, err = component(h[2:4]); err == nil {
				c.B, err = component(h[4:6])
			}
		}
	case 8:
		if c.R, err = component(h[0:2]); err == nil {
			if c.G, err = component(h[2:4]); err == nil {
				if c.B, err = component(h[4:6]); err == nil {
					c.A, err = component(h[6:8])
				}
			}
		}
	default:
		return nil, fmt.Errorf("invalid color %q (use #rgb, #rrggbb, or #rrggbbaa)", s)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid color %q (use #rgb, #rrggbb, or #rrggbbaa)", s)
	}
	return c, nil
}

// printInfo writes the encoding outcome for -i/--info.
func printInfo(w io.Writer, code *qr.Code) {
	fmt.Fprintf(w, "version=%d mask=%d ecc=%s size=%d eci=%v\n",
		code.Version, code.Mask, eccName(code.ErrorCorrectionLevel), code.Size(), code.IsECI)
	for i, s := range code.Segments {
		fmt.Fprintf(w, "segment[%d] mode=%s len=%d data=%q\n", i, s.Mode(), len(s.Data()), s.Data())
	}
}
