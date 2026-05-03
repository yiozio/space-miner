package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerRotateSpeed    = 0.06
	playerOverspeedDecel = 0.10 // 通常最高速度を超えた分を毎フレームこれだけ削る（スラスタ数倍でスケール）
	PlayerHPDefault      = 100  // 初期 HP / 最大 HP
	PlayerFuelDefault    = 100  // 初期燃料 / 最大燃料
	PlayerCreditsDefault = 100  // 初期所持クレジット
	PlayerInvulnFrames   = 30   // 被弾後の無敵フレーム
)

// Player はプレイヤー機。Ship に操作・発射・インベントリ・HP・燃料・クレジットを加えたもの。
type Player struct {
	Ship
	HP             int
	MaxHP          int
	Fuel           float64
	MaxFuel        float64
	Credits        int
	InvulnTimer    int // 被弾後の残無敵フレーム（描画フラッシュにも使う）
	fireTimer      int
	Inventory      map[ResourceType]int // 資源
	PartsInventory map[PartID]int       // 船に未取付のスペアパーツ
	// 動的なスピード上限。ブースト時に boost 上限へ瞬時上昇し、解除後は徐々に通常上限へ減衰する。
	speedCap float64
}

// NewPlayerPebble は初期機体「Pebble」のプレイヤーを生成する。
// 配置は docs/GAME_DESIGN.md「サンプル: スターター艇 Pebble」に対応。
func NewPlayerPebble() *Player {
	return &Player{
		Ship: Ship{
			Parts: []Part{
				{DefID: PartIDThrusterStd, GX: 0, GY: -1},
				{DefID: PartIDGunMkI, GX: -1, GY: 0},
				{DefID: PartIDCockpit, GX: 0, GY: 0},
				{DefID: PartIDGunMkI, GX: 1, GY: 0},
				{DefID: PartIDFuelStd, GX: 0, GY: 1},
			},
			Angle: -math.Pi / 2, // 起動時はビジュアル的に上向き
		},
		HP:             PlayerHPDefault,
		MaxHP:          PlayerHPDefault,
		Fuel:           PlayerFuelDefault,
		MaxFuel:        PlayerFuelDefault,
		Credits:        PlayerCreditsDefault,
		Inventory:      make(map[ResourceType]int),
		PartsInventory: make(map[PartID]int),
	}
}

// thrusterAgg は搭載スラスタの集計値。
// Accel/MaxSpeed/BoostMaxSpeed/BoostFuelCost は全スラスタ分の合算、
// BoostAccelMul は最も性能の良い（倍率の大きい）スラスタの値を採用する。
type thrusterAgg struct {
	count           int
	accel, maxSpeed float64
	boostAccelMul   float64
	boostMaxSpeed   float64
	boostFuelCost   float64
}

func (p *Player) thrusterStats() thrusterAgg {
	var t thrusterAgg
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartThruster {
			continue
		}
		t.count++
		t.accel += d.ThrustAccel
		t.maxSpeed += d.ThrustMaxSpeed
		t.boostMaxSpeed += d.ThrustBoostMaxSpeed
		t.boostFuelCost += d.ThrustBoostFuelCost
		if d.ThrustBoostAccelMul > t.boostAccelMul {
			t.boostAccelMul = d.ThrustBoostAccelMul
		}
	}
	return t
}

// Update はキー入力に応じて機体を1フレーム動かす。発射は Shoot で別途行う。
// 加速度・最高速度・ブースト性能は搭載スラスタの def から集計する。
// スラスタが 0 のときは推力ゼロだが、既存の慣性は保持する。
func (p *Player) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		p.Angle -= playerRotateSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		p.Angle += playerRotateSpeed
	}

	ts := p.thrusterStats()

	accel := 0.0
	p.ThrustState = ThrustOff
	if ts.count > 0 && (ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)) {
		accel = ts.accel
		p.ThrustState = ThrustOn
		// ブーストは燃料が残っているときのみ有効
		boostHeld := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
		if boostHeld && p.Fuel > 0 {
			accel *= ts.boostAccelMul
			p.ThrustState = ThrustBoost
			p.Fuel -= ts.boostFuelCost
			if p.Fuel < 0 {
				p.Fuel = 0
			}
		}
	}
	// 動的スピード上限の更新。
	// ブースト中: boost 上限に瞬時セット。
	// 非ブースト時: 通常上限まで毎フレーム少しずつ降下、超えていなければ即時通常上限。
	if ts.count > 0 {
		normalLimit := ts.maxSpeed
		if p.ThrustState == ThrustBoost {
			p.speedCap = ts.boostMaxSpeed
		} else if p.speedCap > normalLimit {
			p.speedCap -= playerOverspeedDecel * float64(ts.count)
			if p.speedCap < normalLimit {
				p.speedCap = normalLimit
			}
		} else {
			p.speedCap = normalLimit
		}
	}

	p.VX += accel * math.Cos(p.Angle)
	p.VY += accel * math.Sin(p.Angle)

	// Thruster がある場合のみ speedCap でハードクランプ。
	// 通常加速で抜けないようにここでクランプを効かせる。
	if ts.count > 0 {
		speed := math.Hypot(p.VX, p.VY)
		if speed > p.speedCap {
			p.VX = p.VX / speed * p.speedCap
			p.VY = p.VY / speed * p.speedCap
		}
	}

	p.X += p.VX
	p.Y += p.VY

	if p.fireTimer > 0 {
		p.fireTimer--
	}
	if p.InvulnTimer > 0 {
		p.InvulnTimer--
	}
}

// Damage は HP を減らし、無敵フレームを設定する。
// 無敵中、もしくは amount が非正なら何もしない。
func (p *Player) Damage(amount int) {
	if p.InvulnTimer > 0 || amount <= 0 {
		return
	}
	p.HP -= amount
	if p.HP < 0 {
		p.HP = 0
	}
	p.InvulnTimer = PlayerInvulnFrames
}

// Shoot はクールダウンが許せば各 Gun パーツから1発ずつ弾を発射する。
// 各弾はそのガンの def に応じたダメージ・弾速で生成され、
// クールダウンは発射に参加したガンの中で最も長いものを採用する（最遅ガンが律速）。
// 戻り値は今フレームに発射された弾。クールダウン中なら nil。
func (p *Player) Shoot() []Bullet {
	if p.fireTimer > 0 {
		return nil
	}
	var out []Bullet
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	g := float64(GridSize)
	maxCooldown := 0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartGun {
			continue
		}
		// ガンの前端中心（ローカル）。ローカル -y が前方なので GY*g - g/2。
		lx := float64(part.GX) * g
		frontLy := float64(part.GY)*g - g/2
		// ローカル → ワールド: 船体と同じ R(angle + π/2) を適用
		wox := -sin*lx - cos*frontLy
		woy := cos*lx - sin*frontLy
		out = append(out, Bullet{
			X:      p.X + wox,
			Y:      p.Y + woy,
			VX:     cos*d.GunBulletSpeed + p.VX,
			VY:     sin*d.GunBulletSpeed + p.VY,
			Life:   bulletLifeFrames,
			Damage: d.GunDamage,
		})
		if d.GunCooldown > maxCooldown {
			maxCooldown = d.GunCooldown
		}
	}
	if len(out) > 0 {
		p.fireTimer = maxCooldown
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
