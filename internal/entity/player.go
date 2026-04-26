package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerRotateSpeed     = 0.06
	playerThrustAccel     = 0.15
	playerBoostMultiplier = 2.5
	playerMaxSpeed        = 8.0
	playerBoostMaxSpeed   = 14.0
)

// Player はプレイヤー機。Ship に操作と固有状態を加えたもの。
type Player struct {
	Ship
}

// NewPlayerPebble は初期機体「Pebble」のプレイヤーを生成する。
// 配置は docs/GAME_DESIGN.md「サンプル: スターター艇 Pebble」に対応。
func NewPlayerPebble() *Player {
	return &Player{
		Ship: Ship{
			Parts: []Part{
				{Kind: PartThruster, GX: 0, GY: -1},
				{Kind: PartGun, GX: -1, GY: 0},
				{Kind: PartCockpit, GX: 0, GY: 0},
				{Kind: PartGun, GX: 1, GY: 0},
				{Kind: PartFuel, GX: 0, GY: 1},
			},
			Angle: -math.Pi / 2, // 起動時はビジュアル的に上向き
		},
	}
}

// Update はキー入力に応じて機体を1フレーム動かす。
func (p *Player) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		p.Angle -= playerRotateSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		p.Angle += playerRotateSpeed
	}

	accel := 0.0
	boosting := false
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		accel = playerThrustAccel
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			accel *= playerBoostMultiplier
			boosting = true
		}
	}
	p.VX += accel * math.Cos(p.Angle)
	p.VY += accel * math.Sin(p.Angle)

	speed := math.Hypot(p.VX, p.VY)
	limit := playerMaxSpeed
	if boosting {
		limit = playerBoostMaxSpeed
	}
	if speed > limit {
		p.VX = p.VX / speed * limit
		p.VY = p.VY / speed * limit
	}

	p.X += p.VX
	p.Y += p.VY
}
