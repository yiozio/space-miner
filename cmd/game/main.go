package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yiozio/space-miner/internal/game"
)

func main() {
	g := game.New()
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Space Miner")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
