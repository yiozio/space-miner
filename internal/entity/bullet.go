package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

const (
	bulletLifeFrames = 60 // 約1秒
)

// BulletStyle は弾の見た目分類。
type BulletStyle int

const (
	BulletStyleTrail BulletStyle = iota // 短いライン（モーションブラー風）
	BulletStyleBall                     // 塗り円（プラズマ砲）
	BulletStyleLaser                    // 長いライン（高速ビーム）
)

// Bullet はプレイヤーまたは敵が発射する弾。
// Damage は命中対象に与えるダメージで、発射したガンの def に従う。
// Hostile が true なら敵弾（プレイヤーを攻撃）、false ならプレイヤー弾（小惑星と海賊を攻撃）。
// Style/Width で見た目、ImpactFX で着弾時のエフェクト発生有無を制御する。
type Bullet struct {
	X, Y     float64
	VX, VY   float64
	Life     int // 残存フレーム
	Damage   int
	Hostile  bool
	Style    BulletStyle
	Width    float64 // Trail/Laser ではライン太さ、Ball では半径
	ImpactFX bool
}

func (b *Bullet) Update() {
	b.X += b.VX
	b.Y += b.VY
	b.Life--
}

func (b *Bullet) Alive() bool { return b.Life > 0 }

// Draw は弾を Style に応じて描画する。
// Trail: 1 フレーム前の見かけ位置への線（既定）
// Ball:  塗り円
// Laser: 進行方向に長く伸ばした線（ビーム風）
// 敵弾は赤系で描画してプレイヤー弾と区別する。
// カメラも動くため Trail は見かけのスクリーン速度（ワールド速度 − カメラ速度）で長さを決める。
func (b *Bullet) Draw(dst *ebiten.Image, sx, sy, viewVX, viewVY float64, theme *ui.Theme) {
	c := theme.Line
	if b.Hostile {
		c = pirateBulletColor
	}
	w := float32(b.Width)
	if w <= 0 {
		w = 2
	}
	switch b.Style {
	case BulletStyleBall:
		r := float32(b.Width)
		if r <= 0 {
			r = 3
		}
		vector.DrawFilledCircle(dst, float32(sx), float32(sy), r, c, true)
	case BulletStyleLaser:
		// 進行方向に長く伸ばす（ワールド速度ベクトルを正規化して固定長）
		const laserLen = 60.0
		spd := math.Hypot(b.VX, b.VY)
		if spd < 0.001 {
			spd = 1
		}
		dx := b.VX / spd * laserLen
		dy := b.VY / spd * laserLen
		// 弾の前方（進行方向）にも少し伸ばし、後方にも伸ばすことで beam 感を強調
		x1 := float32(sx - dx*0.2)
		y1 := float32(sy - dy*0.2)
		x2 := float32(sx + dx*0.8)
		y2 := float32(sy + dy*0.8)
		vector.StrokeLine(dst, x1, y1, x2, y2, w, c, true)
	default:
		dvx := b.VX - viewVX
		dvy := b.VY - viewVY
		tailX := sx - dvx
		tailY := sy - dvy
		vector.StrokeLine(dst, float32(tailX), float32(tailY),
			float32(sx), float32(sy), w, c, false)
	}
}
