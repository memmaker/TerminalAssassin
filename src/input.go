package main

import (
	"fmt"
	"github.com/memmaker/terminal-assassin/game/services"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type InputState struct {
	config                console.GridConfig
	MousePosOnScreenGrid  geometry.Point
	LastMousePos          geometry.Point
	movementDelayInTicks  int
	timeSinceLastMovement float64
	sneaking              bool
	dpiScale              float64
	axes                  [6]float64
	inputConsumed         bool
	keyBuffer             []ebiten.Key

	// key definitions
	keyboardMovementKeys      [4]ebiten.Key
	keyboardPeekingKeys       [4]ebiten.Key
	keyboardActionKeys        [4]ebiten.Key
	keyboardSameTileActionKey ebiten.Key
	keyboardSneakModeKey      ebiten.Key
	keyboardDropItemKey       ebiten.Key
	keyboardPutItemAwayKey    ebiten.Key
	keyboardUseItemKey        ebiten.Key
	keyboardInventoryKey      ebiten.Key
}

func (i *InputState) IsShiftPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyShift)
}
func (i *InputState) GetKeyDefinitions() services.KeyDefinitions {
	return services.KeyDefinitions{
		MovementKeys:      i.keyboardMovementKeys,
		PeekingKeys:       i.keyboardPeekingKeys,
		ActionKeys:        i.keyboardActionKeys,
		SameTileActionKey: i.keyboardSameTileActionKey,
		SneakModeKey:      i.keyboardSneakModeKey,
		DropItemKey:       i.keyboardDropItemKey,
		PutItemAwayKey:    i.keyboardPutItemAwayKey,
		UseItemKey:        i.keyboardUseItemKey,
		InventoryKey:      i.keyboardInventoryKey,
	}
}
func (i *InputState) DevTerminalKeyPressed() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyF10)
}
func (i *InputState) ConfirmOrCancel() bool {
	if i.inputConsumed {
		return false
	}
	keyPressed := inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace)
	mousePressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	somethingWasPressed := keyPressed || mousePressed
	i.inputConsumed = true
	return somethingWasPressed
}
func (i *InputState) KeyToKeyCommand(key ebiten.Key, caps bool) core.InputCommand {
	switch key {
	case ebiten.KeyDigit0:
		return core.KeyCommand{Key: "0"}
	case ebiten.KeyDigit1:
		return core.KeyCommand{Key: "1"}
	case ebiten.KeyDigit2:
		return core.KeyCommand{Key: "2"}
	case ebiten.KeyDigit3:
		return core.KeyCommand{Key: "3"}
	case ebiten.KeyDigit4:
		return core.KeyCommand{Key: "4"}
	case ebiten.KeyDigit5:
		return core.KeyCommand{Key: "5"}
	case ebiten.KeyDigit6:
		return core.KeyCommand{Key: "6"}
	case ebiten.KeyDigit7:
		return core.KeyCommand{Key: "7"}
	case ebiten.KeyDigit8:
		return core.KeyCommand{Key: "8"}
	case ebiten.KeyDigit9:
		return core.KeyCommand{Key: "9"}
	case ebiten.KeySpace:
		return core.KeyCommand{Key: " "}
	case ebiten.KeyComma:
		return core.KeyCommand{Key: ","}
	case ebiten.KeyPeriod:
		return core.KeyCommand{Key: "."}
	case ebiten.KeySlash:
		return core.KeyCommand{Key: "/"}
	case ebiten.KeySemicolon:
		return core.KeyCommand{Key: ";"}
	case ebiten.KeyApostrophe:
		return core.KeyCommand{Key: "'"}
	case ebiten.KeyLeftBracket:
		return core.KeyCommand{Key: "["}
	case ebiten.KeyRightBracket:
		return core.KeyCommand{Key: "]"}
	case ebiten.KeyBackslash:
		return core.KeyCommand{Key: "\\"}
	case ebiten.KeyMinus:
		return core.KeyCommand{Key: "-"}
	case ebiten.KeyEqual:
		return core.KeyCommand{Key: "="}
	default:
		keyAsString := key.String()
		if !caps && len(keyAsString) == 1 && keyAsString[0] >= 'A' && keyAsString[0] <= 'Z' {
			keyAsString = strings.ToLower(keyAsString)
		}

		return core.KeyCommand{Key: core.Key(keyAsString)}
	}
}
func (i *InputState) pollKeyBoardForUI() []core.InputCommand {
	i.keyBuffer = i.keyBuffer[:0]
	i.keyBuffer = inpututil.AppendPressedKeys(i.keyBuffer)

	for l := len(i.keyBuffer) - 1; l >= 0; l-- {
		key := i.keyBuffer[l]
		if !i.isStillPressed(key, 14) {
			i.keyBuffer = append(i.keyBuffer[:l], i.keyBuffer[l+1:]...)
		}
	}
	//buffer := make([]ebiten.Key, 0)
	//buffer = inpututil.AppendJustPressedKeys(buffer)
	commands := make([]core.InputCommand, len(i.keyBuffer))
	for index, key := range i.keyBuffer {
		switch key {
		case ebiten.KeyEnter:
			commands[index] = core.MenuConfirm
		case ebiten.KeyE:
			commands[index] = core.MenuConfirm
		case ebiten.KeyQ:
			commands[index] = core.MenuCancel
		case ebiten.KeyEscape:
			commands[index] = core.MenuCancel
		case ebiten.KeyArrowUp:
			commands[index] = core.MenuUp
		case ebiten.KeyW:
			commands[index] = core.MenuUp
		case ebiten.KeyArrowDown:
			commands[index] = core.MenuDown
		case ebiten.KeyS:
			commands[index] = core.MenuDown
		case ebiten.KeyArrowLeft:
			commands[index] = core.MenuLeft
		case ebiten.KeyA:
			commands[index] = core.MenuLeft
		case ebiten.KeyArrowRight:
			commands[index] = core.MenuRight
		case ebiten.KeyD:
			commands[index] = core.MenuRight
		default:
			isCaps := ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyCapsLock)
			commands[index] = i.KeyToKeyCommand(key, isCaps)
		}
	}
	return commands
}

// we want
// GetGameCommands(), will pop the game  commands
// GetUICommands(). will pop the UI commands
//
//	IsFinished() method will do the polling and store the results
func (i *InputState) Update() {
	i.inputConsumed = false
}

func (i *InputState) PollText() []core.InputCommand {
	if i.inputConsumed {
		return []core.InputCommand{}
	}
	i.inputConsumed = true
	return i.pollPrintables()
}

func (i *InputState) pollPrintables() []core.InputCommand {
	var keys []ebiten.Key
	keys = inpututil.AppendJustPressedKeys(keys)
	isCaps := ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyCapsLock)
	var commands = make([]core.InputCommand, len(keys))
	for index, key := range keys {
		commands[index] = i.KeyToKeyCommand(key, isCaps)
	}
	return commands
}
func (i *InputState) PollGameCommands() []core.InputCommand {
	if i.inputConsumed {
		return []core.InputCommand{}
	}
	i.inputConsumed = true
	var commands = make([]core.InputCommand, 0)
	commands = append(commands, i.pollKeyBoardForGameplay()...)
	commands = append(commands, i.pollGamePadForGameplay()...)
	commands = append(commands, i.pollMouse()...)
	return commands
}
func (i *InputState) PollEditorCommands() []core.InputCommand {
	if i.inputConsumed {
		return []core.InputCommand{}
	}
	i.inputConsumed = true
	var commands = make([]core.InputCommand, 0)
	commands = append(commands, i.pollPrintables()...)
	commands = append(commands, i.pollMouse()...)
	return commands
}
func (i *InputState) PollUICommands() []core.InputCommand {
	if i.inputConsumed {
		return []core.InputCommand{}
	}
	i.inputConsumed = true
	var commands = make([]core.InputCommand, 0)
	commands = append(commands, i.pollKeyBoardForUI()...)
	commands = append(commands, i.pollMouse()...)

	return commands
}
func (i *InputState) pollKeyBoardForGameplay() []core.InputCommand {
	var commands = make([]core.InputCommand, 0)

	movementDirectionX, movementDirectionY := i.directionFromKeys(i.keyboardMovementKeys)
	playerWantsToMove := movementDirectionX != 0 || movementDirectionY != 0

	if !(i.mustWaitForSneakDelay()) && playerWantsToMove {
		commands = append(commands, core.DirectionalGameCommand{Command: core.MovementDirection, XAxis: float64(movementDirectionX), YAxis: float64(movementDirectionY)})
		i.timeSinceLastMovement = 0
	}
	peekDirX, peekDirY := i.directionFromKeys(i.keyboardPeekingKeys)
	if peekDirX != 0 || peekDirY != 0 {
		commands = append(commands, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: float64(peekDirX), YAxis: float64(peekDirY)})
	}

	actionDirX, actionDirY := i.directionFromKeys(i.keyboardActionKeys)
	cardinalAction := actionDirX != 0 || actionDirY != 0
	sameTileAction := inpututil.IsKeyJustPressed(i.keyboardSameTileActionKey)

	if cardinalAction {
		commands = append(commands, core.DirectionalGameCommand{Command: core.ActionDirection, XAxis: float64(actionDirX), YAxis: float64(actionDirY)})
	} else if sameTileAction {
		commands = append(commands, core.DirectionalGameCommand{Command: core.ActionDirection, XAxis: 0, YAxis: 0})
	}

	if inpututil.IsKeyJustPressed(i.keyboardSneakModeKey) {
		commands = append(commands, core.StartSneaking)
	} else if inpututil.IsKeyJustReleased(i.keyboardSneakModeKey) {
		commands = append(commands, core.StopSneaking)
	}

	if inpututil.IsKeyJustPressed(i.keyboardInventoryKey) {
		//commands = append(commands, core.NextItem})
		commands = append(commands, core.OpenInventory)
	}

	if inpututil.IsKeyJustPressed(i.keyboardDropItemKey) {
		commands = append(commands, core.DropItem)
	}
	if inpututil.IsKeyJustPressed(i.keyboardPutItemAwayKey) {
		commands = append(commands, core.PutItemAway)
	}

	if inpututil.IsKeyJustPressed(i.keyboardUseItemKey) {
		commands = append(commands, core.UseItem)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		commands = append(commands, core.Confirm)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		commands = append(commands, core.Cancel)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF1)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF2)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF3)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF4)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF5)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF6)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF7)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF8)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF9) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF9)})
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF10) {
		commands = append(commands, core.KeyCommand{Key: keyFromEbiten(ebiten.KeyF10)})
	}
	return commands
}

func (i *InputState) mustWaitForSneakDelay() bool {
	return i.sneaking && i.timeSinceLastMovement < 0.75
}

func (i *InputState) directionFromKeys(directionalKeys [4]ebiten.Key) (int, int) {
	inputDirectionX := 0
	inputDirectionY := 0
	if i.movementKeyIsPressed(directionalKeys[0]) {
		inputDirectionY = -1
	} else if i.movementKeyIsPressed(directionalKeys[1]) {
		inputDirectionX = -1
	} else if i.movementKeyIsPressed(directionalKeys[2]) {
		inputDirectionY = 1
	} else if i.movementKeyIsPressed(directionalKeys[3]) {
		inputDirectionX = 1
	}
	return inputDirectionX, inputDirectionY
}

func (i *InputState) padMovementButtonIsPressed(padId ebiten.GamepadID, button ebiten.StandardGamepadButton) bool {
	return inpututil.IsStandardGamepadButtonJustPressed(padId, button) || inpututil.StandardGamepadButtonPressDuration(padId, button)%i.movementDelayInTicks == 1
}

func (i *InputState) movementKeyIsPressed(key ebiten.Key) bool {

	return inpututil.IsKeyJustPressed(key) || inpututil.KeyPressDuration(key)%i.movementDelayInTicks == 1
}

func (i *InputState) isStillPressed(key ebiten.Key, delayInTicks int) bool {
	return inpututil.IsKeyJustPressed(key) || inpututil.KeyPressDuration(key)%delayInTicks == 0
}

func (i *InputState) getPadID() ebiten.GamepadID {
	pads := make([]ebiten.GamepadID, 1)
	pads = inpututil.AppendJustConnectedGamepadIDs(pads)
	if len(pads) > 0 {
		return pads[0]
	}
	return -1
}

const DefaultDelayForRunning = 10 //12

func NewInput(config console.GridConfig) *InputState {

	inputState := &InputState{
		movementDelayInTicks: DefaultDelayForRunning,
		config:               config,
		// key definitions
		keyboardMovementKeys:      [4]ebiten.Key{ebiten.KeyW, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD},
		keyboardPeekingKeys:       [4]ebiten.Key{ebiten.KeyT, ebiten.KeyF, ebiten.KeyG, ebiten.KeyH},
		keyboardActionKeys:        [4]ebiten.Key{ebiten.KeyI, ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL},
		keyboardSameTileActionKey: ebiten.KeyE,
		keyboardUseItemKey:        ebiten.KeySpace,
		keyboardInventoryKey:      ebiten.KeyQ,
		keyboardDropItemKey:       ebiten.KeyX,
		keyboardPutItemAwayKey:    ebiten.KeyC,
		keyboardSneakModeKey:      ebiten.KeyCapsLock,
	}
	inputState.LoadKeyDefs()
	return inputState
}

func (i *InputState) SaveKeyDefs() {
	file, err := os.Create("keydefs.txt")
	if err != nil {
		println(fmt.Sprintf("Error creating keydefs.txt: %v", err))
		return
	}
	defer file.Close()
	rec_files.Write(file, []rec_files.Record{
		[]rec_files.Field{
			{Name: "MoveUp", Value: i.keyboardMovementKeys[0].String()},
			{Name: "MoveLeft", Value: i.keyboardMovementKeys[1].String()},
			{Name: "MoveDown", Value: i.keyboardMovementKeys[2].String()},
			{Name: "MoveRight", Value: i.keyboardMovementKeys[3].String()},
			{Name: "PeekUp", Value: i.keyboardPeekingKeys[0].String()},
			{Name: "PeekLeft", Value: i.keyboardPeekingKeys[1].String()},
			{Name: "PeekDown", Value: i.keyboardPeekingKeys[2].String()},
			{Name: "PeekRight", Value: i.keyboardPeekingKeys[3].String()},
			{Name: "ActionUp", Value: i.keyboardActionKeys[0].String()},
			{Name: "ActionLeft", Value: i.keyboardActionKeys[1].String()},
			{Name: "ActionDown", Value: i.keyboardActionKeys[2].String()},
			{Name: "ActionRight", Value: i.keyboardActionKeys[3].String()},
			{Name: "ActionHere", Value: i.keyboardSameTileActionKey.String()},
			{Name: "UseItem", Value: i.keyboardUseItemKey.String()},
			{Name: "Inventory", Value: i.keyboardInventoryKey.String()},
			{Name: "DropItem", Value: i.keyboardDropItemKey.String()},
			{Name: "PutItemAway", Value: i.keyboardPutItemAwayKey.String()},
			{Name: "SneakMode", Value: i.keyboardSneakModeKey.String()},
		},
	})
}
func (i *InputState) LoadKeyDefs() {
	file, err := os.Open("keydefs.txt")
	if err != nil {
		println(fmt.Sprintf("Error opening keydefs.txt: %v", err))
		return
	}
	defer file.Close()
	records := rec_files.Read(file)
	if len(records) == 0 {
		println("No records in keydefs.txt")
		return
	}

	record := records[0]
	for _, field := range record {
		if field.Name == "MoveUp" {
			i.keyboardMovementKeys[0] = i.toEbitenKey(field.Value)
		} else if field.Name == "MoveLeft" {
			i.keyboardMovementKeys[1] = i.toEbitenKey(field.Value)
		} else if field.Name == "MoveDown" {
			i.keyboardMovementKeys[2] = i.toEbitenKey(field.Value)
		} else if field.Name == "MoveRight" {
			i.keyboardMovementKeys[3] = i.toEbitenKey(field.Value)
		} else if field.Name == "PeekUp" {
			i.keyboardPeekingKeys[0] = i.toEbitenKey(field.Value)
		} else if field.Name == "PeekLeft" {
			i.keyboardPeekingKeys[1] = i.toEbitenKey(field.Value)
		} else if field.Name == "PeekDown" {
			i.keyboardPeekingKeys[2] = i.toEbitenKey(field.Value)
		} else if field.Name == "PeekRight" {
			i.keyboardPeekingKeys[3] = i.toEbitenKey(field.Value)
		} else if field.Name == "ActionUp" {
			i.keyboardActionKeys[0] = i.toEbitenKey(field.Value)
		} else if field.Name == "ActionLeft" {
			i.keyboardActionKeys[1] = i.toEbitenKey(field.Value)
		} else if field.Name == "ActionDown" {
			i.keyboardActionKeys[2] = i.toEbitenKey(field.Value)
		} else if field.Name == "ActionRight" {
			i.keyboardActionKeys[3] = i.toEbitenKey(field.Value)
		} else if field.Name == "ActionHere" {
			i.keyboardSameTileActionKey = i.toEbitenKey(field.Value)
		} else if field.Name == "UseItem" {
			i.keyboardUseItemKey = i.toEbitenKey(field.Value)
		} else if field.Name == "Inventory" {
			i.keyboardInventoryKey = i.toEbitenKey(field.Value)
		} else if field.Name == "DropItem" {
			i.keyboardDropItemKey = i.toEbitenKey(field.Value)
		} else if field.Name == "PutItemAway" {
			i.keyboardPutItemAwayKey = i.toEbitenKey(field.Value)
		} else if field.Name == "SneakMode" {
			i.keyboardSneakModeKey = i.toEbitenKey(field.Value)
		} else {
			println(fmt.Sprintf("Unknown field %v", field.Name))
		}
	}
	println("Loaded keydefs.txt")
}

func (i *InputState) toEbitenKey(text string) ebiten.Key {
	var currentKey ebiten.Key
	keyRecognizeError := currentKey.UnmarshalText([]byte(text))
	if keyRecognizeError != nil {
		println(fmt.Sprintf("Error recognizing key %v: %v", text, keyRecognizeError))
	}
	return currentKey
}
func keyFromEbiten(key ebiten.Key) core.Key {
	switch key {
	case ebiten.KeyEscape:
		return core.KeyEscape
	case ebiten.KeyEnter:
		return core.KeyEnter
	case ebiten.KeyLeft:
		return core.KeyLeftArrow
	case ebiten.KeyRight:
		return core.KeyRightArrow
	case ebiten.KeyUp:
		return core.KeyUpArrow
	case ebiten.KeyDown:
		return core.KeyDownArrow
	case ebiten.KeyBackspace:
		return core.KeyBackspace
	case ebiten.KeyF1:
		return "F1"
	case ebiten.KeyF2:
		return "F2"
	case ebiten.KeyF3:
		return "F3"
	case ebiten.KeyF4:
		return "F4"
	case ebiten.KeyF5:
		return "F5"
	case ebiten.KeyF6:
		return "F6"
	case ebiten.KeyF7:
		return "F7"
	case ebiten.KeyF8:
		return "F8"
	case ebiten.KeyF9:
		return "F9"
	case ebiten.KeyF10:
		return "F10"
	case ebiten.KeyF11:
		return "F11"
	case ebiten.KeyF12:
		return "F12"
	case ebiten.KeySpace:
		return core.KeySpace
	case ebiten.KeyA:
		return "a"
	case ebiten.KeyB:
		return "b"
	case ebiten.KeyC:
		return "c"
	case ebiten.KeyD:
		return "d"
	case ebiten.KeyE:
		return "e"
	case ebiten.KeyF:
		return "f"
	case ebiten.KeyG:
		return "g"
	case ebiten.KeyH:
		return "h"
	case ebiten.KeyI:
		return "i"
	case ebiten.KeyJ:
		return "j"
	case ebiten.KeyK:
		return "k"
	case ebiten.KeyL:
		return "l"
	case ebiten.KeyM:
		return "m"
	case ebiten.KeyN:
		return "n"
	case ebiten.KeyO:
		return "o"
	case ebiten.KeyP:
		return "p"
	case ebiten.KeyQ:
		return "q"
	case ebiten.KeyR:
		return "r"
	case ebiten.KeyS:
		return "s"
	case ebiten.KeyT:
		return "t"
	case ebiten.KeyU:
		return "u"
	case ebiten.KeyV:
		return "v"
	case ebiten.KeyW:
		return "w"
	case ebiten.KeyX:
		return "x"
	case ebiten.KeyY:
		return "y"
	case ebiten.KeyZ:
		return "z"
	case ebiten.KeyDigit0:
		return "0"
	case ebiten.KeyDigit1:
		return "1"
	case ebiten.KeyDigit2:
		return "2"
	case ebiten.KeyDigit3:
		return "3"
	case ebiten.KeyDigit4:
		return "4"
	case ebiten.KeyDigit5:
		return "5"
	case ebiten.KeyDigit6:
		return "6"
	case ebiten.KeyDigit7:
		return "7"
	case ebiten.KeyDigit8:
		return "8"
	case ebiten.KeyDigit9:
		return "9"
	}
	return core.Key(rune(key))
}

func (i *InputState) SetMovementDelayForSneaking() {
	i.sneaking = true
}
func (i *InputState) SetMovementDelayForWalkingAndRunning() {
	i.sneaking = false
}

func (i *InputState) SetScale(scale float64) {
	i.dpiScale = scale
}

func (i *InputState) pollGamePadForGameplay() []core.InputCommand {
	var msgs = make([]core.InputCommand, 0)

	padId := i.getPadID()
	if padId > -1 {
		standardPressedButtons := make([]ebiten.StandardGamepadButton, 0)
		standardPressedButtons = inpututil.AppendPressedStandardGamepadButtons(padId, standardPressedButtons)
		maxAxis := ebiten.GamepadAxisCount(padId)
		leftStickChanged := false
		rightStickChanged := false
		for a := 0; a < maxAxis; a++ {
			v := ebiten.GamepadAxisValue(padId, a)
			if math.Abs(v-i.axes[a]) > 0.01 {
				if a == 0 || a == 1 {
					leftStickChanged = true
				} else if a == 2 || a == 5 {
					rightStickChanged = true
				}
			}
			i.axes[a] = v
		}
		if leftStickChanged {
			msgs = append(msgs, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: i.axes[0], YAxis: i.axes[1]})
		}
		if rightStickChanged {
			msgs = append(msgs, core.DirectionalGameCommand{Command: core.AimingDirection, XAxis: i.axes[2], YAxis: i.axes[5]})
		}
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontBottomLeft) {
			msgs = append(msgs, core.StartSneaking)
		} else if inpututil.IsStandardGamepadButtonJustReleased(padId, ebiten.StandardGamepadButtonFrontBottomLeft) {
			msgs = append(msgs, core.StopSneaking)
		}

		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontBottomRight) {
			msgs = append(msgs, core.UseEquippedItem)
		}
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontTopLeft) {
			msgs = append(msgs, core.NextItem)
		}

		movementDirectionX := 0
		movementDirectionY := 0
		if i.padMovementButtonIsPressed(padId, ebiten.StandardGamepadButtonLeftBottom) {
			movementDirectionY = 1
		}
		if i.padMovementButtonIsPressed(padId, ebiten.StandardGamepadButtonLeftTop) {
			movementDirectionY = -1
		}
		if i.padMovementButtonIsPressed(padId, ebiten.StandardGamepadButtonLeftLeft) {
			movementDirectionX = -1
		}
		if i.padMovementButtonIsPressed(padId, ebiten.StandardGamepadButtonLeftRight) {
			movementDirectionX = 1
		}
		playerWantsToMove := movementDirectionX != 0 || movementDirectionY != 0

		if !(i.mustWaitForSneakDelay()) && playerWantsToMove {
			msgs = append(msgs, core.DirectionalGameCommand{Command: core.MovementDirection, XAxis: float64(movementDirectionX), YAxis: float64(movementDirectionY)})
			i.timeSinceLastMovement = 0
		}

		actionDirectionX := 0
		actionDirectionY := 0
		actionTriggered := false
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightBottom) {
			actionDirectionY = 1
			actionTriggered = true
		} else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightTop) {
			actionDirectionY = -1
			actionTriggered = true
		} else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightLeft) {
			actionDirectionX = -1
			actionTriggered = true
		} else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightRight) {
			actionDirectionX = 1
			actionTriggered = true
		} else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontTopRight) {
			actionTriggered = true
		}
		if actionTriggered {
			msgs = append(msgs, core.DirectionalGameCommand{Command: core.ActionDirection, XAxis: float64(actionDirectionX), YAxis: float64(actionDirectionY)})
		}

		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftStick) {
			msgs = append(msgs, core.DropItem)
		}
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightStick) {
			msgs = append(msgs, core.PutItemAway)
		}
	}
	return msgs
}

func (i *InputState) pollMouse() []core.InputCommand {
	mx, my := ebiten.CursorPosition()
	xMouse := float64(mx) / (float64(i.config.TileWidth) * i.dpiScale)
	yMouse := float64(my) / (float64(i.config.TileHeight) * i.dpiScale)
	xMouse = common.Clamp(xMouse, 0, float64(i.config.GridWidth-1))
	yMouse = common.Clamp(yMouse, 0, float64(i.config.GridHeight-1))
	i.MousePosOnScreenGrid = geometry.Point{X: int(xMouse), Y: int(yMouse)}

	msgs := make([]core.InputCommand, 0)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseLeft, Pos: i.MousePosOnScreenGrid})
	} else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseLeftHeld, Pos: i.MousePosOnScreenGrid})
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseRight, Pos: i.MousePosOnScreenGrid})
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseLeftReleased, Pos: i.MousePosOnScreenGrid})
	}
	xOff, yOff := ebiten.Wheel()
	if xOff > 0 {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseWheelLeft, Pos: i.MousePosOnScreenGrid})
	} else if xOff < 0 {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseWheelRight, Pos: i.MousePosOnScreenGrid})
	}
	if yOff > 0 {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseWheelUp, Pos: i.MousePosOnScreenGrid})
	} else if yOff < 0 {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseWheelDown, Pos: i.MousePosOnScreenGrid})
	}
	if i.LastMousePos != i.MousePosOnScreenGrid {
		msgs = append(msgs, core.PointerCommand{Action: core.MouseMoved, Pos: i.MousePosOnScreenGrid})
	}
	i.LastMousePos = i.MousePosOnScreenGrid

	return msgs
}
