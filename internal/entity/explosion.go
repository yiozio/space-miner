package entity

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	explosionLifeFrames = 36
	explosionDebrisN    = 10
)

// Explosion は海賊撃墜時などの大型爆発エフェクト。
// 中心のフラッシュ、複数の拡大リング、放射状に飛び散るデブリ線で構成する。
type Explosion struct {
	X, Y   float64
	Age    int
	Color  color.NRGBA
	debris [explosionDebrisN]explosionDebris
}

// explosionDebris は外向きに飛ぶ短い線分の生成パラメータ。
// angle: 飛行方向、speed: 1 フレームの移動量、length: 線の長さ。
type explosionDebris struct {
	angle  float64
	speed  float64
	length float64
}

// NewExplosion は (x, y) に大型爆発を生成する。rng は呼び出し側の乱数源を共有する想定。
func NewExplosion(x, y float64, c color.NRGBA, rng *rand.Rand) Explosion {
	e := Explosion{X: x, Y: y, Color: c}
	for i := range e.debris {
		e.debris[i] = explosionDebris{
			angle:  rng.Float64() * math.Pi * 2,
			speed:  1.4 + rng.Float64()*2.2,
			length: 10 + rng.Float64()*16,
		}
	}
	return e
}

// Update は age を進め、終了したら true を返す。
func (e *Explosion) Update() (done bool) {
	e.Age++
	return e.Age >= explosionLifeFrames
}

// Draw は (sx, sy) を中心に爆発を描画する。
func (e *Explosion) Draw(dst *ebiten.Image, sx, sy float64) {
	p := float64(e.Age) / float64(explosionLifeFrames)
	if p > 1 {
		p = 1
	}

	// 中心フラッシュ: 序盤に強く光り、急速にフェードする塗り円。
	flashP := math.Min(1, p*4)
	if flashP < 1 {
		c := e.Color
		c.A = uint8((1 - flashP) * 220)
		vector.FillCircle(dst, float32(sx), float32(sy), float32(10+30*p), c, true)
	}

	// 拡大リング 3 本。後発のリングほど開始が遅く、外側まで届く。
	const rings = 3
	for i := 0; i < rings; i++ {
		delay := float64(i) * 0.18
		if p < delay {
			continue
		}
		rp := (p - delay) / (1 - delay)
		radius := float32(8 + 70*rp)
		c := e.Color
		c.A = uint8((1 - rp) * 220)
		vector.StrokeCircle(dst, float32(sx), float32(sy), radius, 1.5, c, true)
	}

	// 放射状デブリ: 一定速度で外向きに飛ぶ短い線。
	for _, dbr := range e.debris {
		dist := dbr.speed * float64(e.Age)
		cx, sy0 := math.Cos(dbr.angle), math.Sin(dbr.angle)
		x1 := sx + cx*dist
		y1 := sy + sy0*dist
		x2 := x1 + cx*dbr.length
		y2 := y1 + sy0*dbr.length
		c := e.Color
		c.A = uint8((1 - p) * 200)
		vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 1.5, c, false)
	}
}
