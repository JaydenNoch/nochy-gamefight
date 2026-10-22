package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	frameW    = 80
	frameH    = 64
	sheetCols = 10
)

// AnimID identifies one animation state of the warrior.
type AnimID int

const (
	AnimIdle AnimID = iota
	AnimWalk
	AnimRun
	AnimJump
	AnimSlide
	AnimIdleSword
	AnimWalkSword
	AnimRunSword
	AnimJumpSword
	AnimSlideSword
	AnimAttack
	AnimDeath
	AnimDeathSword
)

// animIndex maps every animation state to the row (animation index) it
// occupies inside the "GandalfHardcore Warrior.png" sprite sheet.
// The sheet is an 800x1088 grid of 80x64 cells laid out as 17 rows x 10
// columns; the numbers below are the exact indices supplied with the asset.
var animIndex = map[AnimID]int{
	AnimIdle:       0,  // idle / standing
	AnimWalk:       1,  // walking
	AnimRun:        3,  // running
	AnimJump:       4,  // jumping
	AnimSlide:      6,  // sliding
	AnimIdleSword:  7,  // standing with sword
	AnimWalkSword:  8,  // walking with sword
	AnimRunSword:   9,  // running with sword
	AnimJumpSword:  11, // jumping with sword
	AnimSlideSword: 13, // sliding with sword
	AnimAttack:     14, // sword attack
	AnimDeath:      15, // defeated / death
	AnimDeathSword: 16, // defeated / death with sword
}

// animFrames is the number of frames each animation contains in the sheet.
var animFrames = map[AnimID]int{
	AnimIdle:       5,
	AnimWalk:       8,
	AnimRun:        8,
	AnimJump:       4,
	AnimSlide:      4,
	AnimIdleSword:  5,
	AnimWalkSword:  8,
	AnimRunSword:   8,
	AnimJumpSword:  4,
	AnimSlideSword: 4,
	AnimAttack:     6,
	AnimDeath:      8,
	AnimDeathSword: 10,
}

// attackDamageFrames lists the attack frames (0-based) during which the
// sword hitbox actually deals damage. Damage is synchronised with the swing
// instead of being applied on every frame of the animation.
var attackDamageFrames = map[int]bool{1: true, 2: true, 3: true, 4: true}

// Animator drives playback of a single animation on top of the sprite sheet.
type Animator struct {
	sheet *ebiten.Image
	id    AnimID
	frame int
	timer int
	fps   int
	done  bool
}

func newAnimator(sheet *ebiten.Image) *Animator {
	return &Animator{sheet: sheet, id: AnimIdle, fps: 8}
}

func (a *Animator) loops() bool {
	switch a.id {
	case AnimIdle, AnimWalk, AnimRun, AnimIdleSword, AnimWalkSword, AnimRunSword:
		return true
	}
	return false
}

func (a *Animator) speed() int {
	switch a.id {
	case AnimAttack, AnimJump, AnimSlide, AnimJumpSword, AnimSlideSword:
		return 11
	case AnimRun, AnimRunSword:
		return 12
	case AnimDeath, AnimDeathSword:
		return 7
	default:
		return a.fps
	}
}

// Set switches to a new animation. Re-selecting the animation that is already
// playing is a no-op, which prevents a held attack button from restarting the
// swing every frame.
func (a *Animator) Set(id AnimID) {
	if a.id == id {
		return
	}
	a.id = id
	a.frame = 0
	a.timer = 0
	a.done = false
}

func (a *Animator) Force(id AnimID) {
	a.id = id
	a.frame = 0
	a.timer = 0
	a.done = false
}

func (a *Animator) Update() {
	n := animFrames[a.id]
	if n <= 0 {
		return
	}
	a.timer++
	if a.timer < 60/a.speed() {
		return
	}
	a.timer = 0
	if a.loops() {
		a.frame = (a.frame + 1) % n
		return
	}
	a.frame++
	if a.frame >= n {
		a.frame = n - 1
		a.done = true
	}
}

func (a *Animator) ID() AnimID           { return a.id }
func (a *Animator) Frame() int           { return a.frame }
func (a *Animator) Done() bool           { return a.done }
func (a *Animator) Sheet() *ebiten.Image { return a.sheet }

// Cell returns the sub-image for the current frame.
func (a *Animator) Cell() *ebiten.Image {
	return a.sheet.SubImage(image.Rect(a.frame*frameW, animIndex[a.id]*frameH,
		a.frame*frameW+frameW, animIndex[a.id]*frameH+frameH)).(*ebiten.Image)
}

// drawSprite renders a sheet cell with a centre-bottom anchor, optionally
// mirrored horizontally.
func drawSprite(dst *ebiten.Image, sheet *ebiten.Image, col, row int, x, y float64, flip bool, scale float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(-float64(frameW)/2*scale, -float64(frameH)*scale)
	if flip {
		op.GeoM.Scale(-1, 1)
	}
	op.GeoM.Translate(x, y)
	sub := sheet.SubImage(image.Rect(col*frameW, row*frameH, col*frameW+frameW, row*frameH+frameH)).(*ebiten.Image)
	dst.DrawImage(sub, op)
}
