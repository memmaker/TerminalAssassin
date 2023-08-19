package console

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/geometry"
)

type OverlayState int

const (
	BothTransparent OverlayState = iota
	RightTransparent
	LeftTransparent
	BothOpaque
)

type ConditionalDeltaIterator struct {
	DeltaIterator
	overlayState       func(squarePos geometry.Point) OverlayState
	stateSource        *geometry.Grid
	transparencyDeltas *AccumulatedFrame
	setOverlayDirty    func(halfwidthPos geometry.Point)
}

func (c ConditionalDeltaIterator) Count() int {
	return len(*c.deltaSource) + len(*c.transparencyDeltas)
}
func (c ConditionalDeltaIterator) Iter(f func(geometry.Point, common.Cell)) {
	for pos, cell := range *c.transparencyDeltas {
		if cell.IsTransparent() { // overlay has been changed to transparent, so redraw the square
			squarePos := geometry.Point{X: pos.X / 2, Y: pos.Y}
			if _, ok := (*c.deltaSource)[squarePos]; !ok {
				(*c.deltaSource)[squarePos] = c.stateSource.At(squarePos)
			}
		}
	}
	for pos, cell := range *c.deltaSource {
		overlayState := c.overlayState(pos)
		if overlayState == BothOpaque {
			continue
		}
		f(pos, cell)
		if overlayState == RightTransparent {
			c.setOverlayDirty(geometry.Point{X: pos.X * 2, Y: pos.Y})
		} else if overlayState == LeftTransparent {
			c.setOverlayDirty(geometry.Point{X: pos.X*2 + 1, Y: pos.Y})
		}
	}
}

func (c *Console) GetSquareDeltaIterator() CellIterator {
	return ConditionalDeltaIterator{
		DeltaIterator: DeltaIterator{
			deltaSource: &c.squareAccumulatedFrame,
			doneFunc: func() {
				c.squareAccumulatedFrame = make(map[geometry.Point]common.Cell, c.squareCurrentGrid.Size().X*c.squareCurrentGrid.Size().Y)
			}},
		overlayState: func(squarePos geometry.Point) OverlayState {
			return c.isHWLayerTransparentAt(squarePos)
		},
		setOverlayDirty: func(halfwidthPos geometry.Point) {
			if _, ok := c.halfWidthAccumulatedFrame[halfwidthPos]; !ok {
				c.halfWidthAccumulatedFrame[halfwidthPos] = c.halfWidthCurrentGrid.At(halfwidthPos)
			}
		},
		stateSource:        &c.squareCurrentGrid, // Or should this be the currentGrid?
		transparencyDeltas: &c.halfWidthAccumulatedFrame,
	}
}

func (c *Console) GetHalfWidthDeltaIterator() CellIterator {
	return DeltaIterator{deltaSource: &c.halfWidthAccumulatedFrame, doneFunc: func() {
		c.halfWidthAccumulatedFrame = make(map[geometry.Point]common.Cell, c.halfWidthCurrentGrid.Size().X*c.halfWidthCurrentGrid.Size().Y)
	}}
}

type DeltaIterator struct {
	deltaSource *AccumulatedFrame
	doneFunc    func()
}

func (d DeltaIterator) Done() {
	d.doneFunc()
}

func (d DeltaIterator) Count() int {
	return len(*d.deltaSource)
}

type CellIterator interface {
	Iter(func(geometry.Point, common.Cell))
	Count() int
	Done()
}

func (d DeltaIterator) Iter(f func(geometry.Point, common.Cell)) {
	cellsInFrame := *d.deltaSource
	if len(cellsInFrame) == 0 {
		return
	}
	for pos, cell := range cellsInFrame {
		f(pos, cell)
	}
}
