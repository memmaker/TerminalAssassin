package ui

import (
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

// everything here is in screen space
type RectSelection struct {
	isDirty            bool
	selectedRegion     geometry.Rect
	startPos           geometry.Point
	onFinished         func(rect geometry.Rect)
	selectionCompleted bool
}

func (r *RectSelection) Update(input services.InputInterface) {
	if r.selectionCompleted {
		return
	}
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.PointerCommand:
			r.handlePointerCommand(typedCmd)
		}
	}

}

func (r *RectSelection) handlePointerCommand(cmd core.PointerCommand) {
	mousePos := cmd.Pos
	switch cmd.Action {
	case core.MouseLeftReleased:
		r.selectedRegion = geometry.NewRect(r.startPos.X, r.startPos.Y, mousePos.X, mousePos.Y)
		r.isDirty = true
		r.selectionCompleted = true
		if r.onFinished != nil {
			r.onFinished(r.selectedRegion)
		}
	case core.MouseMoved:
		r.selectedRegion = geometry.NewRect(r.startPos.X, r.startPos.Y, mousePos.X, mousePos.Y)
		r.isDirty = true
	}
}
func (r *RectSelection) Draw(grid console.CellInterface) {
	if !r.isDirty {
		return
	}

	for x := r.selectedRegion.Min.X; x <= r.selectedRegion.Max.X; x++ {
		drawPosTop := geometry.Point{X: x, Y: r.selectedRegion.Min.Y}
		drawPosBottom := geometry.Point{X: x, Y: r.selectedRegion.Max.Y}
		cellTop := grid.AtSquare(drawPosTop)
		cellBottom := grid.AtSquare(drawPosBottom)
		grid.SetSquare(drawPosTop, cellTop.WithStyle(cellTop.Style.Reversed()))
		grid.SetSquare(drawPosBottom, cellBottom.WithStyle(cellBottom.Style.Reversed()))
	}
	for y := r.selectedRegion.Min.Y; y <= r.selectedRegion.Max.Y; y++ {
		drawPosLeft := geometry.Point{X: r.selectedRegion.Min.X, Y: y}
		drawPosRight := geometry.Point{X: r.selectedRegion.Max.X, Y: y}
		cellLeft := grid.AtSquare(drawPosLeft)
		cellRight := grid.AtSquare(drawPosRight)
		grid.SetSquare(drawPosLeft, cellLeft.WithStyle(cellLeft.Style.Reversed()))
		grid.SetSquare(drawPosRight, cellRight.WithStyle(cellRight.Style.Reversed()))
	}

	r.isDirty = false
}

func (r *RectSelection) SetDirty() {
	r.isDirty = true
}

func NewRectSelection(start geometry.Point) *RectSelection {
	return &RectSelection{
		isDirty:        true,
		startPos:       start,
		selectedRegion: geometry.NewRect(start.X, start.Y, start.X, start.Y),
	}
}

func (r *RectSelection) SetOnFinished(onFinished func(rect geometry.Rect)) {
	r.onFinished = onFinished
}

