package render_test

import (
	"testing"

	"github.com/nachop51/qr-go/render"
)

type plainGrid struct{ n int }

func (g plainGrid) Size() int            { return g.n }
func (g plainGrid) IsDark(x, y int) bool { return false }

type budgetGrid struct {
	n      int
	budget int
}

func (g budgetGrid) Size() int            { return g.n }
func (g budgetGrid) IsDark(x, y int) bool { return false }
func (g budgetGrid) MaxLogoModules() int  { return g.budget }

func TestResolveLogoDefaultAndParity(t *testing.T) {
	for _, c := range []struct {
		configured, want int
	}{
		{0, 5},  // size/5 fallback for grids without a budget
		{7, 7},  // odd span stays
		{8, 7},  // even span nudged down to stay centred
		{-3, 5}, // negative restores the default
	} {
		if got := render.ResolveLogo(plainGrid{n: 25}, c.configured); got != c.want {
			t.Errorf("ResolveLogo(configured=%d) = %d, want %d", c.configured, got, c.want)
		}
	}
}

func TestResolveLogoDefaultsToBudget(t *testing.T) {
	var warns int
	orig := render.Warnf
	render.Warnf = func(string, ...any) { warns++ }
	defer func() { render.Warnf = orig }()

	// Unconfigured -> full budget (8), parity-nudged to 7, no cap warning.
	if got := render.ResolveLogo(budgetGrid{n: 25, budget: 8}, 0); got != 7 {
		t.Errorf("default span = %d, want 7", got)
	}
	if warns != 0 {
		t.Errorf("expected 0 warnings, got %d", warns)
	}
}

type allDarkGrid struct{ n int }

func (g allDarkGrid) Size() int            { return g.n }
func (g allDarkGrid) IsDark(x, y int) bool { return true }

func TestMaskLogo(t *testing.T) {
	// n=21, mods=7 -> region [7,14) on both axes.
	g := render.MaskLogo(allDarkGrid{n: 21}, 7)
	if g.Size() != 21 {
		t.Fatalf("Size() = %d, want 21", g.Size())
	}
	for _, c := range []struct {
		x, y int
		want bool
	}{
		{7, 7, false},   // region corner
		{13, 13, false}, // last module inside
		{10, 10, false}, // centre
		{6, 10, true},   // just left of the region
		{14, 10, true},  // just right of the region
		{10, 6, true},   // just above
		{0, 0, true},    // far corner
	} {
		if got := g.IsDark(c.x, c.y); got != c.want {
			t.Errorf("IsDark(%d, %d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}

	if g := render.MaskLogo(allDarkGrid{n: 21}, 0); !g.IsDark(10, 10) {
		t.Error("MaskLogo with mods <= 0 must leave the grid untouched")
	}
}

func TestLogoBox(t *testing.T) {
	var warns int
	orig := render.Warnf
	render.Warnf = func(string, ...any) { warns++ }
	defer func() { render.Warnf = orig }()

	for _, c := range []struct {
		name                string
		region, mods, scale int
		want, wantWarns     int
	}{
		{"default on a narrow span fills 80%", 60, 3, 0, 48, 0},
		{"default on a mid span fills 75%", 90, 7, 0, 67, 0},
		{"default on a wide span fills 70%", 200, 9, 0, 140, 0},
		{"default still draws in a tiny region", 1, 1, 0, 1, 0},
		{"explicit percent of the region", 100, 5, 40, 40, 0},
		{"full region", 90, 5, 100, 90, 0},
		{"above 100 capped with a warning", 90, 5, 150, 90, 1},
		{"tiny percent still draws", 30, 5, 1, 1, 0},
	} {
		warns = 0
		if got := render.LogoBox(c.region, c.mods, c.scale); got != c.want {
			t.Errorf("%s: LogoBox(%d, %d, %d) = %d, want %d", c.name, c.region, c.mods, c.scale, got, c.want)
		}
		if warns != c.wantWarns {
			t.Errorf("%s: %d warnings, want %d", c.name, warns, c.wantWarns)
		}
	}
}

func TestResolveLogoCaps(t *testing.T) {
	var warns int
	orig := render.Warnf
	render.Warnf = func(string, ...any) { warns++ }
	defer func() { render.Warnf = orig }()

	// budget 4, request 9 -> capped to 4, then parity-nudged to 3, one warning.
	warns = 0
	if got := render.ResolveLogo(budgetGrid{n: 25, budget: 4}, 9); got != 3 {
		t.Errorf("capped span = %d, want 3", got)
	}
	if warns != 1 {
		t.Errorf("expected 1 warning, got %d", warns)
	}

	// Within budget: request 3 <= 4 -> 3, no cap, no warning.
	warns = 0
	if got := render.ResolveLogo(budgetGrid{n: 25, budget: 4}, 3); got != 3 {
		t.Errorf("within-budget span = %d, want 3", got)
	}
	if warns != 0 {
		t.Errorf("expected 0 warnings, got %d", warns)
	}
}
