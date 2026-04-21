package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	rec_files "github.com/memmaker/terminal-assassin/rec-files"
)

const graphicsConfigFile = "graphics.cfg"

type GraphicsConfig struct {
	Fullscreen     bool
	WindowedWidth  int
	WindowedHeight int
}

// LoadGraphicsConfig reads graphics.cfg from disk.
// If the file does not exist it creates it with sensible defaults derived from
// the current monitor resolution and the chosen tile size.
func LoadGraphicsConfig(tileSize int) GraphicsConfig {
	monW, monH := ebiten.Monitor().Size()

	// Default windowed size: 32×18 tiles at the chosen tile size, capped at 2/3 of the monitor.
	// Guard against monW/monH being 0 (monitor not yet initialised before RunGame on some
	// platforms) – without this the cap arithmetic zeros out the defaults.
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

	defaults := GraphicsConfig{
		Fullscreen:     false,
		WindowedWidth:  defW,
		WindowedHeight: defH,
	}

	file, err := os.Open(graphicsConfigFile)
	if err != nil {
		log.Printf("[Graphics] %s not found – writing defaults (windowed %dx%d)", graphicsConfigFile, defW, defH)
		SaveGraphicsConfig(defaults)
		return defaults
	}
	defer file.Close()

	records := rec_files.Read(file)
	if len(records) == 0 {
		log.Printf("[Graphics] %s is empty – using defaults", graphicsConfigFile)
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
		}
	}

	log.Printf("[Graphics] Loaded: fullscreen=%v, windowed=%dx%d", cfg.Fullscreen, cfg.WindowedWidth, cfg.WindowedHeight)
	return cfg
}

// SaveGraphicsConfig writes the current graphics settings to graphics.cfg.
func SaveGraphicsConfig(cfg GraphicsConfig) {
	file, err := os.Create(graphicsConfigFile)
	if err != nil {
		log.Printf("[Graphics] Error saving %s: %v", graphicsConfigFile, err)
		return
	}
	defer file.Close()

	fullscreenVal := "false"
	if cfg.Fullscreen {
		fullscreenVal = "true"
	}

	if err := rec_files.Write(file, []rec_files.Record{
		{
			{Name: "Fullscreen", Value: fullscreenVal},
			{Name: "WindowedWidth", Value: fmt.Sprintf("%d", cfg.WindowedWidth)},
			{Name: "WindowedHeight", Value: fmt.Sprintf("%d", cfg.WindowedHeight)},
		},
	}); err != nil {
		log.Printf("[Graphics] Error writing %s: %v", graphicsConfigFile, err)
	}
}

