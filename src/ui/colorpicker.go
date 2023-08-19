package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type ColorMode bool

const (
	ModeRGB ColorMode = false
	ModeHSV ColorMode = true
)

type ColorPicker struct {
	currentColor      common.Color
	isDirty           bool
	colorLabel        core.StyledText
	boundingBox       geometry.Rect
	changedFunc       func(color common.Color)
	closedFunc        func(color common.Color)
	channelXpositions []int
	rgbScale          float64
	currentMode       ColorMode
	oneScale          float64
}

func NewColorPicker(bounds geometry.Rect) *ColorPicker {
	if bounds.Size().Y != 3 {
		panic("ColorPicker must have a height of 3")
	}
	maxX := bounds.Max.X - 4
	maxChannelValue := 8.0
	scaleFactor := maxChannelValue / float64(maxX)
	return &ColorPicker{
		boundingBox:       bounds,
		isDirty:           true,
		channelXpositions: make([]int, 3),
		rgbScale:          scaleFactor,
		oneScale:          1.0 / float64(maxX),
	}
}

// setter for onChanged and onClosed
func (r *ColorPicker) SetOnChangedFunc(f func(color common.Color)) {
	r.changedFunc = f
}

func (r *ColorPicker) SetOnClosedFunc(f func(color common.Color)) {
	r.closedFunc = f
}
func (r *ColorPicker) SetColor(color common.Color) {
	r.currentColor = color
	r.updateXPositionsFromRGB()
	r.isDirty = true
}

func (r *ColorPicker) updateXPositionsFromRGB() {
	r.channelXpositions[0] = intClamp(int(r.currentColor.RValue()/r.rgbScale), 0, r.boundingBox.Max.X-4)
	r.channelXpositions[1] = intClamp(int(r.currentColor.GValue()/r.rgbScale), 0, r.boundingBox.Max.X-4)
	r.channelXpositions[2] = intClamp(int(r.currentColor.BValue()/r.rgbScale), 0, r.boundingBox.Max.X-4)
}

func (r *ColorPicker) updateXPositionsFromHSV() {
	r.channelXpositions[0] = intClamp(int(r.currentColor.HValue()/r.oneScale), 0, r.boundingBox.Max.X-4)
	r.channelXpositions[1] = intClamp(int(r.currentColor.SValue()/r.oneScale), 0, r.boundingBox.Max.X-4)
	r.channelXpositions[2] = intClamp(int(r.currentColor.VValue()/r.rgbScale), 0, r.boundingBox.Max.X-4)
}

func intClamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func (r *ColorPicker) Update(input services.InputInterface) {
	for _, cmd := range input.PollUICommands() {
		switch typedCmd := cmd.(type) {
		case core.PointerCommand:
			r.handlePointerCommand(typedCmd)
		}
	}

}

func (r *ColorPicker) handlePointerCommand(cmd core.PointerCommand) {
	mousePos := cmd.Pos
	if !r.boundingBox.Contains(mousePos) {
		if cmd.Action == core.MouseLeftReleased && !r.isDirty {
			r.closedFunc(r.currentColor)
		}
		return
	}
	// translate to coordinates relative to the bounding box
	mousePos = mousePos.Sub(r.boundingBox.Min)
	maxX := r.boundingBox.Max.X - 4
	isOnChannel := mousePos.X < maxX
	isOnButtons := mousePos.X >= maxX
	switch {
	case (cmd.Action == core.MouseLeftReleased || cmd.Action == core.MouseLeftHeld) && isOnChannel:
		if r.currentMode == ModeHSV {
			r.updateHSVValues(mousePos)
		} else if r.currentMode == ModeRGB {
			r.updateRGBValues(mousePos)
		}
	case cmd.Action == core.MouseLeftReleased && isOnButtons:
		if mousePos.X > maxX {
			// we clicked on the OK button
			r.closedFunc(r.currentColor)
		} else if r.currentMode == ModeHSV {
			r.switchToRGB()
		} else if r.currentMode == ModeRGB {
			r.switchToHSV()
		}
	}
}

func (r *ColorPicker) updateHSVValues(pos geometry.Point) {
	channel := pos.Y
	r.currentColor = r.HSVcolorFromXandChannel(pos.X, channel)
	r.updateXPositionsFromHSV()
	r.isDirty = true
	r.changedFunc(r.currentColor)
}

func (r *ColorPicker) HSVcolorFromXandChannel(xPos, channel int) common.HSVColor {
	maxX := r.boundingBox.Max.X - 4
	switch channel {
	case 0:
		newValue := common.Clamp(float64(xPos)*r.oneScale, 0.0, r.oneScale*float64(maxX))
		return common.HSVColor{H: newValue, S: r.currentColor.SValue(), V: r.currentColor.VValue()}
	case 1:
		newValue := common.Clamp(float64(xPos)*r.oneScale, 0.0, r.oneScale*float64(maxX))
		return common.HSVColor{H: r.currentColor.HValue(), S: newValue, V: r.currentColor.VValue()}
	case 2:
		newValue := common.Clamp(float64(xPos)*r.rgbScale, 0.0, r.rgbScale*float64(maxX))
		return common.HSVColor{H: r.currentColor.HValue(), S: r.currentColor.SValue(), V: newValue}
	}
	return common.HSVColor{}
}
func (r *ColorPicker) updateRGBValues(mousePos geometry.Point) {
	channel := mousePos.Y
	r.currentColor = r.RGBcolorFromXandChannel(mousePos.X, channel)
	r.updateXPositionsFromRGB()
	r.isDirty = true
	r.changedFunc(r.currentColor)
}

func (r *ColorPicker) RGBcolorFromXandChannel(xPos, channel int) common.Color {
	maxX := r.boundingBox.Max.X - 4
	newValue := common.Clamp(float64(xPos)*r.rgbScale, 0.0, r.rgbScale*float64(maxX))
	switch channel {
	case 0:
		return common.RGBAColor{R: newValue, G: r.currentColor.GValue(), B: r.currentColor.BValue(), A: 1.0}
	case 1:
		return common.RGBAColor{R: r.currentColor.RValue(), G: newValue, B: r.currentColor.BValue(), A: 1.0}
	case 2:
		return common.RGBAColor{R: r.currentColor.RValue(), G: r.currentColor.GValue(), B: newValue, A: 1.0}
	}
	return r.currentColor
}

func (r *ColorPicker) Draw(con console.CellInterface) {
	if !r.isDirty {
		return
	}
	maxX := r.boundingBox.Max.X - 4
	for yCoord := r.boundingBox.Min.Y; yCoord < r.boundingBox.Max.Y; yCoord++ {
		for xCoord := r.boundingBox.Min.X; xCoord < maxX; xCoord++ {
			channel := yCoord - r.boundingBox.Min.Y
			toDraw := '─'
			if xCoord == r.channelXpositions[channel] {
				toDraw = '+'
			}
			backgroundColor := common.Color(common.Black)
			if r.currentMode == ModeRGB {
				backgroundColor = r.RGBcolorFromXandChannel(xCoord, channel)
			} else if r.currentMode == ModeHSV {
				hsvBackground := r.HSVcolorFromXandChannel(xCoord, channel)
				if channel == 0 {
					backgroundColor = hsvBackground.WithS(1.0).WithV(1.0)
				} else {
					backgroundColor = hsvBackground
				}
			}

			con.SetSquare(
				geometry.Point{X: xCoord, Y: yCoord},
				common.Cell{
					Style: common.Style{Foreground: common.White, Background: backgroundColor},
					Rune:  toDraw,
				},
			)
		}
	}
	// we want three buttons at the right corner, taking three cells each
	// the first is labeled RGB and the second is labeled HSV, the last reads "OK"
	r.Print(con, geometry.Point{X: maxX + 1, Y: r.boundingBox.Min.Y + 1}, " OK")
	if r.currentMode == ModeRGB {
		// we want draw it vertically
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y}, common.Cell{Style: common.DefaultStyle, Rune: 'R'})
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y + 1}, common.Cell{Style: common.DefaultStyle, Rune: 'G'})
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y + 2}, common.Cell{Style: common.DefaultStyle, Rune: 'B'})
	} else if r.currentMode == ModeHSV {
		// we want draw it vertically
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y}, common.Cell{Style: common.DefaultStyle, Rune: 'H'})
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y + 1}, common.Cell{Style: common.DefaultStyle, Rune: 'S'})
		con.SetSquare(geometry.Point{X: maxX, Y: r.boundingBox.Min.Y + 2}, common.Cell{Style: common.DefaultStyle, Rune: 'V'})
	}
	r.isDirty = false
}

func (r *ColorPicker) SetDirty() {
	r.isDirty = true
}

func (r *ColorPicker) Print(con console.CellInterface, point geometry.Point, text string) {
	for _, char := range text {
		con.SetSquare(point, common.Cell{Style: common.DefaultStyle, Rune: char})
		point.X++
	}
}

func (r *ColorPicker) switchToRGB() {
	r.currentMode = ModeRGB
	r.updateXPositionsFromRGB()
	r.isDirty = true
}

func (r *ColorPicker) switchToHSV() {
	r.currentMode = ModeHSV
	r.updateXPositionsFromHSV()
	r.isDirty = true
}
