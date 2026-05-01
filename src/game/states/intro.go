package states

import (
	"strings"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/ui"
)

type GameStateNewCareer struct {
	engine    services.Engine
	textInput *ui.TextInput
	isDirty   bool
}

func (g *GameStateNewCareer) ClearOverlay()  {}
func (g *GameStateNewCareer) SetDirty()      { g.isDirty = true }
func (g *GameStateNewCareer) Print(_ string) {}
func (g *GameStateNewCareer) UpdateHUD()     {}

func (g *GameStateNewCareer) Init(engine services.Engine) {
	g.engine = engine
	g.isDirty = true

	gridW := engine.ScreenGridWidth() * 2
	gridH := engine.ScreenGridHeight()
	prompt := "Contractor ID: "
	inputX := (gridW - len(prompt) - 16) / 2
	inputY := gridH / 2

	g.textInput = ui.NewTextInputAt(
		geometry.Point{X: inputX, Y: inputY},
		len(prompt)+20,
		prompt, "",
		func(name string) { g.confirm(name) },
		nil,
	)
	g.textInput.MaxLen = 16
}

func (g *GameStateNewCareer) confirm(name string) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return
	}
	career := g.engine.GetCareer()
	career.PlayerName = name
	career.CurrentCampaignFolder = "first blood"
	career.SaveToFile()
	g.engine.GetGame().PopState()
	g.engine.GetGame().PushState(&GameStateMainMenu{})
}

func (g *GameStateNewCareer) Update(input services.InputInterface) {
	g.textInput.Update(input)
	g.isDirty = true
}

func (g *GameStateNewCareer) Draw(con console.CellInterface) {
	if !g.isDirty {
		return
	}
	gridW := g.engine.ScreenGridWidth() * 2
	gridH := g.engine.ScreenGridHeight()
	bg := common.Cell{Rune: ' ', Style: common.Style{Background: common.TerminalColorBackground}}
	con.HalfWidthFill(geometry.NewRect(0, 0, gridW, gridH), bg)

	title := "[ ICA NETWORK ]"
	titleX := (gridW - len(title)) / 2
	titleY := gridH/2 - 2
	ui.PrintToGrid(con, title, geometry.Point{X: titleX, Y: titleY}, common.TerminalColor, common.TerminalColorBackground)

	sub := "Enter your contractor ID to continue"
	subX := (gridW - len(sub)) / 2
	ui.PrintToGrid(con, sub, geometry.Point{X: subX, Y: titleY + 1}, common.TerminalColor, common.TerminalColorBackground)

	g.textInput.SetDirty()
	g.textInput.Draw(con)
	g.isDirty = false
}
