package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	enemyCellW  = 48
	enemyCellH  = 48
	enemyHP     = 60
	enemySpeed  = 1.3
	enemyRange  = 74.0
	enemyStop   = 60.0
	enemyDamage = 12

	// edgeProbe is how far ahead of its feet a goblin looks for floor before
	// committing to a step.
	edgeProbe = 14.0
)

// Enemy is a patrolling goblin that chases and attacks the player.
type Enemy struct {
	X, Y   float64
	VX, VY float64
	Width  float64
	Height float64

	SpawnX, SpawnY float64
	Grounded       bool

	HP    int
	MaxHP int

	FacingRight  bool
	State        string // walk, attack, hurt, dead
	Timer        int
	AnimT        int
	Frame        int
	HitFlash     int
	AttackCool   int
	Dead         bool
	Remove       bool
	lastHitSwing int
}

func newEnemy(x, y float64) *Enemy {
	return &Enemy{
		X:           x,
		Y:           y,
		SpawnX:      x,
		SpawnY:      y,
		Width:       40,
		Height:      70,
		HP:          enemyHP,
		MaxHP:       enemyHP,
		FacingRight: false,
		Grounded:    true,
		State:       "walk",
	}
}

// Box returns the enemy collision box (feet anchored at Y).
func (e *Enemy) Box() Rect {
	return Rect{e.X - e.Width/2, e.Y - e.Height, e.Width, e.Height}
}

// AttackBox returns the enemy's melee hitbox while attacking.
func (e *Enemy) AttackBox() Rect {
	if e.FacingRight {
		return Rect{e.X + 6, e.Y - e.Height, enemyRange, e.Height}
	}
	return Rect{e.X - 6 - enemyRange, e.Y - e.Height, enemyRange, e.Height}
}

func (e *Enemy) Update(player *Player, game *Game) {
	if e.Dead {
		e.AnimT++
		e.applyPhysics(game.level)
		if e.AnimT >= 8 {
			e.Remove = true
		}
		return
	}

	if e.HitFlash > 0 {
		e.HitFlash--
	}
	if e.AttackCool > 0 {
		e.AttackCool--
	}

	dx := player.X - e.X
	adist := math.Abs(dx)
	if dx > 0 {
		e.FacingRight = true
	} else {
		e.FacingRight = false
	}

	switch e.State {
	case "hurt":
		e.Timer++
		e.VX *= 0.86
		if e.Timer > 14 {
			e.State = "walk"
			e.Timer = 0
		}
	case "attack":
		e.Timer++
		e.VX = 0
		if e.Timer == 10 {
			if player.Dead {
				break
			}
			if e.AttackBox().Intersects(player.Box()) {
				player.Damage(enemyDamage, game)
			}
		}
		if e.Timer >= 18 {
			e.State = "walk"
			e.Timer = 0
			e.AttackCool = 34
		}
	default: // walk / idle
		if player.Dead {
			e.VX = 0
			break
		}
		if adist < enemyRange && e.AttackCool == 0 {
			e.State = "attack"
			e.Timer = 0
			break
		}
		if adist < 760 && adist > enemyStop {
			dir := 1.0
			if dx < 0 {
				dir = -1
			}
			// A goblin refuses to walk off a ledge - it paces at the rim
			// instead of tumbling into every pit. Knockback can still send
			// one over the edge.
			if e.Grounded && !groundAhead(game.level, e.X+dir*(e.Width/2+edgeProbe), e.Y) {
				e.VX = 0
			} else {
				e.VX = dir * enemySpeed
			}
		} else {
			e.VX = 0
		}
	}

	// Friction / knockback decay.
	if e.State != "walk" {
		e.VX *= 0.8
	}

	e.applyPhysics(game.level)

	// Body separation: the goblin may not stand inside the warrior.
	if !player.Dead {
		if pbox := player.Box(); e.Box().Intersects(pbox) {
			if e.X < player.X {
				e.X = player.X - playerWidth/2 - e.Width/2
			} else {
				e.X = player.X + playerWidth/2 + e.Width/2
			}
		}
	}

	// Fell into a pit: same fate as the player.
	if e.Y > game.level.Height+120 {
		e.die(game)
	}

	// Animate.
	e.AnimT++
	switch e.State {
	case "attack":
		e.Frame = 6 + (e.AnimT/6)%2
	case "hurt":
		e.Frame = 0
	default:
		if e.VX != 0 {
			e.Frame = (e.AnimT / 7) % 6
		} else {
			e.Frame = 0
		}
	}
}

// applyPhysics moves the goblin with gravity and resolves it against the level
// exactly the way the player is resolved, so pits and walls affect it too.
func (e *Enemy) applyPhysics(level *Level) {
	e.VY += gravity
	if e.VY > 18 {
		e.VY = 18
	}

	// Horizontal.
	e.X += e.VX
	if e.X < e.Width/2 {
		e.X = e.Width / 2
	}
	if e.X > level.Width-e.Width/2 {
		e.X = level.Width - e.Width/2
	}
	box := e.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if e.VX > 0 {
				e.X = pl.X - e.Width/2
			} else if e.VX < 0 {
				e.X = pl.X + pl.W + e.Width/2
			}
			e.VX = 0
			box = e.Box()
		}
	}

	// Vertical.
	e.Y += e.VY
	e.Grounded = false
	box = e.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if e.VY > 0 {
				e.Y = pl.Y
				e.VY = 0
				e.Grounded = true
			} else if e.VY < 0 {
				e.Y = pl.Y + pl.H + e.Height
				e.VY = 0
			}
			box = e.Box()
		}
	}
}

// groundAhead reports whether a solid surface would support a foot planted at
// (x, y).
func groundAhead(level *Level, x, y float64) bool {
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if x < pl.X || x > pl.X+pl.W {
			continue
		}
		if y >= pl.Y-4 && y <= pl.Y+pl.H+12 {
			return true
		}
	}
	return false
}

// die marks the enemy defeated and pays out the kill.
func (e *Enemy) die(game *Game) {
	if e.Dead {
		return
	}
	e.HP = 0
	e.Dead = true
	e.State = "dead"
	e.AnimT = 0
	e.VX = 0
	e.VY = 0
	game.onEnemyKilled()
}

// Damage applies damage to the enemy and knocks it back.
func (e *Enemy) Damage(amount int, dir float64, game *Game) {
	if e.Dead {
		return
	}
	e.HP -= amount
	e.HitFlash = 12
	e.VX = dir * 7
	if e.HP <= 0 {
		e.HP = 0
		e.die(game)
		return
	}
	e.State = "hurt"
	e.Timer = 0
}

// Draw renders the enemy with a centre-bottom anchor.
func (e *Enemy) Draw(screen *ebiten.Image, sheet *ebiten.Image, camX float64) {
	if e.HitFlash > 0 && e.HitFlash%4 < 2 {
		return
	}
	frame := e.Frame
	if e.Dead {
		frame = 8
		if e.AnimT > 8 {
			frame = 9
		}
	}
	x := e.X - camX
	drawCell(screen, sheet, frame, 0, enemyCellW, enemyCellH, x, e.Y, !e.FacingRight, 2)
}

// drawCell renders an arbitrary sheet cell with a centre-bottom anchor.
func drawCell(dst *ebiten.Image, sheet *ebiten.Image, col, row, cw, ch int, x, y float64, flip bool, scale float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(-float64(cw)/2*scale, -float64(ch)*scale)
	if flip {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Translate(x, y)
	sub := sheet.SubImage(image.Rect(col*cw, row*ch, col*cw+cw, row*ch+ch)).(*ebiten.Image)
	dst.DrawImage(sub, op)
}
