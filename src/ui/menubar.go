package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type MenuBar struct {
	menuItems         []services.MenuItem
	selectedItemIndex int
	bgColor           common.RGBAColor
	boundingBox       geometry.Rect
	isDirty           bool
	title             string
	contextMenuItems  []services.MenuItem
}

func (m *MenuBar) Update(input services.InputInterface) {
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.PointerCommand:
			m.handlePointerCommand(typedCmd)
		case core.GameCommand:
			m.handleGameCommand(typedCmd)
		}
	}
}

func (m *MenuBar) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuConfirm:
		m.activateSelectedItem()
	case core.MenuRight:
		m.selectedItemIndex = (m.selectedItemIndex + 1) % len(m.menuItems)
		m.isDirty = true
	case core.MenuLeft:
		m.selectedItemIndex = m.selectedItemIndex - 1
		if m.selectedItemIndex < 0 {
			m.selectedItemIndex = len(m.menuItems) - 1
		}
		m.isDirty = true
	}
}

func (m *MenuBar) activateSelectedItem() {
	if m.selectedItemIndex < 0 || m.selectedItemIndex >= len(m.menuItems)+len(m.contextMenuItems)+1 {
		return
	}
	if m.selectedItemIndex < len(m.menuItems) {
		item := m.menuItems[m.selectedItemIndex]
		if item.Handler != nil {
			item.Handler()
		}
		m.isDirty = true
		return
	}
	contextIndex := m.selectedItemIndex - len(m.menuItems) - 1
	if contextIndex < len(m.contextMenuItems) {
		item := m.contextMenuItems[contextIndex]
		if item.Handler != nil {
			item.Handler()
		}
		m.isDirty = true
		return
	}
}

func (m *MenuBar) handlePointerCommand(cmd core.PointerCommand) bool {
	mousePos := cmd.Pos
	if mousePos.Y < m.boundingBox.Min.Y || mousePos.Y > m.boundingBox.Max.Y {
		return false
	}
	switch cmd.Action {
	case core.MouseMoved:
		m.selectedItemIndex = (mousePos.X - m.boundingBox.Min.X) / 2
	case core.MouseLeftReleased:
		m.activateSelectedItem()
	}
	m.isDirty = true
	return true
}

func (m *MenuBar) handleKeyCommand(cmd core.KeyCommand) bool {
	for _, item := range m.menuItems {
		if item.QuickKey == cmd.Key && item.Handler != nil {
			item.Handler()
			return true
		}
		m.isDirty = true
	}
	for _, item := range m.contextMenuItems {
		if item.QuickKey == cmd.Key && item.Handler != nil {
			item.Handler()
			return true
		}
		m.isDirty = true
	}
	return false
}

func (m *MenuBar) TryHandle(cmd core.InputCommand) bool {
	switch typedCmd := cmd.(type) {
	case core.PointerCommand:
		return m.handlePointerCommand(typedCmd)
	case core.KeyCommand:
		return m.handleKeyCommand(typedCmd)
	default:
		return false
	}
}
func (m *MenuBar) SetDirty() {
	m.isDirty = true
}
func (m *MenuBar) Draw(grid console.CellInterface) {
	if !m.isDirty {
		return
	}

	// we want to print out the menuitems a horizontal lowerLine layout
	// we want one empty cell before each item and the item itself is printed as the icon only

	xPos := m.boundingBox.Min.X + 1
	yPos := m.boundingBox.Min.Y + 1
	drawPos := geometry.Point{X: xPos, Y: yPos}
	grid.SquareFill(m.boundingBox, common.Cell{Rune: ' ', Style: common.Style{Foreground: m.bgColor, Background: m.bgColor}})

	for i, item := range m.menuItems {
		drawColor := common.RGBAColor{R: 0.5, G: 0.5, B: 0.5, A: 1.0}
		if item.Highlight != nil && item.Highlight() {
			drawColor = common.RGBAColor{R: 1.0, G: 4.0, B: 1.0, A: 1.0}
		}
		if i == m.selectedItemIndex {
			drawColor = common.RGBAColor{R: 1.0, G: 1.0, B: 1.0, A: 1.0}
			m.drawToolTip(grid, drawPos, item.Label)
		}

		grid.SetSquare(drawPos, common.Cell{Rune: item.Icon, Style: common.Style{Foreground: drawColor, Background: m.bgColor}})
		drawPos.X += 2
	}
	grid.SetSquare(drawPos, common.Cell{Rune: '│', Style: common.Style{Foreground: common.White, Background: m.bgColor}})
	drawPos.X += 2

	drawColor := common.RGBAColor{R: 0.5, G: 0.5, B: 0.5, A: 1.0}
	offset := len(m.menuItems) + 1
	for ci, item := range m.contextMenuItems {

		if offset+ci == m.selectedItemIndex {
			drawColor = common.RGBAColor{R: 1.0, G: 1.0, B: 1.0, A: 1.0}
			m.drawToolTip(grid, drawPos, item.Label)
		}

		grid.SetSquare(drawPos, common.Cell{Rune: item.Icon, Style: common.Style{Foreground: drawColor, Background: m.bgColor}})
		drawPos.X += 2
	}

	m.isDirty = false
}

func (m *MenuBar) drawToolTip(grid console.CellInterface, pos geometry.Point, label string) {
	drawPos := pos.Add(geometry.Point{X: 0, Y: -1})
	for _, r := range label {
		grid.SetSquare(drawPos, common.Cell{Rune: r, Style: common.Style{Foreground: common.RGBAColor{R: 1.0, G: 1.0, B: 1.0, A: 1.0}, Background: m.bgColor}})
		drawPos.X++
	}
}

func (m *MenuBar) SetContextMenu(menuItems []services.MenuItem) {
	m.contextMenuItems = menuItems
	m.isDirty = true
}

func NewMenuBar(title string, menuItems []services.MenuItem, boundingBox geometry.Rect) *MenuBar {
	return &MenuBar{
		title: title,
		menuItems: Filtered(menuItems, func(item services.MenuItem) bool {
			if item.Condition == nil {
				return true
			}
			return item.Condition()
		}),
		bgColor:           common.Black,
		boundingBox:       boundingBox,
		isDirty:           true,
		selectedItemIndex: -1,
	}
}
