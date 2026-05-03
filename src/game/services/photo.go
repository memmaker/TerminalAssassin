package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/memmaker/terminal-assassin/game/core"
)

// CapturePhotoMetadata inspects the current game state and builds a
// PhotoMetadata record containing every actor, item and object that is
// currently visible to the player.
func CapturePhotoMetadata(engine Engine) core.PhotoMetadata {
	currentMap := engine.GetGame().GetMap()
	player := currentMap.Player

	mapName := filepath.Base(currentMap.MapFileName())

	metadata := core.PhotoMetadata{
		Timestamp: time.Now().Format(time.RFC3339),
		MapName:   mapName,
	}

	// Active (standing/moving) actors
	for _, actor := range currentMap.Actors() {
		if player.CanSee(actor.Pos()) && actor.IsVisible() {
			zone := currentMap.ZoneAt(actor.Pos())
			metadata.VisibleActors = append(metadata.VisibleActors, core.ActorSighting{
				Name: actor.Name,
				Zone: zone.Name,
			})
		}
	}

	// Downed (unconscious / dead on ground) actors
	for _, actor := range currentMap.DownedActors() {
		if player.CanSee(actor.Pos()) {
			zone := currentMap.ZoneAt(actor.Pos())
			metadata.VisibleActors = append(metadata.VisibleActors, core.ActorSighting{
				Name: actor.Name + " (downed)",
				Zone: zone.Name,
			})
		}
	}

	// Items lying on the map (not held by any actor)
	for _, item := range currentMap.Items() {
		if item.HeldBy == nil && player.CanSee(item.Pos()) {
			zone := currentMap.ZoneAt(item.Pos())
			metadata.VisibleItems = append(metadata.VisibleItems, core.ItemSighting{
				Name: item.Name,
				Zone: zone.Name,
			})
		}
	}

	// Interactive objects (doors, containers, etc.)
	for _, obj := range currentMap.Objects() {
		if player.CanSee(obj.Pos()) {
			zone := currentMap.ZoneAt(obj.Pos())
			metadata.VisibleObjects = append(metadata.VisibleObjects, core.ObjectSighting{
				Description: obj.Description(),
				Zone:        zone.Name,
			})
		}
	}

	return metadata
}

// SavePhotoMetadata serialises metadata to a pretty-printed JSON file.
func SavePhotoMetadata(metadata core.PhotoMetadata, filePath string) {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		println("Camera: failed to marshal photo metadata:", err.Error())
		return
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		println("Camera: failed to save photo metadata:", err.Error())
	}
}
