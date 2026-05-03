package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

// 海賊ビジュアル用のアクセント色。弾と敵識別マーカーに使う。
var (
	pirateBulletColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}
	pirateLineColor   = color.NRGBA{0xff, 0x60, 0x40, 0xff}
)

// Pirate は敵 NPC 機体。Ship を継承し、簡易な追尾 AI と発射制御を持つ。
// 機体構成と行動パラメータは PiratePattern から決まる。
type Pirate struct {
	Ship
	HP        int
	MaxHP     int
	Pattern   *PiratePattern
	fireTimer int
	// 描画キャッシュ（海賊専用色で生成）
	cachedTheme *ui.Theme
	image       *ebiten.Image
	imgOffsetX  float64
	imgOffsetY  float64
}

// NewPirate は (x, y) に pattern に従う海賊機を生成する。最初は player 方向を向く。
func NewPirate(x, y float64, playerX, playerY float64, pattern *PiratePattern) *Pirate {
	parts := make([]Part, len(pattern.Parts))
	copy(parts, pattern.Parts)
	angle := math.Atan2(playerY-y, playerX-x)
	return &Pirate{
		Ship: Ship{
			Parts: parts,
			X:     x, Y: y,
			Angle: angle,
		},
		HP:      pattern.MaxHP,
		MaxHP:   pattern.MaxHP,
		Pattern: pattern,
	}
}

// Update は 1 フレーム分 AI を進め、発射した弾を返す。
// 動作: プレイヤー方向に旋回し、PreferredDist に近づこうとする。
// FireRange 内で機首がほぼ向いているとき発射する。
func (p *Pirate) Update(playerX, playerY float64) []Bullet {
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

	// 発射条件: 射程内 + 機首がほぼ合っている + クールダウンが切れている
	if dist < p.Pattern.FireRange && math.Abs(da) < 0.35 && p.fireTimer == 0 {
		return p.shoot()
	}
	return nil
}

// shoot は装着 Gun から敵弾（Hostile=true）を発射する。
// 弾は各 Gun の Rotation に従う方向に射出される（Player.Shoot と同じロジック）。
// クールダウンは最も遅い Gun の値に揃える。
func (p *Pirate) shoot() []Bullet {
	var out []Bullet
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
		cxL := float64(part.GX) * g
		cyL := float64(part.GY) * g
		frontLx := cxL + fxL*halfG
		frontLy := cyL + fyL*halfG
		wox, woy := toWorld(frontLx, frontLy)
		fwx, fwy := toWorld(fxL, fyL)
		out = append(out, Bullet{
			X:       p.X + wox,
			Y:       p.Y + woy,
			VX:      fwx*d.GunBulletSpeed + p.VX,
			VY:      fwy*d.GunBulletSpeed + p.VY,
			Life:    bulletLifeFrames,
			Damage:  d.GunDamage,
			Hostile: true,
		})
		if d.GunCooldown > maxCD {
			maxCD = d.GunCooldown
		}
	}
	if len(out) > 0 {
		p.fireTimer = maxCD
	}
	return out
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
// プレイヤー機と区別するため、ライン色を pirateLineColor に置換した
// 専用の船体画像を内部キャッシュする。
func (p *Pirate) DrawAt(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	p.ensureImage(theme)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-p.imgOffsetX, -p.imgOffsetY)
	op.GeoM.Rotate(p.Angle + math.Pi/2)
	op.GeoM.Translate(sx, sy)
	dst.DrawImage(p.image, op)

	// 敵識別: 赤い輪郭リング
	vector.StrokeCircle(dst, float32(sx), float32(sy), 30, 1.5, pirateLineColor, true)
}

func (p *Pirate) ensureImage(theme *ui.Theme) {
	if p.cachedTheme == theme && p.image != nil {
		return
	}
	minGX, minGY, maxGX, maxGY := p.bounds()
	w := (maxGX - minGX + 1) * GridSize
	h := (maxGY - minGY + 1) * GridSize
	img := ebiten.NewImage(w, h)
	pirateTheme := *theme
	pirateTheme.Line = pirateLineColor
	for _, part := range p.Parts {
		x := float32((part.GX - minGX) * GridSize)
		y := float32((part.GY - minGY) * GridSize)
		DrawPart(img, part.Kind(), x, y, float32(GridSize), &pirateTheme, part.Rotation)
	}
	p.image = img
	p.imgOffsetX = float64(-minGX*GridSize) + float64(GridSize)/2
	p.imgOffsetY = float64(-minGY*GridSize) + float64(GridSize)/2
	p.cachedTheme = theme
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
