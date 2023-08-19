package core

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/geometry"
)

// Alignment represents left, center or right text alignment. It is used to
// draw text with a given alignment within a grid.
type Alignment int16

// Those constants represent the possible text alignment options. The default
// alignment is AlignCenter.
const (
	AlignCenter Alignment = iota
	AlignLeft
	AlignRight
)

// StyledText is a simple text formatter and styler. The zero value can be
// used, but you may prefer using Text and Textf.
type StyledText struct {
	markups map[rune]common.Style
	text    string
	style   common.Style
}

// Text is a shorthand for StyledText{}.WithText and creates a new styled text
// with the given text and the default style.
func Text(text string) StyledText {
	return StyledText{text: text, style: common.DefaultStyle}
}

// Textf returns a new styled text with the given formatted text, and default style.
func Textf(format string, a ...interface{}) StyledText {
	return StyledText{text: fmt.Sprintf(format, a...)}
}

// NewStyledText returns a new styled text with the given text and style. It is
// a shorthand for StyledText{}.With.
func NewStyledText(text string, st common.Style) StyledText {
	return StyledText{text: text, style: st}
}

// Text returns the current styled text as a string.
func (stt StyledText) Text() string {
	return stt.text
}

// WithText returns a derived styled text with updated text.
func (stt StyledText) WithText(text string) StyledText {
	stt.text = text
	return stt
}

// WithTextf returns a derived styled text with updated formatted text.
func (stt StyledText) WithTextf(format string, a ...interface{}) StyledText {
	stt.text = fmt.Sprintf(format, a...)
	return stt
}

// Style returns the text default style.
func (stt StyledText) Style() common.Style {
	return stt.style
}

// WithStyle returns a derived styled text with a updated default style.
func (stt StyledText) WithStyle(style common.Style) StyledText {
	stt.style = style
	return stt
}

// With returns a derived styled text with new next and style.
func (stt StyledText) With(text string, style common.Style) StyledText {
	stt.text = text
	stt.style = style
	return stt
}

// WithMarkup returns a derived styled text with a new markup @r available for
// a given style.  Markup starts by a @ sign, and is followed then by a rune
// indicating the particular style. Default style is marked by @N.
//
// This simple markup is inspired from github/gdamore/tcell, but with somewhat
// different defaults: unless at least one non-default markup is registered,
// markup commands processing is not activated, and @ is treated as any other
// character.
func (stt StyledText) WithMarkup(r rune, style common.Style) StyledText {
	if r == ' ' || r == '\n' {
		// avoid strange cases that can conflict with format
		return stt
	}
	if len(stt.markups) == 0 {
		stt.markups = map[rune]common.Style{}
	} else {
		omarks := stt.markups
		stt.markups = make(map[rune]common.Style, len(omarks))
		for k, v := range omarks {
			stt.markups[k] = v
		}
	}
	if r == 'N' {
		// N has a built-in meaning
		stt.style = style
		return stt
	}
	stt.markups[r] = style
	return stt
}

// WithMarkups is the same as WithMarkup but passing a whole map of rune-style
// associations.
func (stt StyledText) WithMarkups(markups map[rune]common.Style) StyledText {
	stt.markups = markups
	return stt
}

// Markups returns a copy of the markups currently defined for the styled text.
func (stt StyledText) Markups() map[rune]common.Style {
	markups := make(map[rune]common.Style, len(stt.markups))
	for k, v := range stt.markups {
		markups[k] = v
	}
	return markups
}

// Iter iterates a function for all couples positions and cells representing
// the styled text, and returns the minimum (w, h) size in cells which can fit
// the text.
func (stt StyledText) Iter(fn func(geometry.Point, common.Cell)) geometry.Point {
	x, y := 0, 0
	xmax := 0
	c := common.Cell{Style: stt.style}
	markup := stt.markups != nil // whether markup is activated
	procm := false               // processing markup
	for _, r := range stt.text {
		if markup {
			if procMarkup(procm, r) {
				if procm {
					c.Style = stt.markupStyle(r)
				}
				procm = !procm
				continue
			}
			procm = false
		}
		if r == '\n' {
			if x > xmax {
				xmax = x
			}
			x = 0
			y++
			continue
		}
		c.Rune = r
		fn(geometry.Point{X: x, Y: y}, c)
		x++
	}
	if x > xmax {
		xmax = x
	}
	if xmax > 0 || y > 0 {
		y++ // at least one line
	}
	return geometry.Point{X: xmax, Y: y}
}

func procMarkup(procm bool, r rune) bool {
	if procm {
		return r != '@'
	}
	return r == '@'
}

// Size returns the minimum (w, h) size in cells which can fit the text.
func (stt StyledText) Size() geometry.Point {
	x, y := 0, 0
	xmax := 0
	markup := stt.markups != nil // whether markup is activated
	procm := false               // processing markup
	for _, r := range stt.text {
		if markup {
			if procMarkup(procm, r) {
				procm = !procm
				continue
			}
			procm = false
		}
		if r == '\n' {
			if x > xmax {
				xmax = x
			}
			x = 0
			y++
			continue
		}
		x++
	}
	if x > xmax {
		xmax = x
	}
	if xmax > 0 || y > 0 {
		y++ // at least one line
	}
	return geometry.Point{X: xmax, Y: y}
}

// Format formats the text so that lines longer than a certain width get
// wrapped at word boundaries, if possible. It preserves spaces at the
// beginning of a line.
func (stt StyledText) Format(width int) StyledText {
	s := strings.Builder{}
	wordbuf := bytes.Buffer{}
	col := 0                     // current column (without counting @r markups)
	wantspace := false           // whether we expect currently space (start of a new word that is not at line start)
	wlen := 0                    // current word length
	markup := stt.markups != nil // whether markup is activated
	procm := false               // processing markup
	start := true                // whether at line start
	for _, r := range stt.text {
		if markup {
			if procMarkup(procm, r) {
				procm = !procm
				switch r {
				case '\n', ' ':
				default:
					if wlen == 0 {
						s.WriteRune(r)
					} else {
						wordbuf.WriteRune(r)
					}
					continue
				}
			} else {
				procm = false
			}
		}
		if r == ' ' {
			switch {
			case start:
				s.WriteRune(' ')
				col++
				continue
			case wlen > 0:
				newline := wlen+col+1 > width
				if wantspace {
					addSpace(&s, newline)
					if newline {
						col = 0
					} else {
						col++
					}
				}
				s.Write(wordbuf.Bytes())
				col += wlen
				wordbuf.Reset()
				wlen = 0
				wantspace = true
			}
			continue
		}
		if r == '\n' {
			if wlen > 0 {
				if wantspace {
					addSpace(&s, wlen+col+1 > width)
				}
				s.Write(wordbuf.Bytes())
				wordbuf.Reset()
				wlen = 0
			}
			s.WriteRune('\n')
			col = 0
			wantspace = false
			start = true
			continue
		}
		start = false
		wordbuf.WriteRune(r)
		wlen++
	}
	if wlen > 0 {
		if wantspace {
			addSpace(&s, wlen+col+1 > width)
		}
		s.Write(wordbuf.Bytes())
	}
	stt.text = strings.TrimRight(s.String(), " \n")
	return stt
}

func addSpace(s *strings.Builder, b bool) {
	if b {
		s.WriteRune('\n')
	} else {
		s.WriteRune(' ')
	}
}

func (stt StyledText) markupStyle(r rune) common.Style {
	if r == 'N' {
		return stt.style
	}
	st, ok := stt.markups[r]
	if ok {
		return st
	}
	return stt.style
}

// drawOnHalfWidth displays the styled text in a given grid. It returns the smallest grid
// slice containing the drawn part. Note that the grid is not cleared with
// spaces beforehand by this function, not even the returned one, you should
// use the styled text with a label for this.
func (stt StyledText) drawOnHalfWidth(con console.CellInterface, bbox geometry.Rect) {
	//width := bbox.Size().X
	offset := bbox.Min
	stt.Iter(func(p geometry.Point, c common.Cell) {
		//println(fmt.Sprintf("drawOnHalfWidth: %v %v", offset.Add(p), c))
		con.SetHalfWidth(offset.Add(p), c)
	})
}

func (stt StyledText) drawOnSquare(con console.CellInterface, bbox geometry.Rect) {
	//width := bbox.Size().X
	offset := bbox.Min
	stt.Iter(func(p geometry.Point, c common.Cell) {
		//println(fmt.Sprintf("drawOnSquare: %v %v", offset.Add(p), c))
		con.SetSquare(offset.Add(p), c)
	})
}

func (stt StyledText) DrawHalfWidth(con console.CellInterface, bbox geometry.Rect, align Alignment) int {
	return stt.draw(con, bbox, align, stt.drawOnHalfWidth)
}

func (stt StyledText) DrawSquare(con console.CellInterface, bbox geometry.Rect, align Alignment) {
	stt.draw(con, bbox, align, stt.drawOnSquare)
}

func (stt StyledText) draw(con console.CellInterface, bbox geometry.Rect, align Alignment, drawFunc func(con console.CellInterface, bbox geometry.Rect)) int {
	var tw int
	tw = stt.Size().X
	switch align {
	case AlignCenter:
		w := bbox.Size().X
		dist := ((w - tw) / 2) + 1
		drawFunc(con, bbox.Shift(dist, 0, -dist, 0))
	case AlignLeft:
		drawFunc(con, bbox)
	case AlignRight:

		w := bbox.Max.X
		dist := w - tw
		drawFunc(con, bbox.Shift(dist, 0, 0, 0))
	}
	return tw
}

func (stt StyledText) Empty() bool {
	return stt.text == ""
}

func DrawStyledTextAlignedInsideRect(con console.CellInterface, bounds geometry.Rect, alignment Alignment, lines []StyledText) {
	con.HalfWidthFill(bounds, common.Cell{Rune: ' ', Style: common.DefaultStyle})
	height := bounds.Size().Y
	for i := 0; i < height; i++ {
		lineIndex := i
		if lineIndex >= len(lines) {
			break
		}
		lineBounds := bounds.Line(i).Shift(2, 0, -2, 0)
		lines[lineIndex].DrawHalfWidth(con, lineBounds, alignment)
	}
}

type BriefingAnimation struct {
	Slides []BriefingSlide
}
type BriefingSlide struct {
	Text      []StyledText
	AudioFile string
	ImageFile string
}
