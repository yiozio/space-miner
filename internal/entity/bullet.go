package entity

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

const (
	bulletLifeFrames = 60 // 約1秒
	bulletTrailWidth = 2
)

// Bullet はプレイヤー（または将来の敵）が発射する弾。
// Damage は命中したアステロイドグリッドに与えるダメージで、発射したガンの def に従う。
type Bullet struct {
	X, Y   float64
	VX, VY float64
	Life   int // 残存フレーム
	Damage int
}

func (b *Bullet) Update() {
	b.X += b.VX
	b.Y += b.VY
	b.Life--
}

func (b *Bullet) Alive() bool { return b.Life > 0 }

// Draw は弾の「現在地点と1フレーム前の地点（スクリーン上）」を結ぶ線として描画する。
// カメラ自体も移動するため、ワールド速度をそのままトレイルに使うと方向がズレる。
// 見かけのスクリーン速度 = ワールド速度 − カメラ速度（viewVX, viewVY）で計算する。
func (b *Bullet) Draw(dst *ebiten.Image, sx, sy, viewVX, viewVY float64, theme *ui.Theme) {
	dvx := b.VX - viewVX
	dvy := b.VY - viewVY
	tailX := sx - dvx
	tailY := sy - dvy
	vector.StrokeLine(dst, float32(tailX), float32(tailY),
		float32(sx), float32(sy), bulletTrailWidth, theme.Line, false)
}
