package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

const (
	// mineFuseFrames は設置から起爆までのフレーム数（約1秒）。
	mineFuseFrames = 60
	// mineBurstDirs は起爆時に放射する弾の方向数。
	mineBurstDirs = 6
)

// mineArmedColor は起爆直前の警告色（赤系）。
var mineArmedColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}

// Mine は発射時に機体位置へ設置される機雷。
// Fuse フレーム後に Detonate で mineBurstDirs 方向へ弾をばらまく。
// 弾の威力・速度・見た目は設置元のパーツ def から引き継ぐ。
// 自身は移動しない（設置位置に留まる）。
type Mine struct {
	X, Y            float64
	Fuse            int // 残存フレーム（0 で起爆）
	Damage          int
	BulletSpeed     float64
	Style           BulletStyle
	Width           float64
	ImpactFX        bool
	ExplosionRadius float64 // >0 なら起爆弾が着弾時に範囲ダメージを与える
}

// Update は信管を進め、起爆フレームに達したら true を返す。
func (m *Mine) Update() (detonated bool) {
	m.Fuse--
	return m.Fuse <= 0
}

// Detonate は中心から放射状に mineBurstDirs 方向の弾（プレイヤー弾）を生成する。
func (m *Mine) Detonate() []Bullet {
	bullets := make([]Bullet, 0, mineBurstDirs)
	for i := range mineBurstDirs {
		ang := float64(i) / float64(mineBurstDirs) * (2 * math.Pi)
		dx, dy := math.Cos(ang), math.Sin(ang)
		bullets = append(bullets, Bullet{
			X:               m.X,
			Y:               m.Y,
			VX:              dx * m.BulletSpeed,
			VY:              dy * m.BulletSpeed,
			Life:            bulletLifeFrames,
			Damage:          m.Damage,
			Hostile:         false,
			Style:           m.Style,
			Width:           m.Width,
			ImpactFX:        m.ImpactFX,
			ExplosionRadius: m.ExplosionRadius,
		})
	}
	return bullets
}

// Draw は機雷を点滅する円で描画する。起爆が近づくほど点滅が速くなる。
// sx, sy は描画スクリーン座標。
func (m *Mine) Draw(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	// 残存フレームが少ないほど点滅周期を短くし、起爆が近いことを伝える。
	period := m.Fuse/8 + 2
	if (m.Fuse/period)%2 != 0 {
		return // 消灯フェーズ
	}
	c := theme.Line
	if m.Fuse < 20 {
		c = mineArmedColor // 起爆直前は赤で警告
	}
	x := float32(sx)
	y := float32(sy)
	const r = 5
	vector.StrokeCircle(dst, x, y, r, 1.5, c, true)
	vector.FillCircle(dst, x, y, 1.5, c, true)
}
