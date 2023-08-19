package states

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/game/services"
)

type TrainingHelper struct {
	engine       services.Engine
	defaultStyle common.Style
}

func NewTrainingHelper(engine services.Engine) *TrainingHelper {
	return &TrainingHelper{
		engine:       engine,
		defaultStyle: common.DefaultStyle.Reversed(),
	}
}
func (t TrainingHelper) alertContextAction() {
	textStyle := common.DefaultStyle.Reversed()
	userInterface := t.engine.GetUI()
	input := t.engine.GetInput()
	keyDefs := input.GetKeyDefinitions()
	alertMessage := []core.StyledText{
		t.titleLine("Interactions"),
		core.NewStyledText("", textStyle),
		core.NewStyledText("When you are next to something you can interact with,", textStyle),
		core.NewStyledText("the object will light up, and you will see it displayed", textStyle),
		core.NewStyledText("in your context HUD in the lower right corner.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Press the button that is associated with the direction", textStyle),
		core.NewStyledText("of the object or person you want to interact with.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Target the cell you are standing on, by pressing the '@l "+keyDefs.SameTileActionKey.String()+" @N' key.", textStyle).WithMarkup('l', common.Style{Foreground: common.FourGreen, Background: common.FourWhite}),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Since it's rude to get right in someone's face,", textStyle),
		core.NewStyledText("you'll need to have at least one cell of space between", textStyle),
		core.NewStyledText("you and any person you want to talk to.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("                             ↑                             ", textStyle),
		core.NewStyledText("                             @l"+keyDefs.ActionKeys[0].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("             ←@l"+keyDefs.ActionKeys[1].String()+"@N             [@l"+keyDefs.SameTileActionKey.String()+"@N]             @l"+keyDefs.ActionKeys[3].String()+"@N→             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             @l"+keyDefs.ActionKeys[2].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             ↓                             ", textStyle),
		core.NewStyledText("", textStyle),
		t.confirmLine(),
	}
	userInterface.ShowStyledAlert(alertMessage, textStyle.Background)
}

func (t TrainingHelper) alertInfiltration() {
	textStyle := common.DefaultStyle.Reversed()
	userInterface := t.engine.GetUI()
	alertMessage := []core.StyledText{
		t.titleLine("Infiltration"),
		core.NewStyledText("", textStyle),
		core.NewStyledText("You are currently not allowed here.", textStyle),
		core.NewStyledText("The best way to get around is to blend in.", textStyle),
		core.NewStyledText("Look for a disguise, and put it on,", textStyle),
		core.NewStyledText("in order to move around freely.", textStyle),
		core.NewStyledText("", textStyle),
		t.confirmLine(),
	}
	userInterface.ShowStyledAlert(alertMessage, textStyle.Background)
}

func (t TrainingHelper) alertPeekingThroughKeyholes() {
	textStyle := common.DefaultStyle.Reversed()
	userInterface := t.engine.GetUI()
	input := t.engine.GetInput()
	keyDefs := input.GetKeyDefinitions()
	alertMessage := []core.StyledText{
		t.titleLine("Peeking (Keyholes)"),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Peeking is another directional action.", textStyle),
		core.NewStyledText("It will shift the origin of your point of view by one cell.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Use this to your advantage to see what's behind closed doors.", textStyle),
		core.NewStyledText("", textStyle),

		core.NewStyledText("                             ↑                             ", textStyle),
		core.NewStyledText("                             @l"+keyDefs.PeekingKeys[0].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("             ←@l"+keyDefs.PeekingKeys[1].String()+"@N                             @l"+keyDefs.PeekingKeys[3].String()+"@N→             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             @l"+keyDefs.PeekingKeys[2].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             ↓                             ", textStyle),

		core.NewStyledText("", textStyle),
		t.confirmLine(),
	}
	userInterface.ShowStyledAlert(alertMessage, textStyle.Background)
}

func (t TrainingHelper) alertPeekingPickup() {
	textStyle := common.DefaultStyle.Reversed()
	userInterface := t.engine.GetUI()
	input := t.engine.GetInput()
	keyDefs := input.GetKeyDefinitions()
	alertMessage := []core.StyledText{
		t.titleLine("Peeking (Pickup)"),
		core.NewStyledText("", textStyle),
		core.NewStyledText("It is possible to pick up items with the peeking action.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("You can use this to pick up items that are right next to you,", textStyle),
		core.NewStyledText("without having to move unto the same cell.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("                             ↑                             ", textStyle),
		core.NewStyledText("                             @l"+keyDefs.PeekingKeys[0].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("             ←@l"+keyDefs.PeekingKeys[1].String()+"@N                             @l"+keyDefs.PeekingKeys[3].String()+"@N→             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             @l"+keyDefs.PeekingKeys[2].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             ↓                             ", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText(fmt.Sprintf("Then use @l%s@N to pick up the item.", keyDefs.SameTileActionKey.String()), textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("", textStyle),
		t.confirmLine(),
	}
	userInterface.ShowStyledAlert(alertMessage, textStyle.Background)
}

func (t TrainingHelper) alertTakedown() {
	textStyle := common.DefaultStyle.Reversed()
	userInterface := t.engine.GetUI()
	input := t.engine.GetInput()
	keyDefs := input.GetKeyDefinitions()
	alertMessage := []core.StyledText{
		t.titleLine("Silent Takedown"),
		core.NewStyledText("", textStyle),
		core.NewStyledText("Melee attacks from behind won't make any noise.", textStyle),
		core.NewStyledText("Frontal attacks and gunshots make a lot of noise,", textStyle),
		core.NewStyledText("and will alert anyone nearby.", textStyle),
		core.NewStyledText("", textStyle),
		core.NewStyledText("                             @l"+keyDefs.ActionKeys[0].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("              @l"+keyDefs.ActionKeys[1].String()+"@N              @l"+keyDefs.SameTileActionKey.String()+"@N              @l"+keyDefs.ActionKeys[3].String()+"@N              ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("                             @l"+keyDefs.ActionKeys[2].String()+"@N                             ", textStyle).WithMarkup('l', t.defaultStyle.WithFg(common.FourGreen)),
		core.NewStyledText("", textStyle),
		t.confirmLine(),
	}
	userInterface.ShowStyledAlert(alertMessage, textStyle.Background)
}

func (t TrainingHelper) titleLine(title string) core.StyledText {
	return core.NewStyledText(fmt.Sprintf(" >> %s ", title), t.defaultStyle.WithBg(common.FourWhite))
}

func (t TrainingHelper) confirmLine() core.StyledText {
	return core.NewStyledText("@l > @N Understood ", t.defaultStyle.WithBg(common.FourWhite)).WithMarkup('l', common.Style{Foreground: common.FourWhite, Background: common.Red})
}
