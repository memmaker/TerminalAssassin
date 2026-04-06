# AGENTS.md — Terminal Assassin

A terminal-aesthetic stealth/assassination roguelite written in Go, rendered via a custom Unicode cell grid on top of [Ebiten v2](https://ebitengine.org/).

## Architecture Overview

```
ConsoleEngine (main.go)          ← top-level engine; implements services.Engine
├── game.Model                   ← game state stack + map reference
│   └── currentGameStates []GameState  (push/pop stack)
├── game/ai.AIController         ← drives all NPC behaviour
├── game/actions.ActionProvider  ← item use, projectiles, melee
├── game/services.ExternalData   ← loads items/tiles/clothing from rec-files
├── console.Console              ← Ebiten-backed cell grid renderer
└── ui.Manager                   ← modal menus, pager, ring menus, tooltips
```

**`services.Engine`** (`src/game/services/interfaces.go`) is the single dependency-injection root. Every subsystem is accessed via `engine.GetGame()`, `engine.GetAI()`, `engine.GetUI()`, etc. Never hold direct pointers across packages — always go through the `Engine` interface.

**Game state machine**: states live in `src/game/states/`. Each implements `GameState` (`Init`, `Update`, `Draw`, `SetDirty`). Push/pop via `model.PushState` / `model.PopState` / `model.PopAndInitPrevious`.

## Key Packages

| Package | Role |
|---------|------|
| `game/core` | Pure data types: `Actor`, `Item`, `Clothing`, `IncidentReport`, `Observation` constants, `glyphs.go` (Unicode rune constants) |
| `game/stimuli` | Physical effects (`StimulusType` constants + `Stimulus` interface) applied with a force int |
| `game/objects` | Interactive world objects (`services.Object` interface): doors, containers, triggers, safes |
| `game/director` | Scripted narrative: `Script` (keyframe sequences) + `DialogueInfo`, parsed from text scripts |
| `gridmap` | Generic `GridMap[ActorType, ItemType, ObjectType]` — the live world map |
| `geometry` | Pathfinding (A*, BFS, Dijkstra, JPS), FOV, camera, `Point`/`Rect` |
| `rec-files` | Custom `key: value` text format — blank line = record separator |
| `coroutine` | Go-channel coroutine (`gocoro.go`) used for timed scripted sequences |

## Build & Run

All build commands run from **`src/`**:

```sh
# Local development (macOS native)
cd src && go run .

# Production build (requires CGO)
cd src && CGO_ENABLED=1 go build -tags ebitensinglethread -o ta .

# WASM build
cd src && CGO_ENABLED=1 GOOS=js GOARCH=wasm go build -tags ebitensinglethread,web -o ../release/wasm/ta.wasm .

# Linux cross-compile (requires Docker)
./create_linux.sh          # uses ghcr.io/memmaker/lfgo-amd64 container

# Full multi-platform release
./release.sh <executableName>
```

The `web` build tag activates `web_mode.go` which sets `WEB_MODE = true`. Data files are embedded via `//go:embed datafiles` in `main.go`; the `Files` struct (`files.go`) falls back to disk before the embedded FS.

## Critical Conventions

**Item/clothing names are identifiers.** They must be unique — `ExternalData.ItemByName` looks them up by exact name string. Rename with care.

**Glyphs are non-ASCII Unicode runes** defined in `src/game/core/glyphs.go`. Always use the named constant (e.g. `core.GlyphPistol`) rather than the raw rune literal.

**Data files use the `rec-files` format** (`src/datafiles/core/base/*.txt`). One record per blank-line-separated block:
```
name: AMT 1911 'Hardballer'
icon: ٤
type: pistol
uses: 7
trigger_effects: BulletTrigger(35)
```
Trigger/stim references in data files are parsed as function-call strings via `core.GetNameAndArgs`.

**AI states return `core.AIUpdate`** — either `NextUpdateIn(seconds)` (timer) or `DeferredUpdate(predicate)` (condition). Switch AI state via `engine.GetAI().SwitchTo*` methods; never mutate actor state directly from outside `ai/`.

**Context actions** implement `services.ContextAction` (Description + Action + IsActionPossible). One action per tile — see `src/game/actions_context.go` for the pattern.

**Events** use a typed filter bus: `engine.PublishEvent(evt)` + `engine.SubscribeToEvents(services.NewFilter[MyEventType](func(e MyEventType) bool { ... }))`.

**Scheduling**: Use `engine.Schedule(delaySecs, fn)` only for effects that must not be cancelled. Prefer `DeferredUpdate` inside AI states for cancellable timed actions.

## Map Authoring

Maps live in `build/datafiles/` (shipped) or `src/datafiles/campaigns/` (dev). The in-game editor (F1–F5 hotkeys) writes them. Map tile layer is a Unicode rune grid; items/actors/objects are separate `rec-files` sections loaded by `MapSerializer` (`src/map_serialization.go`).

## No Tests

There is no test suite. Validate changes by running the game and exercising the affected feature in the editor or gameplay mode.

## For AI Agents: No scripts, no python, no shell

AI Agents should not utilize external scripts or shell commands or python.

AI Agents should use the internal file creation and manipulation capabilities of the IDE and their LLM integration.

The only exception are compile error checks.