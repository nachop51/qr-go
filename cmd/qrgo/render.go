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
	"github.com/nachop51/qr-go/render/style"
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
	st, err := parseStyle(o)
	if err != nil {
		return nil, err
	}

	switch format {
	case "terminal":
		if o.logo != "" {
			return nil, fmt.Errorf("the terminal renderer has no logo support; use -f png or -f svg")
		}
		if o.styleConfigured() {
			return nil, fmt.Errorf("the terminal renderer has no style support; use -f png or -f svg")
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
		warn := o.warn
		if warn == nil {
			warn = render.StderrWarningHandler
		}
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
		p := png.New().Writer(w).WarningHandler(warn).Dark(dark).White(light).Width(width).Height(height)
		p = p.ModuleShape(st.module).EyeFrameShape(st.frame).EyeBallShape(st.ball)
		if o.eyeFrame != "" {
			c, err := parseHexColor(o.eyeFrame)
			if err != nil {
				return nil, fmt.Errorf("--eye-frame: %w", err)
			}
			p = p.EyeFrame(c)
		}
		if o.eyeBall != "" {
			c, err := parseHexColor(o.eyeBall)
			if err != nil {
				return nil, fmt.Errorf("--eye-ball: %w", err)
			}
			p = p.EyeBall(c)
		}
		if st.gradKind != style.GradientNone {
			from, err := parseHexColor(st.gradFrom)
			if err != nil {
				return nil, fmt.Errorf("--gradient: %w", err)
			}
			to, err := parseHexColor(st.gradTo)
			if err != nil {
				return nil, fmt.Errorf("--gradient: %w", err)
			}
			if st.gradKind == style.GradientRadial {
				p = p.GradientRadial(from, to)
			} else {
				p = p.GradientLinear(from, to, st.gradAngle)
			}
		}
		if o.quiet >= 0 {
			p = p.Quiet(o.quiet)
		}
		if o.logo != "" {
			img, err := loadLogo(o.logo)
			if err != nil {
				return nil, err
			}
			p = p.Logo(img).LogoModules(o.logoModules).LogoScale(o.logoScale)
		}
		return p, nil

	case "svg":
		warn := o.warn
		if warn == nil {
			warn = render.StderrWarningHandler
		}
		s := svg.New().Writer(w).WarningHandler(warn).Dark(o.dark).Light(o.light).Module(o.scale)
		s = s.ModuleShape(st.module).EyeFrameShape(st.frame).EyeBallShape(st.ball).
			EyeFrame(o.eyeFrame).EyeBall(o.eyeBall)
		if st.gradKind == style.GradientRadial {
			s = s.GradientRadial(st.gradFrom, st.gradTo)
		} else if st.gradKind == style.GradientLinear {
			s = s.GradientLinear(st.gradFrom, st.gradTo, st.gradAngle)
		}
		if o.quiet >= 0 {
			s = s.Quiet(o.quiet)
		}
		if o.logo != "" {
			img, err := loadLogo(o.logo)
			if err != nil {
				return nil, err
			}
			s = s.Logo(img).LogoModules(o.logoModules).LogoScale(o.logoScale)
		}
		return s, nil

	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

// styleOpts holds the parsed style flags in renderer-neutral form; colors stay
// raw strings so each renderer converts with its own rules.
type styleOpts struct {
	module           style.ModuleShape
	frame, ball      style.EyeShape
	gradKind         style.GradientKind
	gradFrom, gradTo string
	gradAngle        float64
}

func parseStyle(o *options) (styleOpts, error) {
	var s styleOpts

	// --shape is a shared default for modules and eyes; the specific flags
	// override it.
	moduleShape, eyeShape := o.moduleShape, o.eyeShape
	if o.shape != "" {
		m, e, err := splitSharedShape(o.shape)
		if err != nil {
			return s, err
		}
		if moduleShape == "" {
			moduleShape = m
		}
		if eyeShape == "" {
			eyeShape = e
		}
	}

	var err error
	if s.module, err = style.ParseModuleShape(moduleShape); err != nil {
		return s, fmt.Errorf("--module-shape: %w", err)
	}
	base, err := style.ParseEyeShape(eyeShape)
	if err != nil {
		return s, fmt.Errorf("--eye-shape: %w", err)
	}
	s.frame, s.ball = base, base
	if o.eyeFrameShape != "" {
		if s.frame, err = style.ParseEyeShape(o.eyeFrameShape); err != nil {
			return s, fmt.Errorf("--eye-frame-shape: %w", err)
		}
	}
	if o.eyeBallShape != "" {
		if s.ball, err = style.ParseEyeShape(o.eyeBallShape); err != nil {
			return s, fmt.Errorf("--eye-ball-shape: %w", err)
		}
	}
	if o.gradient != "" {
		if err := parseGradientSpec(o.gradient, &s); err != nil {
			return s, err
		}
	}
	return s, nil
}

// splitSharedShape maps a --shape value to its module and eye spellings: the
// round shape is called "dot" on modules and "circle" on eyes, and --shape
// accepts either name.
func splitSharedShape(shape string) (module, eye string, err error) {
	switch shape {
	case "square", "rounded":
		return shape, shape, nil
	case "circle", "dot":
		return "dot", "circle", nil
	}
	return "", "", fmt.Errorf("--shape: unknown shape %q (want square, rounded, or circle)", shape)
}

// parseGradientSpec parses linear:<from>:<to>[:angle] or radial:<from>:<to>.
func parseGradientSpec(spec string, s *styleOpts) error {
	parts := strings.Split(spec, ":")
	usage := fmt.Errorf("--gradient: invalid spec %q (use linear:<from>:<to>[:angle] or radial:<from>:<to>)", spec)
	if len(parts) < 3 {
		return usage
	}
	kind, err := style.ParseGradientKind(parts[0])
	if err != nil || kind == style.GradientNone {
		return usage
	}
	s.gradKind, s.gradFrom, s.gradTo = kind, parts[1], parts[2]
	switch {
	case len(parts) == 3:
		return nil
	case len(parts) == 4 && kind == style.GradientLinear:
		if s.gradAngle, err = strconv.ParseFloat(parts[3], 64); err != nil {
			return usage
		}
		return nil
	}
	return usage
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
		return 0, fmt.Errorf("invalid error-correction level %q (want L, M, Q, or H)", s)
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

// parseHexColor retains its historical name but uses the shared strict CSS
// grammar so PNG and SVG accept exactly the same color inputs.
func parseHexColor(s string) (color.Color, error) {
	c, err := render.ParseCSSColor(s)
	if err != nil {
		return nil, err
	}
	return color.RGBA{R: c.RGBA.R, G: c.RGBA.G, B: c.RGBA.B, A: c.RGBA.A}, nil
}

// printInfo writes the encoding outcome for -i/--info.
func printInfo(w io.Writer, code *qr.Code) {
	fmt.Fprintf(w, "version=%d mask=%d ecc=%s size=%d eci=%v\n",
		code.Version(), code.Mask(), eccName(code.CorrectionLevel()), code.Size(), code.UsesECI())
	for i, s := range code.Segments() {
		fmt.Fprintf(w, "segment[%d] mode=%s len=%d data=%q\n", i, s.Mode(), len(s.Data()), s.Data())
	}
}
