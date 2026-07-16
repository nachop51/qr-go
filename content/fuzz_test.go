package content

import (
	"strings"
	"testing"
)

func FuzzContentEscapingAndFolding(f *testing.F) {
	for _, seed := range []string{"plain", "line1\r\nline2", strings.Repeat("é", 80), `a;b,c\\d`, "\xc3" + strings.Repeat("\x80", 80)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 2048 {
			s = s[:2048]
		}
		out := VCard{FullName: s}.String()
		for _, line := range strings.Split(strings.TrimSuffix(out, "\r\n"), "\r\n") {
			if len(line) > 75 {
				t.Fatalf("folded line is %d octets", len(line))
			}
		}
		if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\r") {
			t.Fatal("bare carriage return")
		}
	})
}
