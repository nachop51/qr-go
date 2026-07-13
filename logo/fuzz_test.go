package logo

import "testing"

func FuzzLogoDetectionAndDecode(f *testing.F) {
	f.Add([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"/>`))
	f.Add([]byte("binary prefix <svg hidden later"))
	f.Add([]byte{0x89, 'P', 'N', 'G'})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			data = data[:1<<20]
		}
		_ = isSVG(data)
		_, _ = DecodeBytes(data)
	})
}
