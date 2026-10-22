package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 960
	screenHeight = 540

	// deathAnimDelay lets the 8-frame death animation finish before the
	// defeat overlay appears.
	deathAnimDelay = 78
)

type gameState int

const (
	statePlaying gameState = iota
	statePaused
	stateGameOver
	stateVictory
)

// HitEffect is a short-lived particle spawned when something takes damage.
type HitEffect struct {
	X, Y   float64
	VX, VY float64
	Life   int
	Max    int
	Size   float64
	Color  color.RGBA
}

// Game owns the whole simulation and rendering of a single run.
type Game struct {
	state   gameState
	assets  *Assets
	level   *Level
	player  *Player
	enemies []*Enemy
	boss    *Boss

	cameraX       float64
	score         int
	shake         float64
	swingID       int
	prevAttacking bool
	deathTimer    int
	hits          []*HitEffect
	time          int
}

// NewGame creates a fresh game instance and loads all assets.
func NewGame() (*Game, error) {
	assets := loadAssets()
	g := &Game{
		state:  statePlaying,
		assets: assets,
	}
	g.reset()
	return g, nil
}

func (g *Game) reset() {
	g.level = NewLevel()
	g.player = newPlayer(g.assets.Player)
	g.enemies = nil
	for _, s := range g.level.Spawns {
		g.enemies = append(g.enemies, newEnemy(s.X, s.Y))
	}
	g.boss = newBoss(g.level.BossSpawn.X, g.level.BossSpawn.Y)
	g.cameraX = 0
	g.score = 0
	g.shake = 0
	g.swingID = 0
	g.hits = nil
	g.time = 0
	g.deathTimer = 0
	g.prevAttacking = false
	g.state = statePlaying
}

func (g *Game) enemiesLeft() int {
	n := 0
	for _, e := range g.enemies {
		if !e.Dead {
			n++
		}
	}
	return n
}

func (g *Game) onEnemyKilled() {
	g.score += 200
	g.spawnBurst(g.player.X, g.player.Y-50, color.RGBA{250, 210, 90, 255}, 10)
}

func (g *Game) onBossKilled() {
	g.score += bossScore
	g.shake = 16
	g.spawnBurst(g.boss.X, g.boss.Y-70, color.RGBA{245, 140, 100, 255}, 26)
}

func (g *Game) onPlayerDeath() {
	g.deathTimer = 0
	g.spawnBurst(g.player.X, g.player.Y-50, color.RGBA{235, 70, 70, 255}, 16)
}

func (g *Game) spawnBurst(x, y float64, c color.RGBA, n int) {
	for i := 0; i < n; i++ {
		a := math.Pi * 2 * float64(i) / float64(n)
		sp := 2 + float64(i%3)
		g.hits = append(g.hits, &HitEffect{
			X:     x,
			Y:     y,
			VX:    math.Cos(a) * sp,
			VY:    math.Sin(a)*sp - 1.5,
			Life:  26,
			Max:   26,
			Size:  3 + float64(i%3),
			Color: c,
		})
	}
}

// Update advances the simulation by one tick.
func (g *Game) Update() error {
	switch g.state {
	case statePaused:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.state = statePlaying
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			g.reset()
		}
		return nil
	case stateGameOver, stateVictory:
		if inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.reset()
			return nil
		}
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mx, my := ebiten.CursorPosition()
			if restartButtonRect().Contains(float64(mx), float64(my)) {
				g.reset()
			}
		}
		return nil
	}

	if !g.player.Dead && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = statePaused
		return nil
	}

	g.time++
	if g.shake > 0 {
		g.shake *= 0.85
		if g.shake < 0.4 {
			g.shake = 0
		}
	}

	g.player.Update(g.level, g)

	// A new swing gets a new id so each enemy is only hit once per attack.
	if g.player.Attacking && !g.prevAttacking {
		g.swingID++
	}
	g.prevAttacking = g.player.Attacking

	// Sword hits: only while the swing is in its damage frames.
	if g.player.Attacking && g.player.AttackActive {
		box := g.player.AttackBox()
		for _, e := range g.enemies {
			if e.Dead || e.lastHitSwing == g.swingID {
				continue
			}
			if box.Intersects(e.Box()) {
				e.lastHitSwing = g.swingID
				dir := 1.0
				if !g.player.FacingRight {
					dir = -1
				}
				e.Damage(attackDamage, dir, g)
				g.spawnBurst(e.X, e.Y-40, color.RGBA{255, 235, 150, 255}, 8)
			}
		}
		// The warden takes the same swing, once per swing.
		if b := g.boss; b != nil && !b.Dead && b.lastHitSwing != g.swingID {
			if box.Intersects(b.Box()) {
				b.lastHitSwing = g.swingID
				dir := 1.0
				if !g.player.FacingRight {
					dir = -1
				}
				b.Damage(attackDamage, dir, g)
				g.spawnBurst(b.X, b.Y-70, color.RGBA{255, 190, 150, 255}, 12)
			}
		}
	}

	g.boss.Update(g.player, g)
	for _, e := range g.enemies {
		e.Update(g.player, g)
	}
	// Remove finished corpses.
	alive := g.enemies[:0]
	for _, e := range g.enemies {
		if !e.Remove {
			alive = append(alive, e)
		}
	}
	g.enemies = alive

	// Hit particles.
	live := g.hits[:0]
	for _, h := range g.hits {
		h.Life--
		h.X += h.VX
		h.Y += h.VY
		h.VY += 0.25
		if h.Life > 0 {
			live = append(live, h)
		}
	}
	g.hits = live

	// Camera follows the player and clamps to the level bounds.
	target := g.player.X - screenWidth/2
	if target < 0 {
		target = 0
	}
	maxCam := g.level.Width - screenWidth
	if target > maxCam {
		target = maxCam
	}
	g.cameraX += (target - g.cameraX) * 0.12

	// Give the death animation time to finish before the defeat overlay.
	if g.player.Dead {
		g.deathTimer++
		if g.deathTimer > deathAnimDelay {
			g.state = stateGameOver
		}
		return nil
	}

	// Victory when every goblin is down and the warden has fallen.
	if g.enemiesLeft() == 0 && len(g.enemies) == 0 && g.boss.Dead {
		g.state = stateVictory
	}

	return nil
}

// Draw renders the whole frame.
func (g *Game) Draw(screen *ebiten.Image) {
	// Background with parallax.
	g.drawBackground(screen)

	ox := -g.cameraX
	if g.shake > 0 {
		ox += math.Sin(float64(g.time)*1.7) * g.shake
	}

	// Level geometry.
	g.drawLevel(screen, ox)

	// Coins.
	for i := range g.level.Coins {
		c := g.level.Coins[i]
		if c.W <= 0 {
			continue
		}
		drawCell(screen, g.assets.Tiles, TileCoin, 0, tileSize, tileSize, c.X+13+ox, c.Y+26, false, 0.85)
	}

	// Enemies.
	for _, e := range g.enemies {
		e.Draw(screen, g.assets.Enemy, g.cameraX)
	}

	// Final boss.
	g.boss.Draw(screen, g.assets.Villain, g.cameraX)

	// Player.
	g.player.Draw(screen, g.cameraX)

	// Hit particles.
	for _, h := range g.hits {
		a := float32(h.Life) / float32(h.Max)
		c := h.Color
		c.A = uint8(255 * a)
		vector.DrawFilledRect(screen, float32(h.X+ox-h.Size/2), float32(h.Y-h.Size/2),
			float32(h.Size), float32(h.Size), c, false)
	}

	drawHUD(screen, g)

	switch g.state {
	case statePaused:
		drawPause(screen)
	case stateGameOver:
		drawGameOver(screen, g)
	case stateVictory:
		drawVictory(screen, g)
	}
}

func (g *Game) drawBackground(screen *ebiten.Image) {
	bg := g.assets.Background
	bw := float64(bg.Bounds().Dx())
	off := math.Mod(g.cameraX*0.3, bw)
	for x := -off; x < screenWidth; x += bw {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(x, 0)
		screen.DrawImage(bg, op)
	}
}

func (g *Game) drawLevel(screen *ebiten.Image, ox float64) {
	for _, p := range g.level.Platforms {
		col := p.Tile
		if col < 0 || col >= tileCols {
			continue
		}
		// Tile the platform, snapped to the 32px grid.
		startX := math.Floor(p.X/tileSize) * tileSize
		for x := startX; x < p.X+p.W; x += tileSize {
			if x+tileSize < p.X {
				continue
			}
			for y := p.Y; y < p.Y+p.H; y += tileSize {
				drawCell(screen, g.assets.Tiles, col, 0, tileSize, tileSize, x+ox+tileSize/2, y+tileSize/2, false, 1)
			}
		}
	}
}

// Layout reports the fixed logical resolution.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}
