package director

import (
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/mapset"
)

type DialogueParser struct {
	*core.Logic
}

func NewDialogueParser(player *core.Actor) *DialogueParser {
	return &DialogueParser{Logic: core.NewLogicCore(player)}
}
func (p *DialogueParser) LoadDialogForActors(dialogue *DialogueInfo) {
	utterances := dialogue.Utterances
	for speaker, lines := range utterances {
		if _, exists := speaker.Dialogue.Conversations[dialogue.Name]; !exists {
			speaker.Dialogue.Conversations[dialogue.Name] = &core.Conversation{
				Responses: core.NewDefaultResponses(),
			}
		}
		if speaker != p.Player {
			speaker.Dialogue.Conversations[dialogue.Name].RequiredClothing = dialogue.RequiredClothing
		}
		speaker.Dialogue.SpeakInState = func(state core.ActorState) bool {
			return state == core.ActorStatusIdle || state == core.ActorStatusScripted || state == core.ActorStatusOnSchedule
		}

		for trigger, line := range lines {
			// a response
			// add it to the dialogue memory of the actor
			speaker.Dialogue.Conversations[dialogue.Name].Responses[trigger] = line
		}
	}
}

func (p *DialogueParser) DialogueFromFile(f fs.File) (*DialogueInfo, error) {
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return p.parse(string(b)), nil
}

func (p *DialogueParser) parse(content string) *DialogueInfo {
	newDialogue := &DialogueInfo{
		Utterances:   make(map[*core.Actor]map[string]core.Utterance),
		Participants: mapset.NewSet[*core.Actor](),
	}
	var preambleState = true
	var currentSpeaker *core.Actor = nil
	var currentSpeechCode string = ""
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "$") {
			if preambleState {
				p.HandleAssignment(line, true)
			} else {
				pattern := regexp.MustCompile(`^(\$[A-Za-z-_]+):`)
				actorVariable := pattern.FindStringSubmatch(line)[1]
				text := strings.TrimSpace(line[len(actorVariable)+1:])
				var nextSpeaker *core.Actor
				if actorVariable == "$PLAYER" {
					nextSpeaker = p.Player
				} else {
					nextSpeaker = p.ResolveVariable(actorVariable).(*core.Actor)
				}
				if newDialogue.InitialSpeaker == nil {
					newDialogue.InitialSpeaker = nextSpeaker
				}
				currentSpeaker = nextSpeaker
				currentSpeechCode = newDialogue.addLineOfSpeech(currentSpeechCode, currentSpeaker, text)
				newDialogue.Participants.Add(currentSpeaker)
			}
		} else if strings.HasPrefix(line, "## DIALOGUE:") {
			preambleState = false
			newDialogue.Name = strings.TrimSpace(line[12:])
			newDialogue.RequiredClothing = ""
			println("NPC MapConversations found: " + newDialogue.Name)
		} else if strings.HasPrefix(line, "## DIALOGUE-PLAYER") {
			preambleState = false
			//## DIALOGUE-PLAYER(Blue Lotus Leader): weapons_deal_blue_player
			pattern := regexp.MustCompile(`^## DIALOGUE-PLAYER\((.+)\): (.+)$`)
			matches := pattern.FindStringSubmatch(line)
			newDialogue.RequiredClothing = strings.TrimSpace(matches[1])
			newDialogue.Name = strings.TrimSpace(matches[2])
			println(fmt.Sprintf("Player MapConversations found: %s, clothing: %s", newDialogue.Name, newDialogue.RequiredClothing))
		}
	}
	newDialogue.LastSpeechCode = currentSpeechCode
	newDialogue.LastSpeaker = currentSpeaker
	return newDialogue
}
