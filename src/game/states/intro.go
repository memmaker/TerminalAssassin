package states

import (
    "fmt"
    "strings"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/ui"
)

type GameStateIntro struct {
    textInput   *ui.TextInput
    label       *ui.FixedLabel
    engine      services.Engine
    ramChecked  int
    isDirty     bool
    clearScreen bool
    powerOn     bool
    ramText     core.StyledText
}

func (g *GameStateIntro) ClearOverlay() {

}

func (g *GameStateIntro) SetDirty() {
    g.isDirty = true
}

func (g *GameStateIntro) Print(text string) {
    g.label.SetText(text)
}

func (g *GameStateIntro) UpdateHUD() {
}

func (g *GameStateIntro) Update(input services.InputInterface) {
    audio := g.engine.GetAudio()
    if input.ConfirmOrCancel() {
        if !g.powerOn {
            g.powerOn = true
            g.label = nil
            audio.PlayCueWithCallback("power_button", func() {
                g.engine.Schedule(0.016, g.checkMoreRAM)
                audio.PlayCueWithCallback("fan_starting", func() {
                    audio.StartLoop("fan_loop")
                })
                g.isDirty = true
                g.clearScreen = true
            })
        } else {
            g.ramChecked = 640
        }
    }
}

func (g *GameStateIntro) checkMoreRAM() {
    audio := g.engine.GetAudio()
    if g.ramChecked < 640 {
        g.ramChecked += 16
        // we want to have leading zeros
        g.ramText = core.NewStyledText(fmt.Sprintf("%03d KB OK", g.ramChecked), common.Style{Foreground: common.TerminalColor, Background: common.TerminalColorBackground})
        g.isDirty = true
        g.engine.Schedule(0.25, g.checkMoreRAM)
    } else { //HAL, ICA, AIM
        audio.PlayCueWithCallback("ram_check_done", func() {
            userInterface := g.engine.GetUI()
            bootUpText := []string{
                "640 KB OK",
                "",
                "The ICA Personal Computer Networked DOS",
                "*Personal Copy - Do not distribute*",
            }
            textFinished := func() {
                g.fakeRemoteConnection()
            }
            userInterface.RenderFancyText(geometry.Point{}, bootUpText, textFinished)
            g.label = nil
        })
    }
}

func (g *GameStateIntro) fakeRemoteConnection() {
    userInterface := g.engine.GetUI()
    audio := g.engine.GetAudio()
    audio.PlayCue("modem")
    bootUpText := []string{
        "Connecting to ICA Network.........OK",
        "Establishing secure connection....OK",
        "Looking up local contractor id....ERROR",
        "",
        "Please enter a new contractor id: ",
    }
    textFinished := func() {
        g.resetTextInput(geometry.Point{X: 34, Y: 9})
    }
    userInterface.RenderFancyText(geometry.Point{X: 0, Y: 5}, bootUpText, textFinished)
    g.label = nil
}

func (g *GameStateIntro) Draw(con console.CellInterface) {
    if !g.isDirty {
        return
    }
    gridWidth := g.engine.ScreenGridWidth()
    if g.clearScreen {
        rect := geometry.NewRect(0, 0, gridWidth*2, g.engine.ScreenGridHeight())
        con.HalfWidthFill(rect, common.Cell{Rune: ' ', Style: common.Style{Background: common.TerminalColorBackground}})
        g.clearScreen = false
    }

    if g.label != nil {
        g.label.Draw(con)
    }
    if g.ramText.Text() != "" {
        con.HalfWidthFill(geometry.NewRect(0, 0, gridWidth, 1), common.Cell{Rune: ' ', Style: common.Style{Background: common.TerminalColorBackground}})
        g.ramText.DrawHalfWidth(con, geometry.NewRect(0, 0, g.ramText.Size().X, 1), core.AlignLeft)
    }
    g.isDirty = false
}

func (g *GameStateIntro) Init(engine services.Engine) {
    g.engine = engine
    gridWidth, gridHeight := engine.ScreenGridWidth()*2, engine.ScreenGridHeight()
    startText := "*PRESS ANY KEY TO BOOT*"
    startForCentering := geometry.Point{X: (gridWidth - len(startText)) / 2, Y: gridHeight / 2}
    g.label = ui.NewHalfLabelWithWidth("", startForCentering, len(startText))
    g.label.SetStyledText(core.NewStyledText(startText, common.Style{Foreground: common.TerminalColor, Background: common.TerminalColorBackground}))
    g.clearScreen = true
    g.isDirty = true
}

func (g *GameStateIntro) resetTextInput(promptPos geometry.Point) {
    onComplete := func(userInput string) {
        g.HandleCommand(userInput)
    }
    userInterface := g.engine.GetUI()
    userInterface.ShowNoAbortTextInputAt(promptPos, g.engine.ScreenGridWidth()*2, "", "", onComplete)
}

func (g *GameStateIntro) HandleCommand(name string) {
    name = strings.ToLower(name)
    switch {
    case name != "":
        career := g.engine.GetCareer()
        career.PlayerName = name
        career.CurrentCampaignFolder = "first blood"
        career.SaveToFile()
        g.showSegue()
    }
}

func (g *GameStateIntro) showSegue() {
    userInterface := g.engine.GetUI()
    audio := g.engine.GetAudio()
    bootUpText := []string{
        "Looking up local contractor id....OK",
        "Enabling True Color support.......OK",
        "Enabling HDR Lighting.............OK",
        "Enabling Terminal Graphics Mode...OK",
        "",
        "Assigning contractor code.........23",
        "",
        "Sequence complete.",
    }
    textFinished := func() {
        audio.Stop("fan_loop")
        g.engine.GetAnimator().ClearParticles()
        g.engine.GetGame().PopState()
        g.engine.GetGame().PushState(&GameStateMainMenu{})
    }
    audio.PlayCue("hdd_read")
    userInterface.RenderFancyText(geometry.Point{X: 0, Y: 10}, bootUpText, textFinished)
}
