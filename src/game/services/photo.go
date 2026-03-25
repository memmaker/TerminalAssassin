package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PhotoMetadata captures a snapshot of what was visible in the scene when a
// photo was taken. It is serialised as JSON alongside the screenshot image.
type PhotoMetadata struct {
	Timestamp      string           `json:"timestamp"`
	MapName        string           `json:"map_name"`
	VisibleActors  []ActorSighting  `json:"visible_actors"`
	VisibleItems   []ItemSighting   `json:"visible_items"`
	VisibleObjects []ObjectSighting `json:"visible_objects"`
}

// ActorSighting records a single visible actor and the zone they were in.
type ActorSighting struct {
	Name string `json:"name"`
	Zone string `json:"zone"`
}

// ItemSighting records a single visible map item and the zone it was in.
type ItemSighting struct {
	Name string `json:"name"`
	Zone string `json:"zone"`
}

// ObjectSighting records a single visible map object and the zone it was in.
type ObjectSighting struct {
	Description string `json:"description"`
	Zone        string `json:"zone"`
}

// CapturePhotoMetadata inspects the current game state and builds a
// PhotoMetadata record containing every actor, item and object that is
// currently visible to the player.
func CapturePhotoMetadata(engine Engine) PhotoMetadata {
	currentMap := engine.GetGame().GetMap()
	player := currentMap.Player

	mapName := filepath.Base(currentMap.MapFileName())

	metadata := PhotoMetadata{
		Timestamp: time.Now().Format(time.RFC3339),
		MapName:   mapName,
	}

	// Active (standing/moving) actors
	for _, actor := range currentMap.Actors() {
		if player.CanSee(actor.Pos()) && actor.IsVisible() {
			zone := currentMap.ZoneAt(actor.Pos())
			metadata.VisibleActors = append(metadata.VisibleActors, ActorSighting{
				Name: actor.Name,
				Zone: zone.Name,
			})
		}
	}

	// Downed (unconscious / dead on ground) actors
	for _, actor := range currentMap.DownedActors() {
		if player.CanSee(actor.Pos()) {
			zone := currentMap.ZoneAt(actor.Pos())
			metadata.VisibleActors = append(metadata.VisibleActors, ActorSighting{
				Name: actor.Name + " (downed)",
				Zone: zone.Name,
			})
		}
	}

	// Items lying on the map (not held by any actor)
	for _, item := range currentMap.Items() {
		if item.HeldBy == nil && player.CanSee(item.Pos()) {
			zone := currentMap.ZoneAt(item.Pos())
			metadata.VisibleItems = append(metadata.VisibleItems, ItemSighting{
				Name: item.Name,
				Zone: zone.Name,
			})
		}
	}

	// Interactive objects (doors, containers, etc.)
	for _, obj := range currentMap.Objects() {
		if player.CanSee(obj.Pos()) {
			zone := currentMap.ZoneAt(obj.Pos())
			metadata.VisibleObjects = append(metadata.VisibleObjects, ObjectSighting{
				Description: obj.Description(),
				Zone:        zone.Name,
			})
		}
	}

	return metadata
}

// SavePhotoMetadata serialises metadata to a pretty-printed JSON file.
func SavePhotoMetadata(metadata PhotoMetadata, filePath string) {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		println("Camera: failed to marshal photo metadata:", err.Error())
		return
	}
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		println("Camera: failed to save photo metadata:", err.Error())
	}
}

