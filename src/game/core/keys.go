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
    HolsterItem
    NextItem
    PreviousItem
    OpenInventory
    UseRangedItem
    UseItem
    Confirm
    Cancel
    MenuUp
    MenuDown
    MenuLeft
    MenuRight
    MenuConfirm
    MenuCancel

    ContextAction // Cross → context-action / dialogue / pickpocket at peek tile
    BeginMouseAiming
    Assassinate
    DiveTackle // Circle + active right stick → dive/tackle in peek direction
    StopAiming // L2 released → exit gamepad aim mode and return to default state
)

const (
    MovementDirection GameDirectionalCommand = iota
    PeekingDirection
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
