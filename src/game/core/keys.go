package core

import "github.com/memmaker/terminal-assassin/geometry"

type Key string

const (
	KeySpace      Key = " "
	KeyBackspace  Key = "Backspace"
	KeyTab        Key = "Tab"
	KeyEscape     Key = "Escape"
	KeyEnter      Key = "Enter"
	KeyLeftArrow  Key = "ArrowLeft"
	KeyRightArrow Key = "ArrowRight"
	KeyUpArrow    Key = "ArrowUp"
	KeyDownArrow  Key = "ArrowDown"
)

type MouseState uint8

const (
	MouseLeft MouseState = iota
	MouseLeftHeld
	MouseRight
	MouseLeftReleased
	MouseMoved
	MouseWheelUp
	MouseWheelDown
	MouseWheelLeft
	MouseWheelRight
)

type GameCommand uint8
type GameDirectionalCommand uint8

const (
	StartSneaking GameCommand = iota
	StopSneaking
	PickUpItem
	DropItem
	PutItemAway
	NextItem
	PreviousItem
	OpenInventory
	UseEquippedItem
	UseItem
	Confirm
	Cancel
	MenuUp
	MenuDown
	MenuLeft
	MenuRight
	MenuConfirm
	MenuCancel
)

const (
	MovementDirection GameDirectionalCommand = iota
	PeekingDirection
	ActionDirection
	AimingDirection
)

type InputCommand interface {
}
type KeyCommand struct {
	Key Key
}

type DirectionalGameCommand struct {
	Command GameDirectionalCommand
	XAxis   float64
	YAxis   float64
}

type PointerCommand struct {
	Action MouseState
	Pos    geometry.Point
}
