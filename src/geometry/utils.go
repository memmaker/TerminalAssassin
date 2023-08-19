package geometry

import (
	"compress/gzip"
	"encoding/gob"
	"io"
	"math"
)

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func IntMax(x int, y int) int {
	if x > y {
		return x
	}
	return y
}
func UIntMax(x uint64, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}

func UIntMin(x uint64, y uint64) uint64 {
	if x < y {
		return x
	}
	return y
}

// LineOfSight returns a list of points that are in the line of sight between the source and the target.
// It will include the source as the first point.
// It will include the last position which did not return true for isTransparent.
// If there is a LoS, the last point will be the target.
func LineOfSight(source Point, target Point, canTraverse func(p Point) bool) []Point {
	results := make([]Point, 0)
	x0, y0 := source.X, source.Y
	x1, y1 := target.X, target.Y
	dx := Abs(x1 - x0)
	var sx, sy int
	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}
	dy := -Abs(y1 - y0)
	err := dx + dy
	var e2 int /* error value e_xy */
	for true {
		gruidPoint := Point{X: x0, Y: y0}
		pointIsVisible := canTraverse(gruidPoint)
		results = append(results, gruidPoint)
		if (x0 == x1 && y0 == y1) || !pointIsVisible {
			break
		}
		e2 = 2 * err
		if e2 > dy {
			err += dy
			x0 += sx
		} /* e_xy+e_x > 0 */
		if e2 < dx {
			err += dx
			y0 += sy
		} /* e_xy+e_y < 0 */
	}
	return results
}

func isLeft(lineStart Point, lineEnd Point, pointToCheck Point) bool {
	return ((lineEnd.X-lineStart.X)*(pointToCheck.Y-lineStart.Y) - (lineEnd.Y-lineStart.Y)*(pointToCheck.X-lineStart.X)) > 0
}

func isRight(lineStart Point, lineEnd Point, pointToCheck Point) bool {
	return ((lineEnd.X-lineStart.X)*(pointToCheck.Y-lineStart.Y) - (lineEnd.Y-lineStart.Y)*(pointToCheck.X-lineStart.X)) < 0
}

func radiansToVector(radians float64) Point {
	x := int(math.Cos(radians) * 1000)
	y := int(math.Sin(radians) * 1000)
	return Point{X: x, Y: y}
}

func degreesToRadians(degrees float64) float64 {
	radians := degrees * math.Pi / 180
	// normalise
	if radians < 0 {
		radians += 2 * math.Pi
	} else if radians > 2*math.Pi {
		radians -= 2 * math.Pi
	}
	return radians
}

func addPoints(p Point, q Point) Point {
	return Point{X: p.X + q.X, Y: p.Y + q.Y}
}

type CompassDirection float64

const (
	East      CompassDirection = 0
	SouthEast CompassDirection = 45
	South     CompassDirection = 90
	SouthWest CompassDirection = 135
	West      CompassDirection = 180
	NorthWest CompassDirection = 225
	North     CompassDirection = 270
	NorthEast CompassDirection = 315
)

func DirectionVectorToAngleInDegrees(direction Point) float64 {
	rawDirection := math.Atan2(float64(direction.Y), float64(direction.X)) * 180 / math.Pi
	if rawDirection < 0 {
		rawDirection += 360
	}
	return rawDirection
}

func DirectionVectorToAngleInDegreesF(directionX float64, directionY float64) float64 {
	rawDirection := math.Atan2(directionY, directionX) * 180 / math.Pi
	if rawDirection < 0 {
		rawDirection += 360
	}
	return rawDirection
}

func InVisionCone(sourcePos Point, targetPos Point, fovLeftBorder Point, fovRightBorder Point) bool {
	isLeftOfRightBorder := isLeft(sourcePos, fovRightBorder, targetPos)
	isRightOfLeftBorder := isRight(sourcePos, fovLeftBorder, targetPos)
	return isLeftOfRightBorder && isRightOfLeftBorder
}
func GetLeftAndRightBorderOfVisionCone(sourcePos Point, directionInDegrees float64, fovInDegrees float64) (Point, Point) {
	vectorDirLeft := radiansToVector(degreesToRadians(directionInDegrees + (fovInDegrees / 2)))
	vectorDirRight := radiansToVector(degreesToRadians(directionInDegrees - (fovInDegrees / 2)))
	fovLeftBorder := addPoints(vectorDirLeft, sourcePos)
	fovRightBorder := addPoints(vectorDirRight, sourcePos)
	return fovLeftBorder, fovRightBorder
}

func RotateVector(point Point, angleInDegrees float64) Point {
	radians := float64(angleInDegrees) * (math.Pi / 180.0)
	cos := math.Cos(radians)
	sin := math.Sin(radians)
	x := float64(point.X)*cos - float64(point.Y)*sin
	y := float64(point.X)*sin + float64(point.Y)*cos
	return Point{X: int(x), Y: int(y)}
}

type FrameEncoder struct {
	gzw *gzip.Writer
	gbe *gob.Encoder
}

func NewFrameEncoder(w io.Writer) *FrameEncoder {
	fe := &FrameEncoder{}
	fe.gzw = gzip.NewWriter(w)
	fe.gbe = gob.NewEncoder(fe.gzw)
	return fe
}

var OnePointF = PointF{X: 1, Y: 1}

type PointF struct {
	X float64
	Y float64
}

func (f PointF) Mul(scalar float64) PointF {
	return PointF{X: f.X * scalar, Y: f.Y * scalar}
}

func (f PointF) MulInt(scalar int) PointF {
	return f.Mul(float64(scalar))
}
func (f PointF) Normalize() PointF {
	length := math.Sqrt(f.X*f.X + f.Y*f.Y)
	if length == 0 {
		return f
	}
	return PointF{X: f.X / length, Y: f.Y / length}
}

func (f PointF) Rotate(degrees int) PointF {
	radians := float64(degrees) * (math.Pi / 180.0)
	cos := math.Cos(radians)
	sin := math.Sin(radians)
	x := f.X*cos - f.Y*sin
	y := f.X*sin + f.Y*cos
	return PointF{X: x, Y: y}
}

func (f PointF) ToPoint() Point {
	return Point{X: int(f.X), Y: int(f.Y)}
}

func (f PointF) Add(other PointF) PointF {
	return PointF{X: f.X + other.X, Y: f.Y + other.Y}
}

func (f PointF) Div(value float64) PointF {
	return PointF{X: f.X / value, Y: f.Y / value}
}

func NewPointF(sub Point) PointF {
	return PointF{X: float64(sub.X), Y: float64(sub.Y)}
}

func VectorInDirectionWithLength(startPos Point, directionX float64, directionY float64, length int) Point {

	// normalize direction vector
	directionLength := math.Sqrt(directionX*directionX + directionY*directionY)
	if directionLength == 0 {
		return startPos
	}
	directionX /= directionLength
	directionY /= directionLength
	startX := float64(startPos.X)
	startY := float64(startPos.Y)
	endX := startX + directionX*float64(length)
	endY := startY + directionY*float64(length)
	return Point{X: int(endX), Y: int(endY)}
}

func MapTo[A, B any](xs []A, f func(A) B) []B {
	ys := make([]B, len(xs))
	for i, x := range xs {
		ys[i] = f(x)
	}
	return ys
}
func Log(msg string) {
	// append to Log file
	// deactivating Log for now
	/*
		file, err := os.OpenFile("Log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer file.UnloadAll()
		timeStamp := time.Now().Format("2006-01-02 15:04:05.000")
		if _, err := file.WriteString(timeStamp + " " + msg + "\n"); err != nil {
			panic(err)
		}
	*/
}
