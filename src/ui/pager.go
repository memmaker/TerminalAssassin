package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

// PagerStyle describes styling options for a Pager.
type PagerStyle struct {
	LineNum common.Style // line num display style (for boxed pager)
}

// Pager represents a pager widget for viewing a long list of lines.
//
// Pager implements gruid.engine and can be used as main model of an
// application.
type Pager struct {
	box    *Box
	bounds geometry.Rect
	lines  []core.StyledText
	style  PagerStyle
	index  int // current index
	action PagerAction
	init   bool // Update received MsgInit
	dirty  bool // state changed in Update and Draw was still not called
	OnQuit func()
}

// PagerAction represents an user action with the pager.
type PagerAction int

const (
	// PagerPass reports that the pager state did not change (for example a
	// mouse motion outside the menu, or within a same entry line).
	PagerPass PagerAction = iota

	// PagerMove reports a scrolling movement.
	PagerMove

	// PagerQuit reports that the user clicked outside the menu, or pressed
	// Esc, Space or X.
	PagerQuit
)

// NewPager returns a new pager with given configuration options.
func (m *Manager) NewPager(title string, lines []core.StyledText) *Pager {
	gridWidth, gridHeight := m.engine.ScreenGridWidth(), m.engine.ScreenGridHeight()
	pg := &Pager{
		box:    &Box{Title: core.NewStyledText(title, common.DefaultStyle), Style: common.DefaultStyle},
		lines:  lines,
		bounds: geometry.NewRect(0, 0, gridWidth, gridHeight),
	}
	pg.dirty = true
	return pg
}
func (pg *Pager) SetDirty() {
	pg.dirty = true
}

// SetCursor updates the pager's currPosition to the given (X, Y) point, where X is
// the indentation level, and Y the line number of the upper-most line of the
// view.
func (pg *Pager) SetCursor(p geometry.Point) {
	nlines := pg.nlines()
	pg.index = p.Y
	if pg.index+nlines-1 >= len(pg.lines) {
		pg.index = len(pg.lines) - nlines
	}
	if pg.index <= 0 {
		pg.index = 0
	}
	pg.dirty = true
}

// SetLines updates the pager currText lines.
func (pg *Pager) SetLines(lines []core.StyledText) {
	nlines := pg.nlines()
	pg.lines = lines
	if pg.index+nlines-1 >= len(pg.lines) {
		pg.index = len(pg.lines) - nlines
		if pg.index <= 0 {
			pg.index = 0
		}
	}
	pg.dirty = true
}

func (pg *Pager) nlines() int {
	h, bh := pg.height()
	return h - bh
}

func (pg *Pager) down(shift int) {
	nlines := pg.nlines()
	if pg.index+nlines+shift-1 >= len(pg.lines) {
		shift = len(pg.lines) - pg.index - nlines
	}
	if shift > 0 {
		pg.action = PagerMove
		pg.index += shift
	}
}

func (pg *Pager) up(shift int) {
	if pg.index-shift < 0 {
		shift = pg.index
	}
	if shift > 0 {
		pg.action = PagerMove
		pg.index -= shift
	}
}

func (pg *Pager) top() {
	if pg.index != 0 {
		pg.index = 0
		pg.action = PagerMove
	}
}

func (pg *Pager) bottom() {
	nlines := pg.nlines()
	if pg.index != len(pg.lines)-nlines {
		pg.index = len(pg.lines) - nlines
		pg.action = PagerMove
	}
}

// Update implements gruid.engine.IsFinished for Pager. It considers mouse message
// coordinates to be absolute in its grid. If a gruid.MsgInit is passed to
// Update, the pager will behave as if it is the main model of an application,
// and send a gruid.Quit() command on PagerQuit action.
func (pg *Pager) Update(input services.InputInterface) {
	pg.action = PagerPass
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.GameCommand:
			pg.handleGameCommand(typedCmd)
		case core.PointerCommand:
			pg.handlePointerCommand(typedCmd)
		}
	}
	if pg.Action() != PagerPass {
		pg.dirty = true
	}
}
func (pg *Pager) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuDown:
		pg.down(1)
	case core.MenuUp:
		pg.up(1)
	case core.MenuConfirm:
		pg.action = PagerQuit
	case core.MenuCancel:
		pg.action = PagerQuit
	}
	if pg.action == PagerQuit && pg.OnQuit != nil {
		pg.OnQuit()
	}
}

func (pg *Pager) handlePointerCommand(cmd core.PointerCommand) {
	switch cmd.Action {
	case core.MouseWheelUp:
		pg.up(1)
	case core.MouseWheelDown:
		pg.down(1)
	}
}

// Action returns the last action performed with the pager.
func (pg *Pager) Action() PagerAction {
	return pg.action
}

func (pg *Pager) height() (h int, bh int) {
	h = pg.bounds.Size().Y
	if pg.box != nil {
		bh = 2
	}
	if h > bh+len(pg.lines) {
		h = bh + len(pg.lines)
	}
	return h, bh
}

// Draw implements gruid.engine.drawOnHalfWidth for Pager. It returns the grid slice that
// was drawn, or the whole grid if it is used as main model.
func (pg *Pager) Draw(con console.CellInterface) {
	if !pg.dirty {
		return
	}
	//con.HalfWidthFill(pg.bounds.Shift(), common.Cell{Rune: ' ', Style: common.DefaultStyle})
	halfwidthBounds := pg.bounds.Shift(0, 0, pg.bounds.Size().X, 0)
	con.SquareFill(pg.bounds, common.Cell{Rune: ' ', Style: common.DefaultStyle})
	con.HalfWidthFill(halfwidthBounds, common.TransparentCell)
	pg.box.Draw(pg.bounds, con)
	// we want to draw the lines of pg.lines inside of pg.bounds
	// but only using the height, that is available
	height := pg.bounds.Size().Y - 2
	for i := 0; i < height; i++ {
		lineIndex := i + pg.index
		if lineIndex >= len(pg.lines) {
			break
		}
		lineBounds := halfwidthBounds.Line(i+1).Shift(2, 0, -2, 0)
		pg.lines[lineIndex].DrawHalfWidth(con, lineBounds, core.AlignLeft)
	}
	pg.dirty = false
}
