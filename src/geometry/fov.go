// This file implements line of sight algorithms.

package geometry

import (
	"bytes"
	"encoding/gob"
)

// FOV represents a field of vision. There are two main algorithms available:
// VisionMap and SCCVisionMap, and their multiple-source versions. Both
// algorithms are symmetric (under certain conditions) with expansive walls,
// and fast.
//
// The VisionMap method algorithm is more permissive and only produces
// continuous light rays. Moreover, it allows for non-binary visibility (some
// obstacles may reduce sight range without blocking it completely).
//
// The SCCVisionMap method algorithm is a symmetric shadow casting algorithm
// based on the one described there:
//
//	https://www.albertford.com/shadowcasting/
//
// It offers more euclidean-like geometry, less permissive and with expansive
// shadows, while still being symmetric and fast enough.
//
// FOV elements must be created with NewFOV.
//
// FOV implements the gob.Decoder and gob.Encoder interfaces for easy
// serialization.
type FOV struct {
	innerFOV
}

type innerFOV struct {
	Costs         []float64 // non-binary visibility
	ShadowCasting []bool    // binary visibility
	Lighted       []LightNode
	Visibles      []Point
	RayCache      []LightNode
	Rg            Rect // range of valid positions
	Src           Point
	passable      func(Point) bool
	tiles         []Point
	Capacity      int
}

// NewFOV returns new ready to use field of view with a given range of valid
// positions.
func NewFOV(rg Rect) *FOV {
	fov := &FOV{}
	fov.Rg = rg
	max := fov.Rg.Size()
	fov.Capacity = max.X * max.Y
	return fov
}

// SetRange updates the range used by the field of view. If the size is
// smaller, cached structures will be preserved, otherwise they will be
// reinitialized.
func (fov *FOV) SetRange(rg Rect) {
	fov.Rg = rg
	max := rg.Size()
	if max.X*max.Y <= fov.Capacity {
		return
	}
	nfov := NewFOV(rg)
	*fov = *nfov
}

// Range returns the current FOV's range of positions.
func (fov *FOV) Range() Rect {
	return fov.Rg
}

// GobDecode implements gob.GobDecoder.
func (fov *FOV) GobDecode(bs []byte) error {
	r := bytes.NewReader(bs)
	gd := gob.NewDecoder(r)
	ifov := &innerFOV{}
	err := gd.Decode(ifov)
	if err != nil {
		return err
	}
	fov.innerFOV = *ifov
	return nil
}

// GobEncode implements gob.GobEncoder.
func (fov *FOV) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	ge := gob.NewEncoder(&buf)
	err := ge.Encode(&fov.innerFOV)
	return buf.Bytes(), err
}

// At returns the total ray cost at a given position from the last source(s)
// given to VisionMap or LightMap. It returns a false boolean if the position
// was out of reach.
func (fov *FOV) At(p Point) (float64, bool) {
	if !p.In(fov.Rg) || fov.Costs == nil {
		return 0, false
	}
	cost := fov.Costs[fov.idx(p)]
	if cost <= 0 {
		return cost, false
	}
	return cost - 1, true
}

// Visible returns true if the given position is visible according to the
// last SCCVisionMap call.
func (fov *FOV) Visible(p Point) bool {
	if !p.In(fov.Rg) || fov.ShadowCasting == nil {
		return false
	}
	return fov.ShadowCasting[fov.idx(p)]
}

func (fov *FOV) idx(p Point) int {
	p = p.Sub(fov.Rg.Min)
	w := fov.Rg.Max.X - fov.Rg.Min.X
	return p.Y*w + p.X
}

// Iter iterates a function on the nodes lighted in the last VisionMap or
// LightMap.
func (fov *FOV) Iter(fn func(LightNode)) {
	for _, n := range fov.Lighted {
		fn(n)
	}
}

// IterSSC iterates a function on the nodes lighted in the last SCCVisionMap or
// SCCLightMap.
func (fov *FOV) IterSSC(fn func(p Point)) {
	for _, p := range fov.Visibles {
		fn(p)
	}
}
func (fov *FOV) octantParents(ps []LightNode, src, p Point) []LightNode {
	q := src.Sub(p)
	r := Point{sign(q.X), sign(q.Y)}
	p0 := p.Add(Point{r.X, r.Y})
	c0 := fov.Costs[fov.idx(p0)]
	if c0 > 0 {
		ps = append(ps, LightNode{P: p0, Cost: c0})
	}
	switch {
	case q.X == 0 || q.Y == 0 || Abs(q.X) == Abs(q.Y):
	case Abs(q.X) > Abs(q.Y):
		p1 := p.Add(Point{r.X, 0})
		c1 := fov.Costs[fov.idx(p1)]
		if c1 > 0 {
			ps = append(ps, LightNode{P: p1, Cost: c1})
		}
	default:
		p1 := p.Add(Point{0, r.Y})
		c1 := fov.Costs[fov.idx(p1)]
		if c1 > 0 {
			ps = append(ps, LightNode{P: p1, Cost: c1})
		}
	}
	return ps
}

// From returns the previous position in the light ray to the given position,
// as computed in the last VisionMap call.  If none, it returns false.
func (fov *FOV) From(lt Lighter, to Point) (LightNode, bool) {
	_, ok := fov.At(to)
	if !ok {
		return LightNode{}, false
	}
	ln := fov.from(lt, to)
	if ln.Cost == 0 {
		return LightNode{}, false
	}
	return ln, true
}

func (fov *FOV) from(lt Lighter, to Point) LightNode {
	var pnodesa [2]LightNode
	pnodes := pnodesa[:0]
	pnodes = fov.octantParents(pnodes, fov.Src, to)
	switch len(pnodes) {
	case 0:
		return LightNode{Cost: 0}
	case 1:
		n := pnodes[0]
		cost := lt.Cost(fov.Src, n.P, to)
		return LightNode{P: n.P, Cost: n.Cost + cost}
	default:
		n := pnodes[0]
		m := pnodes[1]
		nCost := lt.Cost(fov.Src, n.P, to)
		mCost := lt.Cost(fov.Src, m.P, to)
		cost0 := n.Cost + nCost
		cost1 := m.Cost + mCost
		if cost0 <= cost1 {
			return LightNode{P: n.P, Cost: cost0}
		}
		return LightNode{P: m.P, Cost: cost1}
	}
}

// Lighter is the interface that captures the requirements for light ray
// propagation.
type Lighter interface {
	// Cost returns the cost of light propagation from a position to
	// an adjacent one given an original source. If you want the resulting
	// FOV to be symmetric, the function should generate symmetric costs
	// for rays in both directions.
	//
	// Note that the FOV algorithm takes care of only providing (from, to)
	// couples that may belong to a same light ray whose source is src,
	// independently of terrain.  This means that the Cost function should
	// essentially take care of terrain considerations, for example giving
	// a cost of 1 if from is a regular ground cell, and a maximal cost if
	// it is a wall, or something in between for fog, bushes or other
	// terrains.
	//
	// As a special case, you normally want Cost(src, src, to) == 1
	// independently of the terrain in src to guarantee symmetry, except
	// for diagonals in certain cases with 4-way movement, because two
	// walls could block vision (for example).
	Cost(src Point, from Point, to Point) float64

	// MaxCost indicates the cost limit at which light cannot propagate
	// anymore from the given source. It should normally be equal to
	// maximum sight or light distance.
	MaxCost(src Point) float64
}

// VisionMap builds a field of vision map for a viewer at src. It returns a
// cached slice of lighted nodes. Values can also be consulted individually
// with At.
//
// The algorithm works in a way that can remind of the Dijkstra algorithm, but
// within each cone between a diagonal and an orthogonal axis (an octant), only
// movements along those two directions are allowed. This allows the algorithm
// to be a simple pass on squares around the player, starting from radius 1
// until line of sight range.
//
// Going from a Point p to a Point q has a cost, which depends
// essentially on the type of terrain in p, and is determined by a Lighter.
//
// The obtained light rays are lines formed using at most two adjacent
// directions: a diagonal and an orthogonal one (for example north east and
// east).
func (fov *FOV) VisionMap(lt Lighter, src Point) []LightNode {
	fov.Lighted = fov.Lighted[:0]
	if !src.In(fov.Rg) {
		return fov.Lighted
	}
	if fov.Costs == nil {
		fov.Costs = make([]float64, fov.Capacity)
	}
	for i := range fov.Costs {
		fov.Costs[i] = 0
	}
	fov.Src = src
	fov.Costs[fov.idx(src)] = 1
	fov.Lighted = append(fov.Lighted, LightNode{P: src, Cost: 0})
	for d := 1; float64(d) <= lt.MaxCost(src); d++ {
		rg := fov.Rg.Intersect(NewRect(src.X-d, src.Y-d+1, src.X+d+1, src.Y+d))
		if src.Y+d < fov.Rg.Max.Y {
			for x := rg.Min.X; x < rg.Max.X; x++ {
				fov.visionUpdate(lt, src, Point{x, src.Y + d})
			}
		}
		if src.Y-d >= fov.Rg.Min.Y {
			for x := rg.Min.X; x < rg.Max.X; x++ {
				fov.visionUpdate(lt, src, Point{x, src.Y - d})
			}
		}
		if src.X+d < fov.Rg.Max.X {
			for y := rg.Min.Y; y < rg.Max.Y; y++ {
				fov.visionUpdate(lt, src, Point{src.X + d, y})
			}
		}
		if src.X-d >= fov.Rg.Min.X {
			for y := rg.Min.Y; y < rg.Max.Y; y++ {
				fov.visionUpdate(lt, src, Point{src.X - d, y})
			}
		}
	}
	return fov.Lighted
}

func (fov *FOV) visionUpdate(lt Lighter, src Point, to Point) {
	n := fov.from(lt, to)
	if n.Cost > 0 {
		fov.Costs[fov.idx(to)] = n.Cost
		fov.Lighted = append(fov.Lighted, LightNode{P: to, Cost: n.Cost - 1})
	}
}

// LightMap builds a lighting map with given light sources. It returs a cached
// slice of lighted nodes. Values can also be consulted with At.
func (fov *FOV) LightMap(lt Lighter, srcs []Point) []LightNode {
	if fov.Costs == nil {
		fov.Costs = make([]float64, fov.Capacity)
	}
	for i := range fov.Costs {
		fov.Costs[i] = 0
	}
	for _, src := range srcs {
		if !src.In(fov.Rg) {
			continue
		}
		fov.Src = src
		fov.Costs[fov.idx(src)] = 1
		for d := 1; float64(d) <= lt.MaxCost(src); d++ {
			rg := fov.Rg.Intersect(NewRect(src.X-d, src.Y-d+1, src.X+d+1, src.Y+d))
			if src.Y+d < fov.Rg.Max.Y {
				for x := rg.Min.X; x < rg.Max.X; x++ {
					fov.lightUpdate(lt, src, Point{x, src.Y + d})
				}
			}
			if src.Y-d >= fov.Rg.Min.Y {
				for x := rg.Min.X; x < rg.Max.X; x++ {
					fov.lightUpdate(lt, src, Point{x, src.Y - d})
				}
			}
			if src.X+d < fov.Rg.Max.X {
				for y := rg.Min.Y; y < rg.Max.Y; y++ {
					fov.lightUpdate(lt, src, Point{src.X + d, y})
				}
			}
			if src.X-d >= fov.Rg.Min.X {
				for y := rg.Min.Y; y < rg.Max.Y; y++ {
					fov.lightUpdate(lt, src, Point{src.X - d, y})
				}
			}
		}
	}
	fov.computeLighted()
	return fov.Lighted
}

func (fov *FOV) lightUpdate(lt Lighter, src Point, to Point) {
	n := fov.from(lt, to)
	if n.Cost <= 0 {
		return
	}
	c1p := &fov.Costs[fov.idx(to)]
	if *c1p > 0 && *c1p <= n.Cost {
		return
	}
	*c1p = n.Cost
}

func (fov *FOV) computeLighted() {
	fov.Lighted = fov.Lighted[:0]
	w := fov.Rg.Max.X - fov.Rg.Min.X
	h := len(fov.Costs) / w
	i := 0
	for y := 0; y < h; y = y + 1 {
		for x := 0; x < w; x, i = x+1, i+1 {
			c := fov.Costs[i]
			if c > 0 {
				fov.Lighted = append(fov.Lighted, LightNode{P: Point{x, y}.Add(fov.Rg.Min), Cost: c - 1})
			}
		}
	}
}

// LightNode represents the information attached to a given position in a light
// map.
type LightNode struct {
	P    Point   // position in the light ray
	Cost float64 // light cost
}

// Ray returns a single light ray from the source (viewer) position to another.
// It should be preceded by a VisionMap call. If the destination position is
// not within the max distance from the source, a nil slice will be returned.
//
// The returned slice is cached for efficiency, so results will be invalidated
// by future calls.
func (fov *FOV) Ray(lt Lighter, to Point) []LightNode {
	_, okTo := fov.At(to)
	if !okTo {
		return nil
	}
	fov.RayCache = fov.RayCache[:0]
	var n LightNode
	for to != fov.Src {
		n = fov.from(lt, to)
		fov.RayCache = append(fov.RayCache, LightNode{P: to, Cost: n.Cost - 1})
		to = n.P
	}
	fov.RayCache = append(fov.RayCache, LightNode{P: fov.Src, Cost: 0})
	for i := range fov.RayCache[:len(fov.RayCache)/2] {
		fov.RayCache[i], fov.RayCache[len(fov.RayCache)-i-1] = fov.RayCache[len(fov.RayCache)-i-1], fov.RayCache[i]
	}
	return fov.RayCache
}

type row struct {
	depth      int
	slopeStart Point // fractional number
	slopeEnd   Point
}

func (r row) tiles(ts []Point, colmin, colmax int) []Point {
	min := r.depth * r.slopeStart.X
	div, rem := min/r.slopeStart.Y, min%r.slopeStart.Y
	min = div
	switch sign(rem) {
	case 1:
		if 2*rem >= r.slopeStart.Y {
			min = div + 1
		}
	case -1:
		if -2*rem > r.slopeStart.Y {
			min = div - 1
		}
	}
	max := r.depth * r.slopeEnd.X
	div, rem = max/r.slopeEnd.Y, max%r.slopeEnd.Y
	max = div
	switch sign(rem) {
	case 1:
		if 2*rem > r.slopeEnd.Y {
			max = div + 1
		}
	case -1:
		if -rem*2 >= r.slopeEnd.Y {
			max = div - 1
		}
	}
	if min < colmin {
		min = colmin
	}
	if max > colmax {
		max = colmax
	}
	for col := min; col < max+1; col++ {
		ts = append(ts, Point{r.depth, col})
	}
	return ts
}

func (r row) next() row {
	r.depth++
	return r
}

func (r row) isSymmetric(tile Point) bool {
	col := tile.Y
	return col*r.slopeStart.Y >= r.depth*r.slopeStart.X &&
		col*r.slopeEnd.Y <= r.depth*r.slopeEnd.X
}

func slopeDiamond(tile Point) Point {
	depth, col := tile.X, tile.Y
	return Point{2*col - 1, 2 * depth}
}

func slopeSquare(tile Point) Point {
	depth, col := tile.X, tile.Y
	return Point{2*col - 1, 2*depth + 1}
}

type quadDir int

const (
	north quadDir = iota
	east
	south
	west
)

type quadrant struct {
	dir quadDir
	p   Point
}

func (qt quadrant) transform(tile Point) Point {
	switch qt.dir {
	case north:
		return Point{qt.p.X + tile.Y, qt.p.Y - tile.X}
	case south:
		return Point{qt.p.X + tile.Y, qt.p.Y + tile.X}
	case east:
		return Point{qt.p.X + tile.X, qt.p.Y + tile.Y}
	default:
		return Point{qt.p.X - tile.X, qt.p.Y + tile.Y}
	}
}

func (qt quadrant) maxCols(rg Rect) (int, int) {
	switch qt.dir {
	case north, south:
		deltaX := qt.p.X - rg.Min.X
		deltaY := rg.Max.X - qt.p.X - 1
		return -deltaX, deltaY
	default:
		deltaX := qt.p.Y - rg.Min.Y
		deltaY := rg.Max.Y - qt.p.Y - 1
		return -deltaX, deltaY
	}
}

func (qt quadrant) maxDepth(rg Rect) int {
	switch qt.dir {
	case north:
		delta := qt.p.Y - rg.Min.Y
		return delta
	case south:
		delta := rg.Max.Y - qt.p.Y - 1
		return delta
	case east:
		delta := rg.Max.X - qt.p.X - 1
		return delta
	default:
		delta := qt.p.X - rg.Min.X
		return delta
	}
}

func (fov *FOV) reveal(qt quadrant, tile Point) {
	p := qt.transform(tile)
	idx := fov.idx(p)
	v := fov.ShadowCasting[idx]
	if !v {
		fov.ShadowCasting[idx] = true
		fov.Visibles = append(fov.Visibles, p)
	}
}

// SSCVisionMap implements symmetric shadow casting algorithm based on
// algorithm described there:
//
//	https://www.albertford.com/shadowcasting/
//
// It returns a cached slice of visible points. Visibility of positions can
// also be checked with the Visible method.  Contrary to VisionMap and
// LightMap, this algorithm can have some discontinuous rays.
func (fov *FOV) SSCVisionMap(src Point, maxDepth int, passable func(p Point) bool, diags bool) []Point {
	if !src.In(fov.Rg) {
		return nil
	}
	if fov.ShadowCasting == nil {
		fov.ShadowCasting = make([]bool, fov.Capacity)
	}
	for i := range fov.ShadowCasting {
		fov.ShadowCasting[i] = false
	}
	fov.passable = passable
	fov.Visibles = fov.Visibles[:0]
	fov.sscVisionMap(src, maxDepth, diags)
	return fov.Visibles
}

func (fov *FOV) sscVisionMap(src Point, maxDepth int, diags bool) {
	idx := fov.idx(src)
	if !fov.ShadowCasting[idx] {
		fov.ShadowCasting[idx] = true
		fov.Visibles = append(fov.Visibles, src)
	}
	for i := 0; i < 4; i++ {
		fov.sscQuadrant(src, maxDepth, quadDir(i), diags)
	}
}

func (fov *FOV) sscQuadrant(src Point, maxDepth int, dir quadDir, diags bool) {
	qt := quadrant{dir: dir, p: src}
	colmin, colmax := qt.maxCols(fov.Rg)
	dmax := qt.maxDepth(fov.Rg)
	if dmax > maxDepth {
		dmax = maxDepth
	}
	if dmax == 0 {
		return
	}
	unreachable := maxDepth + 1
	r := row{
		depth:      1,
		slopeStart: Point{-1, 1},
		slopeEnd:   Point{1, 1},
	}
	rows := []row{r}
	for len(rows) > 0 {
		r := rows[len(rows)-1]
		rows = rows[:len(rows)-1]
		ptile := Point{unreachable, 0}
		fov.tiles = r.tiles(fov.tiles[:0], colmin, colmax)
		for _, tile := range fov.tiles {
			wall := !fov.passable(qt.transform(tile))
			if wall || r.isSymmetric(tile) {
				if diags || tile.X <= 1 && tile.Y == 0 || tile.X > 1 && fov.passable(qt.transform(tile.Shift(-1, 0))) ||
					tile.Y >= 0 && fov.passable(qt.transform(tile.Shift(0, -1))) ||
					tile.Y <= 0 && fov.passable(qt.transform(tile.Shift(0, 1))) {
					fov.reveal(qt, tile)
				}
			}
			if ptile.X == unreachable {
				ptile = tile
				continue
			}
			pwall := !fov.passable(qt.transform(ptile))
			if pwall && !wall {
				if !diags {
					if tile.X < dmax && !fov.passable(qt.transform(tile.Shift(1, 0))) {
						r.slopeStart = slopeSquare(tile.Shift(1, 0))
					} else if tile.X > 1 && !fov.passable(qt.transform(tile.Shift(-1, 0))) {
						r.slopeStart = slopeDiamond(tile.Shift(-1, 1))
					} else {
						r.slopeStart = slopeDiamond(tile)
					}
				} else {
					r.slopeStart = slopeDiamond(tile)
				}
			}
			if !pwall && wall {
				nr := r.next()
				if !diags {
					if tile.X < dmax && !fov.passable(qt.transform(ptile.Shift(1, 0))) {
						nr.slopeEnd = slopeSquare(tile.Shift(1, 0))
					} else if ptile.X > 1 && !fov.passable(qt.transform(ptile.Shift(-1, 0))) {
						nr.slopeEnd = slopeDiamond(ptile.Shift(-1, 0))
					} else {
						nr.slopeEnd = slopeDiamond(tile)
					}
				} else {
					nr.slopeEnd = slopeDiamond(tile)
				}
				if nr.depth <= dmax {
					rows = append(rows, nr)
				}
			}
			ptile = tile
		}
		if ptile.X == unreachable {
			continue
		}
		if fov.passable(qt.transform(ptile)) {
			if r.depth < dmax {
				rows = append(rows, r.next())
			}
		}
	}
}

// SSCLightMap is the equivalent of SSCVisionMap with several sources.
func (fov *FOV) SSCLightMap(srcs []Point, maxDepth int, passable func(p Point) bool, diags bool) []Point {
	if fov.ShadowCasting == nil {
		fov.ShadowCasting = make([]bool, fov.Capacity)
	}
	for i := range fov.ShadowCasting {
		fov.ShadowCasting[i] = false
	}
	fov.passable = passable
	fov.Visibles = fov.Visibles[:0]
	for _, src := range srcs {
		if !src.In(fov.Rg) {
			continue
		}
		fov.sscVisionMap(src, maxDepth, diags)
	}
	return fov.Visibles
}
