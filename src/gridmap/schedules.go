package gridmap

import (
	"fmt"

	"github.com/memmaker/terminal-assassin/geometry"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

type ScheduledTask struct {
	Location          geometry.Point
	DurationInSeconds float64
	// LookDirections holds 0..n target bearings in degrees [0, 360) the actor
	// should sweep through while dwelling at this waypoint.
	LookDirections []float64
	KnownPath      []geometry.Point
}

func (t ScheduledTask) ToRecord(scheduleName string) rec_files.Record {
	fields := rec_files.Record{
		{Name: "TaskForSchedule", Value: scheduleName},
		{Name: "Location", Value: t.Location.String()},
		{Name: "Duration", Value: fmt.Sprintf("(%.2f)", t.DurationInSeconds)},
	}
	for _, dir := range t.LookDirections {
		fields = append(fields, rec_files.Field{Name: "LookDir", Value: fmt.Sprintf("(%.2f)", dir)})
	}
	return fields
}

func taskFromRecord(record rec_files.Record) ScheduledTask {
	var task ScheduledTask
	task.LookDirections = []float64{}
	for _, field := range record {
		switch field.Name {
		case "Location":
			task.Location, _ = geometry.NewPointFromString(field.Value)
		case "Duration":
			fmt.Sscanf(field.Value, "(%f)", &task.DurationInSeconds)
		case "LookDir":
			var dir float64
			fmt.Sscanf(field.Value, "(%f)", &dir)
			task.LookDirections = append(task.LookDirections, dir)
		}
	}
	return task
}

func (t ScheduledTask) WithLocationShifted(offset geometry.Point, mapSize geometry.Point) ScheduledTask {
	t.Location = t.Location.AddWrapped(offset, mapSize)
	t.KnownPath = []geometry.Point{}
	return t
}

type Schedule struct {
	Name  string
	Tasks []ScheduledTask
}

// TaskAt returns the task at the given index, wrapping if needed.
func (s *Schedule) TaskAt(index int) ScheduledTask {
	return s.Tasks[index%len(s.Tasks)]
}

// NextIndex returns the index after current, wrapping around.
func (s *Schedule) NextIndex(current int) int {
	return (current + 1) % len(s.Tasks)
}

func (s *Schedule) RemoveTask(index int) {
	s.Tasks = append(s.Tasks[:index], s.Tasks[index+1:]...)
}

func (s *Schedule) Clear() {
	s.Tasks = []ScheduledTask{}
}

// ToRecords serialises the schedule as one rec-file record per task.
func (s *Schedule) ToRecords() []rec_files.Record {
	records := make([]rec_files.Record, len(s.Tasks))
	for i, task := range s.Tasks {
		records[i] = task.ToRecord(s.Name)
	}
	return records
}

// ToActorLinkRecord serialises the actor→schedule mapping for actor_schedules.txt.
func (s *Schedule) ToActorLinkRecord(actorName string) []rec_files.Field {
	return []rec_files.Field{
		{Name: "ForActorWithName", Value: actorName},
		{Name: "StartSchedule", Value: s.Name},
	}
}

func (s *Schedule) ShiftBy(offset geometry.Point, mapSize geometry.Point) {
	for i, task := range s.Tasks {
		s.Tasks[i] = task.WithLocationShifted(offset, mapSize)
	}
}

// SchedulesFromTaskRecords rebuilds all named Schedules from a schedules.txt
// record slice (one record per task).
func SchedulesFromTaskRecords(records []rec_files.Record) []*Schedule {
	byName := make(map[string]*Schedule)
	var order []string
	for _, record := range records {
		m := rec_files.Record(record).ToMap()
		name := m["TaskForSchedule"]
		if name == "" {
			continue
		}
		if _, exists := byName[name]; !exists {
			byName[name] = &Schedule{Name: name, Tasks: make([]ScheduledTask, 0)}
			order = append(order, name)
		}
		byName[name].Tasks = append(byName[name].Tasks, taskFromRecord(record))
	}
	result := make([]*Schedule, len(order))
	for i, name := range order {
		result[i] = byName[name]
	}
	return result
}

// ActorScheduleLinkFromRecord extracts the actor name and starting schedule name
// from an actor_schedules.txt record.
func ActorScheduleLinkFromRecord(fields []rec_files.Field) (actorName, scheduleName string) {
	m := rec_files.Record(fields).ToMap()
	return m["ForActorWithName"], m["StartSchedule"]
}
