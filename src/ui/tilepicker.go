package ui

import (
    "math"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
)

// TilePicker is a modal 2-D icon grid palette.
//
// Items are placed as full-tile icons with exactly one tile gap between each
// pair of neighbours (horizontally and vertically).  Hovering over an icon
// highlights it (red foreground) and writes its label into the bottom border
// row of the window using half-width font.  Clicking an icon fires its Handler
// then closes the picker.  Right-click or the cancel key also closes it.
type TilePicker struct {
    items       []services.MenuItem
    cols        int
    boundingBox geometry.Rect // inclusive, same convention as DrawBox
    hoveredIdx  int           // -1 when nothing is hovered
    bgColor     common.RGBAColor
    isDirty     bool
    title       string
    onClose     func()
    onHover     func(label string) // called with the hovered item's label, or "" on leave
}

// NewTilePicker builds a TilePicker.
//   - cols   : number of icon columns.
//   - bbox   : screen rect in the DrawBox inclusive convention (Min is top-left
//     border, Max is bottom-right border inclusive).
//   - onClose: called after a selection or dismissal (use it to PopModal, etc.).
func NewTilePicker(title string, items []services.MenuItem, cols int, bbox geometry.Rect, onHover func(label string), onClose func()) *TilePicker {
    return &TilePicker{
        items:       items,
        cols:        cols,
        title:       title,
        boundingBox: bbox,
        hoveredIdx:  -1,
        bgColor:     common.RGBAColor{R: 0.2, G: 0.2, B: 0.6, A: 1.0},
        isDirty:     true,
        onHover:     onHover,
        onClose:     onClose,
    }
}

// ColsForItems returns a column count that gives the grid the most square
// possible shape for n items.
func ColsForItems(n int) int {
    if n <= 0 {
        return 1
    }
    return int(math.Ceil(math.Sqrt(float64(n))))
}

func (t *TilePicker) SetDirty() {
    t.isDirty = true
}

// itemIndexAt maps a screen-space tile position to a menu-item index.
// Returns -1 when pos is on a border, a gap, or outside the grid.
func (t *TilePicker) itemIndexAt(pos geometry.Point) int {
    relX := pos.X - (t.boundingBox.Min.X + 1)
    relY := pos.Y - (t.boundingBox.Min.Y + 1)
    if relX < 0 || relY < 0 {
        return -1
    }
    // Odd coords are 1-tile gaps between icons.
    if relX%2 != 0 || relY%2 != 0 {
        return -1
    }
    col := relX / 2
    row := relY / 2
    if col >= t.cols {
        return -1
    }
    idx := row*t.cols + col
    if idx >= len(t.items) {
        return -1
    }
    return idx
}

func (t *TilePicker) Update(input services.InputInterface) {
    for _, cmd := range input.PollUICommands() {
        switch typedCmd := cmd.(type) {
        case core.PointerCommand:
            t.handlePointerCommand(typedCmd)
        case core.GameCommand:
            if typedCmd == core.MenuCancel && t.onClose != nil {
                t.onClose()
            }
        }
    }
}

func (t *TilePicker) handlePointerCommand(cmd core.PointerCommand) {
    switch cmd.Action {
    case core.MouseMoved:
        newIdx := t.itemIndexAt(cmd.Pos)
        if newIdx != t.hoveredIdx {
            t.hoveredIdx = newIdx
            t.isDirty = true
            if t.onHover != nil {
                if newIdx >= 0 {
                    item := t.items[newIdx]
                    label := item.Label
                    if item.DynamicLabel != nil {
                        label = item.DynamicLabel()
                    }
                    t.onHover(label)
                } else {
                    t.onHover("")
                }
            }
        }
    case core.MouseLeftReleased:
        idx := t.itemIndexAt(cmd.Pos)
        if idx >= 0 {
            item := t.items[idx]
            if item.Handler != nil {
                item.Handler()
            }
            if t.onClose != nil {
                t.onClose()
            }
        }
    case core.MouseRight:
        // Right-click cancels and closes the picker.
        if t.onClose != nil {
            t.onClose()
        }
    }
}

func (t *TilePicker) Draw(con console.CellInterface) {
    if !t.isDirty {
        return
    }

    // Clear the half-width overlay so square-cell icons are fully visible.
    con.HalfWidthFill(t.boundingBox.ToHalfWidth(), common.TransparentCell)

    // Draw the framing box.  Pass an empty title so DrawBox fills both border
    // rows with plain '─' chars; we render the title ourselves in half-width.
    DrawBox(con, "", t.boundingBox, common.White, t.bgColor)

    // Title in top border — half-width font for maximum character capacity.
    t.drawHalfWidthTextInBorderRow(con, t.title, t.boundingBox.Min.Y)

    // Item icons on every other tile inside the border.
    innerMinX := t.boundingBox.Min.X + 1
    innerMinY := t.boundingBox.Min.Y + 1
    for i, item := range t.items {
        col := i % t.cols
        row := i / t.cols
        pos := geometry.Point{
            X: innerMinX + col*2,
            Y: innerMinY + row*2,
        }

        var iconFg common.Color = common.OffWhite
        if item.IconForegroundColor != nil {
            iconFg = item.IconForegroundColor
        }
        if i == t.hoveredIdx {
            iconFg = common.FourWhite
        }

        icon := item.Icon
        if icon == 0 {
            icon = 'X' // visible fallback for "Clear …" items that omit an icon glyph
        }

        con.SetSquare(pos, common.Cell{
            Rune:  icon,
            Style: common.Style{Foreground: iconFg, Background: t.bgColor},
        })
    }

    // Bottom border is left as plain '─' chars (from DrawBox above).
    // The hover label is shown via the onHover callback in the editor's status bar.

    t.isDirty = false
}

// drawHalfWidthTextInBorderRow centres text in the given border row using
// half-width characters.  The text is drawn on the half-width layer, which
// renders on top of the square-layer '─' chars placed there by DrawBox.
// This gives 2× the character capacity compared to full-width square cells.
func (t *TilePicker) drawHalfWidthTextInBorderRow(con console.CellInterface, text string, y int) {
    runes := []rune(text)
    textLen := len(runes)
    if textLen == 0 {
        return
    }

    // Each full-width inner tile holds two half-width character slots.
    halfInnerWidth := (t.boundingBox.Max.X - t.boundingBox.Min.X - 1) * 2
    halfStartX := (t.boundingBox.Min.X + 1) * 2

    // Truncate if the text is wider than the inner area.
    if textLen > halfInnerWidth {
        runes = runes[:halfInnerWidth]
        textLen = halfInnerWidth
    }

    // Centre the text within the inner half-width span.
    offset := (halfInnerWidth - textLen) / 2

    for i, r := range runes {
        con.SetHalfWidth(geometry.Point{X: halfStartX + offset + i, Y: y}, common.Cell{
            Rune:  r,
            Style: common.Style{Foreground: common.White, Background: t.bgColor},
        })
    }
}
