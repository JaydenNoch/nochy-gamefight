package main

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/player.png
var playerPNG []byte

//go:embed assets/background.png
var backgroundPNG []byte

//go:embed assets/tiles.png
var tilesPNG []byte

//go:embed assets/enemy.png
var enemyPNG []byte

//go:embed assets/villain.png
var villainPNG []byte

// Assets bundles every image the game needs.
type Assets struct {
	Player     *ebiten.Image
	Background *ebiten.Image
	Tiles      *ebiten.Image
	Enemy      *ebiten.Image
	Villain    *ebiten.Image
}

func loadImage(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	// The default draw filter is already nearest-neighbour, which keeps the
	// sprite sheet's hard pixel edges when the window is scaled up.
	return ebiten.NewImageFromImage(img)
}

func loadAssets() *Assets {
	return &Assets{
		Player:     loadImage(playerPNG),
		Background: loadImage(backgroundPNG),
		Tiles:      loadImage(tilesPNG),
		Enemy:      loadImage(enemyPNG),
		Villain:    loadImage(villainPNG),
	}
}
