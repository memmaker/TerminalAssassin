package director

import (
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/utils"
)

type ScriptParser struct {
	*core.Logic
}

type ReadState int

const (
	ReadStatePreamble ReadState = iota
	ReadStateStart
	ReadStateAndStartConditions
	ReadStateOrStartConditions
	ReadStateOrTimeoutConditions
	ReadStateAndTimeoutConditions
	ReadStateActions
	ReadStateTimeoutActions
)

func NewScriptParser(player *core.Actor) *ScriptParser {
	return &ScriptParser{
		Logic: core.NewLogicCore(player),
	}
}

func (p *ScriptParser) ScriptFromFile(f fs.File) (*Script, error) {
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return p.parse(string(b))
}
func (p *ScriptParser) parse(script string) (*Script, error) {
	p.Variables = make(map[string]func() any)
	newScript := &Script{current: -1}
	keyframes := make([]*KeyFrame, 0)

	var finishedCallback func() = nil
	var currentFrame *KeyFrame = nil
	var currentStartConditions core.CombinedPredicate
	var currentTimeoutConditions core.CombinedPredicate

	var state ReadState = ReadStatePreamble
	lines := strings.Split(script, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "$") {
			if state == ReadStateActions {
				ourLineInstance := line
				currentFrame.Actions = append(currentFrame.Actions, func() {
					p.HandleAssignment(ourLineInstance, true)
				})
			} else {
				p.HandleAssignment(line, state == ReadStatePreamble)
			}
		} else if strings.HasPrefix(line, "#") {
			if strings.HasPrefix(line, "# NEWFRAME") {
				if currentFrame != nil {
					keyframes = append(keyframes, currentFrame)
				}
				currentFrame = &KeyFrame{
					Actions: make([]func(), 0),
				}
				currentStartConditions = core.CombinedPredicate{}
				state = ReadStateStart
			} else if strings.HasPrefix(line, "# TIMEOUTACTIONS") {
				state = ReadStateTimeoutActions
			} else if strings.HasPrefix(line, "# AND-TIMEOUTCONDITIONS") {
				state = ReadStateAndTimeoutConditions
			} else if strings.HasPrefix(line, "# OR-TIMEOUTCONDITIONS") {
				state = ReadStateOrTimeoutConditions
			} else if strings.HasPrefix(line, "## AND-STARTCONDITIONS") {
				state = ReadStateAndStartConditions
			} else if strings.HasPrefix(line, "## OR-STARTCONDITIONS") {
				state = ReadStateOrStartConditions
			} else if strings.HasPrefix(line, "## ACTIONS") {
				state = ReadStateActions
			}
		} else if core.LooksLikeAFunction(line) {
			switch state {
			case ReadStateAndTimeoutConditions:
				predicateCall := p.LineToPredicate(newScript, line)
				currentTimeoutConditions = currentTimeoutConditions.And(predicateCall)
			case ReadStateOrTimeoutConditions:
				predicateCall := p.LineToPredicate(newScript, line)
				currentTimeoutConditions = currentTimeoutConditions.Or(predicateCall)
			case ReadStateAndStartConditions:
				predicateCall := p.LineToPredicate(newScript, line)
				currentStartConditions = currentStartConditions.And(predicateCall)
				currentFrame.StartCondition = currentStartConditions.Evaluate
			case ReadStateOrStartConditions:
				predicateCall := p.LineToPredicate(newScript, line)
				currentStartConditions = currentStartConditions.Or(predicateCall)
				currentFrame.StartCondition = currentStartConditions.Evaluate
			case ReadStateActions:
				actionCall := p.LineToAction(line)
				currentFrame.Actions = append(currentFrame.Actions, actionCall)
			case ReadStateTimeoutActions:
				actionCall := p.LineToAction(line)
				newScript.onTimeOut = append(newScript.onTimeOut, actionCall)
			}
		}
	}
	// add the last frame
	keyframes = append(keyframes, currentFrame)

	newScript.frames = keyframes
	newScript.onScriptFinished = finishedCallback
	if !currentTimeoutConditions.IsEmpty() {
		newScript.timeOutCondition = currentTimeoutConditions.Evaluate
	}
	return newScript, nil
}

func (s *MapConversations) AddLineForSpeaker(dialogue string, speaker *core.Actor, triggerCode string, utterance core.Utterance) {
	if s.Dialogues == nil {
		s.Dialogues = make(map[string]*DialogueInfo)
	}
	if _, ok := s.Dialogues[dialogue]; !ok {
		s.Dialogues[dialogue] = &DialogueInfo{
			Utterances: map[*core.Actor]map[string]core.Utterance{
				speaker: {triggerCode: utterance},
			},
		}
		return
	}
	if _, ok := s.Dialogues[dialogue].Utterances[speaker]; !ok {
		s.Dialogues[dialogue].Utterances[speaker] = map[string]core.Utterance{triggerCode: utterance}
		return
	}
	s.Dialogues[dialogue].Utterances[speaker][triggerCode] = utterance
}
func (p *ScriptParser) LineToPredicate(theScript *Script, line string) func() bool {
	name, stringArgs := core.GetNameAndArgs(line)
	println(fmt.Sprintf("Parsed predicate '%s(%s)'", name, stringArgs))

	if name == "IsCurrentFrameOlderThan" {
		return p.isCurrentFrameOlderThanPredicate(theScript, stringArgs[0])
	}

	args := p.StringResolve(stringArgs)

	if predicate, ok := p.PredicateMap[name]; ok {
		predicateCall := func() bool {
			return predicate(p.ResolveArgs(args)...)
		}
		return predicateCall
	}

	println(fmt.Sprintf("Script ERR: Predicate %s not found. Returning always FALSE predicate.", name))
	return func() bool { return false }
}

func (p *ScriptParser) isCurrentFrameOlderThanPredicate(script *Script, maxAgeInSeconds string) func() bool {
	maxAge, err := strconv.Atoi(maxAgeInSeconds)
	if err != nil {
		println(fmt.Sprintf("Script ERR: isCurrentFrameOlderThanPredicate: %s", err.Error()))
		return func() bool { return false }
	}
	return func() bool {
		return script.CurrentFrame().TicksSinceStart > utils.SecondsToTicks(float64(maxAge))
	}
}
