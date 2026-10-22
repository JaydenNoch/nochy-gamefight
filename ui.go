package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var textCache = map[string]*ebiten.Image{}

// drawText renders cached text scaled up for a crisp pixel-art UI.
func drawText(screen *ebiten.Image, s string, x, y float64, c color.RGBA, scale float64) {
	key := fmt.Sprintf("%s|%d", s, c)
	img, ok := textCache[key]
	if !ok {
		tmp := ebiten.NewImage(640, 28)
		ebitenutil.DebugPrint(tmp, s)
		img = tmp
		textCache[key] = img
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	screen.DrawImage(img, op)
}

func bar(screen *ebiten.Image, x, y, w, h float64, ratio float64, fill color.RGBA, back color.RGBA) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(w), float32(h), back, false)
	if ratio > 0 {
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(w*ratio), float32(h), fill, false)
	}
	vector.StrokeRect(screen, float32(x), float32(y), float32(w), float32(h), 2, color.RGBA{20, 20, 24, 255}, false)
}

// drawHUD draws the health bar, score and control hints.
func drawHUD(screen *ebiten.Image, g *Game) {
	// Health bar.
	bar(screen, 16, 16, 240, 22, float64(g.player.HP)/float64(g.player.MaxHP),
		color.RGBA{206, 60, 60, 255}, color.RGBA{40, 26, 28, 255})
	drawText(screen, fmt.Sprintf("HP %d/%d", g.player.HP, g.player.MaxHP), 22, 21,
		color.RGBA{255, 240, 230, 255}, 1.2)

	// Sword indicator.
	swordLabel := "SWORD: OFF  (K to equip)"
	swordColor := color.RGBA{180, 180, 190, 255}
	if g.player.Sword {
		swordLabel = "SWORD: ON   (K to holster)"
		swordColor = color.RGBA{240, 220, 120, 255}
	}
	drawText(screen, swordLabel, 16, 46, swordColor, 1.2)

	// Score + coins.
	drawText(screen, fmt.Sprintf("SCORE  %d", g.score), 720, 18, color.RGBA{250, 240, 200, 255}, 1.6)
	drawText(screen, fmt.Sprintf("COINS  %d", g.player.Gold), 760, 42, color.RGBA{250, 210, 90, 255}, 1.2)

	// Enemies remaining.
	drawText(screen, fmt.Sprintf("ENEMIES LEFT  %d", g.enemiesLeft()), 700, 64, color.RGBA{200, 210, 220, 255}, 1.1)

	// The warden's bar appears once it has noticed the warrior.
	if g.boss != nil && g.boss.Active && !g.boss.Dead {
		const w = 360.0
		x := (screenWidth - w) / 2
		bar(screen, x, 22, w, 16, float64(g.boss.HP)/float64(g.boss.MaxHP),
			color.RGBA{156, 44, 62, 255}, color.RGBA{34, 20, 26, 255})
		drawText(screen, "THE DARK WARDEN", x+8, 42, color.RGBA{226, 186, 196, 255}, 1.1)
	}

	// Controls.
	drawText(screen, "A/D move   SHIFT run   SPACE jump   S slide   Q/E turn   J/Click attack   K sword   ESC pause",
		16, 512, color.RGBA{190, 200, 210, 255}, 1.0)
}

// drawPause draws the pause overlay.
func drawPause(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 0, 0, screenWidth, screenHeight, color.RGBA{8, 10, 18, 190}, false)
	vector.DrawFilledRect(screen, screenWidth/2-260, screenHeight/2-120, 520, 240, color.RGBA{24, 28, 44, 255}, false)
	vector.StrokeRect(screen, screenWidth/2-260, screenHeight/2-120, 520, 240, 3, color.RGBA{120, 140, 180, 255}, false)
	drawText(screen, "PAUSED", screenWidth/2-70, screenHeight/2-86, color.RGBA{250, 240, 200, 255}, 3)
	drawText(screen, "ESC  - resume", screenWidth/2-120, screenHeight/2-30, color.RGBA{210, 220, 230, 255}, 1.6)
	drawText(screen, "R    - restart level", screenWidth/2-120, screenHeight/2+6, color.RGBA{210, 220, 230, 255}, 1.6)
	drawText(screen, "A/D move  SHIFT run  SPACE jump", screenWidth/2-120, screenHeight/2+42, color.RGBA{170, 180, 190, 255}, 1.2)
	drawText(screen, "S slide  Q/E turn  K sword  J/Click attack", screenWidth/2-120, screenHeight/2+64, color.RGBA{170, 180, 190, 255}, 1.2)
}

// Approximate advance of one glyph in the debug font at scale 1. Used to
// centre labels inside the restart button without measuring pixels.
const glyphW = 6.25

// restartButtonRect is the clickable restart button on the end screens.
func restartButtonRect() Rect {
	return Rect{screenWidth/2 - 100, screenHeight/2 + 34, 200, 52}
}

// drawRestartButton draws the restart button and highlights it while the
// cursor is over it.
func drawRestartButton(screen *ebiten.Image) {
	b := restartButtonRect()
	mx, my := ebiten.CursorPosition()
	hover := b.Contains(float64(mx), float64(my))

	fill := color.RGBA{48, 56, 80, 255}
	border := color.RGBA{24, 28, 42, 255}
	if hover {
		fill = color.RGBA{94, 114, 160, 255}
		border = color.RGBA{252, 232, 160, 255}
	}
	vector.DrawFilledRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), fill, false)
	vector.StrokeRect(screen, float32(b.X), float32(b.Y), float32(b.W), float32(b.H), 3, border, false)

	const label = "RESTART"
	const hint = "or press R"
	drawText(screen, label, b.X+(b.W-float64(len(label))*glyphW*1.8)/2, b.Y+8,
		color.RGBA{246, 242, 230, 255}, 1.8)
	drawText(screen, hint, b.X+(b.W-float64(len(hint))*glyphW)/2, b.Y+32,
		color.RGBA{178, 188, 204, 255}, 1.0)
}

// drawGameOver draws the defeat screen with a restart button.
func drawGameOver(screen *ebiten.Image, g *Game) {
	vector.DrawFilledRect(screen, 0, 0, screenWidth, screenHeight, color.RGBA{10, 6, 8, 210}, false)
	vector.DrawFilledRect(screen, screenWidth/2-300, screenHeight/2-140, 600, 260, color.RGBA{40, 16, 20, 255}, false)
	vector.StrokeRect(screen, screenWidth/2-300, screenHeight/2-140, 600, 260, 3, color.RGBA{200, 70, 70, 255}, false)
	drawText(screen, "YOU HAVE FALLEN", screenWidth/2-150, screenHeight/2-108, color.RGBA{240, 90, 90, 255}, 3)
	drawText(screen, fmt.Sprintf("FINAL SCORE   %d", g.score), screenWidth/2-110, screenHeight/2-56,
		color.RGBA{250, 240, 200, 255}, 2)
	drawText(screen, fmt.Sprintf("COINS COLLECTED   %d", g.player.Gold), screenWidth/2-130, screenHeight/2-24,
		color.RGBA{250, 210, 90, 255}, 1.4)
	drawRestartButton(screen)
}

// drawVictory draws the level-complete screen.
func drawVictory(screen *ebiten.Image, g *Game) {
	vector.DrawFilledRect(screen, 0, 0, screenWidth, screenHeight, color.RGBA{6, 12, 10, 205}, false)
	vector.DrawFilledRect(screen, screenWidth/2-300, screenHeight/2-140, 600, 260, color.RGBA{16, 34, 26, 255}, false)
	vector.StrokeRect(screen, screenWidth/2-300, screenHeight/2-140, 600, 260, 3, color.RGBA{110, 200, 130, 255}, false)
	drawText(screen, "LEVEL CLEARED", screenWidth/2-160, screenHeight/2-108, color.RGBA{150, 235, 160, 255}, 3)
	drawText(screen, fmt.Sprintf("FINAL SCORE   %d", g.score), screenWidth/2-110, screenHeight/2-56,
		color.RGBA{250, 240, 200, 255}, 2)
	drawText(screen, fmt.Sprintf("COINS COLLECTED   %d", g.player.Gold), screenWidth/2-130, screenHeight/2-24,
		color.RGBA{250, 210, 90, 255}, 1.4)
	drawRestartButton(screen)
}
