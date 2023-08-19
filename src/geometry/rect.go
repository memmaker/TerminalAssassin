package geometry

import "fmt"

// Rect represents a rectangle in a grid that contains all the positions P
// such that Min <= P < Max coordinate-wise. A range is well-formed if Min <=
// Max. When non-empty, Min represents the upper-left position in the range,
// and Max-(1,1) the lower-right one.
type Rect struct {
	Min, Max Point
}

// NewRect returns a new Rect with coordinates (x0, y0) for Min and (x1, y1)
// for Max. The returned range will have minimum and maximum coordinates
// swapped if necessary, so that the range is well-formed.
func NewRect(x0, y0, x1, y1 int) Rect {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	return Rect{Min: Point{X: x0, Y: y0}, Max: Point{X: x1, Y: y1}}
}

// String returns a string representation of the form "(x0,y0)-(x1,y1)".
func (rg Rect) String() string {
	return fmt.Sprintf("%s-%s", rg.Min, rg.Max)
}

// Size returns the (width, height) of the range in cells.
func (rg Rect) Size() Point {
	return rg.Max.Sub(rg.Min)
}

// Shift returns a new range with coordinates shifted by (x0,y0) and (x1,y1).
func (rg Rect) Shift(x0, y0, x1, y1 int) Rect {
	rg = Rect{Min: rg.Min.Shift(x0, y0), Max: rg.Max.Shift(x1, y1)}
	if rg.Empty() {
		return Rect{}
	}
	return rg
}

// Line reduces the range to relative line y, or an empty range if out of
// bounds.
func (rg Rect) Line(y int) Rect {
	if rg.Min.Shift(0, y).In(rg) {
		rg.Min.Y = rg.Min.Y + y
		rg.Max.Y = rg.Min.Y + 1
	} else {
		rg = Rect{}
	}
	return rg
}

// Lines reduces the range to relative lines between y0 (included) and y1
// (excluded), or an empty range if out of bounds.
func (rg Rect) Lines(y0, y1 int) Rect {
	nrg := rg
	nrg.Min.Y = rg.Min.Y + y0
	nrg.Max.Y = rg.Min.Y + y1
	return rg.Intersect(nrg)
}

// Column reduces the range to relative column x, or an empty range if out of
// bounds.
func (rg Rect) Column(x int) Rect {
	if rg.Min.Shift(x, 0).In(rg) {
		rg.Min.X = rg.Min.X + x
		rg.Max.X = rg.Min.X + 1
	} else {
		rg = Rect{}
	}
	return rg
}

// Columns reduces the range to relative columns between x0 (included) and x1
// (excluded), or an empty range if out of bounds.
func (rg Rect) Columns(x0, x1 int) Rect {
	nrg := rg
	nrg.Min.X = rg.Min.X + x0
	nrg.Max.X = rg.Min.X + x1
	return rg.Intersect(nrg)
}

// Empty reports whether the range contains no positions.
func (rg Rect) Empty() bool {
	return rg.Min.X >= rg.Max.X || rg.Min.Y >= rg.Max.Y
}

// Eq reports whether the two ranges containt the same set of points. All empty
// ranges are considered equal.
func (rg Rect) Eq(r Rect) bool {
	return rg == r || rg.Empty() && r.Empty()
}

// Sub returns a range of same size translated by -p.
func (rg Rect) Sub(p Point) Rect {
	rg.Max = rg.Max.Sub(p)
	rg.Min = rg.Min.Sub(p)
	return rg
}

// Add returns a range of same size translated by +p.
func (rg Rect) Add(p Point) Rect {
	rg.Max = rg.Max.Add(p)
	rg.Min = rg.Min.Add(p)
	return rg
}

// Intersect returns the largest range contained both by rg and r. If the two
// ranges do not overlap, the zero range will be returned.
func (rg Rect) Intersect(r Rect) Rect {
	if rg.Max.X > r.Max.X {
		rg.Max.X = r.Max.X
	}
	if rg.Max.Y > r.Max.Y {
		rg.Max.Y = r.Max.Y
	}
	if rg.Min.X < r.Min.X {
		rg.Min.X = r.Min.X
	}
	if rg.Min.Y < r.Min.Y {
		rg.Min.Y = r.Min.Y
	}
	if rg.Min.X >= rg.Max.X || rg.Min.Y >= rg.Max.Y {
		return Rect{}
	}
	return rg
}

// Union returns the smallest range containing both rg and r.
func (rg Rect) Union(r Rect) Rect {
	if rg.Max.X < r.Max.X {
		rg.Max.X = r.Max.X
	}
	if rg.Max.Y < r.Max.Y {
		rg.Max.Y = r.Max.Y
	}
	if rg.Min.X > r.Min.X {
		rg.Min.X = r.Min.X
	}
	if rg.Min.Y > r.Min.Y {
		rg.Min.Y = r.Min.Y
	}
	return rg
}

// Overlaps reports whether the two ranges have a non-zero intersection.
func (rg Rect) Overlaps(r Rect) bool {
	return !rg.Intersect(r).Empty()
}

// In reports whether range rg is completely contained in range r.
func (rg Rect) In(r Rect) bool {
	return rg.Intersect(r) == rg
}

// Iter calls a given function for all the positions of the range.
func (rg Rect) Iter(fn func(Point)) {
	for y := rg.Min.Y; y < rg.Max.Y; y++ {
		for x := rg.Min.X; x < rg.Max.X; x++ {
			p := Point{X: x, Y: y}
			fn(p)
		}
	}
}

func (rg Rect) Mid() Point {
	return Point{X: (rg.Min.X + rg.Max.X) / 2, Y: (rg.Min.Y + rg.Max.Y) / 2}
}

func (rg Rect) Contains(position Point) bool {
	return position.X >= rg.Min.X && position.X < rg.Max.X && position.Y >= rg.Min.Y && position.Y < rg.Max.Y
}

func (rg Rect) ToHalfWidth() Rect {
	// the new rect will start at the 2*x position of the old rect
	// and it will have double the width
	newWidth := rg.Size().X * 2
	newXStart := rg.Min.X * 2

	return Rect{
		Min: Point{X: newXStart, Y: rg.Min.Y},
		Max: Point{X: newXStart + newWidth, Y: rg.Max.Y},
	}
}
