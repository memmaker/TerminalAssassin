package audio

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"

	"github.com/memmaker/terminal-assassin/common"
	"github.com/memmaker/terminal-assassin/game/services"
	"github.com/memmaker/terminal-assassin/geometry"
)

type Cue struct {
	AudioCues    []string
	IsRandom     bool
	CurrentIndex int
}

func (c *Cue) Next() string {
	if c.IsRandom {
		randomIndex := rand.Intn(len(c.AudioCues))
		return c.AudioCues[randomIndex]
	}
	if c.CurrentIndex >= len(c.AudioCues) {
		c.CurrentIndex = 0
	}
	c.CurrentIndex++
	return c.AudioCues[c.CurrentIndex]
}

type AudioPlayer struct {
	context         *audio.Context
	engine          services.Engine
	availableSounds map[string]*Sound
	activeSounds    map[string][]*ActiveSound
	loopVolume      float64
	sfxVolume       float64
	masterVolume    float64
}

func (ap *AudioPlayer) GetMasterVolume() float64 {
	return ap.masterVolume
}

func (ap *AudioPlayer) GetMusicVolume() float64 {
	return ap.loopVolume
}

func (ap *AudioPlayer) GetSoundVolume() float64 {
	return ap.sfxVolume
}

func (ap *AudioPlayer) effectiveMusicVolume() float64 {
	return ap.loopVolume * ap.masterVolume
}

func (ap *AudioPlayer) effectiveSoundVolume() float64 {
	return ap.sfxVolume * ap.masterVolume
}

func (ap *AudioPlayer) SetMasterVolume(volume float64) {
	ap.masterVolume = common.Clamp(volume, 0, 1)
	ap.updateVolume()
}

func (ap *AudioPlayer) SetMusicVolume(volume float64) {
	ap.loopVolume = common.Clamp(volume, 0, 1)
	ap.updateVolume()
}

func (ap *AudioPlayer) SetSoundVolume(volume float64) {
	ap.sfxVolume = common.Clamp(volume, 0, 1)
	ap.updateVolume()
}

func (ap *AudioPlayer) updateVolume() {
	for cue, activeSounds := range ap.activeSounds {
		sound := ap.availableSounds[cue]
		for _, activeSound := range activeSounds {
			if sound.IsLooping {
				activeSound.player.SetVolume(ap.effectiveMusicVolume())
			} else {
				activeSound.player.SetVolume(ap.effectiveSoundVolume())
			}
		}
	}
}

type Sound struct {
	Cue          string
	Filenames    []string
	Samples      [][]byte
	IsLooping    bool
	IsRandomized bool
	currentIndex int
}

func (s *Sound) Unload() {
	s.Samples = make([][]byte, len(s.Filenames))
}
func (s *Sound) IsLoaded() bool {
	return len(s.Samples) > 0 && s.Samples[0] != nil && len(s.Samples[0]) > 0
}

func (s *Sound) Load(ap *AudioPlayer) {
	for i, filename := range s.Filenames {
		print(fmt.Sprintf("Loading %s...", filename))
		s.Samples[i] = ap.LoadOGG(filename)
		println("done")
	}
}
func (s *Sound) Next() []byte {
	if s.IsRandomized {
		return s.Samples[rand.Intn(len(s.Samples))]
	}
	sample := s.Samples[s.currentIndex]
	s.currentIndex = (s.currentIndex + 1) % len(s.Samples)
	return sample
}

func (s *Sound) nextFilename() string {
	if s.IsRandomized {
		return s.Filenames[rand.Intn(len(s.Filenames))]
	}
	filename := s.Filenames[s.currentIndex]
	s.currentIndex = (s.currentIndex + 1) % len(s.Filenames)
	return filename
}

func (s *Sound) OpenStream(ap *AudioPlayer) *audio.Player {
	return ap.OpenOGG(s.nextFilename())
}

type ActiveSound struct {
	player   *audio.Player
	callback func()
	pos      *geometry.Point
}

type NullHandle struct{}

func (h *NullHandle) Close() error    { return nil }
func (h *NullHandle) IsPlaying() bool { return false }

func NewAudioPlayer(engine services.Engine) *AudioPlayer {
	return &AudioPlayer{
		context:         audio.NewContext(44100),
		engine:          engine,
		availableSounds: make(map[string]*Sound),
		activeSounds:    make(map[string][]*ActiveSound),
		sfxVolume:       0.8,
		loopVolume:      0.5,
		masterVolume:    0.2,
	}
}
func (ap *AudioPlayer) UnloadAll() {
	for _, sound := range ap.availableSounds {
		sound.Unload()
	}
	ap.activeSounds = make(map[string][]*ActiveSound)
}

func (ap *AudioPlayer) StopAll() {
	for cue, sounds := range ap.activeSounds {
		for _, sound := range sounds {
			sound.player.Close()
		}
		delete(ap.activeSounds, cue)
	}
}

// LoadOGG loads an OGG file from the given filename into memory.
func (ap *AudioPlayer) LoadOGG(filename string) []byte {
	files := ap.engine.GetFiles()
	file, filErr := files.Open(filename)
	defer file.Close()
	stream, decodeErr := vorbis.DecodeWithoutResampling(file)
	if decodeErr != nil || filErr != nil {
		return nil
	}
	buffer := make([]byte, stream.Length())
	stream.Read(buffer)
	return buffer
}

// OpenOGG opens an OGG file from the given filename for streaming.
func (ap *AudioPlayer) OpenOGG(filename string) *audio.Player {
	files := ap.engine.GetFiles()
	file, filErr := files.Open(filename)
	defer file.Close()
	stream, decodeErr := vorbis.DecodeWithoutResampling(file)
	if decodeErr != nil || filErr != nil {
		return nil
	}
	player, err := ap.context.NewPlayer(stream)
	if err != nil {
		return nil
	}
	return player
}

func (ap *AudioPlayer) Stop(cue string) {
	if _, ok := ap.activeSounds[cue]; ok {
		for _, sound := range ap.activeSounds[cue] {
			sound.player.Close()
		}
		delete(ap.activeSounds, cue)
	}
}
func (ap *AudioPlayer) StartLoop(cue string) services.AudioHandle {
	if !ap.engine.GetGame().GetConfig().Audio {
		return &NullHandle{}
	}
	if sound, ok := ap.availableSounds[cue]; ok {
		if !sound.IsLoaded() {
			sound.Load(ap)
		}
		sound.IsLooping = true
		return ap.startPlaying(sound, ap.effectiveMusicVolume(), nil)
	} else {
		println("ERR: Sound cue not found: " + cue)
	}
	return &NullHandle{}
}

func (ap *AudioPlayer) StartLoopStream(cue string) services.AudioHandle {
	if !ap.engine.GetGame().GetConfig().Audio {
		return &NullHandle{}
	}
	if sound, ok := ap.availableSounds[cue]; ok {
		sound.IsLooping = true
		player := sound.OpenStream(ap)
		player.Rewind()
		player.SetVolume(ap.effectiveMusicVolume())
		player.Play()
		ap.addActiveSound(sound, player, nil)
		return player
	} else {
		println("ERR: Sound cue not found: " + cue)
	}
	return &NullHandle{}
}

func (ap *AudioPlayer) startPlaying(sound *Sound, volume float64, callback func()) *audio.Player {
	player := ap.context.NewPlayerFromBytes(sound.Next())
	player.Rewind()
	player.SetVolume(volume)
	player.Play()
	ap.addActiveSound(sound, player, callback)
	return player
}

func (ap *AudioPlayer) startPlayingAt(sound *Sound, volume float64, position geometry.Point, callback func()) *audio.Player {
	player := ap.context.NewPlayerFromBytes(sound.Next())
	player.Rewind()
	player.SetVolume(volume)
	player.Play()
	ap.addActiveSoundAt(sound, player, position, callback)
	return player
}

func (ap *AudioPlayer) addActiveSound(sound *Sound, player *audio.Player, callback func()) {
	if _, ok := ap.activeSounds[sound.Cue]; !ok {
		ap.activeSounds[sound.Cue] = make([]*ActiveSound, 0)
	}
	ap.activeSounds[sound.Cue] = append(ap.activeSounds[sound.Cue], &ActiveSound{player: player, callback: callback})
}

func (ap *AudioPlayer) addActiveSoundAt(sound *Sound, player *audio.Player, position geometry.Point, callback func()) {
	if _, ok := ap.activeSounds[sound.Cue]; !ok {
		ap.activeSounds[sound.Cue] = make([]*ActiveSound, 0)
	}
	ap.activeSounds[sound.Cue] = append(ap.activeSounds[sound.Cue], &ActiveSound{player: player, callback: callback, pos: &position})
}
func (ap *AudioPlayer) PlayCue(cue string) services.AudioHandle {
	if !ap.engine.GetGame().GetConfig().Audio {
		return &NullHandle{}
	}
	if sound, ok := ap.availableSounds[cue]; ok {
		if !sound.IsLoaded() {
			sound.Load(ap)
		}
		return ap.startPlaying(sound, ap.effectiveSoundVolume(), nil)
	} else {
		println("ERR: Sound cue not found: " + cue)
	}
	return &NullHandle{}
}

func (ap *AudioPlayer) PlayCueAt(cue string, position geometry.Point) services.AudioHandle {
	if !ap.engine.GetGame().GetConfig().Audio {
		return &NullHandle{}
	}
	player := ap.engine.GetGame().GetMap().Player
	if geometry.Distance(player.Pos(), position)/float64(player.MaxVisionRange) > 4.0 {
		return &NullHandle{}
	}
	if sound, ok := ap.availableSounds[cue]; ok {
		if !sound.IsLoaded() {
			sound.Load(ap)
		}
		volume := ap.getPositionalVolume(position)
		return ap.startPlayingAt(sound, volume, position, nil)
	} else {
		println("ERR: Sound cue not found: " + cue)
	}
	return &NullHandle{}
}

func (ap *AudioPlayer) getPositionalVolume(position geometry.Point) float64 {
	player := ap.engine.GetGame().GetMap().Player
	if player.Pos() == position {
		return ap.effectiveSoundVolume()
	}
	distance := geometry.Distance(player.Pos(), position) / float64(player.MaxVisionRange) // 0.0..1.0..n
	distance = distance - 1.0
	if distance < 0.0 {
		distance = 0.0
	}
	volume := common.Clamp(ap.effectiveSoundVolume()-distance, 0.0, 1.0)
	return volume
}

func (ap *AudioPlayer) PlayCueWithCallback(cue string, callback func()) services.AudioHandle {
	if !ap.engine.GetGame().GetConfig().Audio {
		return &NullHandle{}
	}
	if sound, ok := ap.availableSounds[cue]; ok {
		if !sound.IsLoaded() {
			sound.Load(ap)
		}
		return ap.startPlaying(sound, ap.effectiveSoundVolume(), callback)
	} else {
		println("ERR: Sound cue not found: " + cue)
	}
	return &NullHandle{}
}

func (ap *AudioPlayer) IsCuePlaying(cue string) bool {
	if sounds, ok := ap.activeSounds[cue]; ok {
		return len(sounds) > 0
	}
	return false
}

func (ap *AudioPlayer) Update() {
	if !ap.engine.GetGame().GetConfig().Audio {
		return
	}
	for cue, sounds := range ap.activeSounds {
		soundDef := ap.availableSounds[cue]
		for i := len(sounds) - 1; i >= 0; i-- {
			sound := sounds[i]
			if !sound.player.IsPlaying() {
				if soundDef.IsLooping {
					sound.player.Rewind()
					sound.player.Play()
					continue
				}
				sound.player.Close()
				if sound.callback != nil {
					sound.callback()
				}
				ap.activeSounds[cue] = append(ap.activeSounds[cue][:i], ap.activeSounds[cue][i+1:]...)
			} else if sound.pos != nil {
				volume := ap.getPositionalVolume(*sound.pos)
				sound.player.SetVolume(volume)
			}
		}
	}
}
func (ap *AudioPlayer) RegisterSoundCues(filenames []string) {
	if !ap.engine.GetGame().GetConfig().Audio {
		return
	}
	files := ap.engine.GetFiles()
	for _, filename := range filenames {
		if !files.FileExists(filename) {
			println("ERR: Sound file not found: " + filename)
			continue
		}
		extension := strings.ToLower(filepath.Ext(filename))
		basename := filepath.Base(filename)
		cue := basename[0 : len(basename)-len(extension)]
		s := &Sound{
			Cue:       cue,
			Filenames: []string{filename},
			Samples:   make([][]byte, 1),
		}
		ap.availableSounds[cue] = s
	}
}
func (ap *AudioPlayer) PreLoadCuesIntoMemory(soundCues []string) {
	if !ap.engine.GetGame().GetConfig().Audio {
		return
	}
	for _, soundCue := range soundCues {
		if sound, ok := ap.availableSounds[soundCue]; ok {
			if !sound.IsLoaded() {
				println(fmt.Sprintf("Loading sound cue '%s'", soundCue))
				sound.Load(ap)
			}
		}
	}
}
func (ap *AudioPlayer) UnloadCues(soundCues []string) {
	for _, soundCue := range soundCues {
		if sound, ok := ap.availableSounds[soundCue]; ok {
			if sound.IsLoaded() {
				println(fmt.Sprintf("Unloading sound cue '%s'", soundCue))
				sound.Unload()
			}
		}
	}
}
func (ap *AudioPlayer) RegisterRandomizedSoundCues(directories []string) {
	if !ap.engine.GetGame().GetConfig().Audio {
		return
	}
	for _, directory := range directories {
		cue := filepath.Base(directory)
		files := ap.engine.GetFiles()
		filesInDirectory := files.GetFilesInPath(directory)
		s := &Sound{
			Cue:          cue,
			Filenames:    filesInDirectory,
			Samples:      make([][]byte, len(filesInDirectory)),
			IsRandomized: true,
		}
		ap.availableSounds[cue] = s
	}
}
