package editor

import (
    "fmt"
    "strings"

    "github.com/memmaker/terminal-assassin/game/services"
    "github.com/memmaker/terminal-assassin/geometry"
    "github.com/memmaker/terminal-assassin/gridmap"
)

// ── schedule library ──────────────────────────────────────────────────────────

// openScheduleLibraryMenu shows all named schedules in the model library
// and allows the user to select one to edit or create a new one.
// No actor needs to be selected.
func (g *GameStateEditor) openScheduleLibraryMenu() {
    schedules := g.engine.GetGame().GetMap().ListOfSchedules()
    menuItems := []services.MenuItem{
        {
            Label:    "New Schedule",
            Handler:  g.createNewSchedule,
            Icon:     '+',
            QuickKey: "n",
        },
    }
    for _, sched := range schedules {
        s := sched // capture
        menuItems = append(menuItems, services.MenuItem{
            Label: fmt.Sprintf("%s (%d tasks)", s.Name, len(s.Tasks)),
            Handler: func() {
                g.SelectedSchedule = s
                g.SelectedTaskIndex = -1
                g.PrintAsMessage(fmt.Sprintf("Editing schedule: %s", s.Name))
                g.changeUIStateTo(editScheduleUI)
            },
            Icon: 'S',
        })
    }
	g.OpenMenuBarDropDown("Schedules", (2*6)-2, menuItems)
}

// createNewSchedule prompts for a name
func (g *GameStateEditor) createNewSchedule() {
    g.engine.GetUI().ShowTextInput("Schedule name: ", "", func(name string) {
        if name == "" {
            g.PrintAsMessage("ERR: schedule name cannot be empty")
            return
        }
        currentMap := g.engine.GetGame().GetMap()
        if currentMap.GetSchedule(name) != nil {
            g.PrintAsMessage(fmt.Sprintf("ERR: schedule '%s' already exists", name))
            return
        }
        newSched := &gridmap.Schedule{Name: name, Tasks: make([]gridmap.ScheduledTask, 0)}
        currentMap.AddSchedule(newSched)
        g.SelectedSchedule = newSched
        g.SelectedTaskIndex = -1
        g.PrintAsMessage(fmt.Sprintf("Created schedule: %s", name))
        g.changeUIStateTo(editScheduleUI)
    }, func() {
        g.PrintAsMessage("Cancelled")
    })
}

// renameSelectedSchedule prompts for a new name and renames the currently selected schedule.
func (g *GameStateEditor) renameSelectedSchedule() {
    if g.SelectedSchedule == nil {
        g.PrintAsMessage("ERR: no schedule selected")
        return
    }
    oldName := g.SelectedSchedule.Name
    g.engine.GetUI().ShowTextInput("New schedule name: ", oldName, func(newName string) {
        if newName == "" || newName == oldName {
            g.changeUIStateTo(editScheduleUI)
            return
        }
        currentMap := g.engine.GetGame().GetMap()
        currentMap.RemoveSchedule(oldName)
        g.SelectedSchedule.Name = newName
        currentMap.AddSchedule(g.SelectedSchedule)
        // Keep actor references in sync (actors store the schedule name as a string).
        for _, actor := range currentMap.Actors() {
            if actor.AI.Schedule == oldName {
                actor.AI.Schedule = newName
            }
        }
        g.PrintAsMessage(fmt.Sprintf("Renamed to: %s", newName))
        g.changeUIStateTo(editScheduleUI)
    }, func() {
        g.changeUIStateTo(editScheduleUI)
    })
}

// deleteSelectedSchedule removes the selected schedule from the library.
func (g *GameStateEditor) deleteSelectedSchedule() {
    if g.SelectedSchedule == nil {
        return
    }
    name := g.SelectedSchedule.Name
    g.engine.GetGame().GetMap().RemoveSchedule(name)
    g.PrintAsMessage(fmt.Sprintf("Deleted schedule: %s", name))
    g.SelectedSchedule = nil
    g.SelectedTaskIndex = -1
    g.changeUIStateTo(editMapUI)
}

// assignScheduleToCurrentActor assigns the selected schedule to the selected actor.
func (g *GameStateEditor) assignScheduleToCurrentActor() {
    if g.SelectedSchedule == nil {
        g.PrintAsMessage("ERR: no schedule selected")
        return
    }
    if g.SelectedActor == nil {
        g.PrintAsMessage("ERR: no actor selected — switch to Actors tab and click an actor first")
        return
    }
    g.SelectedActor.AI.Schedule = g.SelectedSchedule.Name
    g.SelectedActor.AI.CurrentTaskIndex = 0
    g.engine.GetAI().CalculateAllTaskPaths(g.SelectedActor)
    g.PrintAsMessage(fmt.Sprintf("Assigned schedule '%s' to %s", g.SelectedSchedule.Name, g.SelectedActor.Name))
}

// assignScheduleFromLibraryToSelectedActor opens the library picker and assigns
// the chosen schedule to the currently selected actor. Called from actor edit menu.
func (g *GameStateEditor) assignScheduleFromLibraryToSelectedActor() {
    if g.SelectedActor == nil {
        g.PrintAsMessage("ERR: select an Actor first")
        return
    }
    schedules := g.engine.GetGame().GetMap().ListOfSchedules()
    if len(schedules) == 0 {
        g.PrintAsMessage("No schedules in library yet — create one via the Schedule tab (F5)")
        return
    }
    menuItems := make([]services.MenuItem, 0, len(schedules))
    for _, sched := range schedules {
        s := sched
        menuItems = append(menuItems, services.MenuItem{
            Label: fmt.Sprintf("%s (%d tasks)", s.Name, len(s.Tasks)),
            Handler: func() {
                g.SelectedActor.AI.Schedule = s.Name
                g.SelectedActor.AI.CurrentTaskIndex = 0
                g.engine.GetAI().CalculateAllTaskPaths(g.SelectedActor)
                g.PrintAsMessage(fmt.Sprintf("Assigned '%s' to %s", s.Name, g.SelectedActor.Name))
            },
            Icon: 'S',
        })
    }
	g.engine.GetUI().OpenFixedWidthAutoCloseMenu("Schedules", menuItems)
}

// ── task editing ──────────────

// addTask places a new task at the current mouse position into the selected schedule.
func (g *GameStateEditor) addTask() {
    if g.SelectedSchedule == nil {
        g.PrintAsMessage("ERR: select a Schedule first (F5)")
        return
    }
    p := g.MousePositionInWorld
    newTask := gridmap.ScheduledTask{
        Location:          p,
        DurationInSeconds: 10,
        LookDirections:    []float64{},
        KnownPath:         make([]geometry.Point, 0),
    }
    g.SelectedSchedule.Tasks = append(g.SelectedSchedule.Tasks, newTask)
    g.SelectedTaskIndex = len(g.SelectedSchedule.Tasks) - 1
    g.LastSelectedPos = p
    g.recalcPathsForScheduleUsers()
    g.recomputeTaskPreviewFov()
    g.changeUIStateTo(editTaskUI)
}

func (g *GameStateEditor) increaseTaskTime() {
    if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 {
        g.PrintAsMessage("ERR: select a Schedule and Task first")
        return
    }
    g.SelectedSchedule.Tasks[g.SelectedTaskIndex].DurationInSeconds += 1
    g.printTask(g.SelectedSchedule.Tasks[g.SelectedTaskIndex])
}

func (g *GameStateEditor) decreaseTaskTime() {
    if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 {
        g.PrintAsMessage("ERR: select a Schedule and Task first")
        return
    }
    t := &g.SelectedSchedule.Tasks[g.SelectedTaskIndex]
    t.DurationInSeconds -= 1
    if t.DurationInSeconds < 0 {
        t.DurationInSeconds = 0
    }
    g.printTask(*t)
}

func (g *GameStateEditor) deleteTask() {
    if g.SelectedSchedule == nil {
        return
    }
    g.SelectedSchedule.RemoveTask(g.SelectedTaskIndex)
    g.SelectedTaskIndex = -1
    g.recalcPathsForScheduleUsers()
}

func (g *GameStateEditor) selectTaskAt(pos geometry.Point) {
    if g.SelectedSchedule == nil {
        return
    }
    for i, task := range g.SelectedSchedule.Tasks {
        if task.Location == pos {
            g.SelectedTaskIndex = i
            g.LastSelectedPos = pos
            g.recomputeTaskPreviewFov()
            g.printTask(task)
            return
        }
    }
}

func (g *GameStateEditor) printTask(task gridmap.ScheduledTask) {
    schedName := "(no schedule)"
    if g.SelectedSchedule != nil {
        schedName = g.SelectedSchedule.Name
    }
    lookInfo := "none"
    if len(task.LookDirections) > 0 {
        parts := make([]string, len(task.LookDirections))
        for i, dir := range task.LookDirections {
            parts[i] = fmt.Sprintf("%.0f°", dir)
        }
        lookInfo = strings.Join(parts, ", ")
    }
    g.PrintAsMessage(fmt.Sprintf("%d. Task of %s: %ds look:[%s]",
        g.SelectedTaskIndex+1, schedName, int(task.DurationInSeconds), lookInfo))
}

// addLookDirection enters a mouse-driven aiming mode.  Moving the mouse aims
// from the task tile toward the cursor; a left-click appends the angle to the
// task's LookDirections list.  Right-click cancels without adding.
func (g *GameStateEditor) addLookDirection() {
    if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 {
        g.PrintAsMessage("ERR: select a Task first")
        return
    }
    g.pendingLookDir = 0
    g.PrintAsMessage("Add look direction: move mouse to aim, click to confirm, right-click to cancel")
    g.handler = UIHandler{
        Name:       "add look direction",
        MouseMoved: g.updatePendingLookDirection,
        CellsSelected: func() {
            if g.pendingLookDir >= 0 {
                g.SelectedSchedule.Tasks[g.SelectedTaskIndex].LookDirections = append(
                    g.SelectedSchedule.Tasks[g.SelectedTaskIndex].LookDirections, g.pendingLookDir)
            }
            g.pendingLookDir = -1
            g.printTask(g.SelectedSchedule.Tasks[g.SelectedTaskIndex])
            g.changeUIStateTo(editTaskUI)
        },
        ContextMenu: []services.MenuItem{
            {
                Label: "Cancel",
                Handler: func() {
                    g.pendingLookDir = -1
                    g.changeUIStateTo(editTaskUI)
                },
            },
        },
    }
}

// removeLookDirection opens a menu listing the task's current look directions
// so the designer can pick one to delete.
func (g *GameStateEditor) removeLookDirection() {
    if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 ||
        g.SelectedTaskIndex >= len(g.SelectedSchedule.Tasks) {
        return
    }
    task := g.SelectedSchedule.Tasks[g.SelectedTaskIndex]
    if len(task.LookDirections) == 0 {
        g.PrintAsMessage("No look directions to remove")
        return
    }
    // Snapshot mutable editor state so handlers remain valid even if the
    // editor selection changes before the menu item is activated.
    sched := g.SelectedSchedule
    taskIdx := g.SelectedTaskIndex
    menuItems := make([]services.MenuItem, len(task.LookDirections))
    for i, dir := range task.LookDirections {
        idx, d := i, dir
        menuItems[i] = services.MenuItem{
            Label: fmt.Sprintf("%.0f°", d),
            Handler: func() {
                if taskIdx >= len(sched.Tasks) {
                    return
                }
                dirs := sched.Tasks[taskIdx].LookDirections
                if idx >= len(dirs) {
                    return
                }
                sched.Tasks[taskIdx].LookDirections = append(dirs[:idx:idx], dirs[idx+1:]...)
                g.printTask(sched.Tasks[taskIdx])
                g.changeUIStateTo(editTaskUI)
            },
        }
    }
    g.engine.GetUI().OpenFixedWidthAutoCloseMenu("Remove Look Direction", menuItems)
}

// updatePendingLookDirection recomputes pendingLookDir from the vector between
// the task tile and the current mouse position.
func (g *GameStateEditor) updatePendingLookDirection() {
    if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 {
        return
    }
    taskPos := g.SelectedSchedule.Tasks[g.SelectedTaskIndex].Location
    direction := g.MousePositionInWorld.Sub(taskPos)
    if direction.X == 0 && direction.Y == 0 {
        return
    }
    g.pendingLookDir = geometry.DirectionVectorToAngleInDegrees(direction)
    g.gridIsDirty = true
}

// recomputeTaskPreviewFov runs the shadow-casting FOV from the currently
// selected task's tile. Called once on task selection; the result is reused
// every frame by the draw code which only applies the cheap cone filter.
func (g *GameStateEditor) recomputeTaskPreviewFov() {
	if g.SelectedSchedule == nil || g.SelectedTaskIndex < 0 || g.SelectedTaskIndex >= len(g.SelectedSchedule.Tasks) {
		g.taskPreviewFov = nil
		return
	}
	currentMap := g.engine.GetGame().GetMap()
	visionRange := currentMap.MaxVisionRange
	taskPos := g.SelectedSchedule.Tasks[g.SelectedTaskIndex].Location

	fovRect := geometry.NewRect(
		-visionRange, -visionRange,
		visionRange+1, visionRange+1,
	).Add(taskPos).Intersect(
		geometry.NewRect(0, 0, currentMap.MapWidth, currentMap.MapHeight),
	)
	if g.taskPreviewFov == nil {
		g.taskPreviewFov = geometry.NewFOV(fovRect)
	} else {
		g.taskPreviewFov.SetRange(fovRect)
	}
	rangeSquared := visionRange * visionRange
	g.taskPreviewFov.SSCVisionMap(taskPos, visionRange, func(p geometry.Point) bool {
		return currentMap.IsTransparent(p) &&
			geometry.DistanceSquared(p, taskPos) <= rangeSquared
	}, false)
}

// recalcPathsForScheduleUsers recalculates task paths for every actor that
// currently references the selected schedule by name.
func (g *GameStateEditor) recalcPathsForScheduleUsers() {
    if g.SelectedSchedule == nil {
        return
    }
    aic := g.engine.GetAI()
    for _, actor := range g.engine.GetGame().GetMap().Actors() {
        if actor.AI.Schedule == g.SelectedSchedule.Name {
            aic.CalculateAllTaskPaths(actor)
        }
    }
}
