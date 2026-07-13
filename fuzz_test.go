package qr

import "testing"

func FuzzTextSegmentation(f *testing.F) {
	for _, seed := range []string{"", "1234567890", "HELLO WORLD", "café 日本語", "A1é漢字-42"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 512 {
			s = s[:512]
		}
		_, _ = NewTextBuilder(s).Build()
	})
}
