package states

import (
	"strings"

	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type GameStateTerminal struct {
	engine services.Engine
}

func (g *GameStateTerminal) ClearOverlay() {

}

func (g *GameStateTerminal) SetDirty() {

}

func (g *GameStateTerminal) Print(text string) {
}

func (g *GameStateTerminal) UpdateHUD() {
}

func (g *GameStateTerminal) Update(input services.InputInterface) {

}

func (g *GameStateTerminal) Draw(con console.CellInterface) {

}

func (g *GameStateTerminal) Init(engine services.Engine) {
	g.engine = engine
	userInterface := g.engine.GetUI()
	userInterface.RenderFancyText(geometry.Point{}, []string{"ICA DOS 1.0"}, func() {
		g.resetTextInput()
	})
}

func (g *GameStateTerminal) resetTextInput() {
	userInterface := g.engine.GetUI()
	userInterface.ShowNoAbortTextInputAt(geometry.Point{X: 0, Y: 1}, g.engine.ScreenGridWidth()*2, "a>", "", g.HandleCommand)
}

func (g *GameStateTerminal) HandleCommand(command string) {
	command = strings.ToLower(command)
	switch {
	case command == "exit":
		//g.engine.SendGameState(&GameStateMainMenu{Model: g.engine})
		//case g.engine.Data.HasItemUnlock(command):
		g.UnlockItems(command)
	default:
		g.resetTextInput()
		g.Print("unknown command")
	}
}

func (g *GameStateTerminal) UnlockItems(command string) {
	g.Print("Codenamed 47 recognized. Unlocking...")
	/*
		items := g.engine.Data.ItemUnlockMap[command]
		for _, item := range items {
			g.engine.Career.UnlockedItems[item.name] = *item
		}
		g.engine.Career.SaveToFile()
		time.AfterFunc(1*time.Second, func() {
			g.engine.SendGameState(&GameStateMainMenu{Model: g.engine})
		})
	*/
}
