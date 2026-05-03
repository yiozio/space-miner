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

	// シールド回復: 最後の被弾から ShieldRegenDelay フレーム経過後に
	// 毎フレーム ShieldRegenPerFrame ずつ回復する。
	ShieldRegenDelay    = 120 // 2 秒（60fps）
	ShieldRegenPerFrame = 0.5 // 30 HP/秒
)

// Player はプレイヤー機。Ship に操作・発射・インベントリ・HP・燃料・クレジットを加えたもの。
type Player struct {
	Ship
	HP                 int
	MaxHP              int // 基本 HP + 全 Armor の ArmorHP 合算
	ShieldHP           int // 現在のシールド HP
	MaxShieldHP        int // 全 Shield パーツの ShieldHP 合算
	Fuel               float64
	MaxFuel            float64
	MaxCargo           float64 // Cockpit + Cargo パーツの CargoCapacity 合算
	Credits            int
	fireTimer          int
	Inventory          map[ResourceType]int    // 資源
	PartsInventory     map[PartID]int          // 船に未取付のスペアパーツ
	VisitedStations    map[string]bool         // 初回入船ダイアログの再生済みステーション名（FullMap 名）
	Tavern             map[string]*TavernBoard // 各ステーションの酒場掲示板（3 スロット）
	PiratesKilledByMap map[string]int          // 各 FullMap での累計海賊撃破数（Bounty 進捗の根拠）
	// 動的なスピード上限（前後の機軸成分に対して別々）。
	// ブースト中は boost 上限へ瞬時上昇し、解除後は徐々に通常上限へ減衰する。
	fwdCap float64
	bckCap float64
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
			// 起点 FullMap (Aurora) のステーション位置からスタートする
			X:     DefaultStationX,
			Y:     DefaultStationY,
			Angle: -math.Pi / 2, // 起動時はビジュアル的に上向き
		},
		HP:                 PlayerHPDefault,
		MaxHP:              PlayerHPDefault,
		Credits:            PlayerCreditsDefault,
		Inventory:          make(map[ResourceType]int),
		PartsInventory:     make(map[PartID]int),
		VisitedStations:    make(map[string]bool),
		Tavern:             make(map[string]*TavernBoard),
		PiratesKilledByMap: make(map[string]int),
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

// thrusterStatsByDir はスラスタ集計を「前向き」「後ろ向き」別に返す。
// 横向き（Rotation=1,3）のスラスタは無視する。
// Thruster が 1 つも無いときに限り Cockpit の最低限スラスタ性能を前向きに使う。
func (p *Player) thrusterStatsByDir() (fwd, bck thrusterAgg) {
	var cockpit *PartDef
	hasThruster := false
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil {
			continue
		}
		switch d.Kind {
		case PartThruster:
			hasThruster = true
			switch part.ThrustDir() {
			case ThrustDirForward:
				accumulateThruster(&fwd, d)
			case ThrustDirBackward:
				accumulateThruster(&bck, d)
			}
			// Sideways はスキップ（推進に寄与しない）
		case PartCockpit:
			cockpit = d
		}
	}
	if !hasThruster && cockpit != nil {
		// 非常用 Cockpit 推進は前向きのみに与える
		accumulateThruster(&fwd, cockpit)
	}
	return fwd, bck
}

// updateCap は方向別の動的スピード上限 cap を更新する。
// ブースト中は boost 上限に瞬時セット、非ブースト時は通常上限へ overspeedDecel*count で減衰。
// agg.count == 0 のときは触らない（既存値保持）。
func updateCap(cap *float64, agg thrusterAgg, boosting bool) {
	if agg.count == 0 {
		return
	}
	normal := agg.maxSpeed
	if boosting {
		*cap = agg.boostMaxSpeed
		return
	}
	if *cap > normal {
		*cap -= playerOverspeedDecel * float64(agg.count)
		if *cap < normal {
			*cap = normal
		}
	} else {
		*cap = normal
	}
}

func accumulateThruster(t *thrusterAgg, d *PartDef) {
	t.count++
	t.accel += d.ThrustAccel
	t.maxSpeed += d.ThrustMaxSpeed
	t.boostMaxSpeed += d.ThrustBoostMaxSpeed
	t.boostFuelCost += d.ThrustBoostFuelCost
	if d.ThrustBoostAccelMul > t.boostAccelMul {
		t.boostAccelMul = d.ThrustBoostAccelMul
	}
}

// Update はキー入力に応じて機体を1フレーム動かす。発射は Shoot で別途行う。
// 加速度・最高速度・ブースト性能は前向き / 後ろ向きスラスタを別々に集計する。
// 横向きスラスタは推進に寄与せず、ブースト燃料も消費しない。
// スラスタが 0 のときは推力ゼロだが、既存の慣性は保持する。
func (p *Player) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		p.Angle -= playerRotateSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		p.Angle += playerRotateSpeed
	}

	fwd, bck := p.thrusterStatsByDir()

	cos, sin := math.Cos(p.Angle), math.Sin(p.Angle)
	forwardPressed := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	backwardPressed := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	boostHeld := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)

	accel := 0.0
	dirSign := 0.0 // 1 = 前進, -1 = 後進
	p.ThrustState = ThrustOff
	p.ThrustActiveDir = ThrustActiveForward
	switch {
	case fwd.count > 0 && forwardPressed:
		accel = fwd.accel
		dirSign = 1
		p.ThrustState = ThrustOn
		p.ThrustActiveDir = ThrustActiveForward
		if boostHeld && p.Fuel > 0 {
			accel *= fwd.boostAccelMul
			p.ThrustState = ThrustBoost
			p.Fuel -= fwd.boostFuelCost
			if p.Fuel < 0 {
				p.Fuel = 0
			}
		}
	case bck.count > 0 && backwardPressed:
		accel = bck.accel
		dirSign = -1
		p.ThrustState = ThrustOn
		p.ThrustActiveDir = ThrustActiveBackward
		if boostHeld && p.Fuel > 0 {
			accel *= bck.boostAccelMul
			p.ThrustState = ThrustBoost
			p.Fuel -= bck.boostFuelCost
			if p.Fuel < 0 {
				p.Fuel = 0
			}
		}
	}

	// 前向き/後ろ向きの動的スピード上限を更新。
	// ブースト中は対応する boost 上限に瞬時セット、非ブースト時は通常上限へ減衰。
	updateCap(&p.fwdCap, fwd, dirSign == 1 && p.ThrustState == ThrustBoost)
	updateCap(&p.bckCap, bck, dirSign == -1 && p.ThrustState == ThrustBoost)

	p.VX += accel * dirSign * cos
	p.VY += accel * dirSign * sin

	// 機軸（前後軸）に沿った成分でクランプ。横方向（接線方向）はクランプしない。
	if fwd.count > 0 || bck.count > 0 {
		fwdDot := p.VX*cos + p.VY*sin
		sideX, sideY := -sin, cos
		sideDot := p.VX*sideX + p.VY*sideY
		// 前向き上限
		if fwdDot > p.fwdCap {
			fwdDot = p.fwdCap
		}
		// 後ろ向き上限（負方向に対する大きさ）
		if -fwdDot > p.bckCap {
			fwdDot = -p.bckCap
		}
		p.VX = fwdDot*cos + sideDot*sideX
		p.VY = fwdDot*sin + sideDot*sideY
	}

	p.X += p.VX
	p.Y += p.VY

	if p.fireTimer > 0 {
		p.fireTimer--
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
// amount が非正なら何もしない。
// 被弾するとシールド回復タイマーがリセットされ、ShieldRegenDelay フレーム経過後に再生開始する。
func (p *Player) Damage(amount int) {
	if amount <= 0 {
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
}

// Shoot はクールダウンが許せば各 Gun パーツから 1 発ずつ発射する。
// 弾は各 Gun の Rotation に従う向きで射出され、発射位置はパーツの前端中心。
// 戻り値: 通常弾 (Trail/Ball) と瞬間命中レーザー要求 (Laser スタイル) の 2 種に分かれる。
// クールダウンは発射に参加したガンの中で最も長いものを採用する（最遅ガンが律速）。
// クールダウン中なら両方 nil。
func (p *Player) Shoot() ([]Bullet, []LaserShot) {
	if p.fireTimer > 0 {
		return nil, nil
	}
	var bullets []Bullet
	var lasers []LaserShot
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	g := float64(GridSize)
	halfG := g / 2
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	maxCooldown := 0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartGun {
			continue
		}
		var fxL, fyL float64
		switch ((part.Rotation % 4) + 4) % 4 {
		case 0:
			fxL, fyL = 0, -1
		case 1:
			fxL, fyL = -1, 0
		case 2:
			fxL, fyL = 0, 1
		case 3:
			fxL, fyL = 1, 0
		}
		cxL := float64(part.GX) * g
		cyL := float64(part.GY) * g
		frontLx := cxL + fxL*halfG
		frontLy := cyL + fyL*halfG
		wox, woy := toWorld(frontLx, frontLy)
		fwx, fwy := toWorld(fxL, fyL)
		ox := p.X + wox
		oy := p.Y + woy

		if d.GunBulletStyle == BulletStyleLaser {
			// レーザーは瞬間命中要求として返す（飛翔しない）
			lasers = append(lasers, LaserShot{
				X: ox, Y: oy,
				DX: fwx, DY: fwy,
				Damage:   d.GunDamage,
				Range:    d.GunBulletSpeed * float64(bulletLifeFrames),
				Hostile:  false,
				Width:    d.GunBulletWidth,
				ImpactFX: d.GunBulletImpact,
			})
		} else {
			bullets = append(bullets, Bullet{
				X:        ox,
				Y:        oy,
				VX:       fwx*d.GunBulletSpeed + p.VX,
				VY:       fwy*d.GunBulletSpeed + p.VY,
				Life:     bulletLifeFrames,
				Damage:   d.GunDamage,
				Style:    d.GunBulletStyle,
				Width:    d.GunBulletWidth,
				ImpactFX: d.GunBulletImpact,
			})
		}
		if d.GunCooldown > maxCooldown {
			maxCooldown = d.GunCooldown
		}
	}
	if len(bullets)+len(lasers) > 0 {
		p.fireTimer = maxCooldown
	}
	return bullets, lasers
}

// AddSparePart はスペアパーツインベントリに id を qty 個加算する。
// 積載超過になる場合は false を返し、加算しない（拾い直し / 拒絶を呼び出し側で扱う）。
// qty が非正の場合は減算となるため重量チェックを行わない。
func (p *Player) AddSparePart(id PartID, qty int) bool {
	if p.PartsInventory == nil {
		p.PartsInventory = make(map[PartID]int)
	}
	if qty > 0 {
		d := PartDefByID(id)
		if d == nil {
			return false
		}
		w := d.Weight * float64(qty)
		if !p.CanAddWeight(w) {
			return false
		}
	}
	p.PartsInventory[id] += qty
	return true
}

// HasWarpDrive は搭載パーツに Warp パーツが含まれているかを返す。
// 未搭載なら恒星マップは表示のみ、ワープ確定は不可となる。
func (p *Player) HasWarpDrive() bool {
	for _, part := range p.Parts {
		d := part.Def()
		if d != nil && d.Kind == PartWarp {
			return true
		}
	}
	return false
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
