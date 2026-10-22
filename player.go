package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	gravity        = 0.6
	jumpVelocity   = -14.5
	walkSpeed      = 3.0
	runSpeed       = 5.4
	slideSpeed     = 7.6
	slideDuration  = 26
	attackCooldown = 14
	playerWidth    = 30.0
	playerHeight   = 90.0
	attackRange    = 96.0
	attackHeight   = 84.0
	attackDamage   = 34
)

// Player is the controllable warrior.
type Player struct {
	X, Y   float64
	VX, VY float64

	FacingRight bool
	HP          int
	MaxHP       int
	Gold        int

	Grounded     bool
	Running      bool
	Sword        bool
	Sliding      bool
	SlideTimer   int
	SlideReady   bool
	Attacking    bool
	AttackTimer  int
	AttackCool   int
	AttackActive bool
	Invuln       int
	Dead         bool
	DeathTimer   int
	HitFlash     int

	Anim *Animator
}

func newPlayer(sheet *ebiten.Image) *Player {
	return &Player{
		X:           120,
		Y:           470,
		FacingRight: true,
		HP:          100,
		MaxHP:       100,
		Grounded:    true,
		Invuln:      40,
		Anim:        newAnimator(sheet),
	}
}

// Box returns the player's collision box (feet anchored at Y).
func (p *Player) Box() Rect {
	return Rect{p.X - playerWidth/2, p.Y - playerHeight, playerWidth, playerHeight}
}

// AttackBox returns the sword hitbox while an attack is active.
func (p *Player) AttackBox() Rect {
	if p.FacingRight {
		return Rect{p.X + 12, p.Y - attackHeight, attackRange, attackHeight}
	}
	return Rect{p.X - 12 - attackRange, p.Y - attackHeight, attackRange, attackHeight}
}

// SwordAnimation returns the armed version of an unarmed animation.
func swordVersion(id AnimID) AnimID {
	switch id {
	case AnimIdle:
		return AnimIdleSword
	case AnimWalk:
		return AnimWalkSword
	case AnimRun:
		return AnimRunSword
	case AnimJump:
		return AnimJumpSword
	case AnimSlide:
		return AnimSlideSword
	case AnimDeath:
		return AnimDeathSword
	}
	return id
}

// CurrentAnimation picks the animation that matches the player's state.
func (p *Player) CurrentAnimation() AnimID {
	if p.Dead {
		if p.Sword {
			return AnimDeathSword
		}
		return AnimDeath
	}
	if p.Attacking && p.Sword {
		return AnimAttack
	}
	if p.Sliding {
		if p.Sword {
			return AnimSlideSword
		}
		return AnimSlide
	}
	if !p.Grounded {
		if p.Sword {
			return AnimJumpSword
		}
		return AnimJump
	}
	if p.VX != 0 {
		if p.Running {
			if p.Sword {
				return AnimRunSword
			}
			return AnimRun
		}
		if p.Sword {
			return AnimWalkSword
		}
		return AnimWalk
	}
	if p.Sword {
		return AnimIdleSword
	}
	return AnimIdle
}

func (p *Player) Update(level *Level, game *Game) {
	if p.Dead {
		p.DeathTimer++
		p.VX = 0
		p.Anim.Set(p.CurrentAnimation())
		p.Anim.Update()
		p.applyGravity(level)
		return
	}

	if p.Invuln > 0 {
		p.Invuln--
	}
	if p.HitFlash > 0 {
		p.HitFlash--
	}
	if p.AttackCool > 0 {
		p.AttackCool--
	}

	// --- horizontal input -------------------------------------------------
	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	down := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	run := ebiten.IsKeyPressed(ebiten.KeyShift)

	p.VX = 0

	// Q/E turns the warrior around in place. Ignored while he is moving so a
	// direction key can never be overridden mid-stride - that is exactly what
	// makes a character moonwalk.
	if !p.Attacking && !p.Sliding && !p.Dead && !left && !right &&
		(inpututil.IsKeyJustPressed(ebiten.KeyQ) || inpututil.IsKeyJustPressed(ebiten.KeyE)) {
		p.FacingRight = !p.FacingRight
	}

	if p.Attacking {
		// Keep momentum small during an attack.
		p.VX = 0
	} else if p.Sliding {
		if p.FacingRight {
			p.VX = slideSpeed
		} else {
			p.VX = -slideSpeed
		}
	} else if left {
		p.VX = -walkSpeed
		if run {
			p.VX = -runSpeed
		}
		p.FacingRight = false
	} else if right {
		p.VX = walkSpeed
		if run {
			p.VX = runSpeed
		}
		p.FacingRight = true
	}

	// Running is derived from the same input that drives VX, so the animator
	// and the tests always agree on which locomotion animation is playing.
	p.Running = run && (left || right) && !p.Attacking && !p.Sliding

	// --- slide ------------------------------------------------------------
	if down && p.Grounded && !p.Sliding && !p.Attacking && p.SlideReady {
		p.Sliding = true
		p.SlideTimer = slideDuration
		p.SlideReady = false
	}
	if p.Sliding {
		p.SlideTimer--
		if p.SlideTimer <= 0 || !p.Grounded {
			p.Sliding = false
			p.SlideReady = true
		}
	}
	if !down && !p.Sliding {
		p.SlideReady = true
	}

	// --- jump -------------------------------------------------------------
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && p.Grounded && !p.Sliding && !p.Attacking {
		p.VY = jumpVelocity
		p.Grounded = false
	}

	// --- sword equip / unequip -------------------------------------------
	if inpututil.IsKeyJustPressed(ebiten.KeyK) && !p.Sliding && !p.Attacking {
		p.Sword = !p.Sword
	}

	// --- attack -----------------------------------------------------------
	if (inpututil.IsKeyJustPressed(ebiten.KeyJ) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)) &&
		p.Sword && !p.Sliding && !p.Attacking && p.AttackCool == 0 {
		p.Attacking = true
		p.AttackTimer = 0
		p.AttackActive = false
		p.AttackCool = attackCooldown
	}
	if p.Attacking {
		p.AttackTimer++
		// Damage window is synced to the swing frames.
		p.AttackActive = attackDamageFrames[p.Anim.Frame()] && p.Anim.ID() == AnimAttack
		if p.Anim.ID() == AnimAttack && p.Anim.Done() {
			p.Attacking = false
			p.AttackActive = false
		}
	}

	p.Anim.Set(p.CurrentAnimation())
	p.Anim.Update()

	// --- physics ----------------------------------------------------------
	p.moveAndCollide(level)

	// --- hazards ----------------------------------------------------------
	box := p.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() && box.Intersects(pl.Rect) && p.Invuln == 0 {
			p.Damage(12, game)
			if p.FacingRight {
				p.VX = -6
			} else {
				p.VX = 6
			}
			p.VY = -8
			p.Grounded = false
		}
	}

	// --- coins ------------------------------------------------------------
	for i := range level.Coins {
		c := &level.Coins[i]
		if c.W > 0 && box.Intersects(*c) {
			c.W = 0
			p.Gold++
			game.score += 50
		}
	}

	if p.Y > level.Height+120 {
		p.HP = 0
	}
	if p.HP <= 0 {
		p.HP = 0
		p.Dead = true
		p.DeathTimer = 0
		game.onPlayerDeath()
	}
}

func (p *Player) applyGravity(level *Level) {
	p.VY += gravity
	if p.VY > 18 {
		p.VY = 18
	}
}

func (p *Player) moveAndCollide(level *Level) {
	p.applyGravity(level)

	// Horizontal.
	p.X += p.VX
	if p.X < playerWidth/2 {
		p.X = playerWidth / 2
	}
	if p.X > level.Width-playerWidth/2 {
		p.X = level.Width - playerWidth/2
	}
	box := p.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if p.VX > 0 {
				p.X = pl.X - playerWidth/2
			} else if p.VX < 0 {
				p.X = pl.X + pl.W + playerWidth/2
			}
			p.VX = 0
			box = p.Box()
		}
	}

	// Vertical.
	p.Y += p.VY
	p.Grounded = false
	box = p.Box()
	for _, pl := range level.Platforms {
		if pl.Hazard() {
			continue
		}
		if box.Intersects(pl.Rect) {
			if p.VY > 0 {
				p.Y = pl.Y
				p.VY = 0
				p.Grounded = true
			} else if p.VY < 0 {
				p.Y = pl.Y + pl.H + playerHeight
				p.VY = 0
			}
			box = p.Box()
		}
	}
}

// Damage applies incoming damage with a brief invulnerability window.
func (p *Player) Damage(amount int, game *Game) {
	if p.Invuln > 0 || p.Dead {
		return
	}
	p.HP -= amount
	p.Invuln = 55
	p.HitFlash = 14
	game.shake = 6
	if p.HP < 0 {
		p.HP = 0
	}
}

// Draw renders the warrior with a centre-bottom anchor.
func (p *Player) Draw(screen *ebiten.Image, camX float64) {
	x := p.X - camX
	if p.HitFlash > 0 && p.HitFlash%4 < 2 {
		// blink while hurt
	} else {
		// The sheet draws the warrior facing LEFT (attack frame 4-5 shows the
		// sword swung forward-to-the-left), so he has to be mirrored whenever
		// he faces right. Getting this backwards makes him moonwalk.
		drawSprite(screen, p.Anim.Sheet(), p.Anim.Frame(), animIndex[p.Anim.ID()], x, p.Y, p.FacingRight, 2)
	}
}
