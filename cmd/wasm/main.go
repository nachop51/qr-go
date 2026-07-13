//go:build js && wasm

// Command wasm exposes the qr-go library to JavaScript as a single `qrgo`
// global:
//
//	qrgo.generate(opts) -> {data, size, version, mask, maxLogoModules, warnings} | {error}
//	qrgo.content.{wifi,vcard,event,url,tel,sms,geo,email}(...) -> string
//
// generate opts (all optional except text):
//
//	{
//	  text:        string,
//	  ecLevel:     "L" | "M" | "Q" | "H",          // default "M"
//	  eciPolicy:   "auto" | "disabled",             // default "auto"
//	  format:      "png" | "svg",                   // default "png"
//	  dark:        "#rrggbb",                       // module color
//	  light:       "#rrggbb",                       // background color
//	  quiet:       int,                             // quiet zone in modules
//	  size:        int,                             // png output px (square)
//	  moduleSize:  int,                             // svg px per module
//	  logo:        Uint8Array,                      // png or jpeg bytes
//	  logoModules: int,                             // logo span in modules; omit or 0 = max the EC level allows
//	  logoScale:   int,                             // % of the logo area the image fills; omit or 0 = default (70-80 by logo span)
//	  version:     int,                             // 1-40; omit or 0 = auto (errors if the data doesn't fit)
//	  mask:        int,                             // 0-7; omit or -1 = auto (best-scoring pattern)
//	  moduleShape: "square" | "rounded" | "dot",    // default "square"
//	  eyeShape:      "square" | "rounded" | "circle", // finder frame + ball at once
//	  eyeFrameShape: "square" | "rounded" | "circle", // overrides eyeShape for the frame
//	  eyeBallShape:  "square" | "rounded" | "circle", // overrides eyeShape for the ball
//	  eyeFrame:    "#rrggbb",                       // finder frame color; omit = follow dark/gradient
//	  eyeBall:     "#rrggbb",                       // finder ball color; omit = follow dark/gradient
//	  gradient:    {kind: "linear" | "radial",      // module fill gradient
//	                from: "#rrggbb", to: "#rrggbb",
//	                angle: number},                 // degrees, linear only (0 = left to right)
//	}
//
// PNG returns data as a Uint8Array, SVG as a string.
package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"syscall/js"
	"time"

	"github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/content"
	qrlogo "github.com/nachop51/qr-go/logo"
	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/png"
	"github.com/nachop51/qr-go/render/style"
	"github.com/nachop51/qr-go/render/svg"
)

func errResult(format string, args ...any) map[string]any {
	return map[string]any{"error": fmt.Sprintf(format, args...)}
}

// safe wraps a handler so a panic surfaces to JS as {error} instead of
// killing the wasm runtime (main never returns, so a crash would leave every
// later call throwing "Go program has already exited").
func safe(fn func(js.Value) any) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) (result any) {
		defer func() {
			if r := recover(); r != nil {
				result = errResult("panic: %v", r)
			}
		}()
		if len(args) < 1 || args[0].Type() != js.TypeObject {
			return errResult("missing options object argument")
		}
		return fn(args[0])
	})
}

func str(v js.Value, key, def string) string {
	p := v.Get(key)
	if p.Type() != js.TypeString {
		return def
	}
	return p.String()
}

func num(v js.Value, key string, def int) int {
	p := v.Get(key)
	if p.Type() != js.TypeNumber {
		return def
	}
	return p.Int()
}

func boolean(v js.Value, key string) bool {
	p := v.Get(key)
	return p.Type() == js.TypeBoolean && p.Bool()
}

func ecLevel(v js.Value) (qr.CorrectionLevel, error) {
	switch str(v, "ecLevel", "M") {
	case "L":
		return qr.CorrectionLevelLow, nil
	case "M":
		return qr.CorrectionLevelMedium, nil
	case "Q":
		return qr.CorrectionLevelQuartile, nil
	case "H":
		return qr.CorrectionLevelHigh, nil
	default:
		return 0, fmt.Errorf("invalid error-correction level")
	}
}

func eciPolicy(v js.Value) (qr.TextECIPolicy, error) {
	switch str(v, "eciPolicy", "auto") {
	case "auto":
		return qr.TextECIPolicyAuto, nil
	case "disabled":
		return qr.TextECIPolicyDisabled, nil
	default:
		return 0, fmt.Errorf("invalid ECI policy")
	}
}

// parseHex parses #rgb and #rrggbb colors.
func parseHex(s string) (color.Color, error) {
	c, err := render.ParseCSSColor(s)
	if err != nil {
		return nil, err
	}
	return c.RGBA, nil
}

// styleSpec carries the parsed style options; colors stay raw strings so the
// PNG branch converts with parseHex while SVG passes them through.
type styleSpec struct {
	module           style.ModuleShape
	frame, ball      style.EyeShape
	eyeFrame         string
	eyeBall          string
	gradKind         style.GradientKind
	gradFrom, gradTo string
	gradAngle        float64
}

func styleOpts(v js.Value) (styleSpec, error) {
	var s styleSpec
	var err error
	if s.module, err = style.ParseModuleShape(str(v, "moduleShape", "")); err != nil {
		return s, err
	}
	base, err := style.ParseEyeShape(str(v, "eyeShape", ""))
	if err != nil {
		return s, err
	}
	s.frame, s.ball = base, base
	if raw := str(v, "eyeFrameShape", ""); raw != "" {
		if s.frame, err = style.ParseEyeShape(raw); err != nil {
			return s, err
		}
	}
	if raw := str(v, "eyeBallShape", ""); raw != "" {
		if s.ball, err = style.ParseEyeShape(raw); err != nil {
			return s, err
		}
	}
	s.eyeFrame = str(v, "eyeFrame", "")
	s.eyeBall = str(v, "eyeBall", "")

	if g := v.Get("gradient"); g.Type() == js.TypeObject {
		if s.gradKind, err = style.ParseGradientKind(str(g, "kind", "")); err != nil {
			return s, err
		}
		if s.gradKind != style.GradientNone {
			s.gradFrom, s.gradTo = str(g, "from", ""), str(g, "to", "")
			if s.gradFrom == "" || s.gradTo == "" {
				return s, fmt.Errorf("gradient needs from and to colors")
			}
			if a := g.Get("angle"); a.Type() == js.TypeNumber {
				s.gradAngle = a.Float()
			}
		}
	}
	return s, nil
}

func logoImage(v js.Value) (image.Image, error) {
	p := v.Get("logo")
	if p.IsUndefined() || p.IsNull() {
		return nil, nil
	}
	if !p.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil, fmt.Errorf("decode logo: logo must be a Uint8Array")
	}
	n := p.Get("length").Int()
	if n < 0 || n > qrlogo.MaxEncodedBytes {
		return nil, fmt.Errorf("decode logo: encoded input exceeds 16 MiB")
	}
	data := make([]byte, n)
	js.CopyBytesToGo(data, p)
	img, err := qrlogo.DecodeBytes(data)
	if err != nil {
		return nil, fmt.Errorf("decode logo: %v", err)
	}
	return img, nil
}

func generate(opts js.Value) any {
	text := str(opts, "text", "")
	if text == "" {
		return errResult("text is required")
	}

	logo, err := logoImage(opts)
	if err != nil {
		return errResult("%v", err)
	}

	st, err := styleOpts(opts)
	if err != nil {
		return errResult("%v", err)
	}

	format := str(opts, "format", "png")
	level, err := ecLevel(opts)
	if err != nil {
		return errResult("%v", err)
	}
	policy, err := eciPolicy(opts)
	if err != nil {
		return errResult("%v", err)
	}
	version := num(opts, "version", 0)
	mask := num(opts, "mask", -1)
	quiet := num(opts, "quiet", -1)
	if version < 0 || version > 40 {
		return errResult("version must be 0 or between 1 and 40")
	}
	if mask < -1 || mask > 7 {
		return errResult("mask must be -1 or between 0 and 7")
	}
	if quiet < -1 || quiet > 256 {
		return errResult("quiet zone must be between 0 and 256")
	}
	logoModules := num(opts, "logoModules", 0)
	if logoModules < 0 {
		return errResult("logoModules must be non-negative")
	}
	var warnings []any
	warn := func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) }
	var renderer render.Renderer
	switch format {
	case "png":
		r := png.New().WarningHandler(warn)
		if c := str(opts, "dark", ""); c != "" {
			col, err := parseHex(c)
			if err != nil {
				return errResult("%v", err)
			}
			r = r.Dark(col)
		}
		if c := str(opts, "light", ""); c != "" {
			col, err := parseHex(c)
			if err != nil {
				return errResult("%v", err)
			}
			r = r.White(col)
		}
		if quiet >= 0 {
			r = r.Quiet(quiet)
		}
		if value := opts.Get("size"); value.Type() == js.TypeNumber {
			n := value.Int()
			if n < 1 || n > 4096 {
				return errResult("PNG size must be between 1 and 4096")
			}
			r = r.Width(n).Height(n)
		}
		if logo != nil {
			r = r.Logo(logo).LogoModules(logoModules).LogoScale(num(opts, "logoScale", 0))
		}
		r = r.ModuleShape(st.module).EyeFrameShape(st.frame).EyeBallShape(st.ball)
		if st.eyeFrame != "" {
			col, err := parseHex(st.eyeFrame)
			if err != nil {
				return errResult("eyeFrame: %v", err)
			}
			r = r.EyeFrame(col)
		}
		if st.eyeBall != "" {
			col, err := parseHex(st.eyeBall)
			if err != nil {
				return errResult("eyeBall: %v", err)
			}
			r = r.EyeBall(col)
		}
		if st.gradKind != style.GradientNone {
			from, err := parseHex(st.gradFrom)
			if err != nil {
				return errResult("gradient: %v", err)
			}
			to, err := parseHex(st.gradTo)
			if err != nil {
				return errResult("gradient: %v", err)
			}
			if st.gradKind == style.GradientRadial {
				r = r.GradientRadial(from, to)
			} else {
				r = r.GradientLinear(from, to, st.gradAngle)
			}
		}
		renderer = r
	case "svg":
		r := svg.New().WarningHandler(warn)
		if c := str(opts, "dark", ""); c != "" {
			r = r.Dark(c)
		}
		if c := str(opts, "light", ""); c != "" {
			r = r.Light(c)
		}
		if quiet >= 0 {
			r = r.Quiet(quiet)
		}
		if value := opts.Get("moduleSize"); value.Type() == js.TypeNumber {
			n := value.Int()
			if n < 1 || n > 1024 {
				return errResult("SVG moduleSize must be between 1 and 1024")
			}
			r = r.Module(n)
		}
		if logo != nil {
			r = r.Logo(logo).LogoModules(logoModules).LogoScale(num(opts, "logoScale", 0))
		}
		r = r.ModuleShape(st.module).EyeFrameShape(st.frame).EyeBallShape(st.ball).
			EyeFrame(st.eyeFrame).EyeBall(st.eyeBall)
		if st.gradKind == style.GradientRadial {
			r = r.GradientRadial(st.gradFrom, st.gradTo)
		} else if st.gradKind == style.GradientLinear {
			r = r.GradientLinear(st.gradFrom, st.gradTo, st.gradAngle)
		}
		renderer = r
	default:
		return errResult("unknown format %q: want png or svg", format)
	}

	builder := qr.NewTextBuilder(text).
		SetRenderer(renderer).
		SetErrorCorrectionLevel(level).
		SetTextECIPolicy(policy)
	if version > 0 {
		builder = builder.SetVersion(version)
	}
	if mask >= 0 {
		builder = builder.SetMask(mask)
	}
	code, err := builder.Build()
	if err != nil {
		return errResult("%v", err)
	}

	data, err := code.Bytes()
	if err != nil {
		return errResult("%v", err)
	}

	var out any
	if format == "svg" {
		out = string(data)
	} else {
		arr := js.Global().Get("Uint8Array").New(len(data))
		js.CopyBytesToJS(arr, data)
		out = arr
	}
	return map[string]any{
		"data":           out,
		"size":           code.Size(),
		"version":        code.Version(),
		"mask":           code.Mask(),
		"maxLogoModules": code.MaxLogoModules(),
		"warnings":       warnings,
	}
}

func parseEventTime(s string) (time.Time, bool, error) {
	if s == "" {
		return time.Time{}, false, nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, false, nil
	}
	// Date parses a datetime-local value in the browser's local timezone.
	if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?$`, s); matched {
		d := js.Global().Get("Date").New(s)
		if !math.IsNaN(d.Call("getTime").Float()) {
			if t, err := time.Parse(time.RFC3339Nano, d.Call("toISOString").String()); err == nil {
				return t, false, nil
			}
		}
	}
	return time.Time{}, false, fmt.Errorf("invalid time %q: want YYYY-MM-DD or RFC 3339", s)
}

func contentAPI() map[string]any {
	return map[string]any{
		"wifi": safe(func(o js.Value) any {
			auth := content.WiFiAuth(str(o, "auth", ""))
			if auth != "" && auth != content.WiFiWPA && auth != content.WiFiWEP && auth != content.WiFiNone {
				return errResult("unsupported Wi-Fi authentication %q", auth)
			}
			return content.WiFi{
				SSID:   str(o, "ssid", ""),
				Pass:   str(o, "pass", ""),
				Auth:   auth,
				Hidden: boolean(o, "hidden"),
			}.String()
		}),
		"vcard": safe(func(o js.Value) any {
			return content.VCard{
				FullName: str(o, "fullName", ""),
				First:    str(o, "first", ""),
				Last:     str(o, "last", ""),
				Org:      str(o, "org", ""),
				Title:    str(o, "title", ""),
				Phone:    str(o, "phone", ""),
				Email:    str(o, "email", ""),
				URL:      str(o, "url", ""),
				Address:  str(o, "address", ""),
			}.String()
		}),
		"event": safe(func(o js.Value) any {
			start, startAllDay, err := parseEventTime(str(o, "start", ""))
			if err != nil {
				return errResult("start: %v", err)
			}
			end, endAllDay, err := parseEventTime(str(o, "end", ""))
			if err != nil {
				return errResult("end: %v", err)
			}
			if !start.IsZero() && !end.IsZero() && startAllDay != endAllDay {
				return errResult("event start and end must both be dates or both be date-times")
			}
			allDay := startAllDay || endAllDay
			if allDay && (start.IsZero() || end.IsZero() || !startAllDay || !endAllDay) {
				return errResult("all-day events require date-only start and end values")
			}
			return content.Event{
				Summary:     str(o, "summary", ""),
				Location:    str(o, "location", ""),
				Description: str(o, "description", ""),
				Start:       start,
				End:         end,
				AllDay:      allDay,
			}.String()
		}),
		"url": safe(func(o js.Value) any {
			return content.URL(str(o, "url", ""))
		}),
		"tel": safe(func(o js.Value) any {
			return content.Tel(str(o, "number", ""))
		}),
		"sms": safe(func(o js.Value) any {
			return content.SMS(str(o, "number", ""), str(o, "message", ""))
		}),
		"geo": safe(func(o js.Value) any {
			lat, lng := o.Get("lat"), o.Get("lng")
			if lat.Type() != js.TypeNumber || lng.Type() != js.TypeNumber {
				return errResult("lat and lng must be numbers")
			}
			la, lo := lat.Float(), lng.Float()
			if math.IsNaN(la) || math.IsInf(la, 0) || la < -90 || la > 90 || math.IsNaN(lo) || math.IsInf(lo, 0) || lo < -180 || lo > 180 {
				return errResult("coordinates must be finite and within latitude [-90,90], longitude [-180,180]")
			}
			return content.Geo(la, lo)
		}),
		"email": safe(func(o js.Value) any {
			return content.Email(str(o, "to", ""), str(o, "subject", ""), str(o, "body", ""))
		}),
	}
}

func main() {
	js.Global().Set("qrgo", map[string]any{
		"generate": safe(generate),
		"content":  contentAPI(),
	})

	select {}
}
