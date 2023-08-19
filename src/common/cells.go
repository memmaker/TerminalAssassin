package common

var TransparentCell = Cell{Rune: ' ', Style: TransparentBackgroundStyle}

type Cell struct {
	Style Style
	Rune  rune
}

func (c Cell) WithBackgroundRGBColor(rgbColor RGBAColor) Cell {
	return Cell{Style: c.Style.WithRGBBg(rgbColor), Rune: c.Rune}
}
func (c Cell) IsTransparent() bool {
	return c.Rune == ' ' && c.Style.Background == Transparent
}
func (c Cell) WithForegroundRGBColor(rgbColor RGBAColor) Cell {
	return Cell{Style: c.Style.WithRGBFg(rgbColor), Rune: c.Rune}
}

func (c Cell) WithBackgroundColor(rgbColor Color) Cell {
	return Cell{Style: c.Style.WithBg(rgbColor), Rune: c.Rune}
}

func (c Cell) WithForegroundColor(rgbColor Color) Cell {
	return Cell{Style: c.Style.WithFg(rgbColor), Rune: c.Rune}
}

func (c Cell) WithRune(i int32) Cell {
	c.Rune = i
	return c
}

func (c Cell) WithStyle(newStyle Style) Cell {
	c.Style = newStyle
	return c
}
