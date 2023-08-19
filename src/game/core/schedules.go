package core

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/geometry"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type ScheduledTask struct {
	Location          geometry.Point
	Description       string
	DurationInSeconds float64
	KnownPath         []geometry.Point
}

func (t ScheduledTask) ToString() string {
	return fmt.Sprintf("Location%s Duration(%.2f) -> %s", t.Location.String(), t.DurationInSeconds, t.Description)
}

func (t ScheduledTask) WithLocationShifted(offset geometry.Point, mapSize geometry.Point) ScheduledTask {
	t.Location = t.Location.AddWrapped(offset, mapSize)
	t.KnownPath = []geometry.Point{}
	return t
}

func NewTaskFromString(str string) ScheduledTask {
	var task ScheduledTask
	var location string
	fmt.Sscanf(str, "Location%s Duration(%f) -> %s", &location, &task.DurationInSeconds, &task.Description)
	task.Location, _ = geometry.NewPointFromString(location)
	return task
}

type Schedule struct {
	Tasks            []ScheduledTask
	CurrentTaskIndex int
}

func (s *Schedule) CurrentTask() ScheduledTask {
	return s.Tasks[s.CurrentTaskIndex]
}
func (s *Schedule) NextTask() {
	s.CurrentTaskIndex = (s.CurrentTaskIndex + 1) % len(s.Tasks)
}
func (s *Schedule) RemoveTask(index int) {
	s.Tasks = append(s.Tasks[:index], s.Tasks[index+1:]...)
}

func (s *Schedule) Clear() {
	s.Tasks = []ScheduledTask{}
	s.CurrentTaskIndex = 0
}

func (s *Schedule) ToRecord(pos geometry.Point) []rec_files.Field {
	var fields []rec_files.Field
	if len(s.Tasks) == 0 {
		return fields
	}
	fields = append(fields, rec_files.Field{Name: "ForActorAt", Value: pos.String()})
	for _, task := range s.Tasks {
		fields = append(fields, rec_files.Field{Name: "Task", Value: task.ToString()})
	}
	return fields
}

func ScheduleFromRecord(fields []rec_files.Field) *Schedule {
	var schedule Schedule
	for _, field := range fields {
		if field.Name == "ForActorAt" {
			continue
		}
		if field.Name == "Task" {
			schedule.Tasks = append(schedule.Tasks, NewTaskFromString(field.Value))
		}
	}
	return &schedule
}
