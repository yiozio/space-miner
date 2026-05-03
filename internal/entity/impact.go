package entity

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const impactLifeFrames = 16

// Impact は弾の着弾エフェクト。広がりながら消える円リング。
// ImpactFX 付きの弾が命中したときに生成される。
type Impact struct {
	X, Y  float64
	Age   int
	Color color.NRGBA
}

// NewImpact は (x, y) に着弾エフェクトを生成する。
func NewImpact(x, y float64, c color.NRGBA) Impact {
	return Impact{X: x, Y: y, Color: c}
}

// Update は age を進め、終了したら true を返す。
func (i *Impact) Update() (done bool) {
	i.Age++
	return i.Age >= impactLifeFrames
}

// Draw は (sx, sy) を中心に拡大しながらフェードする円を描く。
func (i *Impact) Draw(dst *ebiten.Image, sx, sy float64) {
	progress := float64(i.Age) / float64(impactLifeFrames)
	if progress > 1 {
		progress = 1
	}
	radius := float32(3 + 18*progress)
	c := i.Color
	c.A = uint8((1 - progress) * 220)
	vector.StrokeCircle(dst, float32(sx), float32(sy), radius, 1.5, c, true)
}
