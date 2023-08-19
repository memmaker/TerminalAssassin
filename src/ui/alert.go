package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Alert struct {
	content     []core.StyledText
	onConfirmed func()
	bounds      geometry.Rect
	isDirty     bool
	bgColor     common.Color
}

func (a *Alert) SetDirty() {
	a.isDirty = true
}

func (a *Alert) Update(input services.InputInterface) {
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.KeyCommand:
			a.handleKeyCommand(typedCmd)
		case core.GameCommand:
			a.handleGameCommand(typedCmd)
		case core.PointerCommand:
			a.handlePointerCommand(typedCmd)
		}
	}
}

func (a *Alert) handleKeyCommand(cmd core.KeyCommand) {
	switch cmd.Key {
	case core.KeyEscape:
		fallthrough
	case core.KeyEnter:
		fallthrough
	case core.KeySpace:
		a.isDirty = true
		if a.onConfirmed != nil {
			a.onConfirmed()
		}
	}
}

func (a *Alert) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuCancel:
		fallthrough
	case core.MenuConfirm:
		a.isDirty = true
		if a.onConfirmed != nil {
			a.onConfirmed()
		}
	}
}

func (a *Alert) handlePointerCommand(cmd core.PointerCommand) {
	switch cmd.Action {
	case core.MouseRight:
		fallthrough
	case core.MouseLeft:
		a.isDirty = true
		if a.onConfirmed != nil {
			a.onConfirmed()
		}
	}
}

func (a *Alert) Draw(grid console.CellInterface) {
	if !a.isDirty {
		return
	}
	for x := a.bounds.Min.X; x < a.bounds.Max.X; x++ {
		grid.SetHalfWidth(geometry.Point{X: x, Y: a.bounds.Min.Y}, common.Cell{Rune: ' ', Style: common.DefaultStyle.WithBg(a.bgColor)})
	}
	for y := a.bounds.Min.Y + 1; y < a.bounds.Max.Y; y++ {
		cellsDrawn := 0
		yIndex := (y - a.bounds.Min.Y) - 1
		grid.SetHalfWidth(geometry.Point{X: a.bounds.Min.X, Y: y}, common.Cell{Rune: ' ', Style: common.DefaultStyle.WithBg(a.bgColor)})
		if yIndex < len(a.content) {
			line := a.content[yIndex]
			lineBounds := a.bounds.Line(yIndex+1).Shift(1, 0, -1, 0)
			cellsDrawn = line.DrawHalfWidth(grid, lineBounds, core.AlignLeft)
		}
		startAtX := a.bounds.Min.X + 1 + cellsDrawn
		for x := startAtX; x < a.bounds.Max.X; x++ {
			grid.SetHalfWidth(geometry.Point{X: x, Y: y}, common.Cell{Rune: ' ', Style: common.DefaultStyle.WithBg(a.bgColor)})
		}
	}
	a.isDirty = false
}

func (a *Alert) SetBackgroundColor(newBGColor common.Color) {
	a.bgColor = newBGColor
	a.SetDirty()
}

func NewAlert(halfWidthBounds geometry.Rect, content []core.StyledText, onConfirmed func()) *Alert {
	if len(content) > halfWidthBounds.Size().Y {
		println("WARNING: Alert content is too long, it will be truncated")
	}
	return &Alert{
		bounds:      halfWidthBounds,
		content:     content,
		onConfirmed: onConfirmed,
		isDirty:     true,
	}
}
