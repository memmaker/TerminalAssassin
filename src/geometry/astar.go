// code of this file is a strongly modified version of code from
// github.com/beefsack/go-astar, which has the following license:
//
// Copyright (c) 2014 Michael Charles Alexander
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package geometry

// Astar is the interface that allows to use the A* algorithm used by the
// AstarPath function.
type Astar interface {
	Dijkstra

	// Estimation offers an estimation cost for a path from a position to
	// another one. The estimation should always give a value lower or
	// equal to the cost of the best possible path.
	Estimation(Point, Point) int
}

// AstarPath returns a path from a position to another, including thoses
// positions, in the path order. It returns nil if no path was found.
func (pr *PathRange) AstarPath(ast Astar, from, to Point) []Point {
	if !from.In(pr.Rg) || !to.In(pr.Rg) {
		return nil
	}
	pr.initAstar()
	nm := pr.AstarNodes
	nm.Idx++
	defer checkNodesIdx(nm)
	nqs := pr.AstarQueue[:0]
	nq := &nqs
	pqInit(nq)
	fromNode := nm.get(pr, from)
	fromNode.Open = true
	fromNode.Estimation = ast.Estimation(from, to)
	pqPush(nq, fromNode)
	for {
		if nq.Len() == 0 {
			// There's no path.
			return nil
		}
		n := pqPop(nq)
		n.Open = false
		n.Closed = true

		if n.P == to {
			// Found a path to the goal.
			path := []Point{}
			pn := n
			path = append(path, pn.P)
			for {
				if pn.P == from {
					break
				}
				pn = nm.at(pr, pn.Parent)
				path = append(path, pn.P)
			}
			for i := range path[:len(path)/2] {
				path[i], path[len(path)-i-1] = path[len(path)-i-1], path[i]
			}
			return path
		}

		for _, q := range ast.Neighbors(n.P) {
			if !q.In(pr.Rg) {
				continue
			}
			cost := n.Cost + ast.Cost(n.P, q)
			nbNode := nm.get(pr, q)
			if cost < nbNode.Cost {
				if nbNode.Open {
					pqRemove(nq, nbNode.Idx)
				}
				nbNode.Open = false
				nbNode.Closed = false
			}
			if !nbNode.Open && !nbNode.Closed {
				nbNode.Cost = cost
				nbNode.Open = true
				nbNode.Estimation = ast.Estimation(q, to)
				nbNode.Rank = cost + nbNode.Estimation
				nbNode.Parent = n.P
				pqPush(nq, nbNode)
			}
		}
	}
}

func (pr *PathRange) initAstar() {
	if pr.AstarNodes == nil {
		pr.AstarNodes = &nodeMap{}
		max := pr.Rg.Size()
		pr.AstarNodes.Nodes = make([]node, max.X*max.Y)
		pr.AstarQueue = make(priorityQueue, 0, max.X*max.Y)
	}
}

func checkNodesIdx(nm *nodeMap) {
	if nm.Idx+1 > 0 {
		return
	}
	for i, n := range nm.Nodes {
		idx := 0
		if n.Idx == nm.Idx {
			idx = 1
		}
		n.Idx = idx
		nm.Nodes[i] = n
	}
	nm.Idx = 1
}
