package editor

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
)

func (g *GameStateEditor) addTask() {
	aic := g.engine.GetAI()
	p := g.MousePositionInWorld
	newTask := core.ScheduledTask{Location: p, Description: "Dummy Task #1", DurationInSeconds: 10, KnownPath: make([]geometry.Point, 0)}
	aic.AddTask(g.SelectedActor, newTask)
	taskCount := aic.TaskCountFor(g.SelectedActor)
	g.SelectedTaskIndex = taskCount - 1
	g.LastSelectedPos = p
	aic.CalculateAllTaskPaths(g.SelectedActor)
	g.changeUIStateTo(editTaskUI)
}

func (g *GameStateEditor) increaseTaskTime() {
	if g.SelectedActor == nil {
		g.PrintAsMessage("ERR: select an Actor first")
		return
	}
	if g.SelectedTaskIndex == -1 {
		g.PrintAsMessage("ERR: select a Task first")
		return
	}
	g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex].DurationInSeconds += 1
	g.printTask(g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex])
}

func (g *GameStateEditor) decreaseTaskTime() {
	if g.SelectedActor == nil {
		g.PrintAsMessage("ERR: select an Actor first")
		return
	}
	if g.SelectedTaskIndex == -1 {
		g.PrintAsMessage("ERR: select a Task first")
		return
	}
	g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex].DurationInSeconds -= 1
	if g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex].DurationInSeconds < 0 {
		g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex].DurationInSeconds = 0
	}
	g.printTask(g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex])
}

func (g *GameStateEditor) deleteTask() {
	g.SelectedActor.AI.Schedule.RemoveTask(g.SelectedTaskIndex)
	g.SelectedTaskIndex = -1
	g.engine.GetAI().CalculateAllTaskPaths(g.SelectedActor)
}

func (g *GameStateEditor) selectTaskAt(pos geometry.Point) {
	g.SelectedTaskIndex = g.SelectedActor.AI.GetTaskIndexAt(pos)
	g.LastSelectedPos = pos
	task := g.SelectedActor.AI.Schedule.Tasks[g.SelectedTaskIndex]
	g.printTask(task)
}

func (g *GameStateEditor) printTask(task core.ScheduledTask) {
	g.PrintAsMessage(fmt.Sprintf("%d. Task of %s: %d - %s", g.SelectedTaskIndex, g.SelectedActor.Name, int(task.DurationInSeconds), task.Description))
}
