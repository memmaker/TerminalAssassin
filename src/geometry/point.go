package geometry

import (
	"fmt"
	"math"
)

var RelativeSouth = Point{X: 0, Y: 1}
var RelativeNorth = Point{X: 0, Y: -1}
var RelativeEast = Point{X: 1, Y: 0}
var RelativeWest = Point{X: -1, Y: 0}
var PointZero = Point{}

type Point struct {
	X int
	Y int
}

// String returns a string representation of the form "(x,y)".
func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

func NewPointFromString(s string) (Point, error) {
	var x, y int
	_, err := fmt.Sscanf(s, "(%d,%d)", &x, &y)
	if err != nil {
		return Point{}, err
	}
	return Point{X: x, Y: y}, nil
}

// Shift returns a new point with coordinates shifted by (x,y). It's a
// shorthand for p.Add(Point{x,y}).
func (p Point) Shift(x, y int) Point {
	return Point{X: p.X + x, Y: p.Y + y}
}

// Add returns vector p+q.
func (p Point) Add(q Point) Point {
	return Point{X: p.X + q.X, Y: p.Y + q.Y}
}

// Sub returns vector p-q.
func (p Point) Sub(q Point) Point {
	return Point{X: p.X - q.X, Y: p.Y - q.Y}
}

// In reports whether the position is within the given range.
func (p Point) In(rg Rect) bool {
	return p.X >= rg.Min.X && p.X < rg.Max.X && p.Y >= rg.Min.Y && p.Y < rg.Max.Y
}

// Mul returns the vector p*k.
func (p Point) Mul(k int) Point {
	return Point{X: p.X * k, Y: p.Y * k}
}

// Div returns the vector p/k.
func (p Point) Div(k int) Point {
	return Point{X: p.X / k, Y: p.Y / k}
}

func (p Point) AddWrapped(offset, mapSize Point) Point {
	newX := (p.X + offset.X) % mapSize.X
	if newX < 0 {
		newX += mapSize.X
	}
	newY := (p.Y + offset.Y) % mapSize.Y
	if newY < 0 {
		newY += mapSize.Y
	}
	return Point{
		X: newX,
		Y: newY,
	}
}

func (p Point) ToPointF() PointF {
	return PointF{
		X: float64(p.X),
		Y: float64(p.Y),
	}
}

func (p Point) ToHalfWidth() Point {
	return Point{
		X: p.X * 2,
		Y: p.Y,
	}
}

func Distance(p, q Point) float64 {
	// euclidean distance
	return math.Sqrt(float64((p.X-q.X)*(p.X-q.X) + (p.Y-q.Y)*(p.Y-q.Y)))
}
