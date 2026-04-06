# Player Actions

## Movement & Navigation
| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Move (cardinal) | W / A / S / D | D-Pad |
| Run | Move keys rapidly | D-Pad rapidly |
| Sprint | (hold Shift + move) | Hold R1 + D-Pad |
| Sneak (toggle) | CapsLock | Circle (right stick centered) |
| Peek / look around corner | T / F / G / H | Left analog stick |
| Select adjacent target tile | Action keys (I/J/K/L) | Right analog stick (8-directional) |

## Item Management
| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Open inventory ring menu | Q | L1 |
| Drop equipped item | X | Left stick click (L3) |
| Put item away (holster) | C | Right stick click (R3) |

## Combat & Item Use

### Melee / Self-apply (Square button or action keys)
| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Use item at adjacent tile | Action key (I/J/K/L) toward tile | **Square** + right-stick direction |
| Use item on self (e.g. arm mine) | E (same-tile) | **Square** (right stick centered) |
| Bare-hand melee takedown on NPC | Action key toward NPC (no item) | **Square** + right-stick direction (no item) |
| Prod / push actor (bare hands) | E or action key toward actor | **Square** + direction (no NPC on tile) |

### Ranged Aiming (L2 + right stick → R2 to fire)
Hold **L2** while a throwable or ranged weapon is equipped to switch the right
analog stick into ranged aiming mode.

| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Enter aiming mode | Space | Hold **L2** (ranged/throwable item required) |
| Move aim cursor | Mouse move | Right analog stick (while L2 held) |
| Fire / throw at aimed position | Left-click in aiming mode | **R2** |

### Special Actions
| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Assassination (piercing weapon adjacent) | — (context action) | **Triangle** |
| Dive & Tackle | — | **Circle** + right-stick direction |
| Interact / Context action | E or action key | **Cross (X)** + right-stick direction |
| Start dialogue with NPC | Action key toward NPC | **Cross** + direction (while walking) |
| Pickpocket NPC | — | **Cross** + direction (while **sneaking**, behind unaware NPC) |

## Dive & Tackle (Circle + right stick)
Performs a rapid dash in the chosen direction (much faster than sprinting):

- **Both tiles empty** → player dives two tiles instantly (jump/dive).
- **NPC on tile 1 or 2** → NPC is prodded/pushed in that direction (tackle);
  player moves to the first free tile.

## Pickpocket (Cross while sneaking)
Requirements (all must be met):
1. Player is **sneaking**.
2. Player is on the tile **directly behind** the NPC (opposite of their facing direction).
3. NPC is in an **unaware state** — not in combat, not investigating, not searching, not panicking.
4. NPC has at least one item in inventory that is **not their equipped item**.

On success, one random qualifying item is transferred to the player's inventory.

## Item Use Details
Square (or keyboard action keys) selects the item action based on range and item type:

| Item / Situation | No Direction (self) | Adjacent Direction |
|---|---|---|
| No item equipped | — | Melee takedown (NPC) / Prod (empty tile) |
| Item with SelfUse | Arm/place item | — |
| Item with MeleeAttack | — | Melee attack |
| Item with MeleeTool | — | Tool use (e.g. poison container) |
| Ranged/throwable item | Use L2 + right stick + R2 | — |

## Directional Context Actions (Cross / Action keys)
Triggered by Cross + right-stick direction (gamepad) or action key (I/J/K/L) toward
an adjacent tile, or **E** for the player's own tile. Available action depends on tile
contents and situation:

| Action | Trigger condition |
|--------|-------------------|
| **Pick up item** | Item on the tile |
| **Exit level** | At exit tile; all targets dead |
| **Poison food/drink** | Poison equipped; food/drink container on tile |
| **Expose electricity** | Power socket on tile; no voltage yet |
| **Overflow sink** | Sink on tile; no water yet |
| **Strangle (piano wire)** | Piano Wire equipped; active unsuspicious NPC adjacent |
| **Melee takedown** | Active NPC adjacent (bare hands via Square) |
| **Snap neck** | Sleeping/downed NPC on same tile |
| **Change clothes** | Downed NPC with clothing on same tile |
| **Push over edge** | NPC adjacent to a lethal tile |
| **Drown in toilet** | NPC adjacent to toilet, not alert/in combat |
| **Start dialogue** | Scriptable NPC within 2 tiles (Cross while walking) |
| **Pickpocket** | Unaware NPC; player directly behind them; sneaking (Cross while sneaking) |

## UI / System
| Action | Keyboard | Gamepad |
|--------|----------|---------|
| Pause menu | Escape | — |
| Confirm (menus) | Enter | Cross (X) |
| Toggle full map (debug) | F1 | — |
| Show focused actor info | F2 | — |
| Trigger training alert | F3 | — |

