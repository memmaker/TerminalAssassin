package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

const optionsFile = "options.rec"

type Options struct {
	// Graphics
	Fullscreen     bool
	WindowedWidth  int
	WindowedHeight int
	// Audio
	MasterVolume float64
	MusicVolume  float64
	SoundVolume  float64
	// Gameplay
	ShowHints      bool
	ControllerMode string
	TextFont       string
}

func LoadOptions(tileSize int) Options {
	monW, monH := ebiten.Monitor().Size()
	defW := tileSize * 32
	defH := tileSize * 18
	if monW > 0 && monH > 0 {
		if defW > monW*2/3 {
			defW = monW * 2 / 3
		}
		if defH > monH*2/3 {
			defH = monH * 2 / 3
		}
	}

	defaults := Options{
		Fullscreen:     false,
		WindowedWidth:  defW,
		WindowedHeight: defH,
		MasterVolume:   0.2,
		MusicVolume:    0.5,
		SoundVolume:    0.8,
		ShowHints:      true,
		ControllerMode: "Keyboard & Mouse",
	}

	file, err := os.Open(optionsFile)
	if err != nil {
		log.Printf("[Options] %s not found – writing defaults", optionsFile)
		SaveOptions(defaults)
		return defaults
	}
	defer file.Close()

	records := rec_files.Read(file)
	if len(records) == 0 {
		return defaults
	}

	cfg := defaults
	for _, field := range records[0] {
		switch field.Name {
		case "Fullscreen":
			cfg.Fullscreen = field.Value == "true"
		case "WindowedWidth":
			if v, err := strconv.Atoi(field.Value); err == nil && v > 0 {
				cfg.WindowedWidth = v
			}
		case "WindowedHeight":
			if v, err := strconv.Atoi(field.Value); err == nil && v > 0 {
				cfg.WindowedHeight = v
			}
		case "MasterVolume":
			if v, err := strconv.ParseFloat(field.Value, 64); err == nil {
				cfg.MasterVolume = v
			}
		case "MusicVolume":
			if v, err := strconv.ParseFloat(field.Value, 64); err == nil {
				cfg.MusicVolume = v
			}
		case "SoundVolume":
			if v, err := strconv.ParseFloat(field.Value, 64); err == nil {
				cfg.SoundVolume = v
			}
		case "ShowHints":
			cfg.ShowHints = field.Value == "true"
		case "ControllerMode":
			cfg.ControllerMode = field.Value
		case "TextFont":
			cfg.TextFont = field.Value
		}
	}

	log.Printf("[Options] Loaded: fullscreen=%v windowed=%dx%d master=%.1f music=%.1f sound=%.1f hints=%v",
		cfg.Fullscreen, cfg.WindowedWidth, cfg.WindowedHeight,
		cfg.MasterVolume, cfg.MusicVolume, cfg.SoundVolume, cfg.ShowHints)
	return cfg
}

func SaveOptions(cfg Options) {
	file, err := os.Create(optionsFile)
	if err != nil {
		log.Printf("[Options] Error saving %s: %v", optionsFile, err)
		return
	}
	defer file.Close()

	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}

	if err := rec_files.Write(file, []rec_files.Record{
		{
			{Name: "Fullscreen", Value: boolStr(cfg.Fullscreen)},
			{Name: "WindowedWidth", Value: fmt.Sprintf("%d", cfg.WindowedWidth)},
			{Name: "WindowedHeight", Value: fmt.Sprintf("%d", cfg.WindowedHeight)},
			{Name: "MasterVolume", Value: fmt.Sprintf("%.2f", cfg.MasterVolume)},
			{Name: "MusicVolume", Value: fmt.Sprintf("%.2f", cfg.MusicVolume)},
			{Name: "SoundVolume", Value: fmt.Sprintf("%.2f", cfg.SoundVolume)},
			{Name: "ShowHints", Value: boolStr(cfg.ShowHints)},
			{Name: "ControllerMode", Value: cfg.ControllerMode},
			{Name: "TextFont", Value: cfg.TextFont},
		},
	}); err != nil {
		log.Printf("[Options] Error writing %s: %v", optionsFile, err)
	}
}

