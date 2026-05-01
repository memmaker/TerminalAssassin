# Terminal Assassin

A terminal-aesthetic stealth/assassination game written in Go, rendered via a custom Unicode cell grid on top of [Ebiten v2](https://ebitengine.org/).

---

## Gameplay Hotkeys

| Key | Action |
|---|---|
| `WASD` | Move |
| `Shift` + `WASD` | Run |
| `Arrow Keys` | Peek / shift look direction |
| `CapsLock` | Toggle Sneak mode |
| `E` | Context Action (interact, pick up, pickpocket, knock) |
| `Space` | Use equipped item / Begin aiming |
| `V` | Use item at current peek tile |
| `R` | Assassination |
| `F` | Dive & Tackle |
| `Q` | Open Inventory |
| `X` | Drop equipped item |
| `C` | Holster equipped item |
| `Tab` | Toggle Look Mode (free cursor) |
| `Escape` | Pause / Cancel |

---

## Interactions

Context actions are performed at the tile you are currently **peeking toward** (Arrow Keys), or on the tile you are standing on.

Press `E` to trigger the available action. The status line shows what action is available at any time.

| Situation | Result |
|---|---|
| Item on tile | Pick up |
| NPC facing you | Talk / Dialogue |
| Sneaking behind unaware NPC | Pickpocket |
| Facing a plain wall | Knock (noise distraction) |
| Door | Open / Close |
| Locked door (key) | Unlock automatically if key is in inventory |
| Locked door / safe | Lockpick or crowbar with `V` |
| Container / Locker | Enter to hide; interact again to exit |
| Downed body | Drag (while sneaking or while carrying piano wire) |
| Clothes on downed body | Change disguise |
| Food / drink tile | Poison (with poison equipped + `V`) |
| Power outlet | Expose electricity (with screwdriver + `V`) |
| Exit marker | Leave mission (all targets must be eliminated) |

---

## Sneaking

Toggle sneak mode with `CapsLock`. Sneaking makes movement slower and quieter.

Additional sneak actions:

- **Wipe blood**: Equip a cleaning rag → sneak over a blood splatter to remove it.
- **Drag bodies**: Sneak over a downed body to start dragging it. Walk to move it.
  - You can also drag while walking if you have **piano wire** equipped.

---

## Aiming & Ranged Combat

Press `Space` to enter aim mode with a ranged or throwable item equipped.
- The targeting cursor follows your mouse.
- Left-click to fire / throw.
- Scoped weapons (sniper rifles) zoom into a narrow FoV.
- Press `Space` again to leave aim mode.

---

## Gamepad Controls

See [gamepad-controls.md](gamepad-controls.md) for full gamepad reference.

---

## Editor

The editor is accessed from the **Main Menu → Editor**.

### Editor Mode Keys

| Key | Mode |
|---|---|
| `F1` | Tiles |
| `F2` | Items |
| `F3` | Objects |
| `F4` | Actors |
| `F5` | Schedules |
| `F7` | Zones |
| `F8` | Stimuli |
| `F9` | Lights |
| `F10` | Prefabs |
| `F11` | Global Menu (Save / Load / New / Resize / Quit) |

Global menu quick keys: `s` Save, `l` Load, `q` Quit Editor.

### Navigation

| Key | Action |
|---|---|
| `Arrow Keys` | Scroll map view |
| `Alt` + `Arrow Keys` | Shift map content |
| `Shift` + `Arrow Keys` | Resize map |
| `Escape` | Reset selection / return to default state |
| `Tab` | Brush menu |

### Tile Mode (`F1`)

| Key | Action |
|---|---|
| `Space` | Open tile picker |
| `Left Click` | Place selected tile |
| `e` | Eye dropper — pick tile from map |

### Items Mode (`F2`)

| Key | Action |
|---|---|
| `Space` | Open item picker |
| `Left Click` | Place selected item |
| `b` | Toggle buried |
| `k` | Set key |
| `Backspace` | Delete selected item |

### Objects Mode (`F3`)

| Key | Action |
|---|---|
| `Space` | Open object picker |
| `Left Click` | Place selected object |
| `k` | Set key |
| `d` | Set lock difficulty |
| `Backspace` | Delete selected object |

### Actors Mode (`F4`)

| Key | Action |
|---|---|
| `a` | Add actor |
| `q` | Quick add actor |
| `t` | Toggle actor type (Civilian / Guard / Enforcer / Target) |
| `y` | Toggle target flag |
| `r` | Rename actor |
| `w` | Move actor |
| `d` | Adjust look direction |
| `f` | Select leader |
| `T` | Set team |
| `i` | Open actor inventory |
| `s` | Assign schedule |
| `Backspace` | Delete actor |

### Schedule / Task Mode (`F5`)

| Key | Action |
|---|---|
| `a` | Add task |
| `j` / `k` | Decrease / increase task time |
| `l` / `L` | Add / remove look direction |
| `r` | Rename schedule |
| `o` | Assign schedule to selected actor |
| `Backspace` | Delete task / schedule |

---

## Concepts

- One tile per thing — no multi-tile objects.
- One thing per tile — no overlapping things of the same kind.

---

## Features

### Movement & Stealth
- Move in 8 directions; walk, sneak, or run
- Peek and lean around corners without exposing yourself
- Dive-tackle to lunge two tiles and shove NPCs out of the way
- Sneak over a body to drag it to a better hiding spot
- Hide inside containers and lockers; walk in, walk out

### Kills & Takedowns
- Instant silent assassination with any bladed weapon (double assassination with two weapons)
- Melee takedown — non-lethal knock-out
- Strangle with piano wire (also lets you drag bodies while walking)
- Snap the neck of a downed actor
- Drown an NPC
- Push an actor over an edge or into a hazard

### Firearms
- Pistol, shotgun (5-pellet spread), assault rifle (full-auto), submachine gun (full-auto)
- Sniper rifle with scoped field-of-view
- Dart gun (silenced, ranged knock-out)
- Bow (slingshot pull mechanic; range scales with draw strength)
- All firearms can also pistol-whip / melee-strike in close range

### Thrown & Placed Items
- Throw any sharp or blunt weapon as a projectile
- Throw explosives or grenades; place proximity and poison mines
- Remote-detonated explosives and remote taser
- Throw loot or common items as noise distractions

### Poison
- Lethal injection, emetic injection, sleep injection
- Lace food or drink — victim ingests it on their schedule
- Poison grenades (lethal and sleep variants), poison mines
- Emetic poison causes NPCs to seek a bathroom, isolating them

### Explosives & Hazards
- Fused thrown explosive, remote-detonated charge, proximity mine
- Fire spreads to oil-slicked surfaces; chain combustion
- Electrocute NPCs by sabotaging a power outlet into a water-covered tile
- Create fuel leaks from objects to set up chain fires
- Falling-edge kills (push over a ledge)

### Tools & Equipment
- Mechanical and electronic lockpicks for doors and safes
- Crowbar to pry open locked doors
- Screwdriver and wrench for sabotage and device use
- Taser (non-lethal, melee range)
- Flashlight (dynamic light cone, narrow scoped view)
- Shovel to bury items so guards ignore them
- Cleaning rag to wipe blood spatters while sneaking
- Keys and keycards to unlock matching doors

### World Simulation
- Sound propagates in waves — footsteps, gunshots, explosions attract AI
- Vision cones with three suspicion levels
- Fire spreads tile-to-tile and burns actors over time
- Water floods adjacent tiles; electricity propagates through connected water
- Smoke and gas clouds disperse across an area
- Blood spatters remain on the ground until cleaned
- Zone-based trespassing — hostile zones trigger immediate AI response
- Dynamic and baked lighting; ambient light shifts across a full day/night cycle

### NPC Behaviour
- Guards patrol, investigate disturbances, and enter combat
- Civilians panic and flee from violence
- NPCs on daily schedules walk routes, wait, and react to their environment
- Guards clean up discovered weapons and bodies
- Emetic-poisoned NPCs leave their post to vomit
- Frenzy-poisoned NPCs attack anyone nearby
- NPCs report spotted illegal activity; allies share knowledge
- Open carry of illegal weapons raises suspicion
- Discovered bodies, blood, and weapons trigger investigations
- Trespassing in restricted zones is noticed and reported

### Mission Structure
- A built-in career that persists across missions
- Multiple start locations per map
- Post-mission stats: duration, kills, bodies found, spotted status
- Campaigns group maps into a story arc
- Full in-game map editor with tile, item, object, actor, schedule, zone, and light editing

---

## The One Thing

Be a master assassin. Identify your targets, eliminate them cleanly, leave no traces, and evade law enforcement.

**Verbs:** Sneak, hide, distract, kill, poison, drag, throw, shoot, lockpick, disguise.


