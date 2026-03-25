package main

import (
    "fmt"
    "github.com/memmaker/terminal-assassin/game/services"
    rec_files "github.com/memmaker/terminal-assassin/rec-files"
    "log"
    "os"
    "strings"
    "time"

    "github.com/hajimehoshi/ebiten/v2"
    "github.com/hajimehoshi/ebiten/v2/inpututil"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/geometry"
)

var rightAnalogStickAxisX = 3 // for xbox controller
var rightAnalogStickAxisY = 4 // for xbox controller
//var rightAnalogStickAxisX = 2 // for ps4 controller
//var rightAnalogStickAxisY = 5 // for ps4 controller

type InputState struct {
    config               console.GridConfig
    MousePosOnScreenGrid geometry.Point
    LastMousePos         geometry.Point
    movementDelayInTicks int
    lastMovementAt       time.Time // used for ms-based sneak-step delay
    sneaking             bool      // keyboard sneak state (CapsLock)
    padSneaking          bool      // gamepad sneak state (Circle toggle)
    dpiScale             float64
    //axes                  [6]float64
    inputConsumed bool
    keyBuffer     []ebiten.Key

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

    gamepadIDsBuf []ebiten.GamepadID
    gamepadIDs    map[ebiten.GamepadID]struct{}
    axes          map[ebiten.GamepadID][]float64
    //pressedButtons         map[ebiten.GamepadID][]ebiten.GamepadButton
    pressedStandardButtons     map[ebiten.GamepadID][]ebiten.StandardGamepadButton
    justPressedStandardButtons map[ebiten.GamepadID][]ebiten.StandardGamepadButton

    // required here to correctly calculate the mouse position on screen
    effectiveTileW float64
    effectiveTileH float64
    renderOffsetX  float64
    renderOffsetY  float64
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

    i.axes = map[ebiten.GamepadID][]float64{}
    //i.pressedButtons = map[ebiten.GamepadID][]ebiten.GamepadButton{}
    i.pressedStandardButtons = map[ebiten.GamepadID][]ebiten.StandardGamepadButton{}
    i.justPressedStandardButtons = map[ebiten.GamepadID][]ebiten.StandardGamepadButton{}
    for id := range i.gamepadIDs {
        maxAxis := ebiten.GamepadAxisType(ebiten.GamepadAxisCount(id))
        for a := ebiten.GamepadAxisType(0); a < maxAxis; a++ {
            v := ebiten.GamepadAxisValue(id, a)
            i.axes[id] = append(i.axes[id], v)
        }
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
                if ebiten.IsStandardGamepadButtonPressed(id, b) {
                    i.pressedStandardButtons[id] = append(i.pressedStandardButtons[id], b)
                }
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
    padPressed := inpututil.IsStandardGamepadButtonJustPressed(i.getPadID(), ebiten.StandardGamepadButtonRightBottom) || inpututil.IsStandardGamepadButtonJustPressed(i.getPadID(), ebiten.StandardGamepadButtonRightRight) || inpututil.IsStandardGamepadButtonJustPressed(i.getPadID(), ebiten.StandardGamepadButtonRightBottom)
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
    commands := make([]core.InputCommand, 0)

    // Navigation keys: support held + auto-repeat so the user can scroll menus
    // by holding an arrow key.
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
    commands = append(commands, i.pollGamePadForUI()...)
    commands = append(commands, i.pollMouse()...)

    return commands
}
func (i *InputState) pollKeyBoardForGameplay() []core.InputCommand {
    var commands = make([]core.InputCommand, 0)

    movementDirectionX, movementDirectionY := i.directionFromKeys(i.keyboardMovementKeys)
    playerWantsToMove := movementDirectionX != 0 || movementDirectionY != 0

    actionDirX, actionDirY := 0, 0
    cardinalAction := false

    if ebiten.IsKeyPressed(ebiten.KeyShift) {
        actionDirX, actionDirY = movementDirectionX, movementDirectionY
        cardinalAction = playerWantsToMove
    } else {

        if !(i.mustWaitForSneakDelay()) && playerWantsToMove {
            commands = append(commands, core.DirectionalGameCommand{Command: core.MovementDirection, XAxis: float64(movementDirectionX), YAxis: float64(movementDirectionY)})
            i.lastMovementAt = time.Now()
        }

        actionDirX, actionDirY = i.directionFromKeys(i.keyboardActionKeys)
        cardinalAction = actionDirX != 0 || actionDirY != 0
    }

    peekDirX, peekDirY := i.directionFromKeys(i.keyboardPeekingKeys)
    if peekDirX != 0 || peekDirY != 0 {
        commands = append(commands, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: float64(peekDirX), YAxis: float64(peekDirY)})
    }

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
    return i.sneaking && time.Since(i.lastMovementAt).Milliseconds() < int64(core.SneakStepDelayMs)
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
    axes := i.axes[padId]
    if padId > -1 {
        if axes[0] > 0.1 || axes[0] < -0.1 || axes[1] > 0.1 || axes[1] < -0.1 {
            msgs = append(msgs, core.DirectionalGameCommand{Command: core.PeekingDirection, XAxis: axes[0], YAxis: axes[1]})
        }
        if axes[rightAnalogStickAxisX] > 0.1 || axes[rightAnalogStickAxisX] < -0.1 || axes[rightAnalogStickAxisY] > 0.1 || axes[rightAnalogStickAxisY] < -0.1 {
            msgs = append(msgs, core.DirectionalGameCommand{Command: core.AimingDirection, XAxis: axes[rightAnalogStickAxisX], YAxis: axes[rightAnalogStickAxisY]})
        }

        // Circle (RightRight) toggles sneaking on/off.
        if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightRight) {
            i.padSneaking = !i.padSneaking
            if i.padSneaking {
                msgs = append(msgs, core.StartSneaking)
            } else {
                msgs = append(msgs, core.StopSneaking)
            }
        }

        if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontBottomRight) {
            msgs = append(msgs, core.UseEquippedItem)
        }
        if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonFrontTopLeft) {
            msgs = append(msgs, core.NextItem)
        }
        if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightBottom) {
            msgs = append(msgs, core.Assassinate)
        }

        triangleHeld := ebiten.IsStandardGamepadButtonPressed(padId, ebiten.StandardGamepadButtonRightTop)

        if triangleHeld {
            // Triangle held + D-Pad → directional context action
            // Triangle tapped alone (no D-Pad) → same-tile context action
            actionDirX, actionDirY := 0, 0
            actionTriggered := false
            if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftTop) {
                actionDirY = -1
                actionTriggered = true
            } else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftBottom) {
                actionDirY = 1
                actionTriggered = true
            } else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftLeft) {
                actionDirX = -1
                actionTriggered = true
            } else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonLeftRight) {
                actionDirX = 1
                actionTriggered = true
            } else if inpututil.IsStandardGamepadButtonJustPressed(padId, ebiten.StandardGamepadButtonRightTop) {
                // Triangle just tapped with no simultaneous D-Pad → same-tile action
                actionTriggered = true
            }
            if actionTriggered {
                msgs = append(msgs, core.DirectionalGameCommand{Command: core.ActionDirection, XAxis: float64(actionDirX), YAxis: float64(actionDirY)})
            }
        } else {
            // D-Pad controls movement with three explicit speed modes:
            //   Circle toggled (padSneaking) → sneak  → MovementModeSneaking
            //   L1 held                      → sprint → MovementModeRunning
            //   default                      → walk   → MovementModeWalking
            //
            // We enforce the delay via lastMovementAt (shared across all directions)
            // so that neither re-pressing nor changing direction can bypass the delay.
            l1Held := ebiten.IsStandardGamepadButtonPressed(padId, ebiten.StandardGamepadButtonFrontBottomLeft)

            var stepDelayMs int64
            switch {
            case i.padSneaking:
                stepDelayMs = int64(core.SneakStepDelayMs)
            case l1Held:
                stepDelayMs = int64(core.RunningStepDelayMs)
            default:
                stepDelayMs = int64(core.WalkStepDelayMs)
            }

            readyForNextStep := i.lastMovementAt.IsZero() || time.Since(i.lastMovementAt).Milliseconds() >= stepDelayMs

            padCheck := func(button ebiten.StandardGamepadButton) bool {
                return readyForNextStep && ebiten.IsStandardGamepadButtonPressed(padId, button)
            }

            movementDirectionX := 0
            movementDirectionY := 0
            if padCheck(ebiten.StandardGamepadButtonLeftBottom) {
                movementDirectionY = 1
            }
            if padCheck(ebiten.StandardGamepadButtonLeftTop) {
                movementDirectionY = -1
            }
            if padCheck(ebiten.StandardGamepadButtonLeftLeft) {
                movementDirectionX = -1
            }
            if padCheck(ebiten.StandardGamepadButtonLeftRight) {
                movementDirectionX = 1
            }
            if movementDirectionX != 0 || movementDirectionY != 0 {
                msgs = append(msgs, core.DirectionalGameCommand{Command: core.MovementDirection, XAxis: float64(movementDirectionX), YAxis: float64(movementDirectionY)})
                i.lastMovementAt = time.Now()
            }
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
