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
	playerFireCooldown    = 12 // フレーム単位（60fps で約5発/秒）
)

// Player はプレイヤー機。Ship に操作・発射・インベントリを加えたもの。
type Player struct {
	Ship
	fireTimer int
	Inventory map[ResourceType]int
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
		Inventory: make(map[ResourceType]int),
	}
}

// Update はキー入力に応じて機体を1フレーム動かす。発射は Shoot で別途行う。
func (p *Player) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		p.Angle -= playerRotateSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		p.Angle += playerRotateSpeed
	}

	accel := 0.0
	p.ThrustState = ThrustOff
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		accel = playerThrustAccel
		p.ThrustState = ThrustOn
		if ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
			accel *= playerBoostMultiplier
			p.ThrustState = ThrustBoost
		}
	}
	p.VX += accel * math.Cos(p.Angle)
	p.VY += accel * math.Sin(p.Angle)

	speed := math.Hypot(p.VX, p.VY)
	limit := playerMaxSpeed
	if p.ThrustState == ThrustBoost {
		limit = playerBoostMaxSpeed
	}
	if speed > limit {
		p.VX = p.VX / speed * limit
		p.VY = p.VY / speed * limit
	}

	p.X += p.VX
	p.Y += p.VY

	if p.fireTimer > 0 {
		p.fireTimer--
	}
}

// Shoot はクールダウンが許せば各 Gun パーツから1発ずつ弾を発射する。
// 戻り値は今フレームに発射された弾。クールダウン中なら nil。
func (p *Player) Shoot() []Bullet {
	if p.fireTimer > 0 {
		return nil
	}
	var out []Bullet
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	g := float64(GridSize)
	for _, part := range p.Parts {
		if part.Kind != PartGun {
			continue
		}
		// ガンの前端中心（ローカル）。ローカル -y が前方なので GY*g - g/2。
		lx := float64(part.GX) * g
		frontLy := float64(part.GY)*g - g/2
		// ローカル → ワールド: 船体と同じ R(angle + π/2) を適用
		wox := -sin*lx - cos*frontLy
		woy := cos*lx - sin*frontLy
		out = append(out, Bullet{
			X:    p.X + wox,
			Y:    p.Y + woy,
			VX:   cos*bulletSpeed + p.VX,
			VY:   sin*bulletSpeed + p.VY,
			Life: bulletLifeFrames,
		})
	}
	if len(out) > 0 {
		p.fireTimer = playerFireCooldown
	}
	return out
}

// AddResource はインベントリに資源を加算する。
func (p *Player) AddResource(r ResourceType, qty int) {
	if p.Inventory == nil {
		p.Inventory = make(map[ResourceType]int)
	}
	p.Inventory[r] += qty
}
