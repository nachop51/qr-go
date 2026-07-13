package render_test

import (
	"testing"

	"github.com/nachop51/qr-go/render"
)

func TestParseCSSColorStrict(t *testing.T) {
	valid := []string{"red", "transparent", "#abc", "#abcd", "#112233", "#11223344", "rgb(1, 2, 3)", "rgba(100%,0%,0%,50%)", "hsl(120, 100%, 50%)", "hsla(240,100%,50%,.5)"}
	for _, s := range valid {
		if _, err := render.ParseCSSColor(s); err != nil {
			t.Errorf("%q: %v", s, err)
		}
	}
	invalid := []string{"url(https://example.test/x)", "url(#paint)", "var(--x)", "#12", "rgb(256,0,0)", "rgba(0,0,0,2)", "hsl(0,101%,0%)", "red\nfoo", `red\"/><script>`}
	for _, s := range invalid {
		if _, err := render.ParseCSSColor(s); err == nil {
			t.Errorf("%q: expected rejection", s)
		}
	}
}
