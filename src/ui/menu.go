package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Menu struct {
	menuItems         []services.MenuItem
	onClose           func()
	beforeConfirm     func()
	selectedItemIndex int
	bgColor           common.RGBAColor
	boundingBox       geometry.Rect
	isDirty           bool
	title             string
	scrollPos         int
}

func (m *Menu) Update(input services.InputInterface) {
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.PointerCommand:
			m.handlePointerCommand(typedCmd)
		case core.GameCommand:
			m.handleGameCommand(typedCmd)
		case core.KeyCommand:
			m.handleKeyCommand(typedCmd, input.IsShiftPressed())
		}
	}
}

func (m *Menu) handleKeyCommand(cmd core.KeyCommand, shiftPressed bool) {
	for _, menuItem := range m.menuItems {
		if menuItem.ShiftHandler != nil && shiftPressed && menuItem.QuickKey == cmd.Key {
			menuItem.ShiftHandler()
			m.isDirty = true
			return
		} else if menuItem.QuickKey == cmd.Key {
			menuItem.Handler()
			m.isDirty = true
			return
		}
	}
}
func (m *Menu) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuCancel:
		m.isDirty = true
		if m.onClose != nil {
			m.onClose()
		}
	case core.MenuConfirm:
		m.activateSelectedItem()
	case core.MenuDown:
		m.selectedItemIndex += 1
		if m.selectedItemIndex >= len(m.menuItems) {
			m.selectedItemIndex = 0 // jump to the top
			m.scrollPos = 0
		} else if !m.isCurrentItemVisible() {
			m.scrollDown()
		}
		m.isDirty = true
	case core.MenuUp:
		m.selectedItemIndex -= 1
		if m.selectedItemIndex < 0 {
			m.selectedItemIndex = len(m.menuItems) - 1 // jump to the bottom
			m.scrollPos = len(m.menuItems) - m.boundingBox.Size().Y + 1
		} else if !m.isCurrentItemVisible() {
			m.scrollUp()
		}
		m.isDirty = true
	case core.MenuLeft:
		if m.selectedItemIndex >= 0 && m.selectedItemIndex < len(m.menuItems) {
			item := m.menuItems[m.selectedItemIndex]
			if item.LeftHandler != nil {
				item.LeftHandler()
			}
		}
		m.isDirty = true
	case core.MenuRight:
		if m.selectedItemIndex >= 0 && m.selectedItemIndex < len(m.menuItems) {
			item := m.menuItems[m.selectedItemIndex]
			if item.RightHandler != nil {
				item.RightHandler()
			}
		}
		m.isDirty = true
	}
}
func (m *Menu) isCurrentItemVisible() bool {
	return m.selectedItemIndex >= m.scrollPos && m.selectedItemIndex < m.scrollPos+m.boundingBox.Size().Y-1
}

func (m *Menu) activateSelectedItem() {
	if m.selectedItemIndex < 0 || m.selectedItemIndex >= len(m.menuItems) {
		m.isDirty = true
		if m.onClose != nil {
			m.onClose()
		}
		return
	}
	item := m.menuItems[m.selectedItemIndex]
	if m.beforeConfirm != nil {
		m.beforeConfirm()
	}
	if item.Handler != nil {
		item.Handler()
	}
	m.isDirty = true
}

func (m *Menu) handlePointerCommand(cmd core.PointerCommand) {
	mousePos := cmd.Pos
	switch cmd.Action {
	case core.MouseMoved:
		if mousePos.Y >= m.boundingBox.Min.Y && mousePos.Y <= m.boundingBox.Max.Y {
			m.selectedItemIndex = mousePos.Y - m.boundingBox.Min.Y - 1 + m.scrollPos
			m.isDirty = true
		}
	case core.MouseLeftReleased:
		m.activateSelectedItem()
	case core.MouseWheelUp:
		if m.needsScrolling() {
			m.scrollUp()
		}
	case core.MouseWheelDown:
		if m.needsScrolling() {
			m.scrollDown()
		}
	}
}

func (m *Menu) scrollUp() {
	m.scrollPos--
	if m.scrollPos < 0 {
		m.scrollPos = 0
	}
	m.isDirty = true
}

func (m *Menu) scrollDown() {
	m.scrollPos++
	if m.scrollPos > len(m.menuItems)-m.boundingBox.Size().Y+1 {
		m.scrollPos = len(m.menuItems) - m.boundingBox.Size().Y + 1
	}
	m.isDirty = true
}

func (m *Menu) needsScrolling() bool {
	return len(m.menuItems) > m.boundingBox.Size().Y
}

func (m *Menu) canScrollUp() bool {
	return m.scrollPos > 0
}

func (m *Menu) canScrollDown() bool {
	return m.scrollPos < len(m.menuItems)-m.boundingBox.Size().Y+1
}
func (m *Menu) SetDirty() {
	m.isDirty = true
}
func (m *Menu) Draw(grid console.CellInterface) {
	if !m.isDirty {
		return
	}
	halfWidthBounds := geometry.NewRect(m.boundingBox.Min.X*2, m.boundingBox.Min.Y, m.boundingBox.Max.X*2, m.boundingBox.Max.Y+1)
	DrawBox(grid, m.title, m.boundingBox, common.White, m.bgColor)
	if m.needsScrolling() {
		if m.canScrollUp() {
			scrollUpPos := geometry.Point{X: m.boundingBox.Max.X, Y: m.boundingBox.Min.Y + 1}
			grid.SetSquare(scrollUpPos, common.Cell{Rune: ' ', Style: common.Style{Foreground: common.White, Background: m.bgColor}})
			grid.SetHalfWidth(scrollUpPos.ToHalfWidth(), common.Cell{Rune: '↑', Style: common.Style{Foreground: common.White, Background: m.bgColor}})
		} else {
			scrollUpPos := geometry.Point{X: m.boundingBox.Max.X, Y: m.boundingBox.Min.Y + 1}
			grid.SetHalfWidth(scrollUpPos.ToHalfWidth(), common.Cell{Rune: ' ', Style: common.TransparentBackgroundStyle})
		}

		if m.canScrollDown() {
			scrollDownPos := geometry.Point{X: m.boundingBox.Max.X, Y: m.boundingBox.Max.Y - 1}
			grid.SetSquare(scrollDownPos, common.Cell{Rune: ' ', Style: common.Style{Foreground: common.White, Background: m.bgColor}})
			grid.SetHalfWidth(scrollDownPos.ToHalfWidth(), common.Cell{Rune: '↓', Style: common.Style{Foreground: common.White, Background: m.bgColor}})
		} else {
			scrollDownPos := geometry.Point{X: m.boundingBox.Max.X, Y: m.boundingBox.Max.Y - 1}
			grid.SetHalfWidth(scrollDownPos.ToHalfWidth(), common.Cell{Rune: ' ', Style: common.TransparentBackgroundStyle})
		}
	}
	grid.HalfWidthFill(halfWidthBounds, common.Cell{Rune: ' ', Style: common.Style{Foreground: common.Transparent, Background: common.Transparent}})
	for yPos := m.boundingBox.Min.Y + 1; yPos < m.boundingBox.Max.Y; yPos++ {
		itemIndex := yPos - m.boundingBox.Min.Y - 1 + m.scrollPos
		item := m.menuItems[itemIndex]

		xPos := m.boundingBox.Min.X + 1
		fgColor := common.White
		if itemIndex == m.selectedItemIndex {
			fgColor = common.Red
		}

		labelWidth := m.boundingBox.Size().X - 1
		if item.Icon > 0 {
			var iconColor common.Color
			if item.IconForegroundColor != nil {
				iconColor = item.IconForegroundColor.ToRGB()
			} else {
				iconColor = fgColor
			}
			grid.SetSquare(geometry.Point{X: xPos, Y: yPos}, common.Cell{Rune: item.Icon, Style: common.Style{Foreground: iconColor, Background: m.bgColor}})
			xPos += 2
			labelWidth -= 2
		}
		itemLabel := item.Label
		if item.DynamicLabel != nil {
			itemLabel = item.DynamicLabel()
		}
		PrintToGridFixedWidth(grid, itemLabel, geometry.Point{X: xPos, Y: yPos}, labelWidth, m.bgColor, fgColor)
	}

	m.isDirty = false
}

// ishidden when..
// selectedItemIndex < scrollPos
// or
// selectedItemIndex >= scrollPos + squareOuterBounds.Size().Y

func DrawBox(grid console.CellInterface, title string, bbox geometry.Rect, fgColor common.RGBAColor, bgColor common.RGBAColor) {
	hori := '─'
	vert := '│'
	topLeft := '┌'
	topRight := '┐'
	bottomLeft := '└'
	bottomRight := '┘'
	titleLength := len(title)
	titleStart := bbox.Min.X + (bbox.Size().X-titleLength)/2
	titleEnd := titleStart + titleLength
	grid.SquareFill(bbox, common.Cell{Rune: ' ', Style: common.Style{Foreground: fgColor, Background: bgColor}})
	for x := bbox.Min.X; x <= bbox.Max.X; x++ {
		if x >= titleStart && x < titleEnd {
			grid.SetSquare(geometry.Point{X: x, Y: bbox.Min.Y}, common.Cell{Rune: rune(title[x-titleStart]), Style: common.Style{Foreground: fgColor, Background: bgColor}})
		} else {
			grid.SetSquare(geometry.Point{X: x, Y: bbox.Min.Y}, common.Cell{Rune: hori, Style: common.Style{Foreground: fgColor, Background: bgColor}})
		}
		grid.SetSquare(geometry.Point{X: x, Y: bbox.Max.Y}, common.Cell{Rune: hori, Style: common.Style{Foreground: fgColor, Background: bgColor}})
	}
	for y := bbox.Min.Y; y <= bbox.Max.Y; y++ {
		grid.SetSquare(geometry.Point{X: bbox.Min.X, Y: y}, common.Cell{Rune: vert, Style: common.Style{Foreground: fgColor, Background: bgColor}})
		grid.SetSquare(geometry.Point{X: bbox.Max.X, Y: y}, common.Cell{Rune: vert, Style: common.Style{Foreground: fgColor, Background: bgColor}})
	}
	grid.SetSquare(bbox.Min, common.Cell{Rune: topLeft, Style: common.Style{Foreground: fgColor, Background: bgColor}})
	grid.SetSquare(geometry.Point{X: bbox.Max.X, Y: bbox.Min.Y}, common.Cell{Rune: topRight, Style: common.Style{Foreground: fgColor, Background: bgColor}})
	grid.SetSquare(geometry.Point{X: bbox.Min.X, Y: bbox.Max.Y}, common.Cell{Rune: bottomLeft, Style: common.Style{Foreground: fgColor, Background: bgColor}})
	grid.SetSquare(bbox.Max, common.Cell{Rune: bottomRight, Style: common.Style{Foreground: fgColor, Background: bgColor}})
}

func PrintToGridFixedWidth(grid console.CellInterface, text string, point geometry.Point, width int, bgColor common.RGBAColor, fgColor common.RGBAColor) {
	point.X = point.X * 2
	runes := []rune(text)
	for i := 0; i < (width * 2); i++ {
		drawPos := point.Add(geometry.Point{X: i})
		if i < len(text) {
			r := runes[i]
			grid.SetHalfWidth(drawPos, common.Cell{Rune: r, Style: common.Style{Foreground: fgColor, Background: bgColor}})
		} else {
			grid.SetHalfWidth(drawPos, common.Cell{Rune: ' ', Style: common.Style{Foreground: fgColor, Background: bgColor}})
		}
	}
}

func PrintToGrid(grid console.CellInterface, text string, point geometry.Point, fgColor common.RGBAColor, bgColor common.RGBAColor) {
	i := 0
	for _, r := range text {
		grid.SetHalfWidth(point.Add(geometry.Point{X: i}), common.Cell{Rune: r, Style: common.Style{Foreground: fgColor, Background: bgColor}})
		i++
	}
}

func NewMenu(title string, menuItems []services.MenuItem, boundingBox geometry.Rect, onClose func(), beforeConfirm func()) *Menu {
	return &Menu{
		title: title,
		menuItems: Filtered(menuItems, func(item services.MenuItem) bool {
			if item.Condition == nil {
				return true
			}
			return item.Condition()
		}),
		onClose:       onClose,
		beforeConfirm: beforeConfirm,
		bgColor:       common.RGBAColor{R: 0.2, G: 0.2, B: 0.6, A: 1.0},
		boundingBox:   boundingBox,
		isDirty:       true,
	}
}

func Filtered(items []services.MenuItem, predicate func(item services.MenuItem) bool) []services.MenuItem {
	var filtered []services.MenuItem
	for _, item := range items {
		if predicate(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
func widthFromItems(items []services.MenuItem) int {
	maxWidth := 0
	for _, item := range items {
		if item.Condition != nil && !item.Condition() {
			continue
		}
		if len([]rune(item.Label)) > maxWidth {
			maxWidth = len([]rune(item.Label))
		}
	}
	return maxWidth
}

func ConditionalCount(items []services.MenuItem) int {
	count := 0
	for _, item := range items {
		if item.Condition != nil && !item.Condition() {
			continue
		}
		count++
	}
	return count
}
