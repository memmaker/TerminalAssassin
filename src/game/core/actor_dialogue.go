package core

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/mapset"
	"github.com/memmaker/terminal-assassin/utils"
)

type Utterance struct {
	Line      StyledText
	EventCode string
}

type Conversation struct {
	Responses        map[string]Utterance // response code -> utterance
	RequiredClothing string
}
type DialogueComponent struct {
	LastSpokenAtTick    uint64
	LastHeardAtTick     uint64
	NextUtterance       Utterance
	HeardSpeech         mapset.Set[string]
	SpokenSpeech        mapset.Set[string]
	Situation           *OrientedLocation
	SpeakInState        func(ActorState) bool
	IsCurrentlySpeaking bool
	Conversations       map[string]*Conversation // clothing -> conversation
	CurrentDialogue     string
}

func (c *DialogueComponent) HasHeardSpeech(code string) bool {
	return c.HeardSpeech.Contains(code)
}

func (c *DialogueComponent) HasSpoken(speechCode string) bool {
	return c.SpokenSpeech.Contains(speechCode)
}
func (c *DialogueComponent) DidHear(code string, atTick uint64) {
	c.HeardSpeech.Add(code)
	c.LastHeardAtTick = atTick
}
func (c *DialogueComponent) DidSpeak(person *Actor, atTick uint64) {
	spokenCode := person.Dialogue.NextUtterance.EventCode
	c.SpokenSpeech.Add(spokenCode)
	c.LastSpokenAtTick = atTick
	println(fmt.Sprintf("%s spoke %s", person.Name, spokenCode))
	if len(person.Dialogue.Conversations) == 0 {
		return
	}
	if _, ok := person.Dialogue.Conversations[c.CurrentDialogue]; !ok {
		return
	}
	if followup, hasFollowUp := person.Dialogue.Conversations[c.CurrentDialogue].Responses[spokenCode]; hasFollowUp {
		person.Dialogue.NextUtterance = followup
	} else {
		person.Dialogue.NextUtterance = Utterance{}
	}
}

func (c *DialogueComponent) Available(status ActorState) bool {
	hasLineQueued := !c.NextUtterance.Line.Empty()
	if c.SpeakInState == nil {
		return hasLineQueued
	}
	return hasLineQueued && c.SpeakInState(status)
}

func (c *DialogueComponent) HasDialogueFor(actor *Actor) string {
	// actor also needs to wear the right clothes
	for conversationName, conversation := range c.Conversations {
		if conversation.RequiredClothing == actor.NameOfClothing() || conversation.RequiredClothing == "" { // clothing is fine
			if _, ok := actor.Dialogue.Conversations[conversationName]; ok { // actor has the same conversation
				return conversationName
			}
		}
	}
	return ""
}
func (c *DialogueComponent) Active(currentTick uint64) bool {
	if c.CurrentDialogue == "" || len(c.Conversations) == 0 {
		return false
	}
	if _, ok := c.Conversations[c.CurrentDialogue]; !ok {
		return false
	}
	if len(c.Conversations[c.CurrentDialogue].Responses) == 0 {
		return false
	}
	if c.IsCurrentlySpeaking {
		return true
	}
	hasLineQueued := !c.NextUtterance.Line.Empty()
	fiveSecondsInTicks := uint64(utils.SecondsToTicks(5))
	// active means we have said or heard something in the last 5 seconds, but not longer ago
	return hasLineQueued || c.LastSpokenAtTick > currentTick-fiveSecondsInTicks || c.LastHeardAtTick > currentTick-fiveSecondsInTicks
}

func (c *DialogueComponent) TryResponding(code string) (Utterance, bool) {
	for conversationName, conversation := range c.Conversations {
		if _, ok := conversation.Responses[code]; ok {
			c.CurrentDialogue = conversationName
			return conversation.Responses[code], true
		}
	}
	return Utterance{}, false
}
