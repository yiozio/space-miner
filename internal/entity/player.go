package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerRotateSpeed    = 0.06
	playerOverspeedDecel = 0.10 // 通常最高速度を超えた分を毎フレームこれだけ削る（スラスタ数倍でスケール）
	PlayerHPDefault      = 100  // 基本 HP（Armor の ArmorHP 合算で MaxHP が増える）
	// PlayerCreditsDefault = 100  // 初期所持クレジット
	PlayerCreditsDefault = 1000
	PlayerInvulnFrames   = 30 // 被弾後の無敵フレーム

	// シールド回復: 最後の被弾から ShieldRegenDelay フレーム経過後に
	// 毎フレーム ShieldRegenPerFrame ずつ回復する。
	ShieldRegenDelay    = 120 // 2 秒（60fps）
	ShieldRegenPerFrame = 0.5 // 30 HP/秒
)

// Player はプレイヤー機。Ship に操作・発射・インベントリ・HP・燃料・クレジットを加えたもの。
type Player struct {
	Ship
	HP             int
	MaxHP          int // 基本 HP + 全 Armor の ArmorHP 合算
	ShieldHP       int // 現在のシールド HP
	MaxShieldHP    int // 全 Shield パーツの ShieldHP 合算
	Fuel           float64
	MaxFuel        float64
	MaxCargo       float64 // Cockpit + Cargo パーツの CargoCapacity 合算
	Credits        int
	InvulnTimer    int // 被弾後の残無敵フレーム（描画フラッシュにも使う）
	fireTimer      int
	Inventory      map[ResourceType]int // 資源
	PartsInventory map[PartID]int       // 船に未取付のスペアパーツ
	// 動的なスピード上限。ブースト時に boost 上限へ瞬時上昇し、解除後は徐々に通常上限へ減衰する。
	speedCap float64
	// シールド回復制御。
	noDamageFrames int     // 最後の被弾からの経過フレーム
	shieldRegenAcc float64 // 1 を超えるごとに ShieldHP を 1 上げる
}

// NewPlayerPebble は初期機体「Pebble」のプレイヤーを生成する。
// 最初期はコックピット + 最弱の Starter Gun のみ。
// スラスタは無しで Cockpit の最低限推進機能で動き、燃料タンクが無いので MaxFuel=0（ブースト不可）。
// プレイヤーはステーションでパーツを買い足して機体を強化していく。
func NewPlayerPebble() *Player {
	p := &Player{
		Ship: Ship{
			Parts: []Part{
				{DefID: PartIDGunStarter, GX: 0, GY: -1},
				{DefID: PartIDCockpit, GX: 0, GY: 0},
			},
			Angle: -math.Pi / 2, // 起動時はビジュアル的に上向き
		},
		HP:             PlayerHPDefault,
		MaxHP:          PlayerHPDefault,
		Credits:        PlayerCreditsDefault,
		Inventory:      make(map[ResourceType]int),
		PartsInventory: make(map[PartID]int),
	}
	p.recomputeStats()
	// 初期状態は HP/シールドを満タンに
	p.HP = p.MaxHP
	p.ShieldHP = p.MaxShieldHP
	return p
}

// recomputeStats は搭載パーツから派生するステータス（MaxHP / MaxShieldHP / MaxFuel / MaxCargo）を再計算し、
// 現在値を新しい上限にクランプする。
// Armor 0 個 → 基本 HP のみ、Shield 0 個 → MaxShieldHP=0、Fuel 0 個 → MaxFuel=0。
// MaxCargo は Cockpit と全 Cargo パーツの CargoCapacity を合算（Cockpit が最低限の積載量を提供する）。
func (p *Player) recomputeStats() {
	armor := 0
	shield := 0
	fuel := 0.0
	cargo := 0.0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil {
			continue
		}
		switch d.Kind {
		case PartArmor:
			armor += d.ArmorHP
		case PartShield:
			shield += d.ShieldHP
		case PartFuel:
			fuel += d.FuelCapacity
		}
		// Cockpit と Cargo は Kind に関係なく CargoCapacity を加算する（他カテゴリは値が 0）
		cargo += d.CargoCapacity
	}
	p.MaxHP = PlayerHPDefault + armor
	p.MaxShieldHP = shield
	p.MaxFuel = fuel
	p.MaxCargo = cargo
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
	if p.ShieldHP > p.MaxShieldHP {
		p.ShieldHP = p.MaxShieldHP
	}
	if p.Fuel > p.MaxFuel {
		p.Fuel = p.MaxFuel
	}
	// 注: パーツ取り外しで MaxCargo が現在の積載量より小さくなる場合があるが、
	// 既に持っている荷物を強制で破棄するのは避け、超過状態は許容する（買取・売却で減らす）。
}

// CargoLoad は現在の積載重量を返す（資源 + スペアパーツ）。
// 設置済みパーツは「機体構造」として扱い、積載には含めない。
func (p *Player) CargoLoad() float64 {
	total := 0.0
	for r, qty := range p.Inventory {
		if qty <= 0 {
			continue
		}
		total += r.Info().Weight * float64(qty)
	}
	for id, qty := range p.PartsInventory {
		if qty <= 0 {
			continue
		}
		if d := PartDefByID(id); d != nil {
			total += d.Weight * float64(qty)
		}
	}
	return total
}

// CanAddWeight は w 分の重量が積載上限内に収まるか返す。
func (p *Player) CanAddWeight(w float64) bool {
	return p.CargoLoad()+w <= p.MaxCargo+1e-9
}

// OnPartsChanged はエディタなどでパーツ構成が変わった後に呼ぶ。
// 描画キャッシュ無効化 + 派生ステータス再計算をまとめて行う。
func (p *Player) OnPartsChanged() {
	p.Ship.InvalidateImage()
	p.recomputeStats()
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
	var cockpit *PartDef
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil {
			continue
		}
		switch d.Kind {
		case PartThruster:
			t.count++
			t.accel += d.ThrustAccel
			t.maxSpeed += d.ThrustMaxSpeed
			t.boostMaxSpeed += d.ThrustBoostMaxSpeed
			t.boostFuelCost += d.ThrustBoostFuelCost
			if d.ThrustBoostAccelMul > t.boostAccelMul {
				t.boostAccelMul = d.ThrustBoostAccelMul
			}
		case PartCockpit:
			cockpit = d
		}
	}
	// Thruster が 1 つも無いときに限り、Cockpit の最低限スラスタ性能で代替する。
	// Thruster が 1 つでもあれば Cockpit は推進寄与しない（二重計上を避ける）。
	if t.count == 0 && cockpit != nil {
		t.count = 1
		t.accel = cockpit.ThrustAccel
		t.maxSpeed = cockpit.ThrustMaxSpeed
		t.boostMaxSpeed = cockpit.ThrustBoostMaxSpeed
		t.boostFuelCost = cockpit.ThrustBoostFuelCost
		t.boostAccelMul = cockpit.ThrustBoostAccelMul
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

	// シールド回復: 最後の被弾から ShieldRegenDelay フレーム経過後、毎フレーム回復していく。
	p.noDamageFrames++
	if p.noDamageFrames >= ShieldRegenDelay && p.ShieldHP < p.MaxShieldHP {
		p.shieldRegenAcc += ShieldRegenPerFrame
		for p.shieldRegenAcc >= 1 && p.ShieldHP < p.MaxShieldHP {
			p.shieldRegenAcc--
			p.ShieldHP++
		}
		if p.ShieldHP >= p.MaxShieldHP {
			p.ShieldHP = p.MaxShieldHP
			p.shieldRegenAcc = 0
		}
	}
}

// Damage はダメージを適用する。シールドが先に吸収し、超過分が HP を削る。
// 無敵中、もしくは amount が非正なら何もしない。
// 被弾するとシールド回復タイマーがリセットされ、ShieldRegenDelay フレーム経過後に再生開始する。
func (p *Player) Damage(amount int) {
	if p.InvulnTimer > 0 || amount <= 0 {
		return
	}
	p.noDamageFrames = 0
	p.shieldRegenAcc = 0
	if p.ShieldHP > 0 {
		if amount <= p.ShieldHP {
			p.ShieldHP -= amount
			amount = 0
		} else {
			amount -= p.ShieldHP
			p.ShieldHP = 0
		}
	}
	if amount > 0 {
		p.HP -= amount
		if p.HP < 0 {
			p.HP = 0
		}
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
// 積載超過になる場合は false を返し、加算しない（呼び出し側で拾い直し等を判断する）。
// qty が非正の場合は減算となるため重量チェックを行わない。
func (p *Player) AddResource(r ResourceType, qty int) bool {
	if p.Inventory == nil {
		p.Inventory = make(map[ResourceType]int)
	}
	if qty > 0 {
		w := r.Info().Weight * float64(qty)
		if !p.CanAddWeight(w) {
			return false
		}
	}
	p.Inventory[r] += qty
	return true
}
