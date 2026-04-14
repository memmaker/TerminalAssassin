package game

import (
    "math/rand"
    "path"

    "github.com/hajimehoshi/ebiten/v2"

    "github.com/memmaker/terminal-assassin/common"
    "github.com/memmaker/terminal-assassin/console"
    "github.com/memmaker/terminal-assassin/game/core"
    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/game/stimuli"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/utils"
)

type ActiveAnimation struct {
    nextFrame func(frameIndex int, ticksAlive uint64) bool
    DrawFrame func(con console.CellInterface, frameIndex int)
    // frameCount is the number of frames in the animation. the animation will stop after this many frames, EXCEPT if FinishCondition is present
    frameCount   int
    CurrentFrame int
    ID           uint64
    // Update is called only on frame change
    Update func(frameIndex int)
    // FinishedCallback is called when the animation is finished
    FinishedCallback func()
    // FinishCondition if present, will be called to check if the animation should finish, ignoring the frame count.
    // The finish condition will be checked upon FRAME CHANGE.
    FinishCondition func() bool
    // CancelCondition if present, will be called to check if the animation should be cancelled. The cancel condition will be checked upon EVERY TICK.
    CancelCondition func() bool
    // CancelCallback is called when the animation is cancelled
    CancelCallback            func()
    TicksAliveForCurrentFrame uint64
}

func (a ActiveAnimation) NextFrame() bool {
    return a.nextFrame(a.CurrentFrame, a.TicksAliveForCurrentFrame)
}

// 1. Trigger Animation
// 2. Render first frame
// 3. Send message for next frame with delay
// 4. Render next frame
func (a *Animator) cellOnScreen(grid console.CellInterface, worldPos geometry.Point) common.Cell {
    camera := a.engine.GetGame().GetCamera()
    screenPos := camera.WorldToScreen(worldPos)
    return grid.AtSquare(screenPos)
}

func (a *Animator) drawWorldToScreen(grid console.CellInterface, p geometry.Point, c common.Cell) {
    player := a.engine.GetGame().GetMap().Player
    if !player.CanSee(p) {
        return
    }
    mapWidth, mapHeight := a.engine.MapWindowWidth(), a.engine.MapWindowHeight()
    camera := a.engine.GetGame().GetCamera()
    screenPos := camera.WorldToScreen(p)

    if screenPos.X < 0 || screenPos.X >= mapWidth || screenPos.Y < 0 || screenPos.Y >= mapHeight {
        return
    }
    grid.SetSquare(screenPos, c)
}

// AssassinationAnimation plays a brief blinking animation of the weapon icon on
// each target's position to give the assassination action a moment of weight.
// It lasts ~0.6 s (4 frames × 0.15 s) and calls finishedCallback when done.
func (a *Animator) AssassinationAnimation(targets []*core.Actor, icon rune, finishedCallback func()) {
    // Capture positions at the moment the animation starts.
    targetPositions := make([]geometry.Point, len(targets))
    for i, t := range targets {
        targetPositions[i] = t.Pos()
    }

    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        for _, pos := range targetPositions {
            if !game.IsOnScreen(pos) {
                continue
            }
            cellAt := a.cellOnScreen(grid, pos)
            style := cellAt.Style
            if frameIndex%2 == 0 {
                style.Foreground = core.CurrentTheme.BloodForeground
            }
            a.drawWorldToScreen(grid, pos, common.Cell{Rune: icon, Style: style})
        }
    }

    const frameDelayInSeconds = 0.15
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }

    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       4,
        FinishedCallback: finishedCallback,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}

func (a *Animator) ActorEngagedAnimation(person *core.Actor, r rune, actionPosition geometry.Point, timeNeededInSeconds float64, finishedCallback func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(actionPosition) {
            return
        }
        cellAt := a.cellOnScreen(grid, actionPosition)
        style := cellAt.Style
        if frameIndex%2 == 0 {
            a.drawWorldToScreen(grid, actionPosition, common.Cell{Rune: r, Style: style})
        }
    }
    frameDelayInSeconds := 0.5
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    advanceToNextFrame := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }

    animation := &ActiveAnimation{
        nextFrame:        advanceToNextFrame,
        DrawFrame:        drawFunc,
        frameCount:       int(timeNeededInSeconds / frameDelayInSeconds),
        FinishedCallback: finishedCallback,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }

    a.addAnimation(animation)
}

func (a *Animator) PlayerChangeClothesAnimation(actionPosition geometry.Point, otherClothes core.Clothing, finishedCallback, cancelCallback func()) {
	missionMap := a.engine.GetGame().GetMap()
	player := missionMap.Player
	otherClothesColor := otherClothes.FgColor()
	onFinish := func() {
		oldClothing := player.Clothes // still old before finishedCallback runs
		finishedCallback()
		a.engine.PublishEvent(services.PlayerChangedClothesEvent{
			OldClothing: oldClothing,
			NewClothing: otherClothes,
		})
	}
	a.engagedWithSoundAnimation(player, actionPosition, "get-dressed", core.GlyphClothing, otherClothesColor, onFinish, cancelCallback)
}

func (a *Animator) ActorEngagedIllegalAnimation(person *core.Actor, r rune, actionPosition geometry.Point, timeNeededInSeconds float64, finishedCallback func(), cancelCallback func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(actionPosition) {
            return
        }
        cellAt := a.cellOnScreen(grid, actionPosition)
        style := cellAt.Style
        if frameIndex%2 == 0 {
            style.Foreground = core.CurrentTheme.DangerForeground
        }
        a.drawWorldToScreen(grid, actionPosition, common.Cell{Rune: r, Style: style})
    }
    frameDelayInSeconds := 0.5
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       int(timeNeededInSeconds / frameDelayInSeconds),
        CurrentFrame:     0,
        ID:               0,
        Update:           nil,
        FinishCondition:  nil,
        FinishedCallback: finishedCallback,
        CancelCondition:  ActorMovedOrIncapacitated(person),
        CancelCallback:   cancelCallback,
    }
    a.addAnimation(animation)
}

func (a *Animator) ActorEngagedIllegalAnimationWithSound(person *core.Actor, r rune, actionPosition geometry.Point, audioCue string, finishedCallback func(), cancelCallback func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(actionPosition) {
            return
        }
        cellAt := a.cellOnScreen(grid, actionPosition)
        style := cellAt.Style
        if frameIndex%2 == 0 {
            style.Foreground = core.CurrentTheme.DangerForeground
        }
        a.drawWorldToScreen(grid, actionPosition, common.Cell{Rune: r, Style: style})
    }
    frameDelayInSeconds := 0.5
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }
    audio := a.engine.GetAudio()
    handle := audio.PlayCueAt(audioCue, actionPosition)
    onFinish := func() {
        handle.Close()
        finishedCallback()
    }
    onCancel := func() {
        handle.Close()
        cancelCallback()
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       0,
        CurrentFrame:     0,
        ID:               0,
        Update:           nil,
        FinishCondition:  SoundStopped(handle),
        FinishedCallback: onFinish,
        CancelCondition:  ActorMovedOrIncapacitated(person),
        CancelCallback:   onCancel,
    }
    a.addAnimation(animation)
}

func (a *Animator) FoodAnimation(person *core.Actor, actionPosition geometry.Point, completed func()) {
    a.engagedWithSoundAnimation(person, actionPosition, "eating", 'm', core.CurrentTheme.EmeticPoisonForeground, completed, completed)
}

func (a *Animator) FallingAnimation(actionPosition geometry.Point, completed func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(actionPosition) {
            return
        }
        cellAt := a.cellOnScreen(grid, actionPosition)
        style := cellAt.Style
        if frameIndex%2 == 0 {
            style.Foreground = core.CurrentTheme.DangerForeground
        }
        a.drawWorldToScreen(grid, actionPosition, common.Cell{Rune: 'f', Style: style})
    }
    frameDelayInSeconds := 0.5
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       int(2 / frameDelayInSeconds),
        CurrentFrame:     0,
        ID:               0,
        Update:           nil,
        FinishCondition:  nil,
        FinishedCallback: completed,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}
func (a *Animator) TaskAnimation(person *core.Actor, timeInSeconds float64, lookDirs []float64, cancelCallback func(), finishedCallback func()) {
    updateFunc := func(frameIndex int) {
        a.engine.GetAI().UpdateVision(person)
    }
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(person.Pos()) {
            return
        }
        cellAt := a.cellOnScreen(grid, person.Pos())
        style := cellAt.Style
        if frameIndex%2 == 0 {
            style.Background = core.CurrentTheme.EngagedInTaskBackground
        }
        a.drawWorldToScreen(grid, person.Pos(), common.Cell{Rune: cellAt.Rune, Style: style})
    }
    frameDelay := uint64(utils.SecondsToTicks(0.5))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    finish := func() {
        if finishedCallback != nil {
            finishedCallback()
        }
    }

    animation := &ActiveAnimation{
        Update:           updateFunc,
        DrawFrame:        drawFunc,
        nextFrame:        nextFrameFunc,
        frameCount:       int(timeInSeconds * 2),
        FinishCondition:  nil,
        FinishedCallback: finish,
        CancelCondition:  ActorMovedOrStateChanged(person),
        CancelCallback:   cancelCallback,
    }
    a.addAnimation(animation)

    if len(lookDirs) > 0 {
        a.addAnimation(a.addLookDirectionsSweepAnimation(person, lookDirs, timeInSeconds))
    }
}

// addLookDirectionsSweepAnimation builds a rotation animation that sweeps the
// actor through every direction in lookDirs (in order) and then back to the
// initial look direction, all within timeInSeconds.
//
// With N directions the animation has N+1 equal segments:
//
//	startDir → lookDirs[0] → lookDirs[1] → … → lookDirs[N-1] → startDir
func (a *Animator) addLookDirectionsSweepAnimation(person *core.Actor, lookDirs []float64, timeInSeconds float64) *ActiveAnimation {
    startDir := person.LookDirection

    // Build the waypoint sequence.
    waypoints := make([]float64, 0, len(lookDirs)+2)
    waypoints = append(waypoints, startDir)
    waypoints = append(waypoints, lookDirs...)
    waypoints = append(waypoints, startDir)

    // Pre-compute the shortest-arc signed delta for every consecutive pair.
    deltas := make([]float64, len(waypoints)-1)
    for i := range deltas {
        d := waypoints[i+1] - waypoints[i]
        for d > 180 {
            d -= 360
        }
        for d <= -180 {
            d += 360
        }
        deltas[i] = d
    }

    normAngle := func(angle float64) float64 {
        for angle < 0 {
            angle += 360
        }
        for angle >= 360 {
            angle -= 360
        }
        return angle
    }

    totalFrames := utils.SecondsToTicks(timeInSeconds)
    segCount := len(deltas) // = len(lookDirs) + 1

    rotUpdate := func(frame int) {
        if totalFrames <= 0 || segCount <= 0 {
            return
        }
        // Map frame → segment using uniform integer division so every segment
        // gets its fair share even when totalFrames % segCount != 0.
        seg := frame * segCount / totalFrames
        if seg >= segCount {
            seg = segCount - 1
        }
        segStart := seg * totalFrames / segCount
        segEnd := (seg + 1) * totalFrames / segCount
        if segEnd > totalFrames {
            segEnd = totalFrames
        }
        var t float64
        if segLen := segEnd - segStart; segLen > 0 {
            t = float64(frame-segStart) / float64(segLen)
        }
        person.LookDirection = normAngle(waypoints[seg] + deltas[seg]*t)
        a.engine.GetAI().UpdateVision(person)
    }

    restore := func() { person.LookDirection = startDir }

    return &ActiveAnimation{
        Update:           rotUpdate,
        nextFrame:        func(_ int, ticksAlive uint64) bool { return ticksAlive >= 1 },
        frameCount:       totalFrames,
        FinishedCallback: restore,
        CancelCondition:  ActorMovedOrStateChanged(person),
        CancelCallback:   restore,
    }
}

func (a *Animator) VomitingAnimation(person *core.Actor, actionPosition geometry.Point, completed func()) {
    a.engagedWithSoundAnimation(person, actionPosition, "vomiting", '&', core.CurrentTheme.EmeticPoisonForeground, completed, completed)
}

func (a *Animator) SleepingAnimation(person *core.Actor, finishedCallback func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        cellAt := a.cellOnScreen(grid, person.Pos())
        style := cellAt.Style
        glyph := 'Z'
        if frameIndex%2 == 0 {
            style.Foreground = core.CurrentTheme.SleepPoisonForeground
            glyph = 'z'
        }
        a.drawWorldToScreen(grid, person.Pos(), common.Cell{Rune: glyph, Style: style})
    }
    frameDelay := uint64(utils.SecondsToTicks(1))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       10,
        Update:           nil,
        FinishCondition:  nil,
        FinishedCallback: finishedCallback,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}

func (a *Animator) engagedWithSoundAnimation(person *core.Actor, animPosition geometry.Point, audioCue string, icon rune, fgColor common.Color, finishedCallback, cancelCallback func()) {
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        game := a.engine.GetGame()
        if !game.IsOnScreen(animPosition) {
            return
        }
        cellAt := a.cellOnScreen(grid, animPosition)
        style := cellAt.Style
        if frameIndex%2 == 0 {
            style.Foreground = fgColor
        }
        a.drawWorldToScreen(grid, animPosition, common.Cell{Rune: icon, Style: style})
    }
    frameDelay := uint64(utils.SecondsToTicks(0.5))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    audio := a.engine.GetAudio()
    handle := audio.PlayCueAt(audioCue, animPosition)
    onFinish := func() {
        handle.Close()
        finishedCallback()
    }
    onCancel := func() {
        handle.Close()
        cancelCallback()
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        FinishCondition:  SoundStopped(handle),
        FinishedCallback: onFinish,
        CancelCondition:  ActorMovedOrIncapacitated(person),
        CancelCallback:   onCancel,
    }
    a.addAnimation(animation)
}

func (a *Animator) ElectricityAnimation(tiles []geometry.Point, source core.EffectSource, stim stimuli.Stimulus) {
    updateFunc := func(frameIndex int) {
        if frameIndex == 5 {
            for _, tile := range tiles {
                a.engine.GetGame().ApplyStimulusToThings(tile, source, stim)
            }
        }
    }
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        if frameIndex%2 == 0 {
            for _, tile := range tiles {
                game := a.engine.GetGame()
                if !game.IsOnScreen(tile) {
                    continue
                }
                cellAt := a.cellOnScreen(grid, tile)
                style := cellAt.Style
                style.Foreground = core.CurrentTheme.ElectricityForeground
                a.drawWorldToScreen(grid, tile, common.Cell{Rune: core.GlyphElectric, Style: style})
            }
        }
    }
    frameDelay := uint64(utils.SecondsToTicks(0.05))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       10,
        Update:           updateFunc,
        FinishedCallback: nil,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}

func (a *Animator) SoundPropagationAnimation(sound core.Observation, tiles map[int][]geometry.Point, completed func()) {
    game := a.engine.GetGame()

    drawFunc := func(grid console.CellInterface, frameIndex int) {
        if sound.IsSpeech() {
            return
        }
        tilesAtRange := tiles[frameIndex]
        for _, point := range tilesAtRange {
            if !game.IsOnScreen(point) {
                continue
            }
            cellAt := a.cellOnScreen(grid, point)
            a.drawWorldToScreen(grid, point, cellAt.WithBackgroundColor(cellAt.Style.Background.Lerp(common.White, 0.1)))
        }
    }
    frameDelay := uint64(utils.SecondsToTicks(0.075))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       len(tiles),
        Update:           nil,
        FinishedCallback: completed,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}

func (a *Animator) BlastDistribution(location geometry.Point, source core.EffectSource, applyStim []stimuli.Stimulus, size int, pressure int) {
    distributedEffect := stimuli.StimEffect{Stimuli: applyStim}
    gridmap := a.engine.GetGame().GetMap()
    a.engine.GetGame().Apply(location, source, distributedEffect)
    animationTiles := gridmap.WavePropagationFrom(location, size, pressure)
    updateFunc := func(frameIndex int) {
        for _, point := range animationTiles[frameIndex] {
            a.engine.GetGame().Apply(point, source, distributedEffect)
        }
    }
    drawFunc := func(grid console.CellInterface, frameIndex int) {
        if frameIndex == 0 || frameIndex == 1 {
            a.drawWorldToScreen(grid, location, common.Cell{Rune: '*', Style: common.Style{Foreground: core.CurrentTheme.ExplosionForeground, Background: core.CurrentTheme.ExplosionBackground}})
        }
        tilesAtRange := animationTiles[frameIndex]
        for _, point := range tilesAtRange {
            game := a.engine.GetGame()
            if !game.IsOnScreen(point) {
                continue
            }
            explosionStyle := common.Style{Foreground: core.CurrentTheme.ExplosionForeground, Background: core.CurrentTheme.ExplosionBackground}
            explodingRune := '*'
            if frameIndex%2 == 0 {
                explosionStyle = common.Style{Foreground: core.CurrentTheme.ExplosionBackground, Background: core.CurrentTheme.ExplosionForeground}
                cellAt := a.cellOnScreen(grid, point)
                explodingRune = cellAt.Rune
            }
            newCell := common.Cell{Rune: explodingRune, Style: explosionStyle}
            a.drawWorldToScreen(grid, point, newCell)
        }
    }
    frameDelay := uint64(utils.SecondsToTicks(0.1))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }

    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       len(animationTiles),
        Update:           updateFunc,
        FinishedCallback: nil,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}
func (a *Animator) LiquidDistribution(location geometry.Point, source core.EffectSource, applyStim []stimuli.Stimulus, size int) {
    distributedEffect := stimuli.StimEffect{Stimuli: applyStim}
    gridmap := a.engine.GetGame().GetMap()
    useForSpill := func(p geometry.Point) bool {
        return gridmap.IsTileWalkable(p)
    }
    spillingCells := gridmap.GetFreeCellsForDistribution(location, size, useForSpill)
    a.engine.GetGame().Apply(location, source, distributedEffect)
    updateFunc := func(frameIndex int) {
        if len(spillingCells) > 0 {
            nextCell := spillingCells[0]
            spillingCells = spillingCells[1:]
            a.engine.GetGame().Apply(nextCell, source, distributedEffect)
        }
    }
    drawFunc := func(grid console.CellInterface, frameIndex int) {}
    frameDelay := uint64(utils.SecondsToTicks(0.6))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelay
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       len(spillingCells),
        Update:           updateFunc,
        FinishedCallback: nil,
        FinishCondition:  nil,
        CancelCondition:  nil,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}

// GasDistribution deploys a gas cloud that marks all reachable tiles within radius
// with the given stimulus. Any actor already on a covered tile is affected immediately;
// any actor who later walks into the cloud is affected via the normal ReEmitStimuliOnTileToThings
// hook in ActorEnteredCell. Each tile dissipates independently at a randomised time so
// the cloud dissolves gradually rather than all at once.
func (a *Animator) GasDistribution(location geometry.Point, source core.EffectSource, stims []stimuli.Stimulus, radius int, durationSecs int) {
    game := a.engine.GetGame()
    gmap := game.GetMap()

    // Collect tiles grouped by wave distance so outer tiles can bias towards
    // clearing sooner, giving a natural outward-dissipation feel.
    waveMap := gmap.WavePropagationFrom(location, radius, 0)

    directEffect := stimuli.StimEffect{Stimuli: stims}

    base := float64(durationSecs)
    // Jitter window: each tile's removal is scheduled anywhere between
    // 0.5× and 1.5× the base duration.
    jitter := base * 0.5

    scheduleTileRemoval := func(p geometry.Point, distanceBias float64) {
        // distanceBias nudges outer tiles to clear sooner (max −25% at full radius).
        delay := base - distanceBias + (rand.Float64()*jitter*2 - jitter)
        if delay < base*0.25 {
            delay = base * 0.25 // never clear in less than a quarter of base time
        }
        a.engine.Schedule(delay, func() {
            for _, s := range stims {
                gmap.RemoveStimulusFromTile(p, s.Type())
            }
        })
    }

    // Apply gas to the origin tile.
    for _, s := range stims {
        gmap.AddStimulusToTile(location, s)
    }
    game.Apply(location, source, directEffect)
    scheduleTileRemoval(location, 0)

    // Apply gas to all wave tiles; outer tiles get a small bias to clear first.
    maxBias := base * 0.25
    for dist, tiles := range waveMap {
        distanceBias := float64(dist) / float64(radius) * maxBias
        for _, p := range tiles {
            for _, s := range stims {
                gmap.AddStimulusToTile(p, s)
            }
            game.Apply(p, source, directEffect)
            scheduleTileRemoval(p, distanceBias)
        }
    }
}

func (a *Animator) BriefingAnimation(script *core.BriefingAnimation, finish func()) {
    audio := a.engine.GetAudio()
    files := a.engine.GetFiles()
    currentMap := a.engine.GetGame().GetMap()
    audioFileList := make([]string, 0)
    for _, slide := range script.Slides {
        if slide.AudioFile != "" {
            audioFilePath := path.Join(currentMap.MapFileName(), "briefing", slide.AudioFile+".ogg")
            audioFileList = append(audioFileList, audioFilePath)
        }
    }
    audio.RegisterSoundCues(audioFileList)

    var handle services.AudioHandle
    updateFunc := func(frameIndex int) {
        currentSlide := script.Slides[frameIndex]
        if currentSlide.AudioFile != "" {
            handle = audio.PlayCue(currentSlide.AudioFile)
        }
    }
    gridWidth, gridHeight := a.engine.ScreenGridWidth(), a.engine.ScreenGridHeight()

    drawFunc := func(con console.CellInterface, frameIndex int) {
        con.ClearConsole()
        currentFrame := script.Slides[frameIndex]

        // draw image
        imagePath := path.Join(currentMap.MapFileName(), "briefing", currentFrame.ImageFile+".cmg")
        image := utils.LoadCellImageFromDisk(files, imagePath)
        image.DrawCentered(con)

        if frameIndex == 0 {
            core.DrawStyledTextAlignedInsideRect(con, geometry.NewRect(0, 1, gridWidth*2, 2), core.AlignCenter, currentFrame.Text)
        } else {
            core.DrawStyledTextAlignedInsideRect(con, geometry.NewRect(0, gridHeight-8, gridWidth*2, gridHeight-1), core.AlignCenter, currentFrame.Text)
        }
    }

    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        currentSlide := script.Slides[frameIndex]
        if currentSlide.AudioFile == "" {
            return ticksAlive >= uint64(utils.SecondsToTicks(3)) // TODO: make the delay depend on the length of the text
        }
        return handle != nil && !handle.IsPlaying()
    }

    onFinish := func() {
        if handle != nil {
            handle.Close()
        }
        finish()
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       len(script.Slides),
        Update:           updateFunc,
        FinishedCallback: onFinish,
        FinishCondition:  nil,
        CancelCondition:  a.ConfirmOrCancel,
        CancelCallback:   onFinish,
    }
    a.addAnimation(animation)
}

func (a *Animator) ImageFadeIn(pixels [][]common.Color, cancel, finish func()) {
    frameCount := 500
    gridWidth, gridHeight := a.engine.ScreenGridWidth(), a.engine.ScreenGridHeight()
    drawFunc := func(con console.CellInterface, frameIndex int) {
        completionInPercent := common.Clamp(float64(frameIndex-100)/float64(frameCount-200), 0.0, 1.0)
        con.HalfWidthTransparent()
        for y := 0; y < gridHeight; y++ {
            for x := 0; x < gridWidth; x++ {
                pos := geometry.Point{X: x, Y: y}
                pixel := pixels[y][x]
                pixel = pixel.MultiplyWithScalar(completionInPercent) // 0.0 --> 1.0
                con.SetSquare(pos, common.Cell{Rune: ' ', Style: common.Style{Background: pixel}})
            }
        }
    }
    frameDelayInSeconds := 1.0 / float64(ebiten.TPS())
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       frameCount,
        Update:           nil,
        FinishedCallback: finish,
        FinishCondition:  nil,
        CancelCondition:  a.ConfirmOrCancel,
        CancelCallback:   cancel,
    }
    a.addAnimation(animation)
}

func (a *Animator) ImageToImageFade(src, dest [][]common.Color, draw, cancel, finish func()) {
    frameCount := 500
    gridWidth, gridHeight := a.engine.ScreenGridWidth(), a.engine.ScreenGridHeight()
    drawFunc := func(con console.CellInterface, frameIndex int) {
        completionInPercent := common.Clamp(float64(frameIndex)/float64(frameCount), 0.0, 1.0)
        con.HalfWidthTransparent()
        for y := 0; y < gridHeight; y++ {
            for x := 0; x < gridWidth; x++ {
                pos := geometry.Point{X: x, Y: y}
                srcPixel := src[y][x]
                destPixel := dest[y][x]
                if srcPixel == destPixel {
                    con.SetSquare(pos, common.Cell{Rune: ' ', Style: common.Style{Background: srcPixel}})
                    continue
                }
                pixel := srcPixel.Lerp(destPixel, completionInPercent)
                //pixel = pixel.MultiplyWithScalar(completionInPercent) // 0.0 --> 1.0
                con.SetSquare(pos, common.Cell{Rune: ' ', Style: common.Style{Background: pixel}})
            }
        }
        if draw != nil {
            draw()
        }
    }
    frameDelayInSeconds := 1.0 / float64(ebiten.TPS())
    frameDelayInTicks := uint64(utils.SecondsToTicks(frameDelayInSeconds))
    nextFrameFunc := func(frameIndex int, ticksAlive uint64) bool {
        return ticksAlive >= frameDelayInTicks
    }
    stopFunc := func() bool {
        if a.ConfirmOrCancel() {
            if cancel != nil {
                cancel()
            }
            return true
        }
        return false
    }
    animation := &ActiveAnimation{
        nextFrame:        nextFrameFunc,
        DrawFrame:        drawFunc,
        frameCount:       frameCount,
        Update:           nil,
        FinishedCallback: finish,
        FinishCondition:  nil,
        CancelCondition:  stopFunc,
        CancelCallback:   nil,
    }
    a.addAnimation(animation)
}
