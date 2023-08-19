package common

import (
	"fmt"
	"image/color"
	"math"
	"time"
)

// Cell represents a cell in the console
var White = RGBAColor{R: 1.0, G: 1.0, B: 1.0, A: 1.0}

var FourWhite = RGBAColor{R: 4.0, G: 4.0, B: 4.0, A: 1.0}
var FourGreen = RGBAColor{R: 0.2, G: 4.0, B: 0.2, A: 1.0}

var Gray = RGBAColor{R: 0.5, G: 0.5, B: 0.5, A: 1.0}
var Black = RGBAColor{R: 0.0, G: 0.0, B: 0.0, A: 1.0}
var Yellow = RGBAColor{R: 1.0, G: 1.0, B: 0.0, A: 1.0}
var TerminalColor = RGBAColor{R: 1.0 * 3, G: 0.635294117647059 * 3, B: (1 / 255.0) * 3, A: 1.0}
var TerminalColorBackground = RGBAColor{R: 13 / 255.0, G: 8 / 255.0, B: 1 / 255.0, A: 1.0}

var Transparent = RGBAColor{R: 0, G: 0, B: 0, A: 0}
var Red = RGBAColor{R: 1.0, G: 0.0, B: 0.0, A: 1.0}
var Green = RGBAColor{R: 0.0, G: 1.0, B: 0.0, A: 1.0}
var Blue = RGBAColor{R: 0.0, G: 0.0, B: 1.0, A: 1.0}

var LegalActionGreen = RGBAColor{R: 51 / 255.0, G: 255 / 255.0, B: 93 / 255.0, A: 1.0}
var IllegalActionRed = RGBAColor{R: 191 / 255.0, G: 53 / 255.0, B: 29 / 255.0, A: 1.0}

var TransparentBackgroundStyle = Style{Foreground: White, Background: Transparent}

// HSVColor represents a RGBA color in the console
type HSVColor struct {
	H float64 // [0, 1]
	S float64 // [0, 1]
	V float64 // [0, 1]
}

func (c HSVColor) EncodeAsString() string {
	return c.ToRGB().EncodeAsString()
}

func (c HSVColor) Multiply(color Color) Color {
	return c.ToRGBColor().Multiply(color)
}

func (c HSVColor) AValue() float64 {
	return 1.0
}

func (c HSVColor) Lerp(pixel Color, percent float64) Color {
	return c.ToRGBColor().Lerp(pixel, percent)
}

func (c HSVColor) ToRGB() RGBAColor {
	return c.ToRGBColor()
}

func (c HSVColor) HValue() float64 {
	return c.H
}

func (c HSVColor) SValue() float64 {
	return c.S
}

func (c HSVColor) VValue() float64 {
	return c.V
}

func (c HSVColor) MultiplyWithScalar(f float64) Color {
	return c.WithV(c.V * f)
}

func (c HSVColor) Desaturate() Color {
	return c.WithS(0.0)
}

func (c HSVColor) ToHSV() HSVColor {
	return c
}

func (c HSVColor) RValue() float64 {
	return c.ToRGBColor().R
}

func (c HSVColor) GValue() float64 {
	return c.ToRGBColor().G
}

func (c HSVColor) BValue() float64 {
	return c.ToRGBColor().B
}

type RGBAColor struct {
	R float64
	G float64
	B float64
	A float64
}

func (R RGBAColor) EncodeAsString() string {
	return fmt.Sprintf("(%.2f, %.2f, %.2f)", R.R, R.G, R.B)
}
func NewColorFromString(colorString string) Color {
	var r, g, b float64
	fmt.Sscanf(colorString, "(%f, %f, %f)", &r, &g, &b)
	return RGBAColor{r, g, b, 1.0}
}
func (R RGBAColor) Multiply(color Color) Color {
	return RGBAColor{
		R: R.R * color.RValue(),
		G: R.G * color.GValue(),
		B: R.B * color.BValue(),
		A: R.A * color.AValue(),
	}
}

func (R RGBAColor) Lerp(otherColor Color, percent float64) Color {
	return RGBAColor{
		R: Lerp(R.R, otherColor.RValue(), percent),
		G: Lerp(R.G, otherColor.GValue(), percent),
		B: Lerp(R.B, otherColor.BValue(), percent),
		A: Lerp(R.A, otherColor.AValue(), percent),
	}
}

func Lerp(first float64, second float64, percent float64) float64 {
	return first + percent*(second-first)
}

func (R RGBAColor) ToRGB() RGBAColor {
	return R
}

func (R RGBAColor) HValue() float64 {
	return R.ToHSV().H
}

func (R RGBAColor) SValue() float64 {
	return R.ToHSV().S
}

func (R RGBAColor) VValue() float64 {
	return R.ToHSV().V
}

func (R RGBAColor) AValue() float64 {
	return R.A
}
func (R RGBAColor) Desaturate() Color {
	luminance := Clamp(R.RelativeLuminance(), 0.0, 1.0)
	//luminance := Clamp(R.Lightness(), 0.0, 1.0)
	return RGBAColor{luminance, luminance, luminance, 1.0}
}

func (R RGBAColor) MultiplyWithScalar(factor float64) Color {
	return RGBAColor{R.R * factor, R.G * factor, R.B * factor, R.A}
}

func (R RGBAColor) RValue() float64 {
	return R.R
}

func (R RGBAColor) GValue() float64 {
	return R.G
}

func (R RGBAColor) BValue() float64 {
	return R.B
}

func (R RGBAColor) ExposureToneMapping() (r, g, b, a uint32) {
	exposure := 1.0
	//lightness := R.Lightness()
	scale := float64(0xffff) // * math.Sqrt(lightness)
	// vec3 mapped = hdrColor / (hdrColor + vec3(1.0));
	//gamma := 2.2
	mappedR := 1.0 - math.Exp(-(R.R * exposure))
	r = uint32(mappedR * scale)
	mappedG := 1.0 - math.Exp(-(R.G * exposure))
	g = uint32(mappedG * scale)
	mappedB := 1.0 - math.Exp(-(R.B * exposure))
	b = uint32(mappedB * scale)
	a = uint32(R.A * scale)
	return r, g, b, a
}

func ACESFilm(x float64) float64 {
	a := 2.51
	b := 0.03
	c := 2.43
	d := 0.59
	e := 0.14
	return Clamp((x*(a*x+b))/(x*(c*x+d)+e), 0.0, 1.0)
}

func (R RGBAColor) ACESFilmMapping() (r, g, b, a uint32) {

	//lightness := R.Lightness()
	scale := float64(0xffff) // * math.Sqrt(lightness)
	// vec3 mapped = hdrColor / (hdrColor + vec3(1.0));
	//gamma := 2.2
	mappedR := ACESFilm(R.R)
	r = uint32(mappedR * scale)
	mappedG := ACESFilm(R.G)
	g = uint32(mappedG * scale)
	mappedB := ACESFilm(R.B)
	b = uint32(mappedB * scale)
	a = uint32(R.A * scale)
	return r, g, b, a
}
func (R RGBAColor) ReinhardToneMapping() (r, g, b, a uint32) {
	lightness := R.Lightness()
	scale := 0xffff * math.Sqrt(lightness)
	// vec3 mapped = hdrColor / (hdrColor + vec3(1.0));
	//gamma := 2.2
	mappedR := (R.R * scale) / (R.R + 1.0)
	//mappedR = math.Pow(mappedR, 1.0/gamma)
	r = uint32(mappedR)
	mappedG := (R.G * scale) / (R.G + 1.0)
	//mappedG = math.Pow(mappedG, 1.0/gamma)
	g = uint32(mappedG)
	mappedB := (R.B * scale) / (R.B + 1.0)
	//mappedB = math.Pow(mappedB, 1.0/gamma)
	b = uint32(mappedB)
	a = uint32(0xffff)
	return r, g, b, a
}
func (R RGBAColor) LightnessScaledToneMapping() (r, g, b, a uint32) {
	lightness := R.Luminance()
	scale := 0xffff * math.Sqrt(lightness)
	r = uint32(R.R * scale)
	g = uint32(R.G * scale)
	b = uint32(R.B * scale)
	a = uint32(0xffff)
	return r, g, b, a
}

// RelativeLuminance() is gamma-compressed
func (R RGBAColor) RelativeLuminance() float64 {
	return 0.2126*R.R + 0.7152*R.G + 0.0722*R.B
}

// Luminance() is not gamma-compressed
func (R RGBAColor) Luminance() float64 {
	return 0.2126*degamma(R.R) + 0.7152*degamma(R.G) + 0.0722*degamma(R.B)
}

func (R RGBAColor) Lightness() float64 {
	y := R.RelativeLuminance()
	var result float64
	if y <= (216.0 / 24389) {
		result = y * (24389.0 / 27)
	} else {
		result = math.Pow(y, 1/3.0)*116.0 - 16.0
	}
	result /= 100.0
	return result
}
func degamma(channelValue float64) float64 {
	// Send this function a decimal sRGB gamma encoded color value
	// between 0.0 and 1.0, and it returns a linearized value.
	if channelValue <= 0.04045 {
		return channelValue / 12.92
	} else {
		return math.Pow((channelValue+0.055)/1.055, 2.4)
	}
}

func (R RGBAColor) WithClampTo(intensity float64) RGBAColor {
	return RGBAColor{
		R: Clamp(R.R, 0, intensity),
		G: Clamp(R.G, 0, intensity),
		B: Clamp(R.B, 0, intensity),
		A: R.A,
	}
}

func (R RGBAColor) ToHSV() HSVColor {
	return NewHSVColorFromRGB(R.R, R.G, R.B)
}

func NewHSVColor(h, s, v float64) HSVColor {
	return HSVColor{h, s, v}
}

// NewHSVColorFromRGBBytes creates a new color from R,G,B values
func NewHSVColorFromRGBBytes(r, g, b byte) HSVColor {
	return NewHSVColor(RGBtoHSV(float64(r)/255.0, float64(g)/255.0, float64(b)/255.0))
}
func NewRGBColorFromBytes(r, g, b byte) RGBAColor {
	rgbColor := RGBAColor{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0, 1.0}
	return rgbColor
}
func NewHSVColorFromRGB(r, g, b float64) HSVColor {
	return NewHSVColor(RGBtoHSV(r, g, b))
}

func HSLColor(h, s, l float64) HSVColor {
	return NewHSVColor(HSLtoHSV(h, s, l))
}

func HSVtoHSL(h float64, s float64, v float64) (float64, float64, float64) {
	// both hsv and hsl values are in [0, 1]
	l := (2 - s) * v / 2
	if l != 0 {
		if l == 1 {
			s = 0
		} else if l < 0.5 {
			s = s * v / (l * 2)
		} else {
			s = s * v / (2 - l*2)
		}
	}

	return h, s, l
}

func HSLtoHSV(hslH float64, hslS float64, hslL float64) (float64, float64, float64) {
	// both hsv and hsl values are in [0, 1]
	var hsvH, hsvS, hsvV float64
	hsvH = hslH
	hsvV = hslL + hslS*math.Min(hslL, 1-hslL)
	if hsvV == 0 {
		hsvS = 0
	} else {
		hsvS = 2 * (1 - hslL/hsvV)
	}
	return hsvH, hsvS, hsvV
}
func RGBtoHSV(fR float64, fG float64, fB float64) (h, s, v float64) {
	max := math.Max(math.Max(fR, fG), fB)
	min := math.Min(math.Min(fR, fG), fB)
	d := max - min
	s, v = 0, max
	if max > 0 {
		s = d / max
	}
	if max == min {
		// Achromatic.
		h = 0
	} else {
		// Chromatic.
		switch max {
		case fR:
			h = (fG - fB) / d
			if fG < fB {
				h += 6
			}
		case fG:
			h = (fB-fR)/d + 2
		case fB:
			h = (fR-fG)/d + 4
		}
		h /= 6
	}
	return
}

// RGBA returns the color values as uint32s
func (c HSVColor) RGBA() (r, g, b, a uint32) {
	cr, cg, cb := HSVtoRGB(c.H, c.S, c.V)
	col := RGBAColor{
		R: cr,
		G: cg,
		B: cb,
		A: 1.0,
	}
	return col.ExposureToneMapping()
	//return uint32(cr * 0xFFFF), uint32(cg * 0xFFFF), uint32(cb * 0xFFFF), uint32(0xFFFF)
}
func RGBAFrom32Bits(r, g, b, a uint32) RGBAColor {
	return RGBAColor{
		R: float64(r) / float64(0xFFFF),
		G: float64(g) / float64(0xFFFF),
		B: float64(b) / float64(0xFFFF),
		A: float64(a) / float64(0xFFFF),
	}
}
func (R RGBAColor) RGBA() (r, g, b, a uint32) {
	return R.ExposureToneMapping()
	/*
		scale := float64(0xffff)
		clampedR := Clamp(R.R, 0.0, 1.0) * scale
		clampedG := Clamp(R.G, 0.0, 1.0) * scale
		clampedB := Clamp(R.B, 0.0, 1.0) * scale
		return uint32(clampedR), uint32(clampedG), uint32(clampedB), uint32(1.0 * scale)
	*/
}

func (R RGBAColor) WithAlpha(f float64) RGBAColor {
	return RGBAColor{
		R: R.R,
		G: R.G,
		B: R.B,
		A: f,
	}
}

func (R RGBAColor) AddRGB(lightColor Color) RGBAColor {
	return RGBAColor{
		R: R.R + lightColor.RValue(),
		G: R.G + lightColor.GValue(),
		B: R.B + lightColor.BValue(),
		A: R.A,
	}
}
func AlphaBlend(new, curr color.Color) color.Color {
	nr, ng, nb, na := new.RGBA()
	if na == 0xFFFF {
		return new
	}
	if na == 0 {
		return curr
	}
	cr, cg, cb, ca := curr.RGBA()
	if ca == 0 {
		return new
	}

	return color.RGBA64{
		R: uint16((nr*0xFFFF + cr*(0xFFFF-na)) / 0xFFFF),
		G: uint16((ng*0xFFFF + cg*(0xFFFF-na)) / 0xFFFF),
		B: uint16((nb*0xFFFF + cb*(0xFFFF-na)) / 0xFFFF),
		A: uint16((na*0xFFFF + ca*(0xFFFF-na)) / 0xFFFF),
	}
}

// P returns a pointer to the color
func (c HSVColor) P() *HSVColor {
	return &c
}

func (c HSVColor) Lighten(scale float64) HSVColor {
	// scale must be between 0 and 1
	oldV := c.V
	interval := 1.0 - oldV
	newV := math.Min(1.0, oldV+interval*scale)
	return NewHSVColor(c.H, c.S, newV)
}

func (c HSVColor) WithV(value float64) HSVColor {
	return NewHSVColor(c.H, c.S, value)
}

func (c HSVColor) ToRGBColor() RGBAColor {
	r, g, b := HSVtoRGB(c.H, c.S, c.V)
	return RGBAColor{r, g, b, 1.0}
}

func HSVtoRGB(h float64, s float64, v float64) (float64, float64, float64) {
	hThreeSixty := h * 360.0
	Hp := hThreeSixty / 60.0
	c := v * s
	x := c * (1.0 - math.Abs(math.Mod(Hp, 2.0)-1.0))

	m := v - c
	r, g, b := 0.0, 0.0, 0.0

	switch {
	case 0.0 <= Hp && Hp < 1.0:
		r = c
		g = x
	case 1.0 <= Hp && Hp < 2.0:
		r = x
		g = c
	case 2.0 <= Hp && Hp < 3.0:
		g = c
		b = x
	case 3.0 <= Hp && Hp < 4.0:
		g = x
		b = c
	case 4.0 <= Hp && Hp < 5.0:
		r = x
		b = c
	case 5.0 <= Hp && Hp < 6.0:
		r = c
		b = x
	}
	return m + r, m + g, m + b
}

func (c HSVColor) BlendRGB(rgbColor RGBAColor, value float64) HSVColor {

	solidR, solidG, solidB := HSVtoRGB(c.H, c.S, c.V)
	lightR := rgbColor.R
	lightG := rgbColor.G
	lightB := rgbColor.B

	mixedR := Clamp(lightR+solidR, 0, 1)
	mixedG := Clamp(lightG+solidG, 0, 1)
	mixedB := Clamp(lightB+solidB, 0, 1)

	return NewHSVColorFromRGB(mixedR, mixedG, mixedB)
}

func NewHSVColorFromRGBA(r float64, g float64, b float64, a float64) HSVColor {
	h, s, v := RGBAtoHSV(r, g, b, a)
	return HSVColor{h, s, v}
}

func RGBAtoHSV(r float64, g float64, b float64, a float64) (float64, float64, float64) {
	max := math.Max(math.Max(r, g), b)
	min := math.Min(math.Min(r, g), b)
	delta := max - min
	h := 0.0
	s := 0.0
	v := max

	if max != 0 {
		s = delta / max
	}

	if s != 0 {
		if r == max {
			h = (g - b) / delta
		} else if g == max {
			h = 2 + (b-r)/delta
		} else {
			h = 4 + (r-g)/delta
		}
		h *= 60
		if h < 0 {
			h += 360
		}
	}

	return h, s, v
}

func (c HSVColor) WithH(h float64) HSVColor {
	return NewHSVColor(h, c.S, c.V)
}

func (c HSVColor) LerpH(h float64, ratio float64) HSVColor {
	return NewHSVColor(c.H+(h-c.H)*ratio, c.S, c.V)
}

func (c HSVColor) WithS(saturation float64) HSVColor {
	return NewHSVColor(c.H, saturation, c.V)
}

func Clamp(f float64, min float64, max float64) float64 {
	if f < min {
		return min
	}
	if f > max {
		return max
	}
	return f
}

type Color interface {
	RGBA() (r uint32, g uint32, b uint32, a uint32)

	RValue() float64
	GValue() float64
	BValue() float64
	AValue() float64

	ToRGB() RGBAColor
	HValue() float64
	SValue() float64
	VValue() float64
	ToHSV() HSVColor

	Desaturate() Color
	MultiplyWithScalar(f float64) Color
	Lerp(pixel Color, percent float64) Color
	Multiply(color Color) Color
	EncodeAsString() string
}

type Style struct {
	Foreground Color
	Background Color
}

// WithFg returns a derived style with a new foreground color.
func (st Style) WithFg(cl Color) Style {
	st.Foreground = cl
	return st
}

// WithBg returns a derived style with a new background color.
func (st Style) WithBg(cl Color) Style {
	st.Background = cl
	return st
}

func (st Style) WithRGBBg(rgbColor RGBAColor) Style {
	st.Background = rgbColor
	return st
}

func (st Style) WithRGBFg(rgbColor RGBAColor) Style {
	st.Foreground = rgbColor
	return st
}

func (st Style) Reversed() Style {
	return Style{Foreground: st.Background, Background: st.Foreground}
}

func (st Style) Desaturate() Style {
	return Style{Foreground: st.Foreground.Desaturate(), Background: st.Background.Desaturate()}
}

func (st Style) Darken(f float64) Style {
	return Style{Foreground: st.Foreground.MultiplyWithScalar(f), Background: st.Background.MultiplyWithScalar(f)}
}

var DefaultStyle = Style{Foreground: White, Background: Black}

var TerminalStyle = Style{Foreground: TerminalColor, Background: TerminalColorBackground}

func GetAmbientLightFromDayTime(timeOfDay time.Time) Color {
	morning := RGBAColor{R: 0.4, G: 0.368, B: 0.3466666666666667, A: 1.0}
	noon := RGBAColor{R: 1.8, G: 1.7966666666666666, B: 1.7666666666666666, A: 1.0}
	evening := RGBAColor{R: 0.2177777777777778, G: 0.2177777777777778, B: 0.26666666666666666, A: 1.0}
	night := RGBAColor{R: 0.1111111111111111, G: 0.1111111111111111, B: 0.13333333333333333, A: 1.0}

	secondsSinceMidnight := timeOfDay.Hour()*3600 + timeOfDay.Minute()*60 + timeOfDay.Second()

	// we need to determine the percentage of the interval between the two times
	// for example, if it's 9:00, we need to know how far we are between 6:00 and 12:00
	// 9:00 is 50% of the way between 6:00 and 12:00
	// BUT: we want to be precise, so we need to know how many seconds are in the interval
	// 6:00 - 12:00 is 6 hours, so 6 * 3600 = 21600 seconds
	// that means we got these intervals:
	// 0:00 - 6:00 = 0 - 21600
	// 6:00 - 12:00 = 21600 - 43200
	// 12:00 - 18:00 = 43200 - 64800
	// 18:00 - 24:00 = 64800 - 86400

	var ambientLightColor Color
	startColor := morning
	endColor := noon
	// 6:00 - 12:00
	if secondsSinceMidnight >= 21600 && secondsSinceMidnight < 43200 {
		intervalPercentage := float64(secondsSinceMidnight-21600) / 21600.0
		ambientLightColor = startColor.Lerp(endColor, intervalPercentage)
	} else if secondsSinceMidnight >= 43200 && secondsSinceMidnight < 64800 {
		startColor = noon
		endColor = evening
		intervalPercentage := float64(secondsSinceMidnight-43200) / 21600.0
		ambientLightColor = startColor.Lerp(endColor, intervalPercentage)
	} else if secondsSinceMidnight >= 64800 && secondsSinceMidnight < 86400 {
		startColor = evening
		endColor = night
		intervalPercentage := float64(secondsSinceMidnight-64800) / 21600.0
		ambientLightColor = startColor.Lerp(endColor, intervalPercentage)
	} else if secondsSinceMidnight >= 0 && secondsSinceMidnight < 21600 {
		startColor = night
		endColor = morning
		intervalPercentage := float64(secondsSinceMidnight) / 21600.0
		ambientLightColor = startColor.Lerp(endColor, intervalPercentage)
	}
	return ambientLightColor
}
