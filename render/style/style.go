// Package style holds the shared vocabulary for styled QR rendering: module
// and eye shapes, the fixed geometry of the three finder eyes, and
// neighbour-aware corner classification. Renderers translate these into their
// own output (SVG path data, rasterized fills) through the Path sink in
// path.go, so the geometry lives in exactly one place.
//
// Zones are classified geometrically, not from the matrix: the three finder
// eyes are 7x7 squares at fixed corners for every version, so Size() and
// IsDark() are all a renderer needs.
package style

import (
	"fmt"
	"image/color"
	"math"

	"github.com/nachop51/qr-go/render"
)

type ModuleShape uint8

const (
	ModuleSquare ModuleShape = iota
	ModuleRounded
	ModuleDot
)

func (m ModuleShape) Valid() bool { return m <= ModuleDot }

func (m ModuleShape) String() string {
	switch m {
	case ModuleRounded:
		return "rounded"
	case ModuleDot:
		return "dot"
	default:
		return "square"
	}
}

func ParseModuleShape(s string) (ModuleShape, error) {
	switch s {
	case "", "square":
		return ModuleSquare, nil
	case "rounded":
		return ModuleRounded, nil
	case "dot":
		return ModuleDot, nil
	}
	return ModuleSquare, fmt.Errorf("unknown module shape %q (want square, rounded or dot)", s)
}

type EyeShape uint8

const (
	EyeSquare EyeShape = iota
	EyeRounded
	EyeCircle
)

func (e EyeShape) Valid() bool { return e <= EyeCircle }

func (e EyeShape) String() string {
	switch e {
	case EyeRounded:
		return "rounded"
	case EyeCircle:
		return "circle"
	default:
		return "square"
	}
}

func ParseEyeShape(s string) (EyeShape, error) {
	switch s {
	case "", "square":
		return EyeSquare, nil
	case "rounded":
		return EyeRounded, nil
	case "circle":
		return EyeCircle, nil
	}
	return EyeSquare, fmt.Errorf("unknown eye shape %q (want square, rounded or circle)", s)
}

type GradientKind uint8

const (
	GradientNone GradientKind = iota
	GradientLinear
	GradientRadial
)

func (g GradientKind) Valid() bool { return g <= GradientRadial }

func ParseGradientKind(s string) (GradientKind, error) {
	switch s {
	case "", "none":
		return GradientNone, nil
	case "linear":
		return GradientLinear, nil
	case "radial":
		return GradientRadial, nil
	}
	return GradientNone, fmt.Errorf("unknown gradient kind %q (want linear or radial)", s)
}

// Rect is a rectangle in whole-module units.
type Rect struct{ X, Y, W, H int }

// EyeRects returns the three 7x7 finder-eye rectangles: top-left, top-right,
// bottom-left. Their positions are fixed by the QR spec for every version.
func EyeRects(gridSize int) [3]Rect {
	return [3]Rect{
		{0, 0, 7, 7},
		{gridSize - 7, 0, 7, 7},
		{0, gridSize - 7, 7, 7},
	}
}

// InEye reports whether the module at (x, y) belongs to one of the three
// finder eyes.
func InEye(x, y, gridSize int) bool {
	return (x < 7 && y < 7) ||
		(x >= gridSize-7 && y < 7) ||
		(x < 7 && y >= gridSize-7)
}

// Corners is a bitmask of a module's exposed corners: a corner is exposed
// when neither orthogonal neighbour on that side is dark, so rounding it
// cannot open a gap against an adjacent module.
type Corners uint8

const (
	CornerTL Corners = 1 << iota
	CornerTR
	CornerBR
	CornerBL
)

// CornerMask classifies the module at (x, y). Out-of-bounds neighbours count
// as light.
func CornerMask(g render.Grid, x, y int) Corners {
	dark := func(x, y int) bool {
		return x >= 0 && y >= 0 && x < g.Size() && y < g.Size() && g.IsDark(x, y)
	}
	var c Corners
	up, down := dark(x, y-1), dark(x, y+1)
	left, right := dark(x-1, y), dark(x+1, y)
	if !up && !left {
		c |= CornerTL
	}
	if !up && !right {
		c |= CornerTR
	}
	if !down && !right {
		c |= CornerBR
	}
	if !down && !left {
		c |= CornerBL
	}
	return c
}

// WarnContrast emits a renderer-local warning when fg is too close in
// luminance to bg for reliable scanning. The threshold (2:1 WCAG-style
// contrast ratio) is deliberately loose: it flags decorative choices that
// will genuinely break decoding, not tasteful low-contrast palettes.
func WarnContrast(warn render.WarningHandler, name string, fg, bg color.Color) {
	if warn != nil && contrastRatio(fg, bg) < 2 {
		warn("%s has low contrast against the background; the code may not scan", name)
	}
}

// ParseHex parses #rgb and #rrggbb colors. It exists so renderers whose
// colors are CSS strings (SVG) can still run contrast checks; anything that
// is not simple hex reports ok=false and should be skipped silently.
func ParseHex(s string) (color.Color, bool) {
	if len(s) == 0 || s[0] != '#' {
		return nil, false
	}
	hex := s[1:]
	digit := func(b byte) (uint8, bool) {
		switch {
		case b >= '0' && b <= '9':
			return b - '0', true
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10, true
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10, true
		}
		return 0, false
	}
	var v [6]uint8
	switch len(hex) {
	case 3:
		for i := range 3 {
			d, ok := digit(hex[i])
			if !ok {
				return nil, false
			}
			v[2*i], v[2*i+1] = d, d
		}
	case 6:
		for i := range 6 {
			d, ok := digit(hex[i])
			if !ok {
				return nil, false
			}
			v[i] = d
		}
	default:
		return nil, false
	}
	return color.RGBA{v[0]<<4 | v[1], v[2]<<4 | v[3], v[4]<<4 | v[5], 0xff}, true
}

func contrastRatio(a, b color.Color) float64 {
	la, lb := relLuminance(a), relLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		s := float64(v) / 0xffff
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}
