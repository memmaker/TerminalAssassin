package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/memmaker/terminal-assassin/game/services"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/console"
	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

type InputState struct {
	config               console.GridConfig
	MousePosOnScreenGrid geometry.Point
	LastMousePos         geometry.Point
	movementDelayInTicks int
	lastMovementAt       time.Time // used for ms-based sneak-step delay
	lastStepWasDiagonal  bool      // true when the last pad move was diagonal; used to apply sqrt(2) cool-down
	sneaking             bool      // keyboard sneak state (CapsLock)
	padSneaking          bool      // gamepad sneak state (Circle toggle)
	dpiScale             float64
	//axes                  [6]float64
	inputConsumed bool
	keyBuffer     []ebiten.Key

	// key definitions
	keyboardMovementKeys      [4]ebiten.Key
	keyboardPeekingKeys       [4]ebiten.Key
	keyboardSameTileActionKey ebiten.Key
	keyboardSneakModeKey      ebiten.Key
	keyboardDropItemKey       ebiten.Key
	keyboardHolsterItemKey    ebiten.Key
	keyboardUseItemKey        ebiten.Key
	keyboardInventoryKey      ebiten.Key

	gamepadIDsBuf              []ebiten.GamepadID
	gamepadIDs                 map[ebiten.GamepadID]struct{}
	justPressedStandardButtons map[ebiten.GamepadID][]ebiten.StandardGamepadButton
	lastUIScrollAt             time.Time // rate-limits analog-stick UI scrolling

	// required here to correctly calculate the mouse position on screen
	effectiveTileW float64
	effectiveTileH float64
	renderOffsetX  float64
	renderOffsetY  float64

	controllerMode string // services.ControllerKeyboardMouse or services.ControllerGamepad
}

func (i *InputState) PollGamepad() {
	if i.gamepadIDs == nil {
		i.gamepadIDs = map[ebiten.GamepadID]struct{}{}
	}

	// Log the gamepad connection events.
	i.gamepadIDsBuf = inpututil.AppendJustConnectedGamepadIDs(i.gamepadIDsBuf[:0])
	for _, id := range i.gamepadIDsBuf {
		log.Printf("gamepad connected: id: %d, SDL ID: %s", id, ebiten.GamepadSDLID(id))
		i.gamepadIDs[id] = struct{}{}
	}
	for id := range i.gamepadIDs {
		if inpututil.IsGamepadJustDisconnected(id) {
			log.Printf("gamepad disconnected: id: %d", id)
			delete(i.gamepadIDs, id)
		}
	}

	i.justPressedStandardButtons = map[ebiten.GamepadID][]ebiten.StandardGamepadButton{}
	for id := range i.gamepadIDs {
		/*
		   maxButton := ebiten.GamepadButton(ebiten.GamepadButtonCount(id))
		   for b := ebiten.GamepadButton(0); b < maxButton; b++ {
		       if ebiten.IsGamepadButtonPressed(id, b) {
		           i.pressedButtons[id] = append(i.pressedButtons[id], b)
		       }

		       // Log button events.
		       if inpututil.IsGamepadButtonJustPressed(id, b) {
		           log.Printf("button pressed: id: %d, button: %d", id, b)
		       }
		       if inpututil.IsGamepadButtonJustReleased(id, b) {
		           //log.Printf("button released: id: %d, button: %d", id, b)
		       }
		   }
		*/
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
				// Log button events.
				if inpututil.IsStandardGamepadButtonJustPressed(id, b) {
					//log.Printf("standard button pressed: id: %d, button: %d", id, b)
					i.justPressedStandardButtons[id] = append(i.justPressedStandardButtons[id], b)
				}
				if inpututil.IsStandardGamepadButtonJustReleased(id, b) {
					//log.Printf("standard button released: id: %d, button: %d", id, b)
				}
			}
		}
	}
}
func (i *InputState) VibrateGamepad(padId ebiten.GamepadID, duration time.Duration, strong float64, weak float64) {
	if strong > 0 || weak > 0 {
		op := &ebiten.VibrateGamepadOptions{
			Duration:        duration,
			StrongMagnitude: strong,
			WeakMagnitude:   weak,
		}
		ebiten.VibrateGamepad(padId, op)
	}
}

func (i *InputState) IsShiftPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyShift)
}
func (i *InputState) GetKeyDefinitions() services.KeyDefinitions {
	return services.KeyDefinitions{
		MovementKeys:      i.keyboardMovementKeys,
		PeekingKeys:       i.keyboardPeekingKeys,
		SameTileActionKey: i.keyboardSameTileActionKey,
		SneakModeKey:      i.keyboardSneakModeKey,
		DropItemKey:       i.keyboardDropItemKey,
		HolsterItemKey:    i.keyboardHolsterItemKey,
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
	padPressed := inpututil.IsStandardGamepadButtonJustPressed(i.getPadID(), ebiten.StandardGamepadButtonRightBottom) || inpututil.IsStandardGamepadButtonJustPressed(i.getPadID(), ebiten.StandardGamepadButtonRightRight)
	somethingWasPressed := keyPressed || mousePressed || padPressed
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
		if caps {
			return core.KeyCommand{Key: ":"}
		}
		return core.KeyCommand{Key: ";"}
	// add case for ":"
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
	commands := make([]core.InputCommand, 0)

	// Navigation keys: support held + auto-repeat so the user can scroll menus
	// by holding an arrow key.
	//fmt.Println("pollKeyBoardForUI")
	i.keyBuffer = i.keyBuffer[:0]
	i.keyBuffer = inpututil.AppendPressedKeys(i.keyBuffer)
	for _, key := range i.keyBuffer {
		if !i.isStillPressed(key, 14) {
			continue
		}
		switch key {
		case ebiten.KeyArrowUp, ebiten.KeyW:
			commands = append(commands, core.MenuUp)
		case ebiten.KeyArrowDown, ebiten.KeyS:
			commands = append(commands, core.MenuDown)
		case ebiten.KeyArrowLeft, ebiten.KeyA:
			commands = append(commands, core.MenuLeft)
		case ebiten.KeyArrowRight, ebiten.KeyD:
			commands = append(commands, core.MenuRight)
		}
	}

	// Action and text keys: only on the very first press.
	// Using AppendJustPressedKeys here prevents spurious extra triggers when
	// the OS fires a new keydown event after the window regains focus during a
	// fullscreen/windowed mode switch.
	var justPressed []ebiten.Key
	justPressed = inpututil.AppendJustPressedKeys(justPressed)
	isCaps := ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyCapsLock)
	for _, key := range justPressed {
		switch key {
		case ebiten.KeyEnter:
			commands = append(commands, core.MenuConfirm)
		case ebiten.KeyE:
			commands = append(commands, core.MenuConfirm)
		case ebiten.KeyQ:
			commands = append(commands, core.MenuCancel)
		case ebiten.KeyEscape:
			commands = append(commands, core.MenuCancel)
		case ebiten.KeyArrowUp, ebiten.KeyW,
			ebiten.KeyArrowDown, ebiten.KeyS,
			ebiten.KeyArrowLeft, ebiten.KeyA,
			ebiten.KeyArrowRight, ebiten.KeyD:
			// Already handled with repeat above; skip the first-press duplicate.
		default:
			commands = append(commands, i.KeyToKeyCommand(key, isCaps))
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
	i.PollGamepad()
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
	if i.controllerMode != services.ControllerGamepad {
		commands = append(commands, i.pollKeyBoardForGameplay()...)
		commands = append(commands, i.pollMouse()...)
	}
	if i.controllerMode != services.ControllerKeyboardMouse {
		commands = append(commands, i.pollGamePadForGameplay()...)
	}
	return commands
}
func (i *InputState) PollEditorCommands() []core.InputCommand {
	if i.inputConsumed {
		return []core.InputCommand{}
	}
	i.inputConsumed = true
	var commands = make([]core.InputCommand, 0)
	commands = append(commands, i.pollPrintables()...)
	// Continuous scrolling: repeat arrow keys while held (first press already covered by pollPrintables).
	for _, key := range []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyArrowUp, ebiten.KeyArrowDown} {
		if dur := inpututil.KeyPressDuration(key); dur > 1 && dur%i.movementDelayInTicks == 0 {
			commands = append(commands, i.KeyToKeyCommand(key, false))
		}
	}
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
	commands = append(commands, i.pollGamePadForUI()...)
	commands = append(commands, i.pollMouse()...)

	return commands
}
func (i *InputState) pollKeyBoardForGameplay() []core.InputCommand {
	var commands = make([]core.InputCommand, 0)

	// ── WASD movement — Shift=run, sneak-toggle=slow ──────────────────────────
	movementDirectionX, movementDirectionY := i.heldDirection(i.keyboardMovementKeys)
	if movementDirectionX != 0 || movementDirectionY != 0 {
		shiftHeld := ebiten.IsKeyPressed(ebiten.KeyShift)
		var delay int64
		switch {
		case shiftHeld && !i.sneaking:
			delay = int64(core.RunningStepDelayMs)
		case i.sneaking:
			delay = int64(core.SneakStepDelayMs)
		default:
			delay = int64(core.WalkStepDelayMs)
		}
		if i.lastMovementAt.IsZero() || time.Since(i.lastMovementAt).Milliseconds() >= delay {
			commands = append(commands, core.DirectionalGameCommand{Command: core.MovementDirection, XAxis: float64(movementDirectionX), YAxis: float64(movementDirectionY)})
			i.lastMovementAt = time.Now()
		}
	}

	// ── Arrow key peek ────────────────────────────────────────────────────────
	peekDirX, peekDirY := i.heldDirection(i.keyboardPeekingKeys)
	if peekDirX != 0 || peekDirY != 0 {
		commands = append(commands, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: float64(peekDirX), YAxis: float64(peekDirY)})
	}

	// ── Sneak toggle (CapsLock) ───────────────────────────────────────────────
	if inpututil.IsKeyJustPressed(i.keyboardSneakModeKey) {
		if i.sneaking {
			commands = append(commands, core.StopSneaking)
		} else {
			commands = append(commands, core.StartSneaking)
		}
	}

	// ── Inventory / item management ───────────────────────────────────────────
	if inpututil.IsKeyJustPressed(i.keyboardInventoryKey) {
		commands = append(commands, core.OpenInventory)
	}
	if inpututil.IsKeyJustPressed(i.keyboardDropItemKey) {
		commands = append(commands, core.DropItem)
	}
	if inpututil.IsKeyJustPressed(i.keyboardHolsterItemKey) {
		commands = append(commands, core.HolsterItem)
	}

	// ── Mouse aim / ranged fire (Space) ───────────────────────────────────────
	if inpututil.IsKeyJustPressed(i.keyboardUseItemKey) {
		commands = append(commands, core.BeginMouseAiming)
	}

	// ── Context action — interact / pickpocket / knock (E) ───────────────────
	if inpututil.IsKeyJustPressed(i.keyboardSameTileActionKey) {
		commands = append(commands, core.ContextAction)
	}

	// ── Combat: assassination (R), dive-tackle (F), use item at peek (V) ─────
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		commands = append(commands, core.Assassinate)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		commands = append(commands, core.DiveTackle)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		commands = append(commands, core.UseItem)
	}

	// ── Free look cursor (Tab) ────────────────────────────────────────────────
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		commands = append(commands, core.ToggleLookMode)
	}

	// ── Confirm / cancel ──────────────────────────────────────────────────────
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		commands = append(commands, core.Confirm)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		commands = append(commands, core.Cancel)
	}

	// ── Function keys ─────────────────────────────────────────────────────────
	for _, fk := range []ebiten.Key{
		ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5,
		ebiten.KeyF6, ebiten.KeyF7, ebiten.KeyF8, ebiten.KeyF9, ebiten.KeyF10,
	} {
		if inpututil.IsKeyJustPressed(fk) {
			commands = append(commands, core.KeyCommand{Key: keyFromEbiten(fk)})
		}
	}

	return commands
}

// heldDirection returns a cardinal direction from whichever of the four keys
// is currently held. Order: up, left, down, right.
func (i *InputState) heldDirection(keys [4]ebiten.Key) (int, int) {
	if ebiten.IsKeyPressed(keys[0]) {
		return 0, -1
	}
	if ebiten.IsKeyPressed(keys[1]) {
		return -1, 0
	}
	if ebiten.IsKeyPressed(keys[2]) {
		return 0, 1
	}
	if ebiten.IsKeyPressed(keys[3]) {
		return 1, 0
	}
	return 0, 0
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

// isAnalogDiagonal reports whether (x, y) maps to a diagonal direction using
// the same equal-45°-sector logic as toIntDirection in gameplay.go.
// Returns true when the ratio |y|/|x| falls in [tan(22.5°), tan(67.5°)].
func isAnalogDiagonal(x, y float64) bool {
	const tanOf22_5 = 0.41421356237 // √2 − 1
	const tanOf67_5 = 2.41421356237 // √2 + 1
	absX := math.Abs(x)
	absY := math.Abs(y)
	if absX == 0 || absY == 0 {
		return false
	}
	ratio := absY / absX
	return ratio >= tanOf22_5 && ratio <= tanOf67_5
}

func (i *InputState) movementKeyIsPressed(key ebiten.Key) bool {

	return inpututil.IsKeyJustPressed(key) || inpututil.KeyPressDuration(key)%i.movementDelayInTicks == 1
}

func (i *InputState) isStillPressed(key ebiten.Key, delayInTicks int) bool {
	return inpututil.IsKeyJustPressed(key) || inpututil.KeyPressDuration(key)%delayInTicks == 0
}

func (i *InputState) getPadID() ebiten.GamepadID {
	if len(i.gamepadIDs) == 0 {
		return -1
	}
	for id := range i.gamepadIDs {
		return id
	}
	return -1
}

func NewInput(config console.GridConfig) *InputState {

	inputState := &InputState{
		movementDelayInTicks: core.RunningStepDelayMs * 60 / 1000, // = 7 at 60 TPS
		config:               config,
		controllerMode:       services.ControllerKeyboardMouse,
		// key definitions
		keyboardMovementKeys:      [4]ebiten.Key{ebiten.KeyW, ebiten.KeyA, ebiten.KeyS, ebiten.KeyD},
		keyboardPeekingKeys:       [4]ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowLeft, ebiten.KeyArrowDown, ebiten.KeyArrowRight},
		keyboardSameTileActionKey: ebiten.KeyE,
		keyboardUseItemKey:        ebiten.KeySpace,
		keyboardInventoryKey:      ebiten.KeyQ,
		keyboardDropItemKey:       ebiten.KeyX,
		keyboardHolsterItemKey:    ebiten.KeyC,
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
			{Name: "ActionHere", Value: i.keyboardSameTileActionKey.String()},
			{Name: "UseItem", Value: i.keyboardUseItemKey.String()},
			{Name: "Inventory", Value: i.keyboardInventoryKey.String()},
			{Name: "DropItem", Value: i.keyboardDropItemKey.String()},
			{Name: "HolsterItem", Value: i.keyboardHolsterItemKey.String()},
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
		} else if field.Name == "ActionUp" || field.Name == "ActionLeft" || field.Name == "ActionDown" || field.Name == "ActionRight" {
			// removed concept – ignore legacy values
		} else if field.Name == "ActionHere" {
			i.keyboardSameTileActionKey = i.toEbitenKey(field.Value)
		} else if field.Name == "UseItem" {
			i.keyboardUseItemKey = i.toEbitenKey(field.Value)
		} else if field.Name == "Inventory" {
			i.keyboardInventoryKey = i.toEbitenKey(field.Value)
		} else if field.Name == "DropItem" {
			i.keyboardDropItemKey = i.toEbitenKey(field.Value)
		} else if field.Name == "HolsterItem" || field.Name == "PutItemAway" {
			i.keyboardHolsterItemKey = i.toEbitenKey(field.Value)
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

func (i *InputState) SetControllerMode(mode string) {
	i.controllerMode = mode
}

func (i *InputState) SetScale(scale float64) {
	i.dpiScale = scale
	i.effectiveTileW = float64(i.config.TileSize) * scale
	i.effectiveTileH = float64(i.config.TileSize) * scale
	i.renderOffsetX = 0
	i.renderOffsetY = 0
}

func (i *InputState) SetRenderParams(tileW, tileH, offsetX, offsetY float64) {
	i.effectiveTileW = tileW
	i.effectiveTileH = tileH
	i.renderOffsetX = offsetX
	i.renderOffsetY = offsetY
}

func (i *InputState) pollGamePadForGameplay() []core.InputCommand {

	var msgs = make([]core.InputCommand, 0)

	padId := i.getPadID()

	if padId > -1 {

		// ── Right analog stick → peek ─────────
		rightX := ebiten.StandardGamepadAxisValue(padId, ebiten.StandardGamepadAxisRightStickHorizontal)
		rightY := ebiten.StandardGamepadAxisValue(padId, ebiten.StandardGamepadAxisRightStickVertical)
		const rightDeadzone = 0.2
		rightStickActive := rightX > rightDeadzone || rightX < -rightDeadzone ||
			rightY > rightDeadzone || rightY < -rightDeadzone

		// ── D-Pad Left -> Open Inventory ─────────────────────────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftLeft) {
			msgs = append(msgs, core.OpenInventory)
		}

		// ── D-Pad Down -> Drop Item ─────────────────────────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftBottom) {
			msgs = append(msgs, core.DropItem)
		}

		// ── D-Pad Up -> Holster Item ─────────────────────────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftTop) {
			msgs = append(msgs, core.HolsterItem)
		}

		// ── D-Pad Right → toggle sneaking ───────────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftRight) {
			i.padSneaking = !i.padSneaking
			if i.padSneaking {
				msgs = append(msgs, core.StartSneaking)
			} else {
				msgs = append(msgs, core.StopSneaking)
			}
		}

		// ── Triangle (RightTop) → Assassination ─────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightTop) {
			msgs = append(msgs, core.Assassinate)
		}

		// ── Square (RightLeft) → use item at current peek tile ───────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightLeft) {
			msgs = append(msgs, core.UseItem)
		}

		// ── Circle (RightBottom) → dive tackle ─────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightRight) {
			msgs = append(msgs, core.DiveTackle)
		}

		// ── R1 (FrontTopRight) / ✕·A (RightBottom) → interact at current peek tile ──
		// Dialogue when walking/standing; pickpocket when sneaking; knock when facing a wall.
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontTopRight) ||
			inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightBottom) {
			msgs = append(msgs, core.ContextAction)
		}

		// ── R2 (FrontBottomRight) → fire/throw at aimed position ────────────
		if ebiten.IsStandardGamepadButtonPressed(padId, ebiten.StandardGamepadButtonFrontBottomRight) {
			msgs = append(msgs, core.UseRangedItem)
		}

		// -- L1 (FrontTopLeft) held -> Run ─────────────────────────
		l1Held := ebiten.IsStandardGamepadButtonPressed(padId, ebiten.StandardGamepadButtonFrontTopLeft)
		var baseStepDelayMs int64
		switch {
		case l1Held:
			baseStepDelayMs = int64(core.RunningStepDelayMs)
			if i.padSneaking {
				i.padSneaking = false
				msgs = append(msgs, core.StopSneaking)
			}
		case i.padSneaking:
			baseStepDelayMs = int64(core.SneakStepDelayMs)
		default:
			baseStepDelayMs = int64(core.WalkStepDelayMs)
		}

		if rightStickActive {
			// -- L2 (FrontBottomLeft) held -> Aim instead of peek ─────────────────────────
			if ebiten.IsStandardGamepadButtonPressed(padId, ebiten.StandardGamepadButtonFrontBottomLeft) {
				msgs = append(msgs, core.DirectionalGameCommand{Command: core.AimingDirection, XAxis: rightX, YAxis: rightY})
			} else {
				msgs = append(msgs, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: rightX, YAxis: rightY})
			}
		}

		// -- L2 released → exit pad aim mode regardless of stick position ────────────
		if inpututil.IsStandardGamepadButtonJustReleased(padId, ebiten.StandardGamepadButtonFrontBottomLeft) {
			msgs = append(msgs, core.StopAiming)
		}

		// ── Left analog stick → movement ─────────────────────────────────────
		effectiveDelay := baseStepDelayMs
		if i.lastStepWasDiagonal {
			effectiveDelay = int64(math.Round(float64(baseStepDelayMs) * math.Sqrt2))
		}
		readyForNextStep := i.lastMovementAt.IsZero() || time.Since(i.lastMovementAt).Milliseconds() >= effectiveDelay
		if readyForNextStep {
			command := core.DirectionalGameCommand{
				Command: core.MovementDirection,
			}
			// ── Left analog stick supplements D-Pad for movement ─────────────────
			const leftStickDeadzone = 0.1

			if v := ebiten.StandardGamepadAxisValue(padId, ebiten.StandardGamepadAxisLeftStickHorizontal); v > leftStickDeadzone || v < -leftStickDeadzone {
				command.XAxis = v
			}
			if v := ebiten.StandardGamepadAxisValue(padId, ebiten.StandardGamepadAxisLeftStickVertical); v > leftStickDeadzone || v < -leftStickDeadzone {
				command.YAxis = v
			}

			if command.XAxis != 0 || command.YAxis != 0 {
				i.lastStepWasDiagonal = isAnalogDiagonal(command.XAxis, command.YAxis)
				msgs = append(msgs, command)
				i.lastMovementAt = time.Now()
			}
		}

		// ── Options (CenterRight) → pause menu ───────────────────────────────
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonCenterRight) {
			msgs = append(msgs, core.Cancel)
		}

		// ── Select (CenterLeft) or R3 (RightStick click) → toggle look cursor mode
		if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonCenterLeft) ||
			inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightStick) {
			msgs = append(msgs, core.ToggleLookMode)
		}
	}
	return msgs
}
func (i *InputState) pollGamePadForUI() []core.InputCommand {
	padId := i.getPadID()
	pressed := i.justPressedStandardButtons[padId]
	commands := make([]core.InputCommand, 0, 4)
	for _, button := range pressed {
		if button == ebiten.StandardGamepadButtonRightBottom {
			commands = append(commands, core.MenuConfirm)
		} else if button == ebiten.StandardGamepadButtonRightRight {
			commands = append(commands, core.MenuCancel)
		} else if button == ebiten.StandardGamepadButtonLeftTop {
			commands = append(commands, core.MenuUp)
		} else if button == ebiten.StandardGamepadButtonLeftBottom {
			commands = append(commands, core.MenuDown)
		} else if button == ebiten.StandardGamepadButtonLeftLeft {
			commands = append(commands, core.MenuLeft)
		} else if button == ebiten.StandardGamepadButtonLeftRight {
			commands = append(commands, core.MenuRight)
		}
	}

	// ── Left analog stick → continuous scroll in UI (e.g. pager) ────────────
	if padId > -1 {
		const leftDeadzone = 0.2
		leftY := ebiten.StandardGamepadAxisValue(padId, ebiten.StandardGamepadAxisLeftStickVertical)
		if leftY > leftDeadzone || leftY < -leftDeadzone {
			// Scale scroll speed with stick deflection: full push → ~100 ms per line,
			// half push → ~200 ms per line.
			delay := time.Duration(float64(100*time.Millisecond) / math.Abs(leftY))
			if i.lastUIScrollAt.IsZero() || time.Since(i.lastUIScrollAt) >= delay {
				if leftY > 0 {
					commands = append(commands, core.MenuDown)
				} else {
					commands = append(commands, core.MenuUp)
				}
				i.lastUIScrollAt = time.Now()
			}
		} else {
			// Reset so the next stick push scrolls immediately.
			i.lastUIScrollAt = time.Time{}
		}
	}

	return commands
}

func (i *InputState) pollMouse() []core.InputCommand {
	mx, my := ebiten.CursorPosition()
	tileW := i.effectiveTileW
	tileH := i.effectiveTileH
	if tileW == 0 {
		tileW = float64(i.config.TileSize) * i.dpiScale
	}
	if tileH == 0 {
		tileH = float64(i.config.TileSize) * i.dpiScale
	}
	xMouse := (float64(mx) - i.renderOffsetX) / tileW
	yMouse := (float64(my) - i.renderOffsetY) / tileH
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
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
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
