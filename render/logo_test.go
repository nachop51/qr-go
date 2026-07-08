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
