# Terminal Assassin — Map Editor Manual

## Overview

The map editor is a tile-based level designer for Terminal Assassin. It opens over the game world and lets you build and modify maps in real time. The interface consists of:

- **Map viewport** — the main editing canvas.
- **Status bar** (second row from the bottom) — shows the current edit mode, active brush, and active place icon, plus the currently selected foreground/background colours.
- **Message bar** (bottom row) — feedback and error messages from the last action.
- **Menu bar** (bottom area) — mode-switching buttons and context-sensitive action menus.

---

## UI Layout

```
┌─────────────────────────────────┐
│                                 │
│         Map Viewport            │
│                                 │
├─────────────────────────────────┤
│  Status: mode | brush icon | fg/bg colours   │  ← status bar
├─────────────────────────────────┤
│  [Brush][Tiles][Items][Objects]…│  ← menu bar
├─────────────────────────────────┤
│  Message / error text           │  ← message bar
└─────────────────────────────────┘
```

---

## Global Navigation (always active)

These keys work in every edit mode.

| Key | Action |
|---|---|
| `Arrow Keys` | Scroll/pan the camera one tile |
| `Alt + Arrow Keys` | Shift the entire map one tile in that direction |
| `Shift + Left / Right Arrow` | Shrink / grow map width by 1 |
| `Shift + Up / Down Arrow` | Shrink / grow map height by 1 |
| `Mouse Wheel` | Scroll the camera |
| `n` | Step time of day back by 30 minutes |
| `m` | Step time of day forward by 30 minutes |
| `Escape` | Cancel current selection, clear all selections, return to Tile edit mode |
| `Shift + hover` | Show tooltip for the tile/actor/object/item under the cursor |

---

## Tooltip Information

Hold **Shift** and hover over any tile to see:

- **Actor** — name, look direction, task count, items.
- **Object** — description, and its key string if it has one.
- **Item** — name, and its key string if it has one.
- **Empty tile** — tile type, zone name, named location (if any).

---

## Menu Bar Modes

The menu bar at the bottom switches the editor between major modes. Each mode changes what left-click does on the map and which context actions are available.

| Shortcut | Mode | What left-click does |
|---|---|---|
| `Tab` | **Brush** | Opens brush selector (see Brushes) |
| `F1` | **Tiles** | Places the selected tile |
| `F2` | **Items** | Places / removes items |
| `F3` | **Objects** | Places / removes objects |
| `F4` | **Actors** | Selects or edits actors |
| `F5` | **Schedule** | Adds patrol tasks to the selected actor |
| `F6` | **Clothes** | Places clothing items or dresses actors |
| `F7` | **Zones** | Paints zone regions |
| `F8` | **Stimuli** | Paints persistent stimuli onto tiles |
| `F9` | **Lights** | Places and edits baked light sources |
| `F10` | **Prefabs** | Captures a region as a reusable prefab |
| `F11` | **Global** | Map-level operations (new, load, save, resize, quit) |

---

## Brushes (`Tab`)

The brush controls the *shape* of what gets painted on the map. Choose a brush from the drop-down, then use any placement mode.

| Brush | Behaviour |
|---|---|
| **Pencil** | Paints individual tiles as you click and drag |
| **Fill Bucket** | Flood-fills all connected tiles of the same type |
| **Lines** | Draws a straight line from the click point to the release point |
| **Outlined Rectangle** | Draws only the border of a rectangle |
| **Filled Rectangle** | Draws a fully filled rectangle |
| **Outlined Circle/Ellipse** | Draws only the border of an ellipse defined by the drag bounds |
| **Filled Circle/Ellipse** | Draws a filled ellipse |

The current brush icon is shown in the status bar.

---

## Tiles (`F1`)

Opens a drop-down listing all tile types defined in `tiles.txt`. Click a tile type to select it, then paint on the map.

- **Player Spawn** tile is special: placing it moves the spawn marker (`@` shown in green) rather than painting a tile.
- The currently selected **foreground** and **background** colours (see Colour Tools) are applied to tiles when they differ from the tile's defaults.

### Context actions (right-click / context menu in Tiles mode)

| Key | Action |
|---|---|
| `Space` | Open the tile picker drop-down |

---

## Items (`F2`)

Opens a drop-down listing all items from `items.txt` plus any factory-generated complex items.

- **Place on empty tile** — drops the item on the floor.
- **Place on actor** — adds the item directly to that actor's inventory.
- **"Clear items"** entry removes whichever item is at the target tile.

### Context actions while an item is selected

| Key | Action |
|---|---|
| `Space` | Reopen the item picker |
| `k` | Set a key string on the selected item (used by script triggers) |
| `Backspace` | Delete the selected item from the map or inventory |

---

## Objects (`F3`)

Opens a drop-down listing all simple objects from the object factory.

- Objects are placed on the floor tile at the target position. The tile's colour is updated to match the current foreground/background colours.
- Objects can have a **key string** used to link them to script triggers or doors.

### Context actions

| Key | Action |
|---|---|
| `Space` | Reopen the object picker |
| `k` | Set a key string on the selected object |
| `Backspace` | Delete the selected object |

---

## Actors (`F4`)

Actors are NPCs and enemies on the map.

### Placing actors

| Action | Result |
|---|---|
| `a` (from context menu) | Enter a name, then click a walkable tile to place the actor |
| `q` (Quick Add) | Place an auto-named actor immediately on click |

### Selecting an actor

Left-click any actor to select it. The status bar shows the actor's name, task count and inventory.

### Context actions when an actor is selected

| Key | Action |
|---|---|
| `t` | Cycle actor type: Civilian → Guard → Enforcer → Target → Civilian |
| `f` | Set leader — click another actor to make the selected actor follow them |
| `r` | Rename the selected actor |
| `w` | Move actor — click a passable tile to relocate |
| `i` | Open actor's inventory ring menu |
| `d` | Adjust look direction — move the mouse to set gaze angle, click to confirm |
| `Backspace` | Delete the selected actor |

---

## Schedule (`F5`)

Requires an actor to be selected first (select via F4). Each actor has an ordered list of **patrol tasks** — locations they walk to and wait at.

### Adding a task

Switch to Schedule mode (`F5`), then left-click any walkable tile to place a new task waypoint there. Tasks are numbered in order and paths between them are computed automatically.

### Context actions

| Key | Action |
|---|---|
| `a` | Enter Add Task mode (then click a tile) |
| `j` | Decrease the dwell time of the selected task by 1 second |
| `k` | Increase the dwell time of the selected task by 1 second |
| `Backspace` | Delete the selected task |

> **Tip:** Click an existing task waypoint to select it before adjusting its time.

---

## Clothes (`F6`)

Opens a drop-down listing all clothing types from `clothing.txt`.

- **Place on empty tile** — drops the clothing pickup on the floor.
- **Place on actor** — immediately dresses the actor in that clothing.
- **"Clear clothes"** entry removes the clothing item at the target position.

---

## Zones (`F7`)

Zones are named regions that control NPC access rules and disguise requirements.

### Zone types (cycled with `p`)

| Type | Colour in editor | Meaning |
|---|---|---|
| Standard | default | General zone |
| Public | Green | Anyone can enter |
| High Security | Red | Restricted access |
| Drop-off | Blue | Loot drop-off point |

### Workflow

1. Press **F7**, then `a` to create a new zone (enter a name).
2. Paint tiles to assign them to the zone.
3. Use `p` to cycle the zone's type.
4. Use `c` to add allowed clothing types for the zone.
5. Open the zone drop-down (`Space`) to switch between existing zones.

---

## Stimuli (`F8`)

Paints persistent environmental stimuli onto individual tiles. These influence AI behaviour and fire/water interactions.

| Stimulus | Effect |
|---|---|
| **Fire** | Tile is on fire (force 80) |
| **Water** | Tile has water (force 80) |
| **Blood** | Blood stain on tile (force 10) |
| **Burnable** | Tile contains burnable liquid (force 80) |
| **Clear** | Removes all stimuli from the tile |

---

## Lights (`F9`)

Manages **baked** (static) light sources that are pre-computed into the map.

### Placing a light

Switch to Lights mode and left-click any tile to place a light source. New lights default to a radius of 7 and inherit the ambient light colour (or the colour of the last selected light source).

### Selecting a light

Left-click an existing light (shown as `*`) to select it.

### Context actions

| Key | Action |
|---|---|
| `s` | Decrease the radius of the selected light |
| `d` | Increase the radius of the selected light |
| `f` | Open colour picker for the selected light's colour |
| `o` | Open colour picker for the map's ambient light |
| `i` | Recalculate all lights (baked + dynamic) |

### Time of day

The ambient light colour is derived from the map's **time of day**. Use `n` / `m` globally to step it in 30-minute increments. The ambient colour updates live so you can preview day/night lighting before baking.

---

## Prefabs (`F10`)

Prefabs are reusable tile/object/actor snapshots.

### Creating a prefab

1. Press **F10** to enter Prefab mode.
2. Draw a **Filled Rectangle** selection over the region you want to capture.
3. Release the mouse — the selection becomes a prefab held in memory.

### Placing a prefab

After capture, the editor automatically enters Place Prefab mode.

| Key | Action |
|---|---|
| `e` | Rotate the prefab 90° clockwise |
| Left-click | Stamp the prefab at the clicked position |

---

## Global Menu (`F11`)

| Option | Action |
|---|---|
| **New Map** | Creates a blank map the size of the current viewport |
| **Resize Map** | Prompts for `width height` (e.g. `64 36`) and recreates the map |
| **Load Map** | Opens a map browser to load an existing `.map` folder |
| **Save Map** | Prompts for a file name and saves the map into the current campaign folder |
| **CImage from selection** | Exports the selected rectangle as a `.cmg` cell-image file |
| **Quit Editor** | Returns to the main menu |

---

## Colour Tools

Two colour tools are always accessible from the menu bar. Selected colours are applied when placing tiles, objects, and clothes.

| Button | Action |
|---|---|
| **Tile Background Color** | Opens a colour picker; sets the BG colour used for new tiles |
| **Tile Foreground Color** | Opens a colour picker; sets the FG colour used for new tiles |

The status bar always shows the active FG (`fg:`) and BG (`bg:`) colours as coloured swatches.

### Eye Dropper (`e`)

| Input | Action |
|---|---|
| Left-click a tile | Picks that tile's type **and** its colours, immediately switching to Tile paint mode with those settings |
| `Shift` + left-click a tile | Picks only the tile's **colours** (FG + BG) and switches to colour-paint mode |

---

## Named Locations

Named locations are invisible map markers referenced by scripts and mission briefings.

Access via the **Edit Named Locations** button in the menu bar.

### Context actions

| Key | Action |
|---|---|
| `q` | Place a new named location at the clicked tile (auto-named) |
| `r` | Rename the selected named location |
| `w` | Move the selected named location — click a new tile to confirm |
| `Backspace` | Delete the selected named location |

> Hovering over a named location tile with **Shift** held shows its name in the tooltip.

---

## Smart Selection

Regardless of the current mode, left-clicking the map will **auto-detect** what is at that position and switch to the appropriate edit mode:

| What's at the tile | Mode switched to |
|---|---|
| An actor | Actor edit mode |
| A task waypoint (if actor selected) | Schedule edit mode |
| An object | Object edit mode |
| An item | Item edit mode |
| A baked light source | Light edit mode |
| A named location | Named location edit mode |

---

## Quick Reference Card

```
Tab        Brush picker
F1         Tile mode        (Space = open tile menu)
F2         Item mode        (Space = open item menu, k = set key, Del = remove)
F3         Object mode      (Space = open object menu, k = set key, Del = remove)
F4         Actor mode       (a = add, q = quick-add, t = type, r = rename,
                             w = move, d = direction, f = leader, i = inventory, Del = delete)
F5         Schedule mode    (a = add task, j/k = time ±1s, Del = delete task)
F6         Clothes mode     (Space = open clothes menu)
F7         Zone mode        (a = new zone, p = cycle type, c = clothing rules)
F8         Stimuli mode     (Fire/Water/Blood/Burnable/Clear)
F9         Light mode       (s/d = radius, f = light colour, o = ambient, i = update all)
F10        Prefab mode      (e = rotate, click = stamp)
F11        Global menu      (n = new, r = resize, l = load, s = save, q = quit)

Arrow keys          Pan camera
Alt + Arrow keys    Shift entire map
Shift + Arrow keys  Resize map
n / m               Time of day −/+ 30 min
Escape              Clear selection, back to Tile mode
Shift + hover       Tooltip
```

