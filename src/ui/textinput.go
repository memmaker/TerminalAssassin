package ui

import (
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type TextInput struct {
	prompt       string
	content      string
	onComplete   func(userInput string)
	onAbort      func()
	position     geometry.Point
	isDirty      bool
	width        int
	OnCursorMove func(newPos geometry.Point)
}

func (t *TextInput) SetDirty() {
	t.isDirty = true
}

func (t *TextInput) Update(input services.InputInterface) {
	for _, cmd := range input.PollText() {
		switch typedCmd := cmd.(type) {
		case core.KeyCommand:
			t.handleKeyCommand(typedCmd)
		case core.GameCommand:
			t.handleGameCommand(typedCmd)
		}
	}
}

func (t *TextInput) handleKeyCommand(cmd core.KeyCommand) {
	switch cmd.Key {
	case core.KeyBackspace:
		if len(t.content) > 0 {
			t.content = t.content[:len(t.content)-1]
			t.isDirty = true
		}
	case core.KeyEscape:
		t.isDirty = true
		if t.onAbort != nil {
			t.onAbort()
		}
	case core.KeyEnter:
		t.isDirty = true
		if t.onComplete != nil {
			t.onComplete(t.content)
		}
	default:
		printableChar := string(cmd.Key)
		if len(printableChar) > 1 {
			return
		}
		t.content += printableChar
		t.isDirty = true
	}
	if t.OnCursorMove != nil {
		t.OnCursorMove(geometry.Point{X: t.position.X + len(t.prompt) + len(t.content), Y: t.position.Y})
	}
}

func (t *TextInput) handleGameCommand(cmd core.GameCommand) {
	switch cmd {
	case core.MenuCancel:
		t.isDirty = true
		if t.onAbort != nil {
			t.onAbort()
		}
	case core.MenuConfirm:
		t.isDirty = true
		if t.onComplete != nil {
			t.onComplete(t.content)
		}
	}
}

func (t *TextInput) Draw(grid console.CellInterface) {
	if !t.isDirty {
		return
	}
	startX := t.position.X
	startY := t.position.Y
	if startY < 0 {
		startY = grid.Size().Y - 1
	}
	endX := startX + t.width
	bbox := geometry.NewRect(startX, startY, endX, startY)
	blankCell := common.Cell{Rune: ' ', Style: common.Style{Foreground: common.TerminalColor, Background: common.TerminalColorBackground}}
	grid.HalfWidthFill(bbox, blankCell)
	PrintToGrid(grid, t.prompt, geometry.Point{X: startX, Y: startY}, common.TerminalColor, common.TerminalColorBackground)
	PrintToGrid(grid, t.content, geometry.Point{X: startX + len(t.prompt), Y: startY}, common.TerminalColor, common.TerminalColorBackground)
	//grid.SetHalfWidth(geometry.Point{X: startX + len(t.prompt) + len(t.content), Y: startY}, common.Cell{Rune: ' ', Style: common.Style{Foreground: common.TerminalColor, Background: common.TerminalColor}})
	t.isDirty = false
}

func NewTextInputAt(pos geometry.Point, width int, prompt string, prefill string, onComplete func(userInput string), onAbort func()) *TextInput {
	return &TextInput{
		position:   pos,
		width:      width,
		prompt:     prompt,
		content:    prefill,
		onComplete: onComplete,
		onAbort:    onAbort,
		isDirty:    true,
	}
}
