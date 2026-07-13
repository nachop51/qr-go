package render_test

import (
	"strings"
	"testing"

	"github.com/nachop51/qr-go/render"
)

func FuzzParseCSSColor(f *testing.F) {
	for _, seed := range []string{"#fff", "rgba(1,2,3,.5)", "hsl(120,100%,50%)", "url(#x)", `red\"/><script>`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c, err := render.ParseCSSColor(s)
		if err == nil && (strings.ContainsAny(c.CSS, `\"'<>`) || strings.Contains(strings.ToLower(c.CSS), "url")) {
			t.Fatalf("unsafe canonical color %q", c.CSS)
		}
	})
}
