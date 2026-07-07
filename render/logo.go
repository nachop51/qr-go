package render

import (
	"fmt"
	"os"
)

// Warnf reports non-fatal rendering adjustments, such as a logo span being
// capped to keep the code scannable. It defaults to writing to stderr; set it
// to nil to silence it, or replace it to route the messages elsewhere.
var Warnf = func(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "qr: "+format+"\n", args...)
}

// LogoBudgeter is implemented by grids (such as *qr.Code) that can report the
// largest scannable centred logo, in modules, for their error-correction level.
// Overlay renderers use it to cap oversized logos.
type LogoBudgeter interface {
	MaxLogoModules() int
}

// ResolveLogo returns the effective centred-logo span, in whole modules, for a
// grid: the configured span (or the size/5 default when configured <= 0),
// capped to the grid's recoverable budget when it implements [LogoBudgeter],
// then snapped to the odd module grid so it centres with equal margins. A cap
// is reported through [Warnf]. It returns 0 when no logo should be drawn.
//
// Both raster and vector renderers call this so a logo behaves identically
// across output formats.
func ResolveLogo(g Grid, configured int) int {
	size := g.Size()

	budget := 0
	if b, ok := g.(LogoBudgeter); ok {
		budget = b.MaxLogoModules()
	}

	mods := configured
	if mods <= 0 {
		mods = size / 5
	}
	capped := false
	if budget > 0 && mods > budget {
		mods, capped = budget, true
	}
	// A QR's module count is always odd, so an odd span centres with equal
	// margins; nudge an even span down to stay aligned.
	if (size-mods)%2 != 0 {
		mods--
	}
	if mods < 1 {
		return 0
	}
	if capped && Warnf != nil {
		Warnf("logo span capped to %d modules to stay scannable at this error-correction level", mods)
	}
	return mods
}
