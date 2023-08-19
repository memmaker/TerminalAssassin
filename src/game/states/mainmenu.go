package states

import (
	"path/filepath"
	"strconv"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/editor"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
	"github.com/memmaker/terminal-assassin/utils"
)

type GameStateMainMenu struct {
	hasBackground    bool
	drawHappened     bool
	engine           services.Engine
	isDirty          bool
	clearScreen      bool
	backgroundPixels [][]common.Color
	showGear         bool
}

func (g *GameStateMainMenu) ClearOverlay() {

}

func (g *GameStateMainMenu) SetDirty() {
	g.isDirty = true
}

func (g *GameStateMainMenu) Init(engine services.Engine) {

	g.engine = engine
	g.isDirty = true
	fileSystem := g.engine.GetFilesystem()
	pixels, err := utils.GetPixelsFromImage(fileSystem, "datafiles/images/title_64_blue.png")

	if err != nil {
		g.hasBackground = false
	} else {
		g.hasBackground = true
		g.backgroundPixels = pixels
	}
	audio := g.engine.GetAudio()
	audio.StopAll()
	audio.UnloadCues([]string{
		"key",
		"fan_loop",
		"fan_starting",
		"hdd_read",
		"modem",
		"power_button",
		"ram_check_done",
	})
	if g.engine.GetGame().GetConfig().MusicStreaming {
		audio.StartLoopStream("Suspense")
	} else {
		audio.StartLoop("Suspense")
	}
	//g.loadMusic(engine)
	//g.RenderTitleScreen()
	//g.openMainMenu()
	g.startBackgroundFadeIn()
}

func (g *GameStateMainMenu) openMainMenu() {
	career := g.engine.GetCareer()
	title := "Agent Menu"
	userInterface := g.engine.GetUI()
	if career.CurrentCampaignFolder != "" {
		title += " - " + career.CurrentCampaignFolder
	}
	menuItems := []services.MenuItem{
		{
			Label: "New Mission",
			Handler: func() {
				userInterface.OpenMapsMenu(func(*gridmap.GridMap[*core.Actor, *core.Item, services.Object]) {
					g.engine.GetAudio().StopAll()
					g.openBriefingMenu()
				})
			},
		},
		{
			Label:   "Change campaign",
			Handler: g.openCampaignMenu,
		},
		{
			Label: "Career",
			Handler: func() {
				g.engine.GetGame().PushState(&GameStateCareerViewer{})
			},
		},
		{
			Label: "Editor",
			Handler: func() {
				userInterface.PopModal()
				g.engine.GetAudio().StopAll()
				g.engine.GetGame().PushState(&editor.GameStateEditor{})
			},
		},
		{
			Label:   "Options",
			Handler: g.openOptionsMenu,
		}, /*
			{
				Label:   "Dev Test",
				Handler: g.debugStuff,
				//Handler: g.openReplayMenu,
			},*/
		{
			Label:   "Quit",
			Handler: g.engine.QuitGame,
		},
	}
	g.engine.GetAudio().PlayCue("open_menu")
	g.engine.GetUI().OpenFancyMenu(menuItems)
}

func (g *GameStateMainMenu) openOptionsMenu() {
	audio := g.engine.GetAudio()
	config := g.engine.GetGame().GetConfig()
	menuItems := []services.MenuItem{
		{
			Label:   "Change font",
			Handler: g.openFontsMenu,
		},
		{
			DynamicLabel: func() string {
				return "Master Volume: " + strconv.Itoa(int(audio.GetMasterVolume()*100)) + "%"
			},
			LeftHandler: func() {
				audio.SetMasterVolume(audio.GetMasterVolume() - 0.1)
			},
			RightHandler: func() {
				audio.SetMasterVolume(audio.GetMasterVolume() + 0.1)
			},
		},
		{
			DynamicLabel: func() string {
				return "Music Volume: " + strconv.Itoa(int(audio.GetMusicVolume()*100)) + "%"
			},
			LeftHandler: func() {
				audio.SetMusicVolume(audio.GetMusicVolume() - 0.1)
			},
			RightHandler: func() {
				audio.SetMusicVolume(audio.GetMusicVolume() + 0.1)
			},
		},
		{
			DynamicLabel: func() string {
				return "Sound Volume: " + strconv.Itoa(int(audio.GetSoundVolume()*100)) + "%"
			},
			LeftHandler: func() {
				audio.SetSoundVolume(audio.GetSoundVolume() - 0.1)
			},
			RightHandler: func() {
				audio.SetSoundVolume(audio.GetSoundVolume() + 0.1)
			},
		},
		{
			DynamicLabel: func() string {
				return "Show Hints: " + strconv.FormatBool(config.ShowHints)
			},
			Handler: func() {
				config.ShowHints = !config.ShowHints
			},
		},
	}
	g.engine.GetUI().OpenFixedWidthStackedMenu("options", menuItems)
}

func (g *GameStateMainMenu) Update(input services.InputInterface) {
	if g.isDirty {
		audio := g.engine.GetAudio()
		if !audio.IsCuePlaying("Suspense") {
			audio.StopAll()
			if g.engine.GetGame().GetConfig().MusicStreaming {
				audio.StartLoopStream("Suspense")
			} else {
				audio.StartLoop("Suspense")
			}
		}
	}
	if input.DevTerminalKeyPressed() {
		g.engine.GetGame().PushState(&GameStateTerminal{})
	}
}

func (g *GameStateMainMenu) Draw(con console.CellInterface) {
	if !g.isDirty {
		return
	}
	if g.clearScreen {
		con.ClearSurface()
		g.clearScreen = false
	}
	con.ClearConsole()
	g.drawBackground(con)
	if g.showGear {
		g.RenderChosenGear(con)
	}
	//con.SquareBlack()
	//con.HalfWidthTransparent()
	g.isDirty = false
}

func (g *GameStateMainMenu) startBackgroundFadeIn() {
	if !g.hasBackground {
		return
	}
	//userInterface := g.engine.GetUI()
	//userInterface.HideModal()
	//fileSystem := g.engine.GetFilesystem()
	//pixels, err := utils.GetPixelsFromImage(fileSystem, "embedded/images/title_64_black.png")

	animator := g.engine.GetAnimator()
	onFinish := func() {
		g.openMainMenu()
		g.isDirty = true
	}
	animator.ImageFadeIn(g.backgroundPixels, onFinish, onFinish)
}

func (g *GameStateMainMenu) openCampaignMenu() {
	career := g.engine.GetCareer()
	userInterface := g.engine.GetUI()
	files := g.engine.GetFiles()
	config := g.engine.GetGame().GetConfig()
	// get all subdirectories of the "campaigns" folder
	campaignFolderName := config.CampaignDirectory
	campaigns := files.GetSubdirectories(campaignFolderName)
	menuItems := make([]services.MenuItem, len(campaigns))
	userInterface.PopModal()
	g.isDirty = true
	for i, campaign := range campaigns {
		currentCampaign := filepath.Base(campaign)
		menuItems[i] = services.MenuItem{
			Label: currentCampaign,
			Handler: func() {
				career.CurrentCampaignFolder = currentCampaign
				career.SaveToFile()
				g.openMainMenu()
				g.isDirty = true
				//g.clearScreen = true
			}}
	}
	onClose := func() {
		g.openMainMenu()
		g.isDirty = true
	}
	userInterface.OpenFixedWidthAutoCloseMenuWithCallback("Choose campaign", menuItems, onClose)
}

/*
func (g *GameStateMainMenu) printVersionText(text string, topLeft gruid.Point, yPos int, style common.Style) {
	m := g.engine
	versionTextWidth := len(text)
	versionTextTopLeft := gruid.Point{X: (ScreenGridWidth - versionTextWidth) / 2, Y: yPos}
	for x, char := range text {
		pointToPlace := gruid.Point{X: versionTextTopLeft.X + x, Y: versionTextTopLeft.Y}
		m.ScreenGrid.SetSquare(pointToPlace, gruid.Cell{Rune: char, Style: style.WithFg(ColorFromCode(ColorBrightRed))})
	}
}*/

func (g *GameStateMainMenu) openFontsMenu() {
	userInterface := g.engine.GetUI()
	// get all subdirectories of the "campaigns" folder
	allFonts := g.engine.GetAvailableTextFonts()
	menuItems := make([]services.MenuItem, len(allFonts))
	for i, fontName := range allFonts {
		currentFont := fontName
		menuItems[i] = services.MenuItem{Label: currentFont, Handler: func() {
			g.engine.SetTextFont(currentFont)
			g.clearScreen = true
			g.isDirty = true
		}}
	}
	userInterface.OpenFixedWidthAutoCloseMenu("Choose font", menuItems)
}

func (g *GameStateMainMenu) drawBackground(con console.CellInterface) {
	if !g.hasBackground {
		return
	}
	gridWidth, gridHeight := g.engine.ScreenGridWidth(), g.engine.ScreenGridHeight()
	for y := 0; y < gridHeight; y++ {
		for x := 0; x < gridWidth; x++ {
			pos := geometry.Point{X: x, Y: y}
			con.SetSquare(pos, common.Cell{
				Style: common.Style{Background: g.backgroundPixels[y][x]},
				Rune:  ' ',
			})
		}
	}
}

func (g *GameStateMainMenu) debugStuff() {

}
