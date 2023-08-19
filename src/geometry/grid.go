package geometry

import (
	"fmt"
	"strings"

	"github.com/memmaker/terminal-assassin/common"
)

type Grid struct {
	innerGrid
}

type innerGrid struct {
	Ug *grid // underlying whole grid
	Rg Rect  // range within the whole grid
}

type grid struct {
	Cells  []common.Cell
	Width  int
	Height int
}

func NewGrid(w, h int) Grid {
	gd := Grid{}
	gd.Ug = &grid{}
	if w < 0 || h < 0 {
		panic(fmt.Sprintf("negative dimensions: NewGrid(%d,%d)", w, h))
	}
	gd.Rg.Max = Point{w, h}
	gd.Ug.Width = w
	gd.Ug.Height = h
	gd.Ug.Cells = make([]common.Cell, w*h)
	gd.Fill(common.Cell{Rune: ' ', Style: common.Style{Foreground: common.White, Background: common.Black}})
	return gd
}

// String returns a simplified string representation of the grid's runes,
// without the styling.
func (gd Grid) String() string {
	b := strings.Builder{}
	it := gd.Iterator()
	for it.Next() {
		b.WriteRune(rune(it.Cell().Rune))
		p := it.P()
		if p.X == gd.Rg.Max.X-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

// Bounds returns the range that is covered by this grid slice within the
// underlying original grid.
func (gd Grid) Bounds() Rect {
	return gd.Rg
}

// Range returns the range with Min set to (0,0) and Max set to gd.Size(). It
// may be convenient when using Slice with a range Shift.
func (gd Grid) Range() Rect {
	return gd.Rg.Sub(gd.Rg.Min)
}

// Slice returns a rectangular slice of the grid given by a range relative to
// the grid. If the range is out of bounds of the parent grid, it will be
// reduced to fit to the available space. The returned grid shares memory with
// the parent.
//
// This makes it easy to use relative coordinates when working with UI
// elements.
func (gd Grid) Slice(rg Rect) Grid {
	if rg.Min.X < 0 {
		rg.Min.X = 0
	}
	if rg.Min.Y < 0 {
		rg.Min.Y = 0
	}
	max := gd.Rg.Size()
	if rg.Max.X > max.X {
		rg.Max.X = max.X
	}
	if rg.Max.Y > max.Y {
		rg.Max.Y = max.Y
	}
	min := gd.Rg.Min
	rg.Min = rg.Min.Add(min)
	rg.Max = rg.Max.Add(min)
	return Grid{innerGrid{Ug: gd.Ug, Rg: rg}}
}

// Size returns the grid (width, height) in cells, and is a shorthand for
// gd.Range().Size().
func (gd Grid) Size() Point {
	return gd.Rg.Size()
}

// Resize is similar to Slice, but it only specifies new dimensions, and if the
// range goes beyond the underlying original grid range, it will grow the
// underlying grid. In case of growth, it preserves the content, and new cells
// are initialized to Cell{Rune: ' '}.
func (gd Grid) Resize(w, h int) Grid {
	max := gd.Size()
	ow, oh := max.X, max.Y
	if ow == w && oh == h {
		return gd
	}
	if w <= 0 || h <= 0 {
		gd.Rg.Max = gd.Rg.Min
		return gd
	}
	if gd.Ug == nil {
		gd.Ug = &grid{}
	}
	gd.Rg.Max = gd.Rg.Min.Shift(w, h)
	uh := gd.Ug.Height
	nw := gd.Ug.Width
	if w+gd.Rg.Min.X > gd.Ug.Width {
		nw = w + gd.Rg.Min.X
	}
	nh := uh
	if h+gd.Rg.Min.Y > uh {
		nh = h + gd.Rg.Min.Y
	}
	if nw > gd.Ug.Width || nh > uh {
		ngd := NewGrid(nw, nh)
		ngd.Copy(Grid{innerGrid{Ug: gd.Ug, Rg: NewRect(0, 0, gd.Ug.Width, uh)}})
		*gd.Ug = *ngd.Ug
	}
	return gd
}

// Contains returns true if the given relative position is within the grid.
func (gd Grid) Contains(p Point) bool {
	return p.Add(gd.Rg.Min).In(gd.Rg)
}

// Set draws cell content and styling at a given position in the grid. If the
// position is out of range, the function does nothing.
func (gd Grid) Set(p Point, c common.Cell) {
	q := p.Add(gd.Rg.Min)
	if !q.In(gd.Rg) {
		return
	}
	i := q.Y*gd.Ug.Width + q.X
	gd.Ug.Cells[i] = c
}

// At returns the cell content and styling at a given position. If the position
// is out of range, it returns the zero value.
func (gd Grid) At(p Point) common.Cell {
	q := p.Add(gd.Rg.Min)
	if !q.In(gd.Rg) {
		return common.Cell{}
	}
	i := q.Y*gd.Ug.Width + q.X
	return gd.Ug.Cells[i]
}

// Fill sets the given cell as content for all the grid positions.
func (gd Grid) Fill(c common.Cell) {
	if gd.Ug == nil {
		return
	}
	w := gd.Rg.Max.X - gd.Rg.Min.X
	switch {
	case w >= 8:
		gd.fillcp(c)
	case w == 1:
		gd.fillv(c)
	default:
		gd.fill(c)
	}
}

func (gd Grid) fillcp(c common.Cell) {
	w := gd.Ug.Width
	ymin := gd.Rg.Min.Y * w
	gdw := gd.Rg.Max.X - gd.Rg.Min.X
	cells := gd.Ug.Cells
	for xi := ymin + gd.Rg.Min.X; xi < ymin+gd.Rg.Max.X; xi++ {
		cells[xi] = c
	}
	idxmax := (gd.Rg.Max.Y-1)*w + gd.Rg.Max.X
	for idx := ymin + w + gd.Rg.Min.X; idx < idxmax; idx += w {
		copy(cells[idx:idx+gdw], cells[ymin+gd.Rg.Min.X:ymin+gd.Rg.Max.X])
	}
}

func (gd Grid) fill(c common.Cell) {
	w := gd.Ug.Width
	cells := gd.Ug.Cells
	yimax := gd.Rg.Max.Y * w
	for yi := gd.Rg.Min.Y * w; yi < yimax; yi += w {
		ximax := yi + gd.Rg.Max.X
		for xi := yi + gd.Rg.Min.X; xi < ximax; xi++ {
			cells[xi] = c
		}
	}
}

func (gd Grid) fillv(c common.Cell) {
	w := gd.Ug.Width
	cells := gd.Ug.Cells
	ximax := gd.Rg.Max.Y*w + gd.Rg.Min.X
	for xi := gd.Rg.Min.Y*w + gd.Rg.Min.X; xi < ximax; xi += w {
		cells[xi] = c
	}
}

// Iter iterates a function on all the grid positions and cells.
func (gd Grid) Iter(fn func(Point, common.Cell)) {
	if gd.Ug == nil {
		return
	}
	w := gd.Ug.Width
	yimax := gd.Rg.Max.Y * w
	cells := gd.Ug.Cells
	for y, yi := 0, gd.Rg.Min.Y*w; yi < yimax; y, yi = y+1, yi+w {
		ximax := yi + gd.Rg.Max.X
		for x, xi := 0, yi+gd.Rg.Min.X; xi < ximax; x, xi = x+1, xi+1 {
			c := cells[xi]
			p := Point{X: x, Y: y}
			fn(p, c)
		}
	}
}

// Map updates the grid content using the given mapping function.
func (gd Grid) Map(fn func(Point, common.Cell) common.Cell) {
	if gd.Ug == nil {
		return
	}
	w := gd.Ug.Width
	cells := gd.Ug.Cells
	yimax := gd.Rg.Max.Y * w
	for y, yi := 0, gd.Rg.Min.Y*w; yi < yimax; y, yi = y+1, yi+w {
		ximax := yi + gd.Rg.Max.X
		for x, xi := 0, yi+gd.Rg.Min.X; xi < ximax; x, xi = x+1, xi+1 {
			c := cells[xi]
			p := Point{X: x, Y: y}
			cells[xi] = fn(p, c)
		}
	}
}

// Copy copies elements from a source grid src into the destination grid gd,
// and returns the copied grid-slice size, which is the minimum of both grids
// for each dimension. The result is independent of whether the two grids
// referenced memory overlaps or not.
func (gd Grid) Copy(src Grid) Point {
	if gd.Ug == nil {
		return Point{}
	}
	if gd.Ug != src.Ug {
		if src.Rg.Max.X-src.Rg.Min.X <= 4 {
			return gd.cpv(src)
		}
		return gd.cp(src)
	}
	if gd.Rg == src.Rg {
		return gd.Rg.Size()
	}
	if !gd.Rg.Overlaps(src.Rg) || gd.Rg.Min.Y <= src.Rg.Min.Y {
		return gd.cp(src)
	}
	return gd.cprev(src)
}

func (gd Grid) cp(src Grid) Point {
	w := gd.Ug.Width
	wsrc := src.Ug.Width
	max := gd.Range().Intersect(src.Range()).Size()
	idxmin := gd.Rg.Min.Y*w + gd.Rg.Min.X
	idxsrcmin := src.Rg.Min.Y*w + src.Rg.Min.X
	idxmax := (gd.Rg.Min.Y + max.Y) * w
	for idx, idxsrc := idxmin, idxsrcmin; idx < idxmax; idx, idxsrc = idx+w, idxsrc+wsrc {
		copy(gd.Ug.Cells[idx:idx+max.X], src.Ug.Cells[idxsrc:idxsrc+max.X])
	}
	return max
}

func (gd Grid) cpv(src Grid) Point {
	w := gd.Ug.Width
	wsrc := src.Ug.Width
	max := gd.Range().Intersect(src.Range()).Size()
	yimax := (gd.Rg.Min.Y + max.Y) * w
	cells := gd.Ug.Cells
	srccells := src.Ug.Cells
	for yi, yisrc := gd.Rg.Min.Y*w, src.Rg.Min.Y*wsrc; yi < yimax; yi, yisrc = yi+w, yisrc+wsrc {
		ximax := yi + max.X
		for xi, xisrc := yi+gd.Rg.Min.X, yisrc+src.Rg.Min.X; xi < ximax; xi, xisrc = xi+1, xisrc+1 {
			cells[xi] = srccells[xisrc]
		}
	}
	return max
}

func (gd Grid) cprev(src Grid) Point {
	w := gd.Ug.Width
	wsrc := src.Ug.Width
	max := gd.Range().Intersect(src.Range()).Size()
	idxmax := (gd.Rg.Min.Y+max.Y-1)*w + gd.Rg.Min.X
	idxsrcmax := (src.Rg.Min.Y+max.Y-1)*w + src.Rg.Min.X
	idxmin := gd.Rg.Min.Y * w
	for idx, idxsrc := idxmax, idxsrcmax; idx >= idxmin; idx, idxsrc = idx-w, idxsrc-wsrc {
		copy(gd.Ug.Cells[idx:idx+max.X], src.Ug.Cells[idxsrc:idxsrc+max.X])
	}
	return max
}

// GridIterator represents a stateful iterator for a grid. They are created
// with the Iterator method.
type GridIterator struct {
	cells  []common.Cell // grid cells
	p      Point         // iterator's current position
	max    Point         // last position
	i      int           // current position's index
	w      int           // underlying grid's width
	nlstep int           // newline step
	rg     Rect          // grid range
}

// Iterator returns an iterator that can be used to iterate on the grid. It may
// be convenient when more flexibility than the provided by the other iteration
// functions is needed. It is used as follows:
//
//	it := gd.Iterator()
//	for it.Next() {
//		// call it.P() or it.Cell() or it.SetSquare() as appropriate
//	}
func (gd Grid) Iterator() GridIterator {
	if gd.Ug == nil {
		return GridIterator{}
	}
	w := gd.Ug.Width
	it := GridIterator{
		w:      w,
		cells:  gd.Ug.Cells,
		max:    gd.Size().Shift(-1, -1),
		rg:     gd.Rg,
		nlstep: gd.Rg.Min.X + (w - gd.Rg.Max.X + 1),
	}
	it.Reset()
	return it
}

// Reset resets the iterator's state so that it can be used again.
func (it *GridIterator) Reset() {
	it.p = Point{-1, 0}
	it.i = it.rg.Min.Y*it.w + it.rg.Min.X - 1
}

// Next advances the iterator the next position in the grid.
func (it *GridIterator) Next() bool {
	if it.p.X < it.max.X {
		it.p.X++
		it.i++
		return true
	}
	if it.p.Y < it.max.Y {
		it.p.Y++
		it.p.X = 0
		it.i += it.nlstep
		return true
	}
	return false
}

// P returns the iterator's current position.
func (it *GridIterator) P() Point {
	return it.p
}

// SetP sets the iterator's current position.
func (it *GridIterator) SetP(p Point) {
	q := p.Add(it.rg.Min)
	if !q.In(it.rg) {
		return
	}
	it.p = p
	it.i = q.Y*it.w + q.X
}

// Cell returns the Cell in the grid at the iterator's current position.
func (it *GridIterator) Cell() common.Cell {
	return it.cells[it.i]
}

// SetCell updates the grid cell at the iterator's current position.
func (it *GridIterator) SetCell(c common.Cell) {
	it.cells[it.i] = c
}
