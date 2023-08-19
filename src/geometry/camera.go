package geometry

type Camera struct {
	ViewPort Rect
}

func NewCamera(x0, y0, x1, y1 int) *Camera {
	return &Camera{ViewPort: NewRect(x0, y0, x1, y1)}
}
func (c *Camera) WorldToScreen(p Point) Point {
	return p.Sub(c.ViewPort.Min)
}

func (c *Camera) ScreenToWorld(p Point) Point {
	return p.Add(c.ViewPort.Min)
}
func (c *Camera) CenterOn(targetWorldPosition Point, mapWidth int, mapHeight int) {
	centerOfCameraInWorld := c.ViewPort.Mid()
	deltaMovement := targetWorldPosition.Sub(centerOfCameraInWorld)
	c.MoveBy(deltaMovement, mapWidth, mapHeight)
}
func (c *Camera) MoveBy(delta Point, mapWidth int, mapHeight int) {
	deltaMin := c.ViewPort.Min.Add(delta)
	deltaMax := c.ViewPort.Max.Add(delta)
	if deltaMin.X < 0 {
		delta.X = delta.X - deltaMin.X
	}
	if deltaMin.Y < 0 {
		delta.Y = delta.Y - deltaMin.Y
	}
	if deltaMax.X > mapWidth {
		delta.X = delta.X - (deltaMax.X - mapWidth)
	}
	if deltaMax.Y > mapHeight {
		delta.Y = delta.Y - (deltaMax.Y - mapHeight)
	}
	c.ViewPort = c.ViewPort.Add(delta)
}
