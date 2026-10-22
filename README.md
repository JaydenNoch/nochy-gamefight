# Gandalf Hardcore Warrior

A 2D action platformer written in Go with [Ebitengine](https://ebitengine.org). The warrior
starts unarmed, can draw a sword, and has to cut through twenty goblins spread across a
3600-pixel level before facing the Dark Warden at the end, collecting coins and avoiding
pits and spike traps along the way.

The character animation comes straight from `GandalfHardcore Warrior.png` — a 800x1088 sprite
sheet laid out as a 10x17 grid of 80x64 cells, where the row number *is* the animation index.
The boss uses `assets/villain.png` (500x500), which is **not** on a uniform grid: every row has
its own frame count and pitch, so each frame carries its own measured source rectangle
(`boss.go`).

## Run it

Requires Go 1.27+ (any platform Ebitengine supports; verified on Windows 11).

```bat
cd D:\fighting-game
go mod tidy
go run .
```

To produce a standalone executable:

```bat
go build -o gandalf.exe .
gandalf.exe
```

## Controls

| Key                | Action                       |
| ------------------ | ---------------------------- |
| `A` / `Left`       | Move left                    |
| `D` / `Right`      | Move right                   |
| `Shift`            | Run (while moving)           |
| `Space`            | Jump                         |
| `S` / `Down`       | Slide                        |
| `Q` / `E`          | Turn around on the spot       |
| `J` / Left mouse   | Sword attack                 |
| `K`                | Draw / holster the sword     |
| `Esc`              | Pause / resume               |
| `R` / `Enter`      | Restart (while paused or on an end screen) |
| Left mouse         | Click **RESTART** on the end screens |

## Gameplay

* **Combat is hit-synced.** A swing only damages during its active frames, and each swing
  carries an id so no enemy is ever hit twice by the same attack.
* **Sword must be drawn** (`K`) to attack; unarmed attacks are disabled.
* **Goblins** patrol, chase within a range, stop at striking distance, swing with a wind-up
  frame, get knocked back on hits and die after a few blows.
* **Goblins obey the terrain.** They have gravity, land on platforms, refuse to walk off a
  ledge (they pace at the rim instead), and can be knocked into a pit — where they fall and
  are finished off, exactly like the warrior.
* **The Dark Warden** waits at the end of the level and wakes up when the warrior comes within
  620 pixels. It has 320 HP, hits for 22, chases on foot, leaps over gaps, and its sword
  damage is synced to the middle of the swing animation. A boss health bar appears once it
  has noticed you.
* **Score** goes up for every goblin (+200), coin (+50) and the boss (+1500).
* **Spikes** hurt and bounce you; **gaps in the ground** are instant death.
* Clearing every goblin *and* the boss wins the level; losing all HP shows the defeat screen.

## Project layout

| File            | Contents                                                            |
| --------------- | ------------------------------------------------------------------- |
| `main.go`       | Window setup and entry point                                        |
| `game.go`       | Game state machine, world update, camera, parallax, render order    |
| `player.go`     | Warrior physics, input, states, attack hitbox                       |
| `enemy.go`      | Goblin AI (idle / chase / attack / hurt / death), physics, ledges   |
| `boss.go`       | The Dark Warden: measured frame boxes, AI, physics, sword hitbox    |
| `animation.go`  | Animation table, `Animator`, sprite-sheet draw helper               |
| `collision.go`  | `Rect` helpers used for every hitbox                                |
| `level.go`      | Level layout: ground strips, platforms, coins, spikes, spawns       |
| `ui.go`         | HUD, boss bar, pause menu, defeat/victory screens, restart button   |
| `assets.go`     | `go:embed` asset loader                                             |
| `game_test.go`  | Unit tests for the animation table, physics, AI and state machine   |
| `assets/`       | `player.png`, `background.png`, `tiles.png`, `enemy.png`, `villain.png` |

## Animation mapping

The sprite sheet rows are the animation ids. Nothing is ever played back at random or
restarted while a key is held.

| Animation            | Row | Frames |
| -------------------- | --- | ------ |
| Idle                 | 0   | 5      |
| Walk                 | 1   | 8      |
| Run                  | 3   | 8      |
| Jump                 | 4   | 4      |
| Slide                | 6   | 4      |
| Idle (sword drawn)   | 7   | 5      |
| Walk (sword drawn)   | 8   | 8      |
| Run (sword drawn)    | 9   | 8      |
| Jump (sword drawn)   | 11  | 4      |
| Slide (sword drawn)  | 13  | 4      |
| Attack               | 14  | 6      |
| Death                | 15  | 8      |
| Death (sword drawn)  | 16  | 10     |

## Tests

```bat
go test ./...
go vet ./...
gofmt -l .
```

The tests pin the sprite-sheet geometry (800x1088, 10x17 cells), every animation row and
frame count, the state-machine priority order (death > attack > slide > airborne >
locomotion), gravity/ground collision, the attack hitbox mirroring, goblin body separation
and the game-over / victory / restart transitions.

They also cover the newer work: twenty spawn points that all stand on solid ground, a goblin
refusing to walk off a ledge, a goblin falling into a pit and dying, the boss frame boxes
staying inside `villain.png` with a consistent foot line, the boss hitbox mirroring with its
facing, the boss sleeping until approached, the boss turning to face the warrior, a long
drive through every boss state, and a victory that now waits on the boss as well as the
goblins.
