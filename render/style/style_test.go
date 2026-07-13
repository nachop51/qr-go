package style

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/nachop51/qr-go/render"
)

// recorder captures Path commands as a comparable stream.
type recorder struct{ cmds []string }

func (r *recorder) MoveTo(x, y float64) { r.add("M", x, y) }
func (r *recorder) LineTo(x, y float64) { r.add("L", x, y) }
func (r *recorder) CubeTo(c1x, c1y, c2x, c2y, x, y float64) {
	r.add("C", c1x, c1y, c2x, c2y, x, y)
}
func (r *recorder) Close() { r.cmds = append(r.cmds, "Z") }
func (r *recorder) add(op string, vals ...float64) {
	parts := []string{op}
	for _, v := range vals {
		parts = append(parts, fmt.Sprintf("%.3f", v))
	}
	r.cmds = append(r.cmds, strings.Join(parts, " "))
}
func (r *recorder) String() string { return strings.Join(r.cmds, ";") }

func TestParseShapes(t *testing.T) {
	if s, err := ParseModuleShape("rounded"); err != nil || s != ModuleRounded {
		t.Fatalf("ParseModuleShape(rounded) = %v, %v", s, err)
	}
	if s, err := ParseModuleShape(""); err != nil || s != ModuleSquare {
		t.Fatalf("ParseModuleShape(\"\") = %v, %v", s, err)
	}
	if _, err := ParseModuleShape("blob"); err == nil {
		t.Fatal("expected error for unknown module shape")
	}
	if s, err := ParseEyeShape("circle"); err != nil || s != EyeCircle {
		t.Fatalf("ParseEyeShape(circle) = %v, %v", s, err)
	}
	if _, err := ParseEyeShape("blob"); err == nil {
		t.Fatal("expected error for unknown eye shape")
	}
	if k, err := ParseGradientKind("radial"); err != nil || k != GradientRadial {
		t.Fatalf("ParseGradientKind(radial) = %v, %v", k, err)
	}
	if _, err := ParseGradientKind("conic"); err == nil {
		t.Fatal("expected error for unknown gradient kind")
	}
}

func TestEyeRects(t *testing.T) {
	rects := EyeRects(21)
	want := [3]Rect{{0, 0, 7, 7}, {14, 0, 7, 7}, {0, 14, 7, 7}}
	if rects != want {
		t.Fatalf("EyeRects(21) = %v, want %v", rects, want)
	}
}

func TestInEye(t *testing.T) {
	for _, tc := range []struct {
		x, y int
		want bool
	}{
		{0, 0, true}, {6, 6, true}, {7, 0, false}, {0, 7, false},
		{14, 0, true}, {13, 0, false}, {20, 6, true},
		{0, 14, true}, {6, 20, true}, {7, 14, false},
		{14, 14, false}, {10, 10, false},
	} {
		if got := InEye(tc.x, tc.y, 21); got != tc.want {
			t.Errorf("InEye(%d, %d, 21) = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

// crossGrid is dark at the centre and its four orthogonal neighbours.
type crossGrid struct{}

func (crossGrid) Size() int { return 5 }
func (crossGrid) IsDark(x, y int) bool {
	return (x == 2 && y >= 1 && y <= 3) || (y == 2 && x >= 1 && x <= 3)
}

var _ render.Grid = crossGrid{}

func TestCornerMask(t *testing.T) {
	g := crossGrid{}
	for _, tc := range []struct {
		x, y int
		want Corners
	}{
		{2, 2, 0},                                         // centre: neighbours on all sides
		{2, 1, CornerTL | CornerTR},                       // top arm: only below is dark
		{2, 3, CornerBL | CornerBR},                       // bottom arm
		{1, 2, CornerTL | CornerBL},                       // left arm
		{3, 2, CornerTR | CornerBR},                       // right arm
		{0, 0, CornerTL | CornerTR | CornerBR | CornerBL}, // isolated light cell: all exposed
	} {
		if got := CornerMask(g, tc.x, tc.y); got != tc.want {
			t.Errorf("CornerMask(%d, %d) = %04b, want %04b", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestAddModuleSquare(t *testing.T) {
	var r recorder
	AddModule(&r, 3, 4, ModuleSquare, 0)
	want := "M 3.000 4.000;L 4.000 4.000;L 4.000 5.000;L 3.000 5.000;L 3.000 4.000;Z"
	if r.String() != want {
		t.Fatalf("square module:\ngot  %s\nwant %s", r.String(), want)
	}
}

func TestAddModuleDot(t *testing.T) {
	var r recorder
	AddModule(&r, 0, 0, ModuleDot, CornerTL) // corners ignored for dots
	s := r.String()
	if !strings.HasPrefix(s, "M 1.000 0.500;C ") {
		t.Fatalf("dot should start at right edge midpoint: %s", s)
	}
	if got := strings.Count(s, "C "); got != 4 {
		t.Fatalf("dot should be 4 cubic arcs, got %d: %s", got, s)
	}
}

// A rounded module with only TL exposed rounds exactly one corner.
func TestAddModuleRoundedPartial(t *testing.T) {
	var r recorder
	AddModule(&r, 0, 0, ModuleRounded, CornerTL)
	s := r.String()
	if got := strings.Count(s, "C "); got != 1 {
		t.Fatalf("expected 1 rounded corner, got %d: %s", got, s)
	}
	// The TL arc must end back at the subpath start (0.5, 0).
	if !strings.HasSuffix(s, "0.500 0.000;Z") {
		t.Fatalf("TL arc should close the outline at (0.5, 0): %s", s)
	}
}

func TestAddEyeFrame(t *testing.T) {
	for shape, wantCubes := range map[EyeShape]int{
		EyeSquare:  0,
		EyeRounded: 8, // 4 outer + 4 inner corners
		EyeCircle:  8, // two 4-arc circles
	} {
		var r recorder
		AddEyeFrame(&r, Rect{0, 0, 7, 7}, shape)
		s := r.String()
		if got := strings.Count(s, "C "); got != wantCubes {
			t.Errorf("%v frame: %d cubics, want %d", shape, got, wantCubes)
		}
		if got := strings.Count(s, "Z"); got != 2 {
			t.Errorf("%v frame: %d subpaths, want 2 (outer + cutout)", shape, got)
		}
	}
}

// The cutout must wind opposite to the outer outline so nonzero filling
// leaves a ring. For the square frame, check directions explicitly.
func TestEyeFrameWinding(t *testing.T) {
	var r recorder
	AddEyeFrame(&r, Rect{0, 0, 7, 7}, EyeSquare)
	cmds := r.cmds
	// Outer: M(0,0) then L(7,0) -> travelling +x along the top (clockwise, y-down).
	if cmds[0] != "M 0.000 0.000" || cmds[1] != "L 7.000 0.000" {
		t.Fatalf("outer should start clockwise from origin: %v", cmds[:2])
	}
	// Cutout: starts at (6,1) then L(1,1) -> travelling -x (counter-clockwise).
	zi := 0
	for i, c := range cmds {
		if c == "Z" {
			zi = i
			break
		}
	}
	if cmds[zi+1] != "M 6.000 1.000" || cmds[zi+2] != "L 1.000 1.000" {
		t.Fatalf("cutout should start counter-clockwise at (6,1): %v", cmds[zi+1:zi+3])
	}
}

func TestAddEyeBall(t *testing.T) {
	var r recorder
	AddEyeBall(&r, Rect{14, 0, 7, 7}, EyeSquare)
	// Ball is the centred 3x3: (16,2)..(19,5).
	if r.cmds[0] != "M 16.000 2.000" {
		t.Fatalf("ball should start at (16,2): %s", r.cmds[0])
	}

	var c recorder
	AddEyeBall(&c, Rect{0, 0, 7, 7}, EyeCircle)
	if !strings.HasPrefix(c.String(), "M 5.000 3.500") {
		t.Fatalf("circle ball should start at (cx+r, cy) = (5, 3.5): %s", c.String())
	}
}

func TestWarnContrast(t *testing.T) {
	var warns []string
	warn := func(f string, a ...any) { warns = append(warns, fmt.Sprintf(f, a...)) }

	WarnContrast(warn, "eye frame color", color.Black, color.White)
	if len(warns) != 0 {
		t.Fatalf("black on white should not warn: %v", warns)
	}
	WarnContrast(warn, "eye frame color", color.RGBA{240, 240, 200, 255}, color.White)
	if len(warns) != 1 || !strings.Contains(warns[0], "eye frame color") {
		t.Fatalf("pale yellow on white should warn once, got %v", warns)
	}
}
