package style

// Path receives shape outlines in module units; renderers scale to device
// units in their implementation. Subpaths emitted for ring cutouts run
// counter-clockwise, so consumers must fill with the nonzero winding rule
// (the default for both SVG and rasterx).
type Path interface {
	MoveTo(x, y float64)
	LineTo(x, y float64)
	CubeTo(c1x, c1y, c2x, c2y, x, y float64)
	Close()
}

// kappa scales a radius to the cubic-Bezier control offset that best
// approximates a quarter circle.
const kappa = 0.5522847498307936

// Radii in module units. Module corners round by half a module (a quarter
// circle spanning the full corner); eye radii are tuned to the classic
// styled-QR look where the 7x7 frame rounds harder than its 5x5 cutout.
const (
	moduleRadius   = 0.5
	frameOuterR    = 2.0
	frameInnerR    = 1.0
	ballRadius     = 0.75
	frameOuterCirc = 3.5
	frameInnerCirc = 2.5
	ballCircleR    = 1.5
)

// AddModule emits the outline of the 1x1 module at (x, y). For ModuleRounded
// only the corners present in c are rounded, so runs of adjacent modules keep
// flush shared edges.
func AddModule(p Path, x, y float64, shape ModuleShape, c Corners) {
	switch shape {
	case ModuleDot:
		addCircle(p, x+0.5, y+0.5, moduleRadius, false)
	case ModuleRounded:
		r := func(corner Corners) float64 {
			if c&corner != 0 {
				return moduleRadius
			}
			return 0
		}
		addRect(p, x, y, 1, 1, r(CornerTL), r(CornerTR), r(CornerBR), r(CornerBL))
	default:
		addRect(p, x, y, 1, 1, 0, 0, 0, 0)
	}
}

// AddEyeFrame emits the finder ring for eye r: the 7x7 outline clockwise and
// the 5x5 cutout counter-clockwise, so a nonzero-winding fill leaves the ring.
func AddEyeFrame(p Path, r Rect, shape EyeShape) {
	x, y := float64(r.X), float64(r.Y)
	switch shape {
	case EyeCircle:
		cx, cy := x+3.5, y+3.5
		addCircle(p, cx, cy, frameOuterCirc, false)
		addCircle(p, cx, cy, frameInnerCirc, true)
	case EyeRounded:
		addRect(p, x, y, 7, 7, frameOuterR, frameOuterR, frameOuterR, frameOuterR)
		addRectCCW(p, x+1, y+1, 5, 5, frameInnerR)
	default:
		addRect(p, x, y, 7, 7, 0, 0, 0, 0)
		addRectCCW(p, x+1, y+1, 5, 5, 0)
	}
}

// AddEyeBall emits the 3x3 pupil centred in eye r.
func AddEyeBall(p Path, r Rect, shape EyeShape) {
	x, y := float64(r.X)+2, float64(r.Y)+2
	switch shape {
	case EyeCircle:
		addCircle(p, x+1.5, y+1.5, ballCircleR, false)
	case EyeRounded:
		addRect(p, x, y, 3, 3, ballRadius, ballRadius, ballRadius, ballRadius)
	default:
		addRect(p, x, y, 3, 3, 0, 0, 0, 0)
	}
}

// addRect emits a rectangle clockwise (in the y-down module space) with
// per-corner radii. A zero radius keeps that corner sharp.
func addRect(p Path, x, y, w, h, tl, tr, br, bl float64) {
	p.MoveTo(x+tl, y)
	p.LineTo(x+w-tr, y)
	if tr > 0 {
		p.CubeTo(x+w-tr+kappa*tr, y, x+w, y+tr-kappa*tr, x+w, y+tr)
	}
	p.LineTo(x+w, y+h-br)
	if br > 0 {
		p.CubeTo(x+w, y+h-br+kappa*br, x+w-br+kappa*br, y+h, x+w-br, y+h)
	}
	p.LineTo(x+bl, y+h)
	if bl > 0 {
		p.CubeTo(x+bl-kappa*bl, y+h, x, y+h-bl+kappa*bl, x, y+h-bl)
	}
	p.LineTo(x, y+tl)
	if tl > 0 {
		p.CubeTo(x, y+tl-kappa*tl, x+tl-kappa*tl, y, x+tl, y)
	}
	p.Close()
}

// addRectCCW emits a rectangle counter-clockwise with a uniform corner
// radius, for ring cutouts under the nonzero winding rule.
func addRectCCW(p Path, x, y, w, h, r float64) {
	p.MoveTo(x+w-r, y)
	p.LineTo(x+r, y)
	if r > 0 {
		p.CubeTo(x+r-kappa*r, y, x, y+r-kappa*r, x, y+r)
	}
	p.LineTo(x, y+h-r)
	if r > 0 {
		p.CubeTo(x, y+h-r+kappa*r, x+r-kappa*r, y+h, x+r, y+h)
	}
	p.LineTo(x+w-r, y+h)
	if r > 0 {
		p.CubeTo(x+w-r+kappa*r, y+h, x+w, y+h-r+kappa*r, x+w, y+h-r)
	}
	p.LineTo(x+w, y+r)
	if r > 0 {
		p.CubeTo(x+w, y+r-kappa*r, x+w-r+kappa*r, y, x+w-r, y)
	}
	p.Close()
}

// addCircle approximates a circle with four cubic arcs, clockwise by default
// or counter-clockwise for cutouts.
func addCircle(p Path, cx, cy, r float64, ccw bool) {
	k := kappa * r
	p.MoveTo(cx+r, cy)
	if ccw {
		p.CubeTo(cx+r, cy-k, cx+k, cy-r, cx, cy-r)
		p.CubeTo(cx-k, cy-r, cx-r, cy-k, cx-r, cy)
		p.CubeTo(cx-r, cy+k, cx-k, cy+r, cx, cy+r)
		p.CubeTo(cx+k, cy+r, cx+r, cy+k, cx+r, cy)
	} else {
		p.CubeTo(cx+r, cy+k, cx+k, cy+r, cx, cy+r)
		p.CubeTo(cx-k, cy+r, cx-r, cy+k, cx-r, cy)
		p.CubeTo(cx-r, cy-k, cx-k, cy-r, cx, cy-r)
		p.CubeTo(cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	}
	p.Close()
}
