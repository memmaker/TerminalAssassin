# Gamepad Controls — Terminal Assassin

> This guide covers every in-game action and how to perform it with a standard
> gamepad (PlayStation / Xbox layout). All controls use Ebiten's **Standard
> Gamepad Layout**, so any controller with a recognised mapping works.

---

## Quick-Reference Cheat Sheet

| Button | Gameplay | Menus |
|---|---|---|
| **Left Stick** | Move (8 directions) | Scroll list (up / down) |
| **Right Stick** | Peek / look around | — |
| **L1** *(hold)* | Run | — |
| **L2** *(hold)* | Switch right stick to Aim mode | — |
| **R1** *(or* **✕ / A***)* | Context Action | — |
| **R2** *(hold)* | Fire / Throw equipped item | — |
| **△ / Y** | Assassinate | — |
| **□ / X** | Use item at peeked tile | — |
| **○ / B** | Dive & Tackle | Cancel / Back |
| **✕ / A** | Context Action *(alias for R1)* | Confirm |
| **D-Pad ↑** | Holster item | — |
| **D-Pad ↓** | Drop item | — |
| **D-Pad ←** | Open Inventory | Navigate left |
| **D-Pad →** | Toggle Sneak | Navigate right |
| **Select / Back** | Toggle Look mode | — |
| **R3** *(right stick click)* | Toggle Look mode *(alias)* | — |
| **Options / Start** | Pause menu | — |

---

## Movement & Positioning

### Moving Around
- **Left Stick** — Move in any of 8 directions, one tile per step.
- **L1 held** — Run. Steps fire more rapidly so your character covers ground faster.
- **D-Pad →** — Toggle **Sneak mode**. Press once to enter sneak, press again to
  leave. While sneaking, steps are slower and quieter. A sneak indicator appears
  in the HUD.

### Peeking & Looking
- **Right Stick** — Peek in any direction without moving. The camera shifts and
  your FoV cone rotates so you can scout around corners. The peeked tile is also
  used as the target for several actions (see below).

### Look Mode (Free Cursor)
- **Select / Back** or **R3** *(right stick click)* — Toggle look mode. A cursor appears at your position and a
  tooltip describes whatever is under it — actors, items, objects, or terrain.
- **Right Stick** *(while in look mode)* — Moves the look cursor freely across
  the map (not limited to adjacent tiles). The camera scrolls to follow.
- **Select / Back** or **R3** again — Exit look mode, return to normal play.

---

## Aiming & Ranged Combat

### Aim Mode
- **L2 held** — Switches the right stick from *peek* to *aim*. While L2 is held,
  the right stick steers a targeting cursor across the map.
- **L2 released** — Exits aim mode and returns to the default peek/move state.

### Firing & Throwing
- **R2 held** — Fire your equipped ranged weapon, or throw your equipped throwable
  item, towards the aimed position. Hold for sustained fire with automatic weapons.

---

## Interactions & Context Actions

### Context Action (R1 / ✕·A)
**R1** or **✕ / A** is the smart interaction button. What it does depends on what is in front
of you (your current peek tile):

| Situation | Result |
|---|---|
| Facing an NPC that wants to talk | Start dialogue |
| Standing behind an unaware NPC while **sneaking** | Pickpocket |
| Facing a wall with no special tile | Knock on the wall (creates a noise distraction) |

### Using an Item at the Peeked Tile (□ / X)
Press **□ / X** to activate your equipped item *at the tile you are currently
peeking toward*. Context-sensitive examples:

- **Poison** equipped → lace a food or drink item on that tile.
- **Lockpick / E-Pick** equipped → begins picking a lock on a door or safe.
- **Screwdriver / Wrench** equipped → use on a device, vent or panel.
- **Crowbar** equipped → pry open a locked mechanical door.

### Assassination (△ / Y)
When an assassination target steps into range, press **△ / Y** for an instant
silent kill. Only available when a valid target is highlighted.  
Your sharpest bladed item in inventory is used automatically. If none is found,
a bare-hands technique is used instead.

### Dive & Tackle (○ / B)
Press **○ / B** to lunge **two tiles** in your current peek direction:

- **Both tiles clear** → full two-tile dive.
- **One tile clear, one blocked** → one-tile dive.
- **NPC(s) on either tile** → they are shoved in that direction (tackle), then
  you land on the nearest clear tile.

---

## Inventory & Equipment

### Open Inventory (D-Pad ←)
Opens the full inventory screen. Use the left stick or D-Pad to navigate;
press **✕ / A** to equip / select an item and **○ / B** to close.

### Holster Item (D-Pad ↑)
Puts the currently equipped item away (back into inventory) without dropping it.
Use this to hide a weapon before an NPC spots you carrying it.

### Drop Item (D-Pad ↓)
Drops the currently equipped item onto the floor at your feet. Dropped illegal
weapons may be spotted by guards and trigger an investigation.

---

## Interacting with the World

Items, bodies, objects (doors, safes, containers, distractors) are interacted
with by **walking into them or by using R1 / □**.

| Object | How to interact |
|---|---|
| Closed door | Walk into it to open/close |
| Locked door (key) | Walk into it while carrying the matching key |
| Locked door (pick) | Stand adjacent, peek toward it, press **□** with a lockpick equipped |
| Locked door (crowbar) | Stand adjacent, peek toward it, press **□** with a crowbar equipped |
| Safe | Stand adjacent, peek toward it, press **□** with the correct pick/key |
| Container / Locker | Walk into it to hide inside; walk into it again to exit |
| Downed body | Walk onto it to drag; walk while dragging to carry it to another tile |
| Changeable clothes | Stand on a downed body tile, press **R1** |
| Distractor (radio…) | Press **R1** while adjacent to toggle it on/off |
| Exit point | Walk into the exit marker once all targets are eliminated |

---

## Menus & Dialogs

| Button | Action |
|---|---|
| **Left Stick / D-Pad** | Navigate up / down / left / right |
| **✕ / A** | Confirm selection |
| **○ / B** | Cancel / go back |
| **Options / Start** | Open / close the pause menu |

Text input prompts (keypad codes, etc.) require a keyboard. On controller-only
setups the on-screen numpad pop-up is controlled with **Left Stick** to move the
cursor and **✕ / A** to confirm a digit.

---

## Movement Speed Reference

| State | Speed |
|---|---|
| **Sneak** (D-Pad → toggled on) | Slowest — silent footsteps |
| **Walk** (default) | Normal — audible but not suspicious |
| **Run** (L1 held) | Fastest — clearly audible, may draw attention |

Diagonal movement automatically applies the correct √2 cooldown so diagonal and
cardinal steps feel consistent.

---

## Tips

- **Peek before you act.** Almost every action (R1, □, △, ○) fires at the tile
  your right stick is pointing toward. Aim your stick first.
- **L2 + right stick = precision aim.** Use this combination to line up a shot
  around a corner before pressing R2.
- **Sneak + R1** near an NPC's back = pickpocket without a sound.
- **Knock on a wall** (R1 facing a plain wall) to lure a nearby guard away from
  their post.
- **Holster before entering a restricted zone.** Guards react to visible weapons
  before they react to trespassing.
- **Drag bodies into lockers** to prevent them being found. Walk over the body to
  grab it, then walk to a locker and walk into it.




