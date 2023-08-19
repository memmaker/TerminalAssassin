package console

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/geometry"
)

// Frame contains the necessary information to draw the squareDeltaFrame changes from a
// squareDeltaFrame to the drawFrame. One is sent to the driver after every draw.
type Frame struct {
	ChangedCells []CellDraw // cells that changed from previous squareDeltaFrame
	IsHalfWidth  bool
}

type AccumulatedFrame map[geometry.Point]common.Cell

// CellDraw represents a cell drawing instruction at a specific absolute
// position in the whole squarePreviousGrid.
type CellDraw struct {
	Cell common.Cell    // cell content and styling
	P    geometry.Point // absolute position in the whole squarePreviousGrid
}
