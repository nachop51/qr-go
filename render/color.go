package render

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/image/colornames"
)

// CSSColor is a validated CSS color in both raster and canonical SVG forms.
type CSSColor struct {
	RGBA color.NRGBA
	CSS  string
}

// ParseCSSColor accepts named CSS colors, transparent, #rgb[a], #rrggbb[aa],
// and the comma forms of rgb[a] and hsl[a]. It deliberately excludes every
// extensible CSS construct (including var() and url()).
func ParseCSSColor(input string) (CSSColor, error) {
	s := strings.TrimSpace(input)
	if s == "" || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return CSSColor{}, fmt.Errorf("invalid CSS color %q", input)
	}
	lower := strings.ToLower(s)
	if lower == "transparent" {
		return makeCSSColor(0, 0, 0, 0), nil
	}
	if c, ok := colornames.Map[lower]; ok {
		return makeCSSColor(c.R, c.G, c.B, 255), nil
	}
	if strings.HasPrefix(lower, "#") {
		return parseHexCSS(lower, input)
	}
	for _, fn := range []string{"rgb", "rgba", "hsl", "hsla"} {
		prefix := fn + "("
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ")") {
			return parseFunctionalCSS(fn, lower[len(prefix):len(lower)-1], input)
		}
	}
	return CSSColor{}, fmt.Errorf("invalid CSS color %q", input)
}

func makeCSSColor(r, g, b, a uint8) CSSColor {
	css := fmt.Sprintf("#%02x%02x%02x", r, g, b)
	if a != 255 {
		css += fmt.Sprintf("%02x", a)
	}
	return CSSColor{RGBA: color.NRGBA{R: r, G: g, B: b, A: a}, CSS: css}
}

func parseHexCSS(s, original string) (CSSColor, error) {
	h := s[1:]
	if len(h) != 3 && len(h) != 4 && len(h) != 6 && len(h) != 8 {
		return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
	}
	if len(h) <= 4 {
		var b strings.Builder
		for _, r := range h {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		h = b.String()
	}
	vals := [4]uint8{0, 0, 0, 255}
	for i := 0; i < len(h)/2; i++ {
		v, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
		}
		vals[i] = uint8(v)
	}
	return makeCSSColor(vals[0], vals[1], vals[2], vals[3]), nil
}

func parseFunctionalCSS(fn, body, original string) (CSSColor, error) {
	parts := strings.Split(body, ",")
	want := 3
	if fn == "rgba" || fn == "hsla" {
		want = 4
	}
	if len(parts) != want {
		return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
		}
	}
	a := uint8(255)
	if want == 4 {
		var err error
		a, err = parseAlpha(parts[3])
		if err != nil {
			return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
		}
	}
	if fn == "rgb" || fn == "rgba" {
		var rgb [3]uint8
		for i := range 3 {
			v, err := parseRGBComponent(parts[i])
			if err != nil {
				return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
			}
			rgb[i] = v
		}
		return makeCSSColor(rgb[0], rgb[1], rgb[2], a), nil
	}
	h, err := parseHue(parts[0])
	if err != nil || math.IsNaN(h) || math.IsInf(h, 0) {
		return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
	}
	sat, err1 := parsePercent(parts[1])
	light, err2 := parsePercent(parts[2])
	if err1 != nil || err2 != nil {
		return CSSColor{}, fmt.Errorf("invalid CSS color %q", original)
	}
	r, g, b := hslToRGB(h, sat, light)
	return makeCSSColor(r, g, b, a), nil
}

func parseHue(s string) (float64, error) {
	factor := 1.0
	switch {
	case strings.HasSuffix(s, "deg"):
		s = strings.TrimSpace(strings.TrimSuffix(s, "deg"))
	case strings.HasSuffix(s, "grad"):
		s, factor = strings.TrimSpace(strings.TrimSuffix(s, "grad")), 0.9
	case strings.HasSuffix(s, "rad"):
		s, factor = strings.TrimSpace(strings.TrimSuffix(s, "rad")), 180/math.Pi
	case strings.HasSuffix(s, "turn"):
		s, factor = strings.TrimSpace(strings.TrimSuffix(s, "turn")), 360
	}
	v, err := strconv.ParseFloat(s, 64)
	return v * factor, err
}

func parseRGBComponent(s string) (uint8, error) {
	if strings.HasSuffix(s, "%") {
		v, err := parsePercent(s)
		return uint8(math.Round(v * 255)), err
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 255 {
		return 0, fmt.Errorf("out of range")
	}
	return uint8(math.Round(v)), nil
}

func parseAlpha(s string) (uint8, error) {
	if strings.HasSuffix(s, "%") {
		v, err := parsePercent(s)
		return uint8(math.Round(v * 255)), err
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
		return 0, fmt.Errorf("out of range")
	}
	return uint8(math.Round(v * 255)), nil
}

func parsePercent(s string) (float64, error) {
	if !strings.HasSuffix(s, "%") {
		return 0, fmt.Errorf("percentage required")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, "%")), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 100 {
		return 0, fmt.Errorf("out of range")
	}
	return v / 100, nil
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch int(h / 60) {
	case 0:
		r, g = c, x
	case 1:
		r, g = x, c
	case 2:
		g, b = c, x
	case 3:
		g, b = x, c
	case 4:
		r, b = x, c
	default:
		r, b = c, x
	}
	return uint8(math.Round((r + m) * 255)), uint8(math.Round((g + m) * 255)), uint8(math.Round((b + m) * 255))
}
