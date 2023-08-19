package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

// Box contains information to draw a rectangle using box characters, with an
// optional title.
type Box struct {
	Style       common.Style    // box style
	Title       core.StyledText // optional top currText
	Footer      core.StyledText // optional bottom currText
	AlignTitle  core.Alignment  // title alignment
	AlignFooter core.Alignment  // footer alignment
}

func (b Box) Draw(bbox geometry.Rect, con console.CellInterface) {
	max := bbox.Size()
	if max.X < 2 || max.Y < 2 {
		return
	}
	for y := 0; y < max.Y; y++ {
		for x := 0; x < max.X; x++ {
			var c common.Cell
			if y == 0 {
				if x == 0 {
					c = common.Cell{Rune: '┌', Style: b.Style}
				} else if x == max.X-1 {
					c = common.Cell{Rune: '┐', Style: b.Style}
				} else {
					c = common.Cell{Rune: '─', Style: b.Style}
				}
			} else if y == max.Y-1 {
				if x == 0 {
					c = common.Cell{Rune: '└', Style: b.Style}
				} else if x == max.X-1 {
					c = common.Cell{Rune: '┘', Style: b.Style}
				} else {
					c = common.Cell{Rune: '─', Style: b.Style}
				}
			} else {
				if x == 0 || x == max.X-1 {
					c = common.Cell{Rune: '│', Style: b.Style}
				}
			}
			if c.Rune != 0 {
				con.SetSquare(bbox.Min.Add(geometry.Point{x, y}), c)
			}
		}
	}
	if b.Title.Text() != "" {
		b.Title.DrawHalfWidth(con, bbox.Shift(0, 0, bbox.Size().X, -(max.Y-1)), b.AlignTitle)
	}

	if b.Footer.Text() != "" {
		b.Footer.DrawHalfWidth(con, bbox.Shift(0, max.Y-1, bbox.Size().X, 0), b.AlignFooter)
	}
}
