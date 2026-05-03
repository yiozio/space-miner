package entity

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// LaserShot は瞬間命中レーザーの発射要求。
// Player.Shoot / Pirate.shoot が返し、シーン側でレイキャスト → 即ダメージ + Beam 描画する。
type LaserShot struct {
	X, Y     float64 // 発射位置（ガン前端）
	DX, DY   float64 // 単位方向ベクトル
	Damage   int
	Range    float64
	Hostile  bool
	Width    float64
	ImpactFX bool
}

const beamLifeFrames = 8

// Beam は瞬間レーザーの可視化エフェクト。短時間の直線として描画される。
type Beam struct {
	X1, Y1 float64
	X2, Y2 float64
	Width  float64
	Color  color.NRGBA
	Age    int
}

// NewBeam は (x1,y1) → (x2,y2) のビームを生成する。
func NewBeam(x1, y1, x2, y2, width float64, c color.NRGBA) Beam {
	return Beam{X1: x1, Y1: y1, X2: x2, Y2: y2, Width: width, Color: c}
}

// Update は age を進め、寿命到達なら true を返す。
func (b *Beam) Update() (done bool) {
	b.Age++
	return b.Age >= beamLifeFrames
}

// DrawScreen は (sx1,sy1) → (sx2,sy2) でビームを描画する（カメラ変換は呼び出し側）。
// 寿命に応じてアルファをフェードアウトする。
func (b *Beam) DrawScreen(dst *ebiten.Image, sx1, sy1, sx2, sy2 float64) {
	progress := float64(b.Age) / float64(beamLifeFrames)
	if progress > 1 {
		progress = 1
	}
	c := b.Color
	c.A = uint8((1 - progress) * 255)
	w := float32(b.Width)
	if w <= 0 {
		w = 2
	}
	vector.StrokeLine(dst, float32(sx1), float32(sy1), float32(sx2), float32(sy2), w, c, true)
}
