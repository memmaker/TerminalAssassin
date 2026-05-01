package states

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/editor"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
	"github.com/memmaker/terminal-assassin/gridmap"
)

type GameStateMainMenu struct {
	engine      services.Engine
	isDirty     bool
	clearScreen bool
}

func (g *GameStateMainMenu) ClearOverlay() {}

func (g *GameStateMainMenu) SetDirty() {
	g.isDirty = true
}

func (g *GameStateMainMenu) Init(engine services.Engine) {
	g.engine = engine
	g.isDirty = true
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
	g.openMainMenu()
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
			Label:   "Control Help",
			Handler: g.showControlHelp,
		},

		{
			DynamicLabel: func() string {
				return "Controller   : " + config.ControllerMode
			},
			Handler: func() {
				if config.ControllerMode == services.ControllerKeyboardMouse {
					g.engine.SetControllerMode(services.ControllerGamepad)
				} else {
					g.engine.SetControllerMode(services.ControllerKeyboardMouse)
				}
				g.engine.SaveOptions()
			},
		},
		{
			DynamicLabel: func() string {
				if config.Fullscreen {
					return "Display      : Fullscreen"
				}
				return "Display      : Windowed"
			},
			Handler: func() {
				g.engine.SetFullscreen(!config.Fullscreen)
				g.engine.SaveOptions()
			},
		},
		{
			DynamicLabel: func() string {
				return fmt.Sprintf("Master Volume: %d%%", int(audio.GetMasterVolume()*100))
			},
			LeftHandler: func() {
				audio.SetMasterVolume(audio.GetMasterVolume() - 0.1)
				g.engine.SaveOptions()
			},
			RightHandler: func() {
				audio.SetMasterVolume(audio.GetMasterVolume() + 0.1)
				g.engine.SaveOptions()
			},
		},
		{
			DynamicLabel: func() string {
				return fmt.Sprintf("Music Volume : %d%%", int(audio.GetMusicVolume()*100))
			},
			LeftHandler: func() {
				audio.SetMusicVolume(audio.GetMusicVolume() - 0.1)
				g.engine.SaveOptions()
			},
			RightHandler: func() {
				audio.SetMusicVolume(audio.GetMusicVolume() + 0.1)
				g.engine.SaveOptions()
			},
		},
		{
			DynamicLabel: func() string {
				return fmt.Sprintf("Sound Volume : %d%%", int(audio.GetSoundVolume()*100))
			},
			LeftHandler: func() {
				audio.SetSoundVolume(audio.GetSoundVolume() - 0.1)
				g.engine.SaveOptions()
			},
			RightHandler: func() {
				audio.SetSoundVolume(audio.GetSoundVolume() + 0.1)
				g.engine.SaveOptions()
			},
		},
		{
			DynamicLabel: func() string {
				return "Show Hints   : " + strconv.FormatBool(config.ShowHints)
			},
			Handler: func() {
				config.ShowHints = !config.ShowHints
				g.engine.SaveOptions()
			},
		},
		{
			Label:   "Change font",
			Handler: g.openFontsMenu,
		},
	}
	g.engine.GetUI().OpenFixedWidthStackedMenu("options", menuItems)
}

func (g *GameStateMainMenu) showControlHelp() {
	config := g.engine.GetGame().GetConfig()
	g.engine.GetUI().ShowPager("Controls", controlHelpLines(config.ControllerMode), nil)
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
	w, h := g.engine.ScreenGridWidth(), g.engine.ScreenGridHeight()
	drawDiagonalTextPattern(con, w, h)
	g.isDirty = false
}

// drawDiagonalTextPattern tiles "TerminalAssassin" across the screen.
// Each row is shifted one character forward, giving a diagonal effect.
// Letter brightness fades from dark to light gray along the diagonal.
func drawDiagonalTextPattern(con console.CellInterface, w, h int) {
	const text = "TerminalAssassin"
	n := len(text)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ch := rune(text[(x+y)%n])
			t := float64(x+y) / float64(w+h-2)
			gray := uint8(20 + t*55)
			fg := common.NewRGBColorFromBytes(gray, gray, gray)
			con.SetSquare(geometry.Point{X: x, Y: y}, common.Cell{
				Style: common.Style{Foreground: fg, Background: common.Black},
				Rune:  ch,
			})
		}
	}
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

// controlHelpLines returns pager lines describing the bindings for the given controller mode.
func controlHelpLines(mode string) []core.StyledText {
	if mode == services.ControllerGamepad {
		return []core.StyledText{
			core.Text("Movement      Left Stick"),
			core.Text("Peek          Right Stick"),
			core.Text("Run           L1 (hold)"),
			core.Text("Sneak Toggle  D-Pad Right"),
			core.Text("Context Act.  R1 / A (cross)"),
			core.Text("Inventory     D-Pad Left"),
			core.Text("Drop Item     D-Pad Down"),
			core.Text("Holster Item  D-Pad Up"),
			core.Text("Aim (hold)    L2 + Right Stick"),
			core.Text("Fire / Throw  R2"),
			core.Text("Assassinate   Triangle (Y)"),
			core.Text("Dive Tackle   Circle (B)"),
			core.Text("Use at Peek   Square (X)"),
			core.Text("Free Look     Select / R3"),
			core.Text("Pause         Options / Start"),
		}
	}
	return []core.StyledText{
		core.Text("Move          W A S D"),
		core.Text("Peek          Arrow Keys"),
		core.Text("Run           Shift + Move"),
		core.Text("Sneak Toggle  CapsLock"),
		core.Text("Context Act.  E"),
		core.Text("Inventory     Q"),
		core.Text("Drop Item     X"),
		core.Text("Holster Item  C"),
		core.Text("Aim / Fire    Space  (+ Mouse aim)"),
		core.Text("Assassinate   R"),
		core.Text("Dive Tackle   F"),
		core.Text("Use at Peek   V"),
		core.Text("Free Look     Tab"),
		core.Text("Confirm       Enter"),
		core.Text("Cancel/Pause  Escape"),
	}
}
