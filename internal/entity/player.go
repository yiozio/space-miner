package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	playerRotateSpeed    = 0.06
	playerOverspeedDecel = 0.10 // 通常最高速度を超えた分を毎フレームこれだけ削る（スラスタ数倍でスケール）
	speedCapDecel        = 0.20 // 速度マグニチュード上限を下げるときの毎フレーム減衰量
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
	HP          int
	MaxHP       int // 基本 HP + 全 Armor の ArmorHP 合算
	ShieldHP    int // 現在のシールド HP
	MaxShieldHP int // 全 Shield パーツの ShieldHP 合算
	Fuel        float64
	MaxFuel     float64
	MaxCargo    float64 // Cockpit + Cargo パーツの CargoCapacity 合算
	Credits     int
	// gunFireTimers はガンごとの発射クールダウン残フレーム。グリッド位置(GX,GY)で
	// 管理し、各ガンが自分の GunCooldown で独立して発射する。
	gunFireTimers      map[[2]int]int
	Inventory          map[ResourceType]int    // 資源
	PartsInventory     map[PartID]int          // 船に未取付のスペアパーツ
	VisitedStations    map[string]bool         // 初回入船ダイアログの再生済みステーション名（FullMap 名）
	Tavern             map[string]*TavernBoard // 各ステーションの酒場掲示板（4 スロット）
	PiratesKilledByMap map[string]int          // 各 FullMap での累計海賊撃破数（Bounty 進捗の根拠）
	// 動的なスピード上限（前後左右の方向別）。
	// ブースト中は boost 上限へ瞬時上昇し、解除後は徐々に通常上限へ減衰する。
	fwdCap float64
	bckCap float64
	lftCap float64
	rgtCap float64
	// 速度マグニチュードの上限。上げるときは即時、下げるときは徐々に追従させ、
	// 高速移動中に低速方向（後退など）へ切り替えても一気に減速しないようにする。
	spdCap float64
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
			BaseID: ShipBasePebble, // 3x3 ベース。コックピット機能はベースが内包する。
			Parts: []Part{
				{DefID: PartIDGunStarter, GX: 0, GY: -1},
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
// 基本 HP・基本積載はベース（土台）が提供し、Armor / Cargo パーツがそれに上乗せする。
func (p *Player) recomputeStats() {
	base := ShipBaseDefByID(p.BaseID)
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
		// Cargo パーツの CargoCapacity を加算（他カテゴリは値が 0）
		cargo += d.CargoCapacity
	}
	p.MaxHP = base.BaseHP + armor
	p.MaxShieldHP = shield
	p.MaxFuel = fuel
	p.MaxCargo = base.BaseCargo + cargo
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

// DirSpeed は1方向の速度性能（表示用）。Active が false の方向は推進が無い。
type DirSpeed struct {
	Active bool
	Max    float64 // 最高速度
	Boost  float64 // ブースト時の最高速度
}

// ShipStats は機体性能の表示用サマリ（エディタ表示など）。
type ShipStats struct {
	TotalDPS           float64 // 全銃の DPS 合計（総火力）
	MaxHP              int     // 耐久値
	MaxShield          int     // シールド
	MaxFuel            float64 // 燃料（>0 ならブースト可）
	Fwd, Bck, Lft, Rgt DirSpeed
}

// Stats は現在の搭載パーツから機体性能サマリを算出する。
// DPS は各銃の GunDamage×60/GunCooldown（個別クールダウン前提）の合計。
func (p *Player) Stats() ShipStats {
	var dps float64
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartGun || d.GunCooldown <= 0 {
			continue
		}
		dps += float64(d.GunDamage) * 60.0 / float64(d.GunCooldown)
	}
	fwd, bck, lft, rgt := p.thrusterStatsByDir()
	toDir := func(a thrusterAgg) DirSpeed {
		return DirSpeed{Active: a.count > 0, Max: a.maxSpeed, Boost: a.boostMaxSpeed}
	}
	return ShipStats{
		TotalDPS:  dps,
		MaxHP:     p.MaxHP,
		MaxShield: p.MaxShieldHP,
		MaxFuel:   p.MaxFuel,
		Fwd:       toDir(fwd),
		Bck:       toDir(bck),
		Lft:       toDir(lft),
		Rgt:       toDir(rgt),
	}
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

// thrusterStatsByDir はスラスタ集計を方向別 (前/後/左/右) に返す。
// Thruster が 1 つも無いときに限り、ベースの非常用スラスタ性能を前向きに使う。
func (p *Player) thrusterStatsByDir() (fwd, bck, lft, rgt thrusterAgg) {
	hasThruster := false
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartThruster {
			continue
		}
		hasThruster = true
		switch part.ThrustDir() {
		case ThrustDirForward:
			accumulateThruster(&fwd, d)
		case ThrustDirBackward:
			accumulateThruster(&bck, d)
		case ThrustDirLeft:
			accumulateThruster(&lft, d)
		case ThrustDirRight:
			accumulateThruster(&rgt, d)
		}
	}
	if !hasThruster {
		// 非常用ベース推進は前向きのみに与える
		accumulateThruster(&fwd, ShipBaseDefByID(p.BaseID).EmergencyThrust())
	}
	return fwd, bck, lft, rgt
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
// 加速度・最高速度・ブースト性能は方向別スラスタ (前後左右) を別々に集計する。
// W/S で機軸方向、Q/E で横方向ストラフ。前後と左右は同時に押せる（斜め推進）。
// スラスタが 0 のときは推力ゼロだが、既存の慣性は保持する。
func (p *Player) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		p.Angle -= playerRotateSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		p.Angle += playerRotateSpeed
	}

	fwd, bck, lft, rgt := p.thrusterStatsByDir()

	cos, sin := math.Cos(p.Angle), math.Sin(p.Angle)
	// 機体の右方向単位ベクトル (機軸を CW 90° 回した方向)
	rightX, rightY := -sin, cos
	forwardPressed := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	backwardPressed := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	leftPressed := ebiten.IsKeyPressed(ebiten.KeyQ)
	rightPressed := ebiten.IsKeyPressed(ebiten.KeyE)
	boostHeld := ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)

	// 軸方向ヘルパ。同方向のスラスタが稼働した場合に true を返す。
	applyThrust := func(agg thrusterAgg, vx, vy float64, activeBit ThrustActiveDir) (active, boosting bool) {
		if agg.count == 0 {
			return false, false
		}
		accel := agg.accel
		boosting = boostHeld && p.Fuel > 0
		if boosting {
			accel *= agg.boostAccelMul
			p.Fuel -= agg.boostFuelCost
			if p.Fuel < 0 {
				p.Fuel = 0
			}
		}
		p.VX += accel * vx
		p.VY += accel * vy
		p.ThrustActiveDir |= activeBit
		return true, boosting
	}

	p.ThrustState = ThrustOff
	p.ThrustActiveDir = 0

	// 前後方向: W/S が同時押しなら前進優先 (既存挙動)。
	fwdActive, fwdBoosting := false, false
	bckActive, bckBoosting := false, false
	switch {
	case forwardPressed:
		fwdActive, fwdBoosting = applyThrust(fwd, cos, sin, ThrustActiveForward)
	case backwardPressed:
		bckActive, bckBoosting = applyThrust(bck, -cos, -sin, ThrustActiveBackward)
	}
	// 左右方向: Q/E が同時押しなら左優先。前後とは独立に同時稼働できる。
	lftActive, lftBoosting := false, false
	rgtActive, rgtBoosting := false, false
	switch {
	case leftPressed:
		lftActive, lftBoosting = applyThrust(lft, -rightX, -rightY, ThrustActiveLeft)
	case rightPressed:
		rgtActive, rgtBoosting = applyThrust(rgt, rightX, rightY, ThrustActiveRight)
	}

	if fwdActive || bckActive || lftActive || rgtActive {
		p.ThrustState = ThrustOn
	}
	if fwdBoosting || bckBoosting || lftBoosting || rgtBoosting {
		p.ThrustState = ThrustBoost
	}

	// 動的スピード上限の更新。各方向はそれぞれブースト中の場合のみ boost 上限へ。
	updateCap(&p.fwdCap, fwd, fwdBoosting)
	updateCap(&p.bckCap, bck, bckBoosting)
	updateCap(&p.lftCap, lft, lftBoosting)
	updateCap(&p.rgtCap, rgt, rgtBoosting)

	// 速度マグニチュードの上限。まず現フレームの目標上限を求める：稼働中の方向の
	// cap の最大値（何も稼働していなければ全方向 cap の最大値で慣性を保つ）。
	target := 0.0
	hasActive := false
	considerCap := func(c float64) {
		if c > target {
			target = c
		}
		hasActive = true
	}
	if fwdActive {
		considerCap(p.fwdCap)
	}
	if bckActive {
		considerCap(p.bckCap)
	}
	if lftActive {
		considerCap(p.lftCap)
	}
	if rgtActive {
		considerCap(p.rgtCap)
	}
	if !hasActive {
		target = math.Max(math.Max(p.fwdCap, p.bckCap), math.Max(p.lftCap, p.rgtCap))
	}
	// 上限は上げるときは即時、下げるときは徐々に。これで高速前進中に後退へ
	// 切り替えても、上限が一気に下がらず緩やかに減速する。
	if target >= p.spdCap {
		p.spdCap = target
	} else {
		p.spdCap = math.Max(target, p.spdCap-speedCapDecel)
	}
	if p.spdCap > 0 {
		speed := math.Hypot(p.VX, p.VY)
		if speed > p.spdCap {
			scale := p.spdCap / speed
			p.VX *= scale
			p.VY *= scale
		}
	}

	p.X += p.VX
	p.Y += p.VY

	// 各ガンの発射クールダウンを 1 フレーム進める。
	for k, t := range p.gunFireTimers {
		if t > 0 {
			p.gunFireTimers[k] = t - 1
		}
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

// Shoot は各 Gun / MineLayer / DroneLauncher パーツを個別のクールダウンで発射する（パーツごとに独立）。
// 弾は各 Gun の Rotation に従う向きで射出され、発射位置はパーツの前端中心。
// MineLayer / DroneLauncher は向きに依らず機体（パーツ）位置へ機雷／ドローンを設置する。
// 戻り値: 通常弾 (Trail/Ball) と瞬間命中レーザー要求 (Laser スタイル)、設置する機雷、設置するドローン、
// および この発射で鳴らすべき発射音の種類（重複は除去）の 5 つ。
// クールダウン中のパーツは発射せず、全パーツが待機中なら全て nil。
func (p *Player) Shoot() ([]Bullet, []LaserShot, []Mine, []Drone, []GunFireSound) {
	if p.gunFireTimers == nil {
		p.gunFireTimers = map[[2]int]int{}
	}
	var bullets []Bullet
	var lasers []LaserShot
	var mines []Mine
	var drones []Drone
	var fireSounds []GunFireSound
	seenSound := map[GunFireSound]bool{}
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	g := float64(GridSize)
	halfG := g / 2
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	// addFireSound は発射音を重複なく登録する小ヘルパ。
	addFireSound := func(s GunFireSound) {
		if s != FireSoundNone && !seenSound[s] {
			seenSound[s] = true
			fireSounds = append(fireSounds, s)
		}
	}
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || (d.Kind != PartGun && d.Kind != PartMineLayer && d.Kind != PartDroneLauncher) {
			continue
		}
		// このパーツがクールダウン中なら発射しない。
		key := [2]int{part.GX, part.GY}
		if p.gunFireTimers[key] > 0 {
			continue
		}

		// 機雷敷設は向きに依らずパーツ位置へ機雷を設置する。
		if d.Kind == PartMineLayer {
			cxL, cyL := PartLocalCenter(part.GX, part.GY)
			wox, woy := toWorld(cxL, cyL)
			mines = append(mines, Mine{
				X:           p.X + wox,
				Y:           p.Y + woy,
				Fuse:        mineFuseFrames,
				Damage:      d.GunDamage,
				BulletSpeed: d.GunBulletSpeed,
				Style:       d.GunBulletStyle,
				Width:       d.GunBulletWidth,
				ImpactFX:    d.GunBulletImpact,
			})
			p.gunFireTimers[key] = d.GunCooldown
			addFireSound(d.GunFireSound)
			continue
		}

		// ドローン射出も向きに依らずパーツ位置へドローンを設置する。
		if d.Kind == PartDroneLauncher {
			cxL, cyL := PartLocalCenter(part.GX, part.GY)
			wox, woy := toWorld(cxL, cyL)
			drones = append(drones, Drone{
				X:     p.X + wox,
				Y:     p.Y + woy,
				Life:  droneLifeFrames,
				Range: d.AutoAimRange,
				DPS:   d.AutoAimDPS,
			})
			p.gunFireTimers[key] = d.GunCooldown
			addFireSound(d.GunFireSound)
			continue
		}
		// 発射方向（ローカル）。描画と同じ回転規約 (x,y)→(-y,x) で前方ベクトル
		// (0,-1) を Rotation ぶん回したもの。横向き(R=1/3)の左右は drawAfterburners
		// の後方ベクトルと整合させる（前方＝後方の逆）。
		var fxL, fyL float64
		switch ((part.Rotation % 4) + 4) % 4 {
		case 0:
			fxL, fyL = 0, -1
		case 1:
			fxL, fyL = 1, 0
		case 2:
			fxL, fyL = 0, 1
		case 3:
			fxL, fyL = -1, 0
		}
		cxL, cyL := PartLocalCenter(part.GX, part.GY)
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
				// 新規弾は同フレーム内で 1 回 Update されるが、機体は発射前に既に
				// 1 フレーム分 (VX,VY) 進んでいる。その移動量ぶん手前から射出して
				// 銃口位置に合わせる（前進しながら撃つと射出地点がズレる補正）。
				X:        ox - p.VX,
				Y:        oy - p.VY,
				VX:       fwx*d.GunBulletSpeed + p.VX,
				VY:       fwy*d.GunBulletSpeed + p.VY,
				Life:     bulletLifeFrames,
				Damage:   d.GunDamage,
				Style:    d.GunBulletStyle,
				Width:    d.GunBulletWidth,
				ImpactFX: d.GunBulletImpact,
			})
		}
		// このガンを自分の GunCooldown でクールダウンに入れる。
		p.gunFireTimers[key] = d.GunCooldown
		addFireSound(d.GunFireSound)
	}
	return bullets, lasers, mines, drones, fireSounds
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
