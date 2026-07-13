package svg

import "testing"

func FuzzRendererOptions(f *testing.F) {
	f.Add(4, 10, "#000", "white")
	f.Add(-1, 0, `url(#paint)`, `red\"/><script>`)
	f.Add(257, 1025, "transparent", "rgba(0,0,0,.5)")
	f.Fuzz(func(t *testing.T, quiet, module int, dark, light string) {
		if len(dark) > 256 {
			dark = dark[:256]
		}
		if len(light) > 256 {
			light = light[:256]
		}
		_, _ = New().Quiet(quiet).Module(module).Dark(dark).Light(light).Bytes(fakeGrid{n: 21})
	})
}
