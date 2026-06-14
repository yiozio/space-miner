package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/yiozio/space-miner/internal/ui"
)

// 海賊ビジュアル用のアクセント色。弾と敵識別マーカーに使う。
var (
	pirateBulletColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}
	pirateLineColor   = color.NRGBA{0xff, 0x60, 0x40, 0xff}
)

// Pirate は敵 NPC 機体。Ship を継承し、簡易な追尾 AI と発射制御を持つ。
// 機体構成と行動パラメータは PiratePattern から決まる。
// 描画・当たり判定はプレイヤー機と同じ「ベース船体 + グリッドパーツ」方式を共有し、
// 敵識別のためライン色を pirateLineColor に上書きして描く（Ship.LineColor）。
type Pirate struct {
	Ship
	HP         int
	MaxHP      int
	Pattern    *PiratePattern
	fireTimer  int
	droneTimer int // ドローン設置のクールダウン（ガン発射とは独立）
}

// NewPirate は (x, y) に pattern に従う海賊機を生成する。最初は player 方向を向く。
func NewPirate(x, y float64, playerX, playerY float64, pattern *PiratePattern) *Pirate {
	parts := make([]Part, len(pattern.Parts))
	copy(parts, pattern.Parts)
	angle := math.Atan2(playerY-y, playerX-x)
	return &Pirate{
		Ship: Ship{
			Parts:  parts,
			BaseID: pattern.BaseID, // ゼロ値は ShipBasePebble（3x3 ベース）
			X:      x, Y: y,
			Angle:     angle,
			LineColor: pirateLineColor, // 敵識別の赤系ラインで機体を描く
		},
		HP:      pattern.MaxHP,
		MaxHP:   pattern.MaxHP,
		Pattern: pattern,
	}
}

// Update は 1 フレーム分 AI を進め、発射した弾・レーザー要求・設置したドローンを返す。
// 動作: プレイヤー方向に旋回し、PreferredDist に近づこうとする。
// FireRange 内で機首がほぼ向いているとき発射する。
// ドローンランチャー搭載時は、射程内で独立クールダウンに従い自機狙いのドローンを設置する。
func (p *Pirate) Update(playerX, playerY float64) ([]Bullet, []LaserShot, []Drone) {
	dx := playerX - p.X
	dy := playerY - p.Y
	dist := math.Hypot(dx, dy)
	targetAngle := math.Atan2(dy, dx)

	// 旋回（最短角度）
	da := normalizeAngle(targetAngle - p.Angle)
	if math.Abs(da) <= p.Pattern.TurnSpeed {
		p.Angle = targetAngle
	} else if da > 0 {
		p.Angle += p.Pattern.TurnSpeed
	} else {
		p.Angle -= p.Pattern.TurnSpeed
	}

	// 推進: PreferredDist より遠ければ前進、十分近ければ慣性で滑る
	accel := 0.0
	p.ThrustState = ThrustOff
	if dist > p.Pattern.PreferredDist+80 {
		accel = p.Pattern.ThrustAccel
		p.ThrustState = ThrustOn
	}
	p.VX += accel * math.Cos(p.Angle)
	p.VY += accel * math.Sin(p.Angle)

	// 軽いドラッグで暴走を抑える
	p.VX *= 0.99
	p.VY *= 0.99

	// 速度上限クランプ
	if speed := math.Hypot(p.VX, p.VY); speed > p.Pattern.MaxSpeed {
		p.VX = p.VX / speed * p.Pattern.MaxSpeed
		p.VY = p.VY / speed * p.Pattern.MaxSpeed
	}

	p.X += p.VX
	p.Y += p.VY

	if p.fireTimer > 0 {
		p.fireTimer--
	}
	if p.droneTimer > 0 {
		p.droneTimer--
	}

	// ドローン設置: 射程内かつ独立クールダウンが切れていれば、向きに依らず設置する。
	var drones []Drone
	if dist < p.Pattern.FireRange && p.droneTimer == 0 {
		drones = p.deployDrones()
	}

	// 発射条件: 射程内 + 機首がほぼ合っている + クールダウンが切れている
	if dist < p.Pattern.FireRange && math.Abs(da) < 0.35 && p.fireTimer == 0 {
		bullets, lasers := p.shoot()
		return bullets, lasers, drones
	}
	return nil, nil, drones
}

// deployDrones は搭載するドローンランチャーから自機狙い（Hostile）のドローンを設置する。
// 設置したら droneTimer を最も遅いランチャーのクールダウンに揃える。
// ランチャー非搭載なら何もしない。
func (p *Pirate) deployDrones() []Drone {
	var drones []Drone
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	maxCD := 0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != PartDroneLauncher {
			continue
		}
		cxL, cyL := PartLocalCenter(part.GX, part.GY)
		wox, woy := toWorld(cxL, cyL)
		drones = append(drones, NewDroneFromDef(d, p.X+wox, p.Y+woy, true))
		if d.GunCooldown > maxCD {
			maxCD = d.GunCooldown
		}
	}
	if len(drones) > 0 {
		p.droneTimer = maxCD
	}
	return drones
}

// shoot は装着 Gun から発射する。Hostile な弾とレーザー要求の 2 種を返す。
// 弾は各 Gun の Rotation に従う方向に射出される（Player.Shoot と同じロジック）。
// クールダウンは最も遅い Gun の値に揃える。
func (p *Pirate) shoot() ([]Bullet, []LaserShot) {
	var bullets []Bullet
	var lasers []LaserShot
	sin, cos := math.Sin(p.Angle), math.Cos(p.Angle)
	g := float64(GridSize)
	halfG := g / 2
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	maxCD := 0
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
		cxL, cyL := PartLocalCenter(part.GX, part.GY)
		frontLx := cxL + fxL*halfG
		frontLy := cyL + fyL*halfG
		wox, woy := toWorld(frontLx, frontLy)
		fwx, fwy := toWorld(fxL, fyL)
		ox := p.X + wox
		oy := p.Y + woy

		if d.GunBulletStyle == BulletStyleLaser {
			lasers = append(lasers, LaserShot{
				X: ox, Y: oy,
				DX: fwx, DY: fwy,
				Damage:   d.GunDamage,
				Range:    d.GunBulletSpeed * float64(bulletLifeFrames),
				Hostile:  true,
				Width:    d.GunBulletWidth,
				ImpactFX: d.GunBulletImpact,
			})
		} else {
			bullets = append(bullets, Bullet{
				X:               ox,
				Y:               oy,
				VX:              fwx*d.GunBulletSpeed + p.VX,
				VY:              fwy*d.GunBulletSpeed + p.VY,
				Life:            bulletLifeFrames,
				Damage:          d.GunDamage,
				Hostile:         true,
				Style:           d.GunBulletStyle,
				Width:           d.GunBulletWidth,
				ImpactFX:        d.GunBulletImpact,
				ExplosionRadius: d.GunExplosionRadius,
			})
		}
		if d.GunCooldown > maxCD {
			maxCD = d.GunCooldown
		}
	}
	if len(bullets)+len(lasers) > 0 {
		p.fireTimer = maxCD
	}
	return bullets, lasers
}

// TakeHit はダメージを適用し、true を返したら撃破。
func (p *Pirate) TakeHit(dmg int) (killed bool) {
	if dmg <= 0 {
		return false
	}
	p.HP -= dmg
	return p.HP <= 0
}

// DrawAt は海賊機をスクリーン (sx, sy) を中心に描画する。
// 機体本体はプレイヤーと共通の Ship.DrawAt（ベース船体 + グリッドパーツ）で描く。
// 敵識別はライン色の赤上書き（Ship.LineColor）で行うため、輪郭リングは描かない。
func (p *Pirate) DrawAt(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	p.Ship.DrawAt(dst, sx, sy, theme)
}

// normalizeAngle は角度を [-π, π] に丸める。
func normalizeAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
