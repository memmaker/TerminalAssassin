package ui

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type RingMenu struct {
	listOfItems       []*core.Item
	currentIndex      int
	selected          func(*core.Item)
	ringSlotCount     int
	isDirty           bool
	itemLabel         core.StyledText
	ringYOffset       int
	boxDef            Box
	squareOuterBounds geometry.Rect
	colorShades       []common.RGBAColor
	cancel            func()
}

func NewRingMenu(currentItem *core.Item, listOfItems []*core.Item, onSelected func(*core.Item), onCancel func(), yOffset, gridWidth int) *RingMenu {
	borderDist := gridWidth / 4
	itemName := ""
	if currentItem != nil {
		itemName = currentItem.Name
	}
	bbox := geometry.NewRect(borderDist, yOffset-3, gridWidth-borderDist, yOffset+4)
	outerBox := Box{
		Style:      common.DefaultStyle,
		AlignTitle: core.AlignCenter,
	}
	brightWhite := common.RGBAColor{R: 4.0, G: 4.0, B: 4.0, A: 1.0}
	gray := common.RGBAColor{R: 0.4, G: 0.4, B: 0.4, A: 1.0}
	darkGray := common.RGBAColor{R: 0.2, G: 0.2, B: 0.2, A: 1.0}
	reallyDarkGray := common.RGBAColor{R: 0.1, G: 0.1, B: 0.1, A: 1.0}
	shades := []common.RGBAColor{brightWhite, gray, darkGray, reallyDarkGray}
	r := &RingMenu{
		listOfItems:       listOfItems,
		currentIndex:      RingIndexOf(currentItem, listOfItems),
		selected:          onSelected,
		cancel:            onCancel,
		ringSlotCount:     SlotCountFromList(listOfItems),
		itemLabel:         core.NewStyledText(itemName, common.DefaultStyle),
		ringYOffset:       yOffset - 1,
		boxDef:            outerBox,
		squareOuterBounds: bbox,
		isDirty:           true,
		colorShades:       shades,
	}
	r.UpdateLabel()
	return r
}

func SlotCountFromList(items []*core.Item) int {
	itemCount := len(items)
	switch {
	case itemCount < 3:
		return 1
	case itemCount < 5:
		return 3
	}
	return 5
}

func RingIndexOf(item *core.Item, items []*core.Item) int {
	for i, v := range items {
		if v == item {
			return i
		}
	}
	return 0
}

func (r *RingMenu) Update(input services.InputInterface) {
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.GameCommand:
			r.handleGameCommand(typedCmd)
		}
	}
}

func (r *RingMenu) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuRight:
		r.currentIndex = r.relativIndex(r.currentIndex, 1)
		r.UpdateLabel()
	case core.MenuLeft:
		r.currentIndex = r.relativIndex(r.currentIndex, -1)
		r.UpdateLabel()
	case core.MenuConfirm:
		r.selected(r.listOfItems[r.currentIndex])
	case core.MenuCancel:
		if r.cancel != nil {
			r.cancel()
		}
	}
}

func (r *RingMenu) UpdateLabel() {
	item := r.listOfItems[r.currentIndex]
	newText := item.Name
	if item.Uses >= 0 {
		newText = fmt.Sprintf("%s (%d)", item.Name, item.Uses)
	}
	r.itemLabel = r.itemLabel.WithText(newText)
	r.isDirty = true
}
func (r *RingMenu) Draw(con console.CellInterface) {
	if !r.isDirty {
		return
	}
	gridWidth := con.Size().X
	centeredX := gridWidth / 2
	centeredY := r.ringYOffset

	con.SquareFill(r.squareOuterBounds, common.Cell{Rune: ' ', Style: common.DefaultStyle})
	r.boxDef.Draw(r.squareOuterBounds, con)

	labelBox := r.squareOuterBounds.Line(4).Shift(1, 0, -1, 0).ToHalfWidth()
	con.HalfWidthFill(labelBox, common.Cell{Rune: ' ', Style: common.DefaultStyle})
	r.itemLabel.DrawHalfWidth(con, labelBox, core.AlignCenter)
	centerPos := geometry.Point{X: centeredX, Y: centeredY}
	if r.ringSlotCount == 1 {
		selectedItem := r.listOfItems[r.currentIndex]

		con.SetSquare(centerPos, common.Cell{Rune: selectedItem.DefinedIcon, Style: common.DefaultStyle.WithFg(r.colorShades[0])})
		if len(r.listOfItems) > 1 {
			r.drawArrows(con, centerPos, 1)
		}
		return
	}
	loopRange := (r.ringSlotCount - 1) / 2
	for i := -loopRange; i <= loopRange; i++ {
		itemIndex := r.relativIndex(r.currentIndex, i)
		itemAt := r.listOfItems[itemIndex]
		xOffset := i * 3
		drawPos := geometry.Point{X: centeredX + xOffset, Y: centeredY}
		drawColor := r.colorShades[AbsInt(i)]
		con.SetSquare(drawPos, common.Cell{Rune: itemAt.DefinedIcon, Style: common.DefaultStyle.WithFg(drawColor)})
	}
	r.drawArrows(con, centerPos, loopRange+1)
	r.isDirty = false
}

func (r *RingMenu) drawArrows(con console.CellInterface, basePos geometry.Point, indexOffset int) {
	leftX := basePos.X - (3 * indexOffset)
	rightX := basePos.X + (3 * indexOffset)
	con.SetSquare(geometry.Point{X: leftX, Y: basePos.Y}, common.Cell{Rune: '<', Style: common.DefaultStyle.WithFg(r.colorShades[indexOffset])})
	con.SetSquare(geometry.Point{X: rightX, Y: basePos.Y}, common.Cell{Rune: '>', Style: common.DefaultStyle.WithFg(r.colorShades[indexOffset])})
}

func AbsInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
func (r *RingMenu) SetDirty() {
	r.isDirty = true
}

func (r *RingMenu) relativIndex(baseIndex int, offset int) int {
	index := baseIndex + offset
	if index < 0 {
		index = index + len(r.listOfItems)
	} else if index >= len(r.listOfItems) {
		index = index - len(r.listOfItems)
	}
	return index
}
