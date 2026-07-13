package render

import (
	"fmt"
	"os"
)

// WarningHandler receives non-fatal rendering recommendations.
type WarningHandler func(format string, args ...any)

// StderrWarningHandler is the default used by command-line renderers.
func StderrWarningHandler(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "qr: "+format+"\n", args...)
}

// LogoBudgeter is implemented by grids (such as *qr.Code) that can report the
// largest scannable centred logo, in modules, for their error-correction level.
// Overlay renderers use it to cap oversized logos.
type LogoBudgeter interface {
	MaxLogoModules() int
}

// logoMasked hides the modules under the centred logo region.
type logoMasked struct {
	Grid
	lo, hi int // region bounds in modules, [lo, hi) on both axes
}

func (m logoMasked) IsDark(x, y int) bool {
	if x >= m.lo && x < m.hi && y >= m.lo && y < m.hi {
		return false
	}
	return m.Grid.IsDark(x, y)
}

// MaskLogo wraps g so the modules inside the centred mods-wide logo region
// (as returned by [ResolveLogo]) read as light. Renderers draw from the
// wrapped grid so shaped modules at the region's edge round toward the
// cleared area instead of connecting to modules the logo overlay is about to
// hide, and so no module is emitted just to be painted over.
func MaskLogo(g Grid, mods int) Grid {
	if mods <= 0 {
		return g
	}
	lo := (g.Size() - mods) / 2
	return logoMasked{Grid: g, lo: lo, hi: lo + mods}
}

// DefaultLogoScale is the percentage of the cleared area a logo fills when no
// explicit scale is configured. Small regions get less breathing room: on a
// narrow span every ring of background is a big slice of the (already small)
// logo, while a wide span can afford a proportionally larger margin.
func DefaultLogoScale(mods int) int {
	switch {
	case mods <= 3:
		return 80
	case mods <= 7:
		return 75
	default:
		return 70
	}
}

// LogoBox returns the edge length of the square the logo image is fitted
// into, inside a cleared region of region device units (the mods-module logo
// span times the per-module scale). scalePct is the percentage of the region
// the logo may fill: 100 covers the whole cleared square, smaller values
// shrink the logo and leave more background around it. A value <= 0 keeps the
// span-dependent default, [DefaultLogoScale]. Values above 100 are capped
// (reported through [Warnf]).
//
// Both raster and vector renderers call this so a logo behaves identically
// across output formats.
func LogoBox(region, mods, scalePct int) int {
	return LogoBoxWithWarnings(region, mods, scalePct, nil)
}

func LogoBoxWithWarnings(region, mods, scalePct int, warn WarningHandler) int {
	if scalePct <= 0 {
		scalePct = DefaultLogoScale(mods)
	}
	if scalePct > 100 {
		if warn != nil {
			warn("logo scale capped to 100%% of the cleared area")
		}
		scalePct = 100
	}
	return max(region*scalePct/100, 1)
}

// ResolveLogo returns the effective centred-logo span, in whole modules, for a
// grid. When configured <= 0 it defaults to the grid's full recoverable budget
// (see [LogoBudgeter]) — the largest span the error-correction level can
// afford to lose — falling back to size/5 for grids without a budget. An
// explicit span is capped to that budget, then snapped to the odd module grid
// so it centres with equal margins. A cap is reported through [Warnf]. It
// returns 0 when no logo should be drawn.
//
// Both raster and vector renderers call this so a logo behaves identically
// across output formats.
func ResolveLogo(g Grid, configured int) int {
	return ResolveLogoWithWarnings(g, configured, nil)
}

func ResolveLogoWithWarnings(g Grid, configured int, warn WarningHandler) int {
	size := g.Size()

	budget := 0
	if b, ok := g.(LogoBudgeter); ok {
		budget = b.MaxLogoModules()
	}

	mods := configured
	if mods <= 0 {
		if budget > 0 {
			mods = budget
		} else {
			mods = size / 5
		}
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
	if capped && warn != nil {
		warn("logo span capped to %d modules; logo sizing is a recommendation, not a scanability guarantee", mods)
	}
	return mods
}
