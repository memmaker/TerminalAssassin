package ui

import (
	"math/rand"
	"path"
	"path/filepath"
	"strings"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/mapset"
)

const HUDHeight = 3

func NewManager(engine services.Engine) *Manager {

	uiMan := &Manager{
		engine:      engine,
		hudheight:   HUDHeight,
		uiScene:     mapset.NewSet[services.UIWidget](),
		hiddenScene: mapset.NewSet[services.UIWidget](),
	}
	return uiMan
}

type Manager struct {
	engine           services.Engine
	hudheight        int
	modalStack       []services.UIWidget
	uiScene          *mapset.MapSet[services.UIWidget]
	hiddenScene      *mapset.MapSet[services.UIWidget]
	isHidden         bool
	tooltipLabel     *MovableLabel
	currentGamestate services.GameState
}

func (m *Manager) currentModal() services.UIWidget {
	if len(m.modalStack) == 0 {
		return nil
	}
	return m.modalStack[len(m.modalStack)-1]
}
func (m *Manager) InitTooltip(boundsFunc func(origin geometry.Point, stringLength int) geometry.Rect) {
	m.tooltipLabel = NewMovableLabel(boundsFunc)
}
func (m *Manager) OpenMapsMenu(afterLoad func(*gridmap.GridMap[*core.Actor, *core.Item, services.Object])) {
	menuItems := make([]services.MenuItem, 0)
	game := m.engine.GetGame()
	config := game.GetConfig()
	career := m.engine.GetCareer()
	files := m.engine.GetFiles()
	campaignFolderName := config.CampaignDirectory
	mapDir := path.Join(campaignFolderName, career.CurrentCampaignFolder)

	entries, err := files.ReadDir(mapDir)
	if err != nil {
		return
	}

	for _, file := range entries {
		if !file.IsDir() || !strings.HasSuffix(file.Name(), ".map") {
			continue
		}
		filename := filepath.Base(file.Name())
		mapname := strings.TrimSuffix(filename, ".map")
		selectedMap := file.Name()
		menuItems = append(menuItems, services.MenuItem{Label: mapname, Handler: func() {
			fileName := path.Join(mapDir, selectedMap)
			game.ResetModel()
			//loadedMap := gridmap.NewMapFromData[*core.Actor, *core.Item, services.Object](fileName)
			loadedMap, loadErr := m.engine.LoadMap(fileName)
			if loadErr != nil {
				println("Error loading map: " + loadErr.Error())
			}

			game.InitLoadedMap(loadedMap)
			if afterLoad != nil {
				afterLoad(loadedMap)
			}
		}})
	}
	m.OpenFixedWidthAutoCloseMenu("Missions", menuItems)
	//m.NewAutoCloseMenu("Missions", menuItems)
}
func (m *Manager) ShowTooltipAt(screenPos geometry.Point, infoString core.StyledText) {
	m.tooltipLabel.Set(screenPos, infoString)
}

func (m *Manager) BoundsForWorldLabel(worldPos geometry.Point, stringLength int) geometry.Rect {
	camera := m.engine.GetGame().GetCamera()
	screenPos := camera.WorldToScreen(worldPos)                    // translate world position to screen position
	labelPos := m.CalculateLabelPlacement(screenPos, stringLength) // reposition the screen position for the label and convert to half position
	return NewBoundsForText(labelPos, stringLength)                // return the bounds for the label (on the halfwidth grid)
}

func NewBoundsForText(fromPos geometry.Point, stringLength int) geometry.Rect {
	return geometry.NewRect(fromPos.X, fromPos.Y, fromPos.X+stringLength, fromPos.Y+1)
}

// CalculateLabelPlacement takes a screen position and a string length and returns a screen position where the string can be placed
func (m *Manager) CalculateLabelPlacement(screenPos geometry.Point, stringLength int) geometry.Point {
	//currentMap := m.engine.GetGame().GetMap()
	screenWidth := m.engine.ScreenGridWidth() * 2
	xOffset := -(stringLength / 2)
	yOffset := -1
	if screenPos.Y < 1 { // are we at the top of the screen? if so, show below
		yOffset = 1
	}

	stringStartX := screenPos.X*2 + xOffset
	stringEndX := stringStartX + stringLength
	if stringStartX < 0 {
		stringStartX = 0
	} else if stringEndX > screenWidth-1 {
		dist := stringEndX - (screenWidth - 1)
		stringStartX = stringStartX - dist
	}

	if stringStartX%2 == 1 {
		stringStartX++
	}
	halfPos := geometry.Point{X: stringStartX, Y: screenPos.Y + yOffset}
	return halfPos
}
func (m *Manager) ClearTooltip() {
	if m.tooltipLabel != nil {
		m.tooltipLabel.Clear()
	}
}

func (m *Manager) TooltipShown() bool {
	return !m.tooltipLabel.IsEmpty()
}
func (m *Manager) IntersectsTooltip(bounds geometry.Rect) bool {
	return m.tooltipLabel.currBounds.Overlaps(bounds)
}

// Update Should be the ONLY function to receive input from the parent
// From here the input gets passed to the current modal or the scene
func (m *Manager) Update(input services.InputInterface) {
	if m.currentModal() != nil && !m.isHidden {
		m.currentModal().Update(input)
		return
	}
	if m.currentGamestate != nil {
		m.currentGamestate.Update(input)
	}
}
func (m *Manager) Draw(con console.CellInterface) {
	if m.currentGamestate != nil {
		m.currentGamestate.Draw(con)
	}
	m.uiScene.Iter(func(widget services.UIWidget) {
		widget.Draw(con)
	})
	if m.currentModal() != nil && !m.isHidden {
		m.currentModal().Draw(con)
	}
	if m.tooltipLabel != nil {
		m.tooltipLabel.Draw(con)
	}
}
func (m *Manager) HUDHeight() int {
	return m.hudheight
}
func (m *Manager) Reset() {
	m.uiScene.Clear()
	m.currentGamestate = nil
	m.modalStack = make([]services.UIWidget, 0)
	m.isHidden = false
	if m.tooltipLabel != nil {
		m.tooltipLabel.Clear()
		m.tooltipLabel = nil
	}
}
func (m *Manager) PopAll() {
	m.modalStack = make([]services.UIWidget, 0)
}

func (m *Manager) HideModal() {
	m.isHidden = true
}
func (m *Manager) ShowModal() {
	m.isHidden = false
	m.currentModal().SetDirty()
}

func (m *Manager) HideWidget(widget services.UIWidget) {
	m.uiScene.Remove(widget)
	m.hiddenScene.Add(widget)
}
func (m *Manager) ShowWidget(widget services.UIWidget) {
	m.hiddenScene.Remove(widget)
	m.uiScene.Add(widget)
	widget.SetDirty()
}
func (m *Manager) PopModal() {
	if len(m.modalStack) == 0 {
		return
	}
	m.modalStack = m.modalStack[:len(m.modalStack)-1]
	if m.currentGamestate != nil {
		m.currentGamestate.ClearOverlay()
		m.currentGamestate.SetDirty()
	}
	for _, widget := range m.uiScene.ToSlice() {
		widget.SetDirty()
	}
	if len(m.modalStack) == 0 {
		return
	}
	m.currentModal().SetDirty()
}
func (m *Manager) pushModal(modal services.UIWidget) {
	m.modalStack = append(m.modalStack, modal)
}

func (m *Manager) AddToScene(widget services.UIWidget) {
	m.uiScene.Add(widget)
}

func (m *Manager) SetGamestate(widget services.GameState) {
	m.currentGamestate = widget
}

func (m *Manager) RemoveFromScene(widget services.UIWidget) {
	m.uiScene.Remove(widget)
}

func (m *Manager) OpenXOffsetAutoCloseMenuWithCallback(xOffset int, items []services.MenuItem, onClose func()) {
	height := len(items) + 1
	yEnd := m.engine.ScreenGridHeight() - 3
	yStart := yEnd - height
	if yStart < 0 {
		yStart = 0
	}
	width := (widthFromItems(items) / 2) + 4
	bbox := geometry.NewRect(xOffset, yStart, xOffset+width, yEnd)
	m.pushModal(NewMenu("", items, bbox, func() {
		m.PopModal()
		if onClose != nil {
			onClose()
		}
	}, func() {
		m.PopModal()
		if onClose != nil {
			onClose()
		}
	}))
}

func (m *Manager) OpenFixedWidthAutoCloseMenuWithCallback(title string, items []services.MenuItem, onClose func()) {
	itemCount := ConditionalCount(items)
	if itemCount == 0 {
		return
	}
	bbox := m.HalfWidthRect(itemCount)
	//currBounds := CenterRectWithYOffset(items, m.engine.ScreenGridWidth(), 3)
	closeFunc := func() {
		m.PopModal()
		if onClose != nil {
			onClose()
		}
	}
	m.pushModal(NewMenu(title, items, bbox, closeFunc, closeFunc))
}
func (m *Manager) OpenFixedWidthAutoCloseMenu(title string, items []services.MenuItem) {
	itemCount := ConditionalCount(items)
	if itemCount == 0 {
		return
	}
	bbox := m.HalfWidthRect(itemCount)
	//currBounds := CenterRectWithYOffset(items, m.engine.ScreenGridWidth(), 3)
	m.pushModal(NewMenu(title, items, bbox, m.PopModal, m.PopModal))
}

func (m *Manager) OpenFixedWidthStackedMenu(title string, items []services.MenuItem) {
	itemCount := ConditionalCount(items)
	if itemCount == 0 {
		return
	}
	bbox := m.HalfWidthRect(itemCount)
	//currBounds := CenterRectWithYOffset(items, m.engine.ScreenGridWidth(), 3)
	m.pushModal(NewMenu(title, items, bbox, m.PopModal, nil))
}

func (m *Manager) HalfWidthRect(itemCount int) geometry.Rect {
	yStart := 3
	yEnd := yStart + itemCount + 1
	xStart := m.engine.ScreenGridWidth() / 4
	xEnd := m.engine.ScreenGridWidth() - xStart
	bbox := geometry.NewRect(xStart, yStart, xEnd, yEnd)
	return bbox
}

func (m *Manager) OpenFancyMenu(menuItems []services.MenuItem) {
	itemCount := ConditionalCount(menuItems)
	bbox := m.HalfWidthRect(itemCount)
	bbox = bbox.Shift(0, 21, 0, 21)
	m.RenderRevealAnimation(common.White.ToHSV(), common.RGBAColor{R: 0.2, G: 0.2, B: 0.6, A: 1.0}.ToHSV(), bbox, func() {
		menu := NewMenu("", menuItems, bbox, nil, nil)
		m.pushModal(menu)
	})
}

func (m *Manager) RenderRevealAnimation(fgColor common.HSVColor, bgColor common.HSVColor, box geometry.Rect, onFinished func()) {
	animator := m.engine.GetAnimator()
	spawnReveal := func() {
		for i := box.Min.X; i <= box.Max.X; i++ {
			pos := geometry.Point{X: i, Y: box.Mid().Y}
			upwardsRevealer := &RevealParticle{
				Box:      box,
				Position: pos,
				OldPos:   pos,
				EndPos:   geometry.Point{X: pos.X, Y: box.Min.Y},
				FgColor:  fgColor,
				BgColor:  bgColor,
			}
			animator.AddParticle(upwardsRevealer)
			downwardsRevealer := &RevealParticle{
				Box:      box,
				Position: pos,
				OldPos:   pos,
				EndPos:   geometry.Point{X: pos.X, Y: box.Max.Y},
				FgColor:  fgColor,
				BgColor:  bgColor,
			}
			animator.AddParticle(downwardsRevealer)
			if i == box.Max.X {
				downwardsRevealer.OnFinish = onFinished
			}
		}
	}

	horizontalLinedrawer := &LineDrawerParticle{
		Box:      box,
		FgColor:  fgColor,
		BgColor:  bgColor,
		OnFinish: spawnReveal,
	}
	animator.AddParticle(horizontalLinedrawer)
}

func (m *Manager) RenderFancyText(lineStart geometry.Point, text []string, finished func()) {
	delay := int(0)
	animator := m.engine.GetAnimator()

	for lineIndex, line := range text {
		for charNumber, c := range line {
			lifetime := uint64(rand.Intn(25)) + 25
			charPos := lineStart.Add(geometry.Point{X: charNumber})
			delay += 4
			charParticle := &AppearingCharacterParticle{
				Char:         c,
				Delay:        delay,
				Pos:          charPos,
				FgColor:      common.TerminalColor.ToHSV(),
				BgColorStart: common.TerminalColor.ToHSV(),
				Lifetime:     lifetime,
			}
			if lineIndex == len(text)-1 && (charNumber == len(line)-1 || len(line) == 0) && finished != nil {
				charParticle.OnFinish = finished
			}
			animator.AddParticle(charParticle)
			if c == '.' || c == '?' || c == '!' {
				delay += 20
			} else if c == ',' {
				delay += 10
			} else if c == ':' {
				delay += 25
			}
		}
		lineStart = lineStart.Add(geometry.Point{Y: 1})
		delay += 10
	}
	//endPos := geometry.Point{X: len(currText[len(currText)-1]), Y: len(currText) - 1}

}

func (m *Manager) IsBlocking() bool {
	return m.currentModal() != nil && !m.isHidden
}

func (m *Manager) ShowTextInputAt(pos geometry.Point, width int, prompt string, prefill string, onComplete func(userInput string), onAbort func()) {
	animator := m.engine.GetAnimator()
	cursorPos := pos.Add(geometry.Point{X: len(prompt) + len(prefill)})
	cursor := &CursorParticle{
		Pos:             cursorPos,
		Color:           common.TerminalColor.ToHSV(),
		DelayAppearance: 35,
	}
	animator.AddParticle(cursor)
	changeCursorPos := func(newPos geometry.Point) {
		cursor.Pos = newPos
	}
	textInput := NewTextInputAt(
		pos,
		width,
		prompt,
		prefill,
		func(userInput string) {
			m.PopModal()
			cursor.Kill()
			onComplete(userInput)
		},
		func() {
			m.PopModal()
			cursor.Kill()
			onAbort()
		})
	textInput.OnCursorMove = changeCursorPos
	m.pushModal(textInput)
}
func (m *Manager) ShowNoAbortTextInputAt(pos geometry.Point, width int, prompt string, prefill string, onComplete func(userInput string)) {
	animator := m.engine.GetAnimator()
	cursorPos := pos.Add(geometry.Point{X: len(prompt) + len(prefill)})
	cursor := &CursorParticle{
		Pos:             cursorPos,
		Color:           common.TerminalColor.ToHSV(),
		DelayAppearance: 35,
	}
	animator.AddParticle(cursor)
	changeCursorPos := func(newPos geometry.Point) {
		cursor.Pos = newPos
	}
	textInput := NewTextInputAt(
		pos,
		width,
		prompt,
		prefill,
		func(userInput string) {
			m.PopModal()
			cursor.Kill()
			onComplete(userInput)
		}, nil)
	textInput.OnCursorMove = changeCursorPos
	m.pushModal(textInput)
}

func (m *Manager) ShowTextInput(prompt string, prefill string, onComplete func(userInput string), onAbort func()) {
	m.pushModal(NewTextInputAt(
		geometry.Point{Y: -1},
		m.engine.ScreenGridWidth(),
		prompt,
		prefill,
		func(userInput string) {
			m.PopModal()
			onComplete(userInput)
		},
		func() {
			m.PopModal()
			onAbort()
		}))
}

func (m *Manager) ShowPager(title string, lines []core.StyledText, onQuit func()) {
	pager := m.NewPager(title, lines)
	pager.OnQuit = func() {
		m.PopModal()
		if onQuit != nil {
			onQuit()
		}
	}
	m.pushModal(pager)
}

func (m *Manager) OpenItemRingMenu(currentItem *core.Item, listOfItems []*core.Item, selectedFunc func(*core.Item), cancelFunc func()) {
	if listOfItems == nil || len(listOfItems) == 0 {
		return
	}
	onSelection := func(item *core.Item) {
		m.PopModal()
		if selectedFunc != nil && item != nil {
			selectedFunc(item)
		}
	}
	onCancel := func() {
		m.PopModal()
		if cancelFunc != nil {
			cancelFunc()
		}
	}
	gridWidth := m.engine.ScreenGridWidth()
	labelYOffset := m.engine.ScreenGridHeight() / 2
	ringMenu := NewRingMenu(currentItem, listOfItems, onSelection, onCancel, labelYOffset, gridWidth)
	m.pushModal(ringMenu)
}

func (m *Manager) OpenColorPicker(color common.Color, onChanged func(color common.Color), onClosed func(color common.Color)) {
	gridW, gridH := m.engine.ScreenGridWidth(), m.engine.ScreenGridHeight()
	bbox := geometry.NewRect(0, gridH-3, gridW, gridH)
	picker := NewColorPicker(bbox)
	closedFunc := func(returnedColor common.Color) {
		m.PopModal()
		onClosed(returnedColor)
	}
	picker.SetOnChangedFunc(onChanged)
	picker.SetOnClosedFunc(closedFunc)
	picker.SetColor(color)
	m.pushModal(picker)
}

func (m *Manager) IsShowingUI() bool {
	return m.uiScene.Cardinality() > 0
}

func (m *Manager) StartRectSelection(startPos geometry.Point, onFinished func(geometry.Rect)) {
	rectSelection := NewRectSelection(startPos)
	closeFunc := func(rect geometry.Rect) {
		m.RemoveFromScene(rectSelection)
		onFinished(rect)
	}
	rectSelection.SetOnFinished(closeFunc)
	m.AddToScene(rectSelection)
}
func (m *Manager) ShowStyledAlert(styledText []core.StyledText, bgColor common.Color) {
	alertHeight := len(styledText) + 2
	alertWidth := 0
	for _, line := range styledText {
		if line.Size().X > alertWidth {
			alertWidth = line.Size().X
		}
	}
	alertWidth += 2

	screenWidth, screenHeight := m.engine.ScreenGridWidth()*2, m.engine.ScreenGridHeight()

	x0 := (screenWidth - alertWidth) / 2
	y0 := (screenHeight - alertHeight) / 2
	boundsForCentering := geometry.NewRect(
		x0,
		y0,
		x0+alertWidth,
		y0+alertHeight,
	)
	alert := NewAlert(boundsForCentering, styledText, func() {
		m.PopModal()

	})
	alert.SetBackgroundColor(bgColor)
	m.pushModal(alert)
}
func (m *Manager) ShowAlert(strings []string) {
	textStyle := common.DefaultStyle.Reversed()
	formatted := make([]core.StyledText, len(strings))
	for i, str := range strings {
		formatted[i] = core.NewStyledText(str, textStyle)
	}
	m.ShowStyledAlert(formatted, textStyle.Background)
}
