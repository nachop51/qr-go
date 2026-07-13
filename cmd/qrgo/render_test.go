package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	qr "github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/render/png"
	"github.com/nachop51/qr-go/render/style"
	"github.com/nachop51/qr-go/render/svg"
	"github.com/nachop51/qr-go/render/terminal"
)

func TestResolveFormat(t *testing.T) {
	cases := []struct {
		name    string
		flag    string
		output  string
		want    string
		wantErr bool
	}{
		{name: "default terminal", want: "terminal"},
		{name: "infer png", output: "out.png", want: "png"},
		{name: "infer svg", output: "diagram.SVG", want: "svg"},
		{name: "unknown ext is terminal", output: "out.txt", want: "terminal"},
		{name: "flag wins over ext", flag: "svg", output: "out.png", want: "svg"},
		{name: "flag alias term", flag: "term", want: "terminal"},
		{name: "invalid flag", flag: "pdf", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveFormat(tc.flag, tc.output)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %q got %q", tc.want, got)
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	cases := []struct {
		in      string
		want    color.RGBA
		wantErr bool
	}{
		{in: "#000000", want: color.RGBA{A: 0xff}},
		{in: "#ffffff", want: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
		{in: "ff0000", wantErr: true},
		{in: "#0f0", want: color.RGBA{G: 0xff, A: 0xff}},
		{in: "#11223344", want: color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x44}},
		{in: "red", want: color.RGBA{R: 0xff, A: 0xff}},
		{in: "#12345", wantErr: true},
		{in: "#gggggg", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseHexColor(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error for %q, got %v", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %+v got %+v", tc.want, got)
			}
		})
	}
}

func TestParseECCRoundTrip(t *testing.T) {
	for _, in := range []string{"L", "m", "Q", "high"} {
		level, err := parseECC(in)
		if err != nil {
			t.Fatalf("parseECC(%q): %v", in, err)
		}
		if name := eccName(level); name == "?" {
			t.Fatalf("eccName did not recognise the level from %q", in)
		}
	}
	if _, err := parseECC("Z"); err == nil {
		t.Fatalf("want error for invalid level")
	}
}

func TestBuildRendererTypes(t *testing.T) {
	var buf bytes.Buffer
	base := options{dark: "#000000", light: "#ffffff", width: 800, height: 800, scale: 10, quiet: -1}

	term, err := buildRenderer("terminal", &base, &buf)
	if err != nil {
		t.Fatalf("terminal: %v", err)
	}
	if _, ok := term.(terminal.Terminal); !ok {
		t.Fatalf("terminal: got %T", term)
	}

	p, err := buildRenderer("png", &base, &buf)
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	if _, ok := p.(png.PNG); !ok {
		t.Fatalf("png: got %T", p)
	}

	s, err := buildRenderer("svg", &base, &buf)
	if err != nil {
		t.Fatalf("svg: %v", err)
	}
	if _, ok := s.(svg.SVG); !ok {
		t.Fatalf("svg: got %T", s)
	}
}

func TestBuildRendererTerminalRejectsLogo(t *testing.T) {
	var buf bytes.Buffer
	o := options{logo: "logo.png", quiet: -1}
	if _, err := buildRenderer("terminal", &o, &buf); err == nil {
		t.Fatalf("want error: terminal renderer cannot take a logo")
	}
}

func TestBuildRendererTerminalRejectsStyle(t *testing.T) {
	var buf bytes.Buffer
	o := options{eyeShape: "circle", quiet: -1}
	if _, err := buildRenderer("terminal", &o, &buf); err == nil {
		t.Fatalf("want error: terminal renderer cannot take style flags")
	}
}

func TestParseStyle(t *testing.T) {
	// --eye-shape sets both; the specific flags override individually.
	o := options{moduleShape: "dot", eyeShape: "circle", eyeBallShape: "rounded",
		gradient: "linear:#0f172a:#7f1d1d:45"}
	st, err := parseStyle(&o)
	if err != nil {
		t.Fatal(err)
	}
	if st.module != style.ModuleDot || st.frame != style.EyeCircle || st.ball != style.EyeRounded {
		t.Fatalf("shapes = %v/%v/%v", st.module, st.frame, st.ball)
	}
	if st.gradKind != style.GradientLinear || st.gradFrom != "#0f172a" || st.gradTo != "#7f1d1d" || st.gradAngle != 45 {
		t.Fatalf("gradient = %+v", st)
	}

	for _, bad := range []options{
		{shape: "blob"},
		{moduleShape: "blob"},
		{eyeShape: "blob"},
		{eyeFrameShape: "blob"},
		{gradient: "linear:#000"},          // missing <to>
		{gradient: "conic:#000:#111"},      // unknown kind
		{gradient: "radial:#000:#111:45"},  // radial takes no angle
		{gradient: "linear:#000:#111:top"}, // non-numeric angle
	} {
		if _, err := parseStyle(&bad); err == nil {
			t.Errorf("expected error for %+v", bad)
		}
	}
}

func TestParseStyleSharedShape(t *testing.T) {
	tests := []struct {
		name   string
		o      options
		module style.ModuleShape
		frame  style.EyeShape
		ball   style.EyeShape
	}{
		{"rounded everywhere", options{shape: "rounded"},
			style.ModuleRounded, style.EyeRounded, style.EyeRounded},
		{"circle maps to dot modules", options{shape: "circle"},
			style.ModuleDot, style.EyeCircle, style.EyeCircle},
		{"dot is accepted for eyes too", options{shape: "dot"},
			style.ModuleDot, style.EyeCircle, style.EyeCircle},
		{"specific flags override", options{shape: "circle", moduleShape: "square", eyeBallShape: "rounded"},
			style.ModuleSquare, style.EyeCircle, style.EyeRounded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := parseStyle(&tt.o)
			if err != nil {
				t.Fatal(err)
			}
			if st.module != tt.module || st.frame != tt.frame || st.ball != tt.ball {
				t.Fatalf("shapes = %v/%v/%v, want %v/%v/%v",
					st.module, st.frame, st.ball, tt.module, tt.frame, tt.ball)
			}
		})
	}
}

// Styled flags survive the full CLI pipeline and still produce a decodable PNG.
func TestStyledPNGRoundTrip(t *testing.T) {
	const payload = "https://example.com/qrgo-styled"

	var buf bytes.Buffer
	o := options{dark: "#000000", light: "#ffffff", width: 600, height: 600, quiet: -1,
		moduleShape: "rounded", eyeShape: "circle", eyeFrame: "#1d4ed8", eyeBall: "#b91c1c"}
	renderer, err := buildRenderer("png", &o, &buf)
	if err != nil {
		t.Fatal(err)
	}
	code, err := qr.NewTextBuilder(payload).
		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
		SetRenderer(renderer).Build()
	if err != nil {
		t.Fatal(err)
	}
	if err := code.Render(); err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	bmp, _ := gozxing.NewBinaryBitmapFromImage(img)
	res, err := qrcode.NewQRCodeReader().Decode(bmp, nil)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res.GetText() != payload {
		t.Fatalf("decoded %q, want %q", res.GetText(), payload)
	}
}

func TestBuildRendererInvalidColor(t *testing.T) {
	var buf bytes.Buffer
	o := options{dark: "notacolor", light: "#ffffff", width: 800, height: 800, quiet: -1}
	if _, err := buildRenderer("png", &o, &buf); err == nil {
		t.Fatalf("want error for invalid --dark color")
	}
}

// TestPNGRoundTrip renders a URL through the CLI's own renderer pipeline and
// decodes the PNG back with gozxing, proving the wiring produces a scannable
// code.
func TestPNGRoundTrip(t *testing.T) {
	const payload = "https://example.com/qrgo"

	var buf bytes.Buffer
	o := options{dark: "#000000", light: "#ffffff", size: 512, width: 800, height: 800, scale: 10, quiet: -1}
	renderer, err := buildRenderer("png", &o, &buf)
	if err != nil {
		t.Fatalf("buildRenderer: %v", err)
	}

	code, err := qr.NewTextBuilder(payload).
		SetErrorCorrectionLevel(qr.CorrectionLevelHigh).
		SetRenderer(renderer).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if err := code.Render(); err != nil {
		t.Fatalf("render: %v", err)
	}

	if got := decodePNG(t, buf.Bytes()); got != payload {
		t.Fatalf("round-trip mismatch: want %q got %q", payload, got)
	}
}

func decodePNG(t *testing.T, data []byte) string {
	t.Helper()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		t.Fatalf("bitmap: %v", err)
	}
	hints := map[gozxing.DecodeHintType]any{
		gozxing.DecodeHintType_PURE_BARCODE: true,
		gozxing.DecodeHintType_TRY_HARDER:   true,
	}
	res, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		t.Fatalf("qr decode failed: %v", err)
	}
	return res.GetText()
}
