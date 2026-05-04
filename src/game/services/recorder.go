package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
	"github.com/memmaker/terminal-assassin/geometry"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

const ReplayDirectory = "replays"

// ReplayEntry is a single recorded input event, keyed to a WorldTick.
type ReplayEntry struct {
	Tick    uint64
	Command core.InputCommand
}

// ReplayFile holds the header data and all recorded input events.
type ReplayFile struct {
	MapPath      string
	MapHash      string
	Seed         int64
	DurationTicks uint64 // total ticks recorded; 0 = truncated/incomplete
	Entries      []ReplayEntry
}

// Recorder captures player inputs during a mission.
type Recorder struct {
	// ShouldRecord is toggled from the briefing menu. Gameplay reads it on Init.
	ShouldRecord bool

	recording  bool
	tickFunc   func() uint64 // returns current WorldTick
	mapPath    string
	mapHash    string
	seed       int64
	entries    []ReplayEntry
}

// SetTickSource wires the recorder to the engine's WorldTick counter.
func (r *Recorder) SetTickSource(f func() uint64) {
	r.tickFunc = f
}

// StartRecording begins capturing inputs with the given map metadata and RNG seed.
func (r *Recorder) StartRecording(mapPath, mapHash string, seed int64) {
	r.recording = true
	r.mapPath = mapPath
	r.mapHash = mapHash
	r.seed = seed
	r.entries = make([]ReplayEntry, 0, 256)
}

// IsRecording returns true while a recording is in progress.
func (r *Recorder) IsRecording() bool {
	return r.recording
}

// Record appends one command entry if recording is active.
func (r *Recorder) Record(cmd core.InputCommand) {
	if !r.recording || r.tickFunc == nil {
		return
	}
	r.entries = append(r.entries, ReplayEntry{
		Tick:    r.tickFunc(),
		Command: cmd,
	})
}

// RecordingProxy wraps an InputInterface and forwards polled commands to a Recorder.
type RecordingProxy struct {
	Inner    InputInterface
	Recorder *Recorder
}

func (p *RecordingProxy) PollGameCommands() []core.InputCommand {
	cmds := p.Inner.PollGameCommands()
	for _, cmd := range cmds {
		p.Recorder.Record(cmd)
	}
	return cmds
}
func (p *RecordingProxy) PollUICommands() []core.InputCommand {
	cmds := p.Inner.PollUICommands()
	for _, cmd := range cmds {
		p.Recorder.Record(cmd)
	}
	return cmds
}
func (p *RecordingProxy) PollEditorCommands() []core.InputCommand { return p.Inner.PollEditorCommands() }
func (p *RecordingProxy) ConfirmOrCancel() bool                   { return p.Inner.ConfirmOrCancel() }
func (p *RecordingProxy) DevTerminalKeyPressed() bool             { return p.Inner.DevTerminalKeyPressed() }
func (p *RecordingProxy) PollText() []core.InputCommand           { return p.Inner.PollText() }
func (p *RecordingProxy) GetKeyDefinitions() KeyDefinitions       { return p.Inner.GetKeyDefinitions() }
func (p *RecordingProxy) IsShiftPressed() bool                    { return p.Inner.IsShiftPressed() }
func (p *RecordingProxy) SetMovementDelayForSneaking()            { p.Inner.SetMovementDelayForSneaking() }
func (p *RecordingProxy) SetMovementDelayForWalkingAndRunning()   { p.Inner.SetMovementDelayForWalkingAndRunning() }

// StopAndSave ends the recording and writes a .rec file to ReplayDirectory.
// Returns the saved file path on success.
func (r *Recorder) StopAndSave() (string, error) {
	var durationTicks uint64
	if r.tickFunc != nil {
		durationTicks = r.tickFunc()
	}
	r.recording = false
	r.ShouldRecord = false

	if err := os.MkdirAll(ReplayDirectory, 0755); err != nil {
		return "", err
	}

	filename := filepath.Join(ReplayDirectory,
		fmt.Sprintf("replay_%s.rec", time.Now().Format("2006-01-02_15-04-05")))
	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	header := rec_files.Record{
		{Name: "MapPath",       Value: r.mapPath},
		{Name: "MapHash",       Value: r.mapHash},
		{Name: "Seed",          Value: strconv.FormatInt(r.seed, 10)},
		{Name: "DurationTicks", Value: strconv.FormatUint(durationTicks, 10)},
	}
	records := []rec_files.Record{header}
	for _, entry := range r.entries {
		records = append(records, encodeEntry(entry))
	}
	rec_files.Write(file, records)
	return filename, nil
}

// encodeEntry serialises one ReplayEntry into a rec-file Record.
func encodeEntry(e ReplayEntry) rec_files.Record {
	fields := rec_files.Record{
		{Name: "Tick", Value: strconv.FormatUint(e.Tick, 10)},
	}
	switch cmd := e.Command.(type) {
	case core.KeyCommand:
		fields = append(fields,
			rec_files.Field{Name: "Type", Value: "key"},
			rec_files.Field{Name: "Key", Value: string(cmd.Key)},
		)
	case core.GameCommand:
		fields = append(fields,
			rec_files.Field{Name: "Type", Value: "gamecommand"},
			rec_files.Field{Name: "Command", Value: strconv.Itoa(int(cmd))},
		)
	case core.DirectionalGameCommand:
		fields = append(fields,
			rec_files.Field{Name: "Type", Value: "directional"},
			rec_files.Field{Name: "DirCommand", Value: strconv.Itoa(int(cmd.Command))},
			rec_files.Field{Name: "XAxis", Value: strconv.FormatFloat(cmd.XAxis, 'f', 6, 64)},
			rec_files.Field{Name: "YAxis", Value: strconv.FormatFloat(cmd.YAxis, 'f', 6, 64)},
		)
	case core.PointerCommand:
		fields = append(fields,
			rec_files.Field{Name: "Type", Value: "pointer"},
			rec_files.Field{Name: "MouseState", Value: strconv.Itoa(int(cmd.Action))},
			rec_files.Field{Name: "PosX", Value: strconv.Itoa(cmd.Pos.X)},
			rec_files.Field{Name: "PosY", Value: strconv.Itoa(cmd.Pos.Y)},
		)
	}
	return fields
}

// LoadReplayFile reads and parses a .rec replay file from disk.
func LoadReplayFile(path string) (*ReplayFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records := rec_files.Read(file)
	if len(records) < 1 {
		return nil, fmt.Errorf("empty replay file")
	}

	header := records[0].ToMap()
	seed, _ := strconv.ParseInt(header["Seed"], 10, 64)
	durationTicks, _ := strconv.ParseUint(header["DurationTicks"], 10, 64)
	rf := &ReplayFile{
		MapPath:       header["MapPath"],
		MapHash:       header["MapHash"],
		Seed:          seed,
		DurationTicks: durationTicks,
		Entries:       make([]ReplayEntry, 0, len(records)-1),
	}

	for _, record := range records[1:] {
		if entry, ok := decodeEntry(record); ok {
			rf.Entries = append(rf.Entries, entry)
		}
	}
	return rf, nil
}

// decodeEntry deserialises one rec-file Record into a ReplayEntry.
func decodeEntry(record rec_files.Record) (ReplayEntry, bool) {
	m := record.ToMap()
	tick, err := strconv.ParseUint(m["Tick"], 10, 64)
	if err != nil {
		return ReplayEntry{}, false
	}
	entry := ReplayEntry{Tick: tick}

	switch m["Type"] {
	case "key":
		entry.Command = core.KeyCommand{Key: core.Key(m["Key"])}
	case "gamecommand":
		v, _ := strconv.Atoi(m["Command"])
		entry.Command = core.GameCommand(v)
	case "directional":
		dc, _ := strconv.Atoi(m["DirCommand"])
		x, _ := strconv.ParseFloat(m["XAxis"], 64)
		y, _ := strconv.ParseFloat(m["YAxis"], 64)
		entry.Command = core.DirectionalGameCommand{
			Command: core.GameDirectionalCommand(dc),
			XAxis:   x,
			YAxis:   y,
		}
	case "pointer":
		ms, _ := strconv.Atoi(m["MouseState"])
		px, _ := strconv.Atoi(m["PosX"])
		py, _ := strconv.Atoi(m["PosY"])
		entry.Command = core.PointerCommand{
			Action: core.MouseState(ms),
			Pos:    geometry.Point{X: px, Y: py},
		}
	default:
		return ReplayEntry{}, false
	}
	return entry, true
}
