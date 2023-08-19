package director

import (
	"fmt"
	"strings"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/mapset"
)

type KeyFrame struct {
	StartCondition   func() bool
	TimeOutCondition func() bool
	Actions          []func()
	TicksSinceStart  int
}

type DialogueInfo struct {
	Utterances       map[*core.Actor]map[string]core.Utterance
	LastSpeechCode   string
	LastSpeaker      *core.Actor
	RequiredClothing string
	Name             string
	InitialSpeaker   *core.Actor
	Participants     *mapset.MapSet[*core.Actor]
}

func (d *DialogueInfo) linesForSpeaker(speaker *core.Actor) int {
	if _, ok := d.Utterances[speaker]; !ok {
		return 0
	}
	return len(d.Utterances[speaker])
}
func (d *DialogueInfo) addLineOfSpeech(triggerCode string, currentSpeaker *core.Actor, currentSpeakersText string) string {
	safeName := strings.Replace(currentSpeaker.Name, " ", "-", -1)
	code := fmt.Sprintf("DLG_%s_%s_%d", d.Name, safeName, d.linesForSpeaker(currentSpeaker))
	utterance := core.Utterance{
		Line:      core.Text(currentSpeakersText),
		EventCode: code,
	}
	if _, ok := d.Utterances[currentSpeaker]; !ok {
		d.Utterances[currentSpeaker] = make(map[string]core.Utterance)
	}
	d.Utterances[currentSpeaker][triggerCode] = utterance
	println(fmt.Sprintf("Adding line '%s' for speaker '%s' with trigger '%s' and code '%s'", currentSpeakersText, currentSpeaker.Name, triggerCode, code))
	return code
}

type Script struct {
	frames           []*KeyFrame
	onTimeOut        []func()
	onScriptFinished func()
	current          int
	timeOutCondition func() bool
}

func (s *Script) CurrentFrame() *KeyFrame {
	if s.current > len(s.frames) || s.current < 0 {
		return &KeyFrame{}
	}
	return s.frames[s.current]
}

func (s *Script) NextFrame() *KeyFrame {
	if s.current+1 < len(s.frames) {
		return s.frames[s.current+1]
	}
	return nil
}

func (s *Script) IsEmpty() bool {
	return len(s.frames) == 0
}

func (s *Script) IsFinished() bool {
	if s.current >= len(s.frames) { // script has finished
		return true
	}
	nextFrame := s.NextFrame()
	if nextFrame != nil { // there are more frames..
		if nextFrame.StartCondition == nil || nextFrame.StartCondition() {
			// can start
			s.current++
			for _, action := range nextFrame.Actions {
				action() // do start..
			}
			println(fmt.Sprintf("Start condition for script frame %d was met and the actions were executed.", s.current))
			if s.current == len(s.frames) {
				println("This was the last frame. Script is finished")
				if s.onScriptFinished != nil {
					s.onScriptFinished()
				}
				return true
			}
		}
	}
	if s.current < 0 { // script hasn't started yet
		return false
	}
	if s.timeOutCondition != nil && s.timeOutCondition() {
		println(fmt.Sprintf("Script timed out. Quitting script at frame %d", s.current))
		s.current = len(s.frames)
		for _, action := range s.onTimeOut {
			action()
		}
		return true
	}
	currentFrame := s.CurrentFrame()
	if currentFrame.TimeOutCondition != nil && currentFrame.TimeOutCondition() {
		println(fmt.Sprintf("Script frame %d timed out. Quitting script", s.current))
		s.current = len(s.frames)
		for _, action := range s.onTimeOut {
			action()
		}
		return true
	}

	currentFrame.TicksSinceStart++
	return false
}

type MapConversations struct {
	Dialogues map[string]*DialogueInfo
}

func (s *MapConversations) linesForSpeaker(currentDialogue string, currentSpeaker *core.Actor) int {
	if s.Dialogues == nil {
		return 0
	}
	if _, ok := s.Dialogues[currentDialogue]; !ok {
		return 0
	}
	if _, ok := s.Dialogues[currentDialogue].Utterances[currentSpeaker]; !ok {
		return 0
	}
	return len(s.Dialogues[currentDialogue].Utterances[currentSpeaker])
}
