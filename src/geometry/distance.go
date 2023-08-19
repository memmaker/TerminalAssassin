package geometry

func DistanceSquared(p, q Point) int {
	p = p.Sub(q)
	return p.X*p.X + p.Y*p.Y
}

// DistanceManhattan computes the taxicab norm (1-norm). See:
//
//	https://en.wikipedia.org/wiki/Taxicab_geometry
//
// It can often be used as A* distance heuristic when 4-way movement is used.
func DistanceManhattan(p, q Point) int {
	p = p.Sub(q)
	return Abs(p.X) + Abs(p.Y)
}

// DistanceChebyshev computes the maximum norm (infinity-norm). See:
//
//	https://en.wikipedia.org/wiki/Chebyshev_distance
//
// It can often be used as A* distance heuristic when 8-way movement is used.
func DistanceChebyshev(p, q Point) int {
	p = p.Sub(q)
	return max(Abs(p.X), Abs(p.Y))
}

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}
