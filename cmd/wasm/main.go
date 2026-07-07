//go:build js && wasm

// Command wasm exposes the qr-go library to JavaScript as a single `qrgo`
// global:
//
//	qrgo.generate(opts) -> {data, size, maxLogoModules, warnings} | {error}
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
//	  logoModules: int,                             // logo span in modules
//	}
//
// PNG returns data as a Uint8Array, SVG as a string.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"syscall/js"
	"time"

	"github.com/nachop51/qr-go"
	"github.com/nachop51/qr-go/content"
	"github.com/nachop51/qr-go/render"
	"github.com/nachop51/qr-go/render/png"
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

func ecLevel(v js.Value) qr.CorrectionLevel {
	switch str(v, "ecLevel", "M") {
	case "L":
		return qr.CorrectionLevelLow
	case "Q":
		return qr.CorrectionLevelQuartile
	case "H":
		return qr.CorrectionLevelHigh
	default:
		return qr.CorrectionLevelMedium
	}
}

func eciPolicy(v js.Value) qr.TextECIPolicy {
	if str(v, "eciPolicy", "auto") == "disabled" {
		return qr.TextECIPolicyDisabled
	}
	return qr.TextECIPolicyAuto
}

// parseHex parses #rgb and #rrggbb colors.
func parseHex(s string) (color.Color, error) {
	if len(s) == 0 || s[0] != '#' {
		return nil, fmt.Errorf("invalid color %q: want #rgb or #rrggbb", s)
	}
	hex := s[1:]
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return nil, fmt.Errorf("invalid color %q: want #rgb or #rrggbb", s)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err != nil {
		return nil, fmt.Errorf("invalid color %q: %v", s, err)
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil
}

func logoImage(v js.Value) (image.Image, error) {
	p := v.Get("logo")
	if p.IsUndefined() || p.IsNull() {
		return nil, nil
	}
	data := make([]byte, p.Get("length").Int())
	js.CopyBytesToGo(data, p)
	img, _, err := image.Decode(bytes.NewReader(data))
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

	format := str(opts, "format", "png")
	var renderer render.Renderer
	switch format {
	case "png":
		r := png.New()
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
		if n := num(opts, "quiet", -1); n >= 0 {
			r = r.Quiet(n)
		}
		if n := num(opts, "size", 0); n > 0 {
			r = r.Width(n).Height(n)
		}
		if logo != nil {
			r = r.Logo(logo).LogoModules(num(opts, "logoModules", 0))
		}
		renderer = r
	case "svg":
		r := svg.New()
		if c := str(opts, "dark", ""); c != "" {
			r = r.Dark(c)
		}
		if c := str(opts, "light", ""); c != "" {
			r = r.Light(c)
		}
		if n := num(opts, "quiet", -1); n >= 0 {
			r = r.Quiet(n)
		}
		if n := num(opts, "moduleSize", 0); n > 0 {
			r = r.Module(n)
		}
		if logo != nil {
			r = r.Logo(logo).LogoModules(num(opts, "logoModules", 0))
		}
		renderer = r
	default:
		return errResult("unknown format %q: want png or svg", format)
	}

	code, err := qr.NewTextBuilder(text).
		SetRenderer(renderer).
		SetErrorCorrectionLevel(ecLevel(opts)).
		SetTextECIPolicy(eciPolicy(opts)).
		Build()
	if err != nil {
		return errResult("%v", err)
	}

	// Collect non-fatal adjustments (e.g. logo span capped) for the UI.
	var warnings []any
	prevWarnf := render.Warnf
	render.Warnf = func(format string, args ...any) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	data, err := code.Bytes()
	render.Warnf = prevWarnf
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
	// datetime-local inputs produce "2006-01-02T15:04" (no zone, no seconds).
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, false, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("invalid time %q: want YYYY-MM-DD or RFC 3339", s)
}

func contentAPI() map[string]any {
	return map[string]any{
		"wifi": safe(func(o js.Value) any {
			return content.WiFi{
				SSID:   str(o, "ssid", ""),
				Pass:   str(o, "pass", ""),
				Auth:   content.WiFiAuth(str(o, "auth", "")),
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
			return content.Event{
				Summary:     str(o, "summary", ""),
				Location:    str(o, "location", ""),
				Description: str(o, "description", ""),
				Start:       start,
				End:         end,
				AllDay:      startAllDay && (end.IsZero() || endAllDay),
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
			return content.Geo(lat.Float(), lng.Float())
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
