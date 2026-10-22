package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// villainRect is one measured source rectangle inside villain.png. The sheet is
// NOT on a uniform grid - every row carries its own frame count and its own
// pitch - so each frame gets an explicit box instead of sharing a cell size.
// Every value below was read off the alpha channel of assets/villain.png
// (500x500); rows 4 and 5 touch in the sheet and had to be split by the
// vertical extent of each column.
type villainRect struct{ X, Y, W, H int }

const (
	bossScale  = 2.0
	bossWidth  = 84.0
	bossHeight = 140.0

	bossHP     = 320
	bossSpeed  = 1.6
	bossRunSpd = 3.4
	bossRange  = 118.0
	bossStop   = 94.0
	bossDamage = 22
	bossWake   = 700.0
	bossScore  = 1500

	// The boss is penned into the last stretch of the level so it cannot chase
	// the warrior back through the whole stage, but the arena is wide enough
	// for it to actually charge out and meet you.
	bossArenaL = 3020.0
	bossArenaR = 3540.0
)

// villain.png draws the warden looking RIGHT - the run frames lean forward to
// right with the cape streaming behind on the left - which is the same way the
// goblin sheet faces. So the sprite has to be mirrored whenever the boss faces
// left. Getting this backwards is what makes it walk toward the player with
// its back turned.
const bossSheetFacesRight = true

// bossFlip reports whether the warden sprite must be mirrored for a given
// facing. The warrior sheet faces the other way, so it uses the opposite test.
func bossFlip(facingRight bool) bool { return facingRight != bossSheetFacesRight }

var (
	bossIdle = []villainRect{
		{6, 8, 30, 56}, {48, 8, 30, 56}, {90, 8, 30, 56}, {131, 8, 30, 56}, {173, 8, 30, 56},
	}
	bossWalk = []villainRect{
		{7, 71, 29, 56}, {48, 71, 29, 56}, {90, 71, 30, 56}, {131, 71, 29, 56},
		{175, 71, 29, 56}, {217, 71, 26, 56}, {256, 71, 31, 56}, {298, 71, 31, 56},
		{340, 71, 30, 56},
	}
	bossRun = []villainRect{
		{6, 133, 32, 56}, {49, 133, 38, 56}, {97, 133, 40, 56}, {148, 133, 42, 56},
		{202, 133, 44, 56}, {256, 133, 37, 56}, {301, 133, 37, 56}, {345, 133, 34, 56},
	}
	bossJump = []villainRect{
		{10, 188, 34, 64}, {61, 188, 40, 64}, {113, 188, 40, 64}, {170, 188, 31, 64},
	}
	bossFall = []villainRect{
		{6, 257, 46, 56}, {58, 257, 48, 56}, {115, 257, 49, 56},
	}
	bossSword = []villainRect{
		{6, 305, 46, 74}, {58, 305, 48, 74}, {115, 305, 49, 74}, {173, 305, 41, 74},
		{220, 305, 42, 74}, {271, 305, 48, 74}, {329, 305, 54, 74}, {394, 305, 45, 74},
		{449, 305, 41, 74},
	}
	bossHurt = []villainRect{
		{12, 381, 40, 60}, {72, 381, 41, 60}, {134, 381, 42, 60}, {197, 381, 41, 60},
	}
	bossDead = []villainRect{
		{16, 444, 38, 56}, {75, 444, 44, 56}, {134, 444, 51, 56},
		{197, 444, 59, 56}, {269, 444, 59, 56}, {340, 444, 60, 56},
	}
)

// bossDamageFrames marks which frames of the sword sweep actually connect, so
// the hitbox is synced to the animation instead of the other way round.
var bossDamageFrames = map[int]bool{4: true, 5: true, 6: true}

// Boss is the Dark Warden waiting at the end of the level.
type Boss struct {
	X, Y   float64
	VX, VY float64
	Width  float64
	Height float64

	HP    int
	MaxHP int

	Grounded     bool
	FacingRight  bool
	Active       bool
	State        string // idle, walk, run, jump, attack, hurt, dead
	AnimT        int
	Timer        int
	HitFlash     int
	AttackCool   int
	JumpCool     int
	HitDone      bool
	Dead         bool
	Defeated     bool
	lastHitSwing int
}

func newBoss(x, y float64) *Boss {
	return &Boss{
		X:           x,
		Y:           y,
		Width:       bossWidth,
		Height:      bossHeight,
		HP:          bossHP,
		MaxHP:       bossHP,
		Grounded:    true,
		FacingRight: false,
		State:       "idle",
		AttackCool:  50,
	}
}

// Box returns the boss collision box (feet anchored at Y).
func (b *Boss) Box() Rect {
	return Rect{b.X - b.Width/2, b.Y - b.Height, b.Width, b.Height}
}

// AttackBox returns the boss's sword hitbox while it is swinging.
func (b *Boss) AttackBox() Rect {
	const top = 18.0
	h := b.Height - top - 24
	if b.FacingRight {
		return Rect{b.X + 18, b.Y - b.Height + top, bossRange, h}
	}
	return Rect{b.X - 18 - bossRange, b.Y - b.Height + top, bossRange, h}
}

// frames returns the sprite strip matching the boss's current state.
func (b *Boss) frames() []villainRect {
	switch b.State {
	case "walk":
		return bossWalk
	case "run":
		return bossRun
	case "jump":
		if b.VY < 0 {
			return bossJump
		}
		return bossFall
	case "attack":
		return bossSword
	case "hurt":
		return bossHurt
	case "dead":
		return bossDead
	}
	return bossIdle
}

// frameTicks is how long each state holds one frame.
func (b *Boss) frameTicks() int {
	switch b.State {
	case "run":
		return 3
	case "attack":
		return 5
	case "hurt":
		return 4
	}
	return 6
}

// animFrame picks the current index inside a strip of n frames.
func (b *Boss) animFrame(n int) int {
	if n <= 0 {
		return 0
	}
	return (b.AnimT / b.frameTicks()) % n
}

func (b *Boss) Update(player *Player, game *Game) {
	if b.Dead {
		b.AnimT++
		b.applyPhysics(game.level)
		if b.AnimT > len(bossDead)*8 {
			b.Defeated = true
		}
		return
	}

	if b.HitFlash > 0 {
		b.HitFlash--
	}
	if b.AttackCool > 0 {
		b.AttackCool--
	}
	if b.JumpCool > 0 {
		b.JumpCool--
	}

	dx := player.X - b.X
	adist := math.Abs(dx)

	// The warden sleeps until the warrior steps into the arena.
	if !b.Active {
		if !player.Dead && adist < bossWake {
			b.Active = true
			game.shake = 12
		} else {
			b.VX = 0
			b.applyPhysics(game.level)
			b.AnimT++
			return
		}
	}

	if !player.Dead && b.State != "hurt" {
		if dx > 0 {
			b.FacingRight = true
		} else {
			b.FacingRight = false
		}
	}

	switch b.State {
	case "hurt":
		b.Timer++
		b.VX *= 0.86
		if b.Timer > 16 {
			b.State = "walk"
			b.Timer = 0
		}
	case "attack":
		b.Timer++
		b.VX = 0
		frames := b.frames()
		if bossDamageFrames[b.animFrame(len(frames))] && !b.HitDone {
			b.HitDone = true
			if !player.Dead && b.AttackBox().Intersects(player.Box()) {
				player.Damage(bossDamage, game)
			}
		}
		if b.Timer >= len(frames)*b.frameTicks()+6 {
			b.State = "walk"
			b.Timer = 0
			b.AttackCool = 46
		}
	case "jump":
		// Keep drifting toward the warrior while airborne.
		if !player.Dead {
			if dx > 0 {
				b.VX = math.Min(b.VX+0.25, bossRunSpd)
			} else {
				b.VX = math.Max(b.VX-0.25, -bossRunSpd)
			}
		}
		if b.Grounded && b.VY >= 0 {
			b.State = "walk"
		}
	default:
		if player.Dead {
			b.VX = 0
			break
		}
		if adist < bossRange && b.AttackCool == 0 {
			b.State = "attack"
			b.Timer = 0
			b.AnimT = 0
			b.HitDone = false
			break
		}
		// Leap when the warrior is above, or to close a big gap.
		if b.Grounded && b.JumpCool == 0 && (player.Y < b.Y-70 || adist > 520) {
			b.VY = -16.5
			b.Grounded = false
			b.State = "jump"
			b.JumpCool = 120
			break
		}
		if adist > bossStop {
			dir := 1.0
			if dx < 0 {
				dir = -1
			}
			if adist > 420 {
				b.State = "run"
				b.VX = dir * bossRunSpd
			} else {
				b.State = "walk"
				b.VX = dir * bossSpeed
			}
		} else {
			b.State = "idle"
			b.VX = 0
		}
	}

	b.applyPhysics(game.level)

	// Pinned inside the arena.
	if b.X < bossArenaL {
		b.X = bossArenaL
		if b.VX < 0 {
			b.VX = 0
		}
	}
	if b.X > bossArenaR {
		b.X = bossArenaR
		if b.VX > 0 {
			b.VX = 0
		}
	}

	b.AnimT++
}

// applyPhysics gives the boss the same gravity and terrain rules as everything
// else, so it lands on platforms and can be knocked into a pit.
func (b *Boss) applyPhysics(level *Level) {
	b.VY += gravity
	if b.VY > 18 {
		b.VY = 18
	}

	b.X += b.VX
	box := b.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if b.VX > 0 {
				b.X = pl.X - b.Width/2
			} else if b.VX < 0 {
				b.X = pl.X + pl.W + b.Width/2
			}
			b.VX = 0
			box = b.Box()
		}
	}

	b.Y += b.VY
	b.Grounded = false
	box = b.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if b.VY > 0 {
				b.Y = pl.Y
				b.VY = 0
				b.Grounded = true
			} else if b.VY < 0 {
				b.Y = pl.Y + pl.H + b.Height
				b.VY = 0
			}
			box = b.Box()
		}
	}
}

// Damage applies a hit and knocks the warden back.
func (b *Boss) Damage(amount int, dir float64, game *Game) {
	if b.Dead {
		return
	}
	b.HP -= amount
	b.HitFlash = 12
	b.VX = dir * 6
	if b.HP <= 0 {
		b.HP = 0
		b.Dead = true
		b.State = "dead"
		b.AnimT = 0
		b.VX = 0
		b.VY = 0
		game.onBossKilled()
		return
	}
	b.State = "hurt"
	b.Timer = 0
}

// Draw renders the boss with a centre-bottom anchor.
func (b *Boss) Draw(screen *ebiten.Image, sheet *ebiten.Image, camX float64) {
	if b.HitFlash > 0 && b.HitFlash%4 < 2 {
		return
	}
	frames := b.frames()
	idx := b.animFrame(len(frames))
	if b.State == "dead" {
		idx = b.AnimT / 7
		if idx > len(frames)-1 {
			idx = len(frames) - 1
		}
	}
	drawSub(screen, sheet, frames[idx], b.X-camX, b.Y, bossFlip(b.FacingRight), bossScale)
}

// drawSub renders an arbitrary sub-rectangle of a sheet with a centre-bottom
// anchor. villain.png is not on a uniform grid, so the frame carries its own
// source box instead of a column/row pair.
func drawSub(dst *ebiten.Image, sheet *ebiten.Image, r villainRect, x, y float64, flip bool, scale float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(-float64(r.W)/2*scale, -float64(r.H)*scale)
	if flip {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Translate(x, y)
	sub := sheet.SubImage(image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)).(*ebiten.Image)
	dst.DrawImage(sub, op)
}
