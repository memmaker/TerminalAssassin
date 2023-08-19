package states

import (
	"bufio"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
)

func (g *GameStateMainMenu) openBriefingMenu() {
	g.isDirty = true
	userInterface := g.engine.GetUI()
	game := g.engine.GetGame()

	currentMap := game.GetMap()
	gear := game.GetMissionPlan()
	gear.SetDefaultStartLocation(currentMap.PlayerSpawn)

	files := g.engine.GetFiles()
	audioPlayer := g.engine.GetAudio()

	if g.engine.GetGame().GetConfig().MusicStreaming {
		audioPlayer.StartLoopStream("Hitman")
	} else {
		audioPlayer.StartLoop("Hitman")
	}

	//recorder := g.engine.GetRecorder()
	title := "Mission" // + currentMap.MapFileName()
	menuItems := []services.MenuItem{
		{
			Label: "Start mission",
			Handler: func() {
				audioPlayer.StopAll()
				//recorder.StartRecording()
				userInterface.PopAll()
				game.PopState()
				game.PushState(&GameStateGameplay{})
			},
		},
		{
			Label:   "Show briefing",
			Handler: g.showBriefingAnimated,
			Condition: func() bool {
				return files.FileExists(g.getBriefingFilePath())
			},
		},
		{
			Label:   "Planning",
			Handler: g.openPlanningMenu,
		},
		{
			Label: "Back",
			Handler: func() {
				audioPlayer.Stop("Hitman")
				userInterface.PopModal()
			},
		},
	}

	userInterface.OpenFixedWidthStackedMenu(title, menuItems)
}

func toStyled(lines []string) []core.StyledText {
	sLines := make([]core.StyledText, len(lines))
	redStyle := common.Style{Foreground: core.ColorFromCode(core.ColorBlood), Background: core.ColorFromCode(core.ColorBlackBackground)}
	for i := range lines {
		sLines[i] = core.Text(lines[i]).WithStyle(common.DefaultStyle).WithMarkup('r', redStyle)
	}
	return sLines
}

func (g *GameStateMainMenu) showBriefingAnimated() {
	files := g.engine.GetFiles()
	userInterface := g.engine.GetUI()
	animator := g.engine.GetAnimator()

	file, err := files.Open(g.getBriefingFilePath())
	if err != nil {
		println("Error opening briefing file: " + err.Error())
		return
	}
	script := NewBriefingFromFile(file)
	userInterface.HideModal()
	onFinish := func() {
		g.isDirty = true
		userInterface.ShowModal()
	}
	animator.BriefingAnimation(script, onFinish)
}

func NewBriefingFromFile(file fs.File) *core.BriefingAnimation {
	audioPattern := regexp.MustCompile(`[Aa]udio: ?"([A-Za-z0-9_\-]+)"`)
	imagePattern := regexp.MustCompile(`[Ii]mage: ?"([A-Za-z0-9_\-]+)"`)
	slides := make([]core.BriefingSlide, 0)
	var currentSlide *core.BriefingSlide
	var currText []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		currLine := strings.TrimSpace(scanner.Text())
		if currLine == "" {
			continue
		}
		if strings.HasPrefix(currLine, "#") { // new slide
			if currentSlide != nil {
				currentSlide.Text = toStyled(currText)
				slides = append(slides, *currentSlide)
			}

			currText = make([]string, 0)
			currentSlide = &core.BriefingSlide{}

			audioMatch := audioPattern.FindStringSubmatch(currLine)
			if len(audioMatch) > 0 {
				currentSlide.AudioFile = audioMatch[1]
			}
			imageMatch := imagePattern.FindStringSubmatch(currLine)
			if len(imageMatch) > 0 {
				currentSlide.ImageFile = imageMatch[1]
			}
		} else {
			currText = append(currText, currLine)
		}
	}
	if currentSlide != nil {
		currentSlide.Text = toStyled(currText)
		slides = append(slides, *currentSlide)
	}
	return &core.BriefingAnimation{Slides: slides}
}

func (g *GameStateMainMenu) getBriefingFilePath() string {
	currentMap := g.engine.GetGame().GetMap()
	mapFolder := currentMap.MapFileName()
	briefingFilePath := path.Join(mapFolder, "briefing", "slides.txt")
	return briefingFilePath
}
