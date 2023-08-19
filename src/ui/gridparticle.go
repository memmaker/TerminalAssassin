package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type CursorParticle struct {
	Pos             geometry.Point
	Color           common.HSVColor
	DelayAppearance int
	IsDeadNow       bool
}

func (s *CursorParticle) Kill() {
	s.IsDeadNow = true
}

func (s *CursorParticle) IsDead() bool {
	return s.IsDeadNow
}

func (s *CursorParticle) Update(engine services.Engine) {
	if s.DelayAppearance > 0 {
		s.DelayAppearance--
		return
	}
}
func (s *CursorParticle) IsDelayed() bool {
	return s.DelayAppearance > 0
}
func (s *CursorParticle) Draw(grid console.CellInterface) {
	if s.IsDelayed() {
		return
	}
	if s.Color.V <= 0.0 {
		s.Color = s.Color.WithV(1.0)
	} else {
		s.Color = s.Color.WithV(s.Color.V - 0.01)
	}
	grid.SetHalfWidth(s.Pos, common.Cell{Rune: ' ', Style: common.Style{Foreground: common.Black, Background: s.Color}})
}

type AppearingCharacterParticle struct {
	Char         int32
	Pos          geometry.Point
	FgColor      common.HSVColor
	BgColorStart common.HSVColor
	Ticks        uint64
	Lifetime     uint64
	Delay        int
	SoundPlayed  bool
	OnFinish     func()
}

func (s *AppearingCharacterParticle) IsDead() bool {
	return s.Ticks >= s.Lifetime
}

func (s *AppearingCharacterParticle) Update(engine services.Engine) {
	if s.Delay > 0 {
		s.Delay--
		return
	}
	if !s.SoundPlayed {
		engine.GetAudio().PlayCue("key")
		s.SoundPlayed = true
	}
	s.Ticks++
	if s.IsDead() && s.OnFinish != nil {
		s.OnFinish()
	}
}
func (s *AppearingCharacterParticle) IsDelayed() bool {
	return s.Delay > 0
}
func (s *AppearingCharacterParticle) IsSettled() bool {
	return s.Ticks >= s.Lifetime-5
}
func (s *AppearingCharacterParticle) Draw(grid console.CellInterface) {
	if s.IsDelayed() {
		return
	}
	percentComplete := common.Clamp(float64(s.Ticks)/float64(s.Lifetime), 0.0, 1.0)
	//bglight := 1.0 - percentComplete
	drawBgColor := s.BgColorStart.ToRGB().Lerp(common.TerminalColorBackground, percentComplete)

	grid.SetHalfWidth(s.Pos, common.Cell{Rune: s.Char, Style: common.Style{Foreground: s.FgColor, Background: drawBgColor}})
}

type FlashyLineParticle struct {
	Char     int32
	Pos      geometry.Point
	FgColor  common.HSVColor
	BgColor  common.HSVColor
	Ticks    uint64
	Lifetime uint64
	Delay    int
}

func (s *FlashyLineParticle) IsDead() bool {
	return s.Ticks >= s.Lifetime
}

func (s *FlashyLineParticle) Update(engine services.Engine) {
	if s.Delay > 0 {
		s.Delay--
		return
	}
	s.Ticks++
}
func (s *FlashyLineParticle) IsDelayed() bool {
	return s.Delay > 0
}
func (s *FlashyLineParticle) IsSettled() bool {
	return s.Ticks >= s.Lifetime-5
}
func (s *FlashyLineParticle) Draw(grid console.CellInterface) {
	if s.IsDelayed() {
		return
	}
	percentComplete := float64(s.Ticks) / float64(s.Lifetime)
	drawFgColor := s.FgColor.WithV(percentComplete)

	grid.SetSquare(s.Pos, common.Cell{Rune: s.Char, Style: common.Style{Foreground: drawFgColor, Background: s.BgColor}})
}

type LineDrawerParticle struct {
	FgColor     common.HSVColor
	BgColor     common.HSVColor
	Ticks       int
	OnFinish    func()
	leftPos     geometry.Point
	rightPos    geometry.Point
	hasTicked   bool
	hasFinished bool
	Box         geometry.Rect
}

func (s *LineDrawerParticle) IsDead() bool {
	return s.hasFinished
}

func (s *LineDrawerParticle) ShouldDraw() bool {
	ticksNeeded := s.Box.Size().X / 2
	return s.Ticks <= ticksNeeded
}

func (s *LineDrawerParticle) Update(engine services.Engine) {
	animator := engine.GetAnimator()
	ticksNeeded := s.Box.Size().X / 2
	if s.Ticks < ticksNeeded {
		s.hasTicked = true
		s.leftPos = geometry.Point{X: s.Box.Mid().X - 1 - s.Ticks, Y: s.Box.Mid().Y}
		s.rightPos = geometry.Point{X: s.Box.Mid().X + s.Ticks, Y: s.Box.Mid().Y}
		s.Ticks++

		animator.AddParticle(&FlashyLineParticle{
			Char:     '─',
			Pos:      s.leftPos,
			FgColor:  s.FgColor,
			BgColor:  s.BgColor,
			Lifetime: uint64(ticksNeeded - s.Ticks),
			Delay:    0,
		})
		animator.AddParticle(&FlashyLineParticle{
			Char:     '─',
			Pos:      s.rightPos,
			FgColor:  s.FgColor,
			BgColor:  s.BgColor,
			Lifetime: uint64(ticksNeeded - s.Ticks),
			Delay:    0,
		})
	} else if !s.hasFinished {
		s.hasFinished = true
		if s.OnFinish != nil {
			s.OnFinish()
		}
	}
}

func (s *LineDrawerParticle) Draw(grid console.CellInterface) {

}

type RevealParticle struct {
	Position geometry.Point
	OldPos   geometry.Point
	EndPos   geometry.Point
	FgColor  common.HSVColor
	BgColor  common.HSVColor
	Ticks    int
	Box      geometry.Rect
	OnFinish func()
}

func (s *RevealParticle) IsDead() bool {
	return s.Position == s.EndPos
}

func (s *RevealParticle) Update(engine services.Engine) {
	animator := engine.GetAnimator()

	s.OldPos = s.Position
	if s.Position.X < s.EndPos.X {
		if s.Ticks > 1 {
			s.Position.X++
		}
	}
	if s.Position.X > s.EndPos.X {
		if s.Ticks > 1 {
			s.Position.X--
		}
	}
	if s.Position.Y < s.EndPos.Y {
		if s.Ticks > 1 {
			s.Position.Y++
		}
	}
	if s.Position.Y > s.EndPos.Y {
		if s.Ticks > 1 {
			s.Position.Y--
		}
	}
	s.Ticks++
	if s.Position == s.EndPos {
		isGoingUp := s.Position.Y < s.OldPos.Y
		if s.OldPos.X != s.Box.Min.X && s.OldPos.X != s.Box.Max.X {
			animator.AddParticle(&FlashyLineParticle{
				Char:     '─',
				Pos:      s.Position,
				FgColor:  s.FgColor,
				BgColor:  s.BgColor,
				Lifetime: 30,
				Delay:    0,
			})
		} else if s.OldPos.X == s.Box.Min.X && isGoingUp {
			animator.AddParticle(&FlashyLineParticle{
				Char:     '┌',
				Pos:      s.Position,
				FgColor:  s.FgColor,
				BgColor:  s.BgColor,
				Lifetime: 30,
				Delay:    0,
			})
		} else if s.OldPos.X == s.Box.Min.X && !isGoingUp {
			animator.AddParticle(&FlashyLineParticle{
				Char:     '└',
				Pos:      s.Position,
				FgColor:  s.FgColor,
				BgColor:  s.BgColor,
				Lifetime: 30,
				Delay:    0,
			})
		} else if s.OldPos.X == s.Box.Max.X && isGoingUp {
			animator.AddParticle(&FlashyLineParticle{
				Char:     '┐',
				Pos:      s.Position,
				FgColor:  s.FgColor,
				BgColor:  s.BgColor,
				Lifetime: 30,
				Delay:    0,
			})
		} else if s.OldPos.X == s.Box.Max.X && !isGoingUp {
			animator.AddParticle(&FlashyLineParticle{
				Char:     '┘',
				Pos:      s.Position,
				FgColor:  s.FgColor,
				BgColor:  s.BgColor,
				Lifetime: 30,
				Delay:    0,
			})
		}
		if s.OnFinish != nil {
			s.OnFinish()
		}
	}
	if s.OldPos != s.Position && (s.OldPos.X == s.Box.Min.X || s.OldPos.X == s.Box.Max.X) {
		//grid.SetSquare(s.OldPos, common.Cell{Rune: drawChar, Foreground: s.FgColor, Background: s.Color})
		animator.AddParticle(&FlashyLineParticle{
			Char:     '│',
			Pos:      s.OldPos,
			FgColor:  s.FgColor,
			BgColor:  s.BgColor,
			Lifetime: 30,
			Delay:    0,
		})
	}
}

func (s *RevealParticle) Draw(grid console.CellInterface) {
	drawChar := '─'

	isGoingUp := s.Position.Y < s.OldPos.Y

	if s.Position.X == s.Box.Min.X && isGoingUp {
		drawChar = '┌'
	} else if s.Position.X == s.Box.Min.X && !isGoingUp {
		drawChar = '└'
	} else if s.Position.X == s.Box.Max.X && isGoingUp {
		drawChar = '┐'
	} else if s.Position.X == s.Box.Max.X && !isGoingUp {
		drawChar = '┘'
	}

	drawColor := s.FgColor.WithS(0.0)

	grid.SetSquare(s.Position, common.Cell{Rune: drawChar, Style: common.Style{Foreground: drawColor, Background: s.BgColor}})
	if s.OldPos != s.Position {
		if s.OldPos.X != s.Box.Min.X && s.OldPos.X != s.Box.Max.X {
			grid.SetSquare(s.OldPos, common.Cell{Rune: ' ', Style: common.Style{Foreground: s.FgColor, Background: s.BgColor}})
		} else {
			grid.SetSquare(s.OldPos, common.Cell{Rune: '│', Style: common.Style{Foreground: drawColor, Background: s.BgColor}})
		}
	}
}
