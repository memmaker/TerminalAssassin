package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/utils"
)

type LabelState int

const (
	Normal LabelState = iota
	NeedsClear
)

type MovableLabel struct {
	currText   core.StyledText
	currOrigin geometry.Point
	currBounds geometry.Rect
	state      LabelState

	newText   core.StyledText
	newOrigin geometry.Point
	newBounds func(origin geometry.Point, length int) geometry.Rect
	isDirty   bool
}

func (l *MovableLabel) Update(input services.InputInterface) {
	if l.currBounds != l.newBounds(l.currOrigin, l.currText.Size().X) && l.state != NeedsClear {
		// camera has moved, we need to move the label
		l.newText = l.currText
		l.needsClear()
	}
}

func (l *MovableLabel) SetDirty() {
	l.isDirty = true
}

func NewMovableLabel(boundsFunc func(origin geometry.Point, length int) geometry.Rect) *MovableLabel {
	return &MovableLabel{currText: core.NewStyledText("", common.DefaultStyle), newBounds: boundsFunc}
}

// the crux is:
// when we want to clear this label, we'll have to wait for one draw to have happened..
// we have to clear the current label, every time we change the currText or the currPosition or do an explicit clear
func (l *MovableLabel) Draw(con console.CellInterface) {
	if l.state == NeedsClear {
		if !l.currBounds.Empty() {
			con.HalfWidthFill(l.currBounds, common.TransparentCell)
		}

		l.state = Normal
		l.updateBounds()
		return
	}

	if l.currBounds.Empty() || !l.isDirty {
		return
	}
	l.currText.DrawHalfWidth(con, l.currBounds, core.AlignLeft)
	l.isDirty = false
}
func (l *MovableLabel) needsClear() {
	if l.currText.Size().X == 0 && l.newText.Size().X == 0 && l.currBounds.Size().X == 0 {
		// no need to clear, we're not drawing anything
		l.state = Normal
		return
	}
	l.state = NeedsClear
}

func (l *MovableLabel) Clear() {
	l.currText = core.NewStyledText("", common.DefaultStyle)
	l.newText = l.currText
	l.needsClear()
}

func (l *MovableLabel) GetText() string {
	return l.currText.Text()
}
func (l *MovableLabel) updateBounds() {
	// we set the screenbox from the current position and text
	l.currText = l.newText
	l.currOrigin = l.newOrigin
	l.currBounds = l.newBounds(l.currOrigin, l.currText.Size().X)
	l.isDirty = true
}

func (l *MovableLabel) Set(origin geometry.Point, infoString core.StyledText) {
	l.newText = infoString
	l.newOrigin = origin
	l.needsClear()
}

func (l *MovableLabel) ScreenBounds() geometry.Rect {
	return l.currBounds
}

func (l *MovableLabel) IsEmpty() bool {
	return l.currText.Size().X == 0
}

type FixedLabel struct {
	text                   core.StyledText
	position               geometry.Point
	textWidth              int
	isDirty                bool
	bbox                   geometry.Rect
	fillFunc               func(con console.CellInterface, bbox geometry.Rect, style common.Style)
	drawFunc               func(con console.CellInterface, bbox geometry.Rect, styledText core.StyledText, align core.Alignment)
	autoClearAndAnimations bool
	clearTimer             int
	clearAnimationRunning  bool
	lerpFactor             float64
}

func NewSquareLabelWithWidth(text string, pos geometry.Point, width int) *FixedLabel {
	s := &FixedLabel{
		text:      core.NewStyledText(text, common.DefaultStyle),
		position:  pos,
		textWidth: width,
		fillFunc: func(con console.CellInterface, bbox geometry.Rect, style common.Style) {
			con.SquareFill(bbox, common.Cell{Rune: ' ', Style: style})
		},
		drawFunc: func(con console.CellInterface, bbox geometry.Rect, styledText core.StyledText, align core.Alignment) {
			styledText.DrawSquare(con, bbox, align)
		},
		isDirty: true,
	}
	s.updateBounds()
	return s
}

func NewHalfLabelWithWidth(text string, pos geometry.Point, width int) *FixedLabel {
	s := &FixedLabel{
		text:      core.NewStyledText(text, common.DefaultStyle),
		position:  pos,
		textWidth: width,
		fillFunc: func(con console.CellInterface, bbox geometry.Rect, style common.Style) {
			con.HalfWidthFill(bbox, common.Cell{Rune: ' ', Style: style})
		},
		drawFunc: func(con console.CellInterface, bbox geometry.Rect, styledText core.StyledText, align core.Alignment) {
			styledText.DrawHalfWidth(con, bbox, align)
		},
		isDirty: true,
	}
	s.updateBounds()
	return s
}

func (s *FixedLabel) Update(input services.InputInterface) {
	if s.autoClearAndAnimations && s.clearTimer > 0 {
		s.clearTimer--
		if s.clearTimer <= 0 {
			s.clearAnimationRunning = true
			s.isDirty = true
		}
	}
}

func (s *FixedLabel) Draw(con console.CellInterface) {
	if !s.isDirty {
		return
	}
	if s.clearAnimationRunning && s.autoClearAndAnimations {
		for i := 0; i < s.bbox.Size().X; i++ {
			textPos := geometry.Point{X: s.bbox.Min.X + i, Y: s.bbox.Min.Y}
			cellAt := con.AtHalfWidth(textPos)
			if cellAt.Style.Foreground != common.Black {
				darkerColor := cellAt.Style.Foreground.Lerp(common.Black, 0.05)
				con.SetHalfWidth(textPos, cellAt.WithForegroundColor(darkerColor))
			} else {
				s.clearAnimationRunning = false
			}
		}
	} else {
		textStyle := s.text.Style()
		if s.lerpFactor > 0 && s.autoClearAndAnimations {
			textStyle = textStyle.WithFg(textStyle.Foreground.Lerp(common.FourWhite, s.lerpFactor))
			s.lerpFactor -= 0.04
		}
		s.fillFunc(con, s.bbox, s.text.Style())
		s.drawFunc(con, s.bbox, s.text.WithStyle(textStyle), core.AlignLeft)
	}
	if !s.autoClearAndAnimations || (s.lerpFactor < 0 && !s.clearAnimationRunning) {
		s.isDirty = false
	}
}

func (s *FixedLabel) SetDirty() { s.isDirty = true }
func (s *FixedLabel) SetText(text string) {
	s.text = core.NewStyledText(text, common.DefaultStyle)
	s.resetClearTimer()
	s.clearAnimationRunning = false
	s.isDirty = true
	s.lerpFactor = 1.0
}

func (s *FixedLabel) SetStyledText(text core.StyledText) {
	s.text = text
	s.resetClearTimer()
	s.clearAnimationRunning = false
	s.isDirty = true
	s.lerpFactor = 1.0
}

func (s *FixedLabel) updateBounds() {
	s.bbox = geometry.NewRect(s.position.X, s.position.Y, s.position.X+s.textWidth, s.position.Y+1)
	s.isDirty = true
}

func (s *FixedLabel) SetAutoClearAndAnimations(autoClear bool) {
	s.autoClearAndAnimations = autoClear
	s.resetClearTimer()
}

func (s *FixedLabel) resetClearTimer() {
	if !s.autoClearAndAnimations || s.text.Empty() {
		return
	}
	s.clearTimer = utils.SecondsToTicks(4)
}
