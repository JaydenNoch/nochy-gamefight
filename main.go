package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game, err := NewGame()
	if err != nil {
		log.Fatal("failed to start game: ", err)
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gandalf Hardcore Warrior - 2D Action Platformer")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
