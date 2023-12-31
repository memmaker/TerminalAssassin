// Package paths provides utilities for efficient pathfinding in rectangular
// maps.
package geometry

import (
	"bytes"
	"encoding/gob"
)

// PathRange allows for efficient path finding within a range. It caches
// structures, so that they can be reused without further memory allocations.
//
// It implements gob.Encoder and gob.Decoder for easy serialization.
type PathRange struct {
	pathRange
}

type pathRange struct {
	diags               bool             // JPS diagonal movement
	passable            func(Point) bool // JPS passable function
	AstarNodes          *nodeMap
	DijkstraNodes       *nodeMap // dijkstra map
	DijkstraIterNodes   []Node
	BfMap               []int  // breadth first map
	BfQueue             []Node // map numbers for caching
	CC                  []int  // connected components
	CCStack             []int
	CCIterCache         []Point
	AstarQueue          priorityQueue
	DijkstraQueue       priorityQueue
	Rg                  Rect
	DijkstraUnreachable int
	BfUnreachable       int // last maxcost + 1
	BfEnd               int // bf map last index
	W                   int // path range width
	Capacity            int
}

// GobDecode implements gob.GobDecoder.
func (pr *PathRange) GobDecode(bs []byte) error {
	r := bytes.NewReader(bs)
	gd := gob.NewDecoder(r)
	ipr := &pathRange{}
	err := gd.Decode(ipr)
	if err != nil {
		return err
	}
	pr.pathRange = *ipr
	return nil
}

// GobEncode implements gob.GobEncoder.
func (pr *PathRange) GobEncode() ([]byte, error) {
	buf := bytes.Buffer{}
	ge := gob.NewEncoder(&buf)
	err := ge.Encode(&pr.pathRange)
	return buf.Bytes(), err
}

// NewPathRange returns a new PathFinder for positions in a given range,
// such as the range occupied by the whole map, or a part of it.
func NewPathRange(rg Rect) *PathRange {
	pr := &PathRange{}
	pr.Rg = rg
	max := pr.Rg.Size()
	pr.W = max.X
	pr.Capacity = max.X * max.Y
	return pr
}

// SetRange updates the range used by the PathFinder. If the size is smaller,
// cached structures will be preserved, otherwise they will be reinitialized.
func (pr *PathRange) SetRange(rg Rect) {
	pr.Rg = rg
	max := rg.Size()
	if max.X*max.Y <= pr.Capacity {
		return
	}
	npr := NewPathRange(rg)
	*pr = *npr
}

// Range returns the current PathRange's range of positions.
func (pr *PathRange) Range() Rect {
	return pr.Rg
}

func (pr *PathRange) idx(p Point) int {
	p = p.Sub(pr.Rg.Min)
	return p.Y*pr.W + p.X
}

func (nm nodeMap) get(pr *PathRange, p Point) *node {
	idx := pr.idx(p)
	n := &nm.Nodes[idx]
	if n.CacheIndex != nm.Idx {
		nm.Nodes[idx] = node{P: p, CacheIndex: nm.Idx}
	}
	return n
}

func (nm nodeMap) at(pr *PathRange, p Point) *node {
	n := &nm.Nodes[pr.idx(p)]
	if n.CacheIndex != nm.Idx {
		return nil
	}
	return n
}

type node struct {
	Open       bool
	Closed     bool
	Parent     Point
	P          Point
	Cost       int
	Rank       int
	Idx        int
	Estimation int
	CacheIndex int
}

type nodeMap struct {
	Nodes []node
	Idx   int
}

// priorityQueue implements a custom heap-like interface with node elements.
type priorityQueue []*node

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].Rank < pq[j].Rank || pq[i].Rank == pq[j].Rank && pq[i].Estimation < pq[j].Estimation
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Idx = i
	pq[j].Idx = j
}

func (pq *priorityQueue) Push(n *node) {
	i := len(*pq)
	n.Idx = i
	*pq = append(*pq, n)
}

func (pq *priorityQueue) Pop() *node {
	old := *pq
	i := len(old)
	n := old[i-1]
	n.Idx = -1
	*pq = old[0 : i-1]
	return n
}
