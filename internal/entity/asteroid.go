package entity

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	asteroidMaxDriftSpeed = 0.4   // px/frame
	asteroidMaxAngularVel = 0.006 // rad/frame
)

// AsteroidGrid は小惑星を構成する1グリッド。
type AsteroidGrid struct {
	GX, GY   int
	Resource ResourceType
	HP       int
}

// Asteroid はワールド座標 (X, Y) を中心とした、共通グリッドサイズの集合体。
// 緩やかに浮遊（VX, VY）し、自転（Angle, AngularVel）する。
// 各グリッドは独自の HP を持ち、破壊されると Pickup として落ちる。
type Asteroid struct {
	X, Y       float64
	VX, VY     float64
	Angle      float64 // 自転角（ラジアン）
	AngularVel float64 // 自転速度（rad/frame）
	Grids      []AsteroidGrid
}

// NewAsteroid は (x, y) を中心とする size マス分の小惑星を、単一素材で生成する。
// 形状はランダムウォーク的にグリッドを連結する。
// 浮遊速度・自転は乱数で軽く揺らす。
func NewAsteroid(seed int64, x, y float64, size int, resource ResourceType) *Asteroid {
	rng := rand.New(rand.NewSource(seed))
	cells := map[[2]int]bool{{0, 0}: true}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(cells) < size {
		var keys [][2]int
		for k := range cells {
			keys = append(keys, k)
		}
		base := keys[rng.Intn(len(keys))]
		d := dirs[rng.Intn(4)]
		next := [2]int{base[0] + d[0], base[1] + d[1]}
		cells[next] = true
	}
	a := &Asteroid{
		X: x, Y: y,
		VX:         (rng.Float64()*2 - 1) * asteroidMaxDriftSpeed,
		VY:         (rng.Float64()*2 - 1) * asteroidMaxDriftSpeed,
		Angle:      rng.Float64() * 2 * math.Pi,
		AngularVel: (rng.Float64()*2 - 1) * asteroidMaxAngularVel,
	}
	maxHP := resource.Info().MaxHP
	for k := range cells {
		a.Grids = append(a.Grids, AsteroidGrid{
			GX: k[0], GY: k[1],
			Resource: resource,
			HP:       maxHP,
		})
	}
	return a
}

// Update は1フレーム分浮遊と自転を進める。
func (a *Asteroid) Update() {
	a.X += a.VX
	a.Y += a.VY
	a.Angle += a.AngularVel
}

// Hit はワールド座標 (bx, by) が小惑星のいずれかのグリッドに含まれていれば
// そのグリッドに damage 分のダメージを与え、絶命したグリッドからは Pickup を返す。
// 自転を考慮して、弾をアステロイドのローカル空間に逆回転で変換してから
// 軸並行のグリッド AABB と比較する。
func (a *Asteroid) Hit(bx, by float64, damage int) (absorbed bool, pickups []Pickup) {
	g := float64(GridSize)
	// ワールド → ローカル（中心基準で逆回転）
	sin, cos := math.Sin(-a.Angle), math.Cos(-a.Angle)
	dx := bx - a.X
	dy := by - a.Y
	lx := cos*dx - sin*dy
	ly := sin*dx + cos*dy
	for i := 0; i < len(a.Grids); i++ {
		gr := &a.Grids[i]
		cx := float64(gr.GX) * g
		cy := float64(gr.GY) * g
		if lx >= cx-g/2 && lx < cx+g/2 && ly >= cy-g/2 && ly < cy+g/2 {
			gr.HP -= damage
			if gr.HP <= 0 {
				// グリッド中心をワールドに戻して Pickup を生成
				fSin, fCos := math.Sin(a.Angle), math.Cos(a.Angle)
				wcx := fCos*cx - fSin*cy
				wcy := fSin*cx + fCos*cy
				pk := NewPickup(a.X+wcx, a.Y+wcy, gr.Resource)
				// 浮遊速度を継承（自転接線速度は微小なため省略）
				pk.VX = a.VX
				pk.VY = a.VY
				pickups = append(pickups, pk)
				a.Grids = append(a.Grids[:i], a.Grids[i+1:]...)
			}
			return true, pickups
		}
	}
	return false, nil
}

// Empty は全グリッドが破壊された状態か返す。
func (a *Asteroid) Empty() bool { return len(a.Grids) == 0 }

// Draw は (sx, sy) を小惑星中心としてグリッド群を描画する。
// 各グリッドは資源色で輪郭描画し、自転に合わせて 4 頂点を回転させる。
// HP が減るほどアルファが下がり「もろさ」を表現する。
func (a *Asteroid) Draw(dst *ebiten.Image, sx, sy float64) {
	g := float64(GridSize)
	sin, cos := math.Sin(a.Angle), math.Cos(a.Angle)
	half := g / 2
	for _, gr := range a.Grids {
		info := gr.Resource.Info()
		// ローカル中心 → 自転後のオフセット
		lcx := float64(gr.GX) * g
		lcy := float64(gr.GY) * g
		wox := cos*lcx - sin*lcy
		woy := sin*lcx + cos*lcy
		gcx := sx + wox
		gcy := sy + woy
		c := info.Color
		if gr.HP < info.MaxHP {
			ratio := float64(gr.HP) / float64(info.MaxHP)
			c.A = uint8(80 + 175*ratio)
		}
		drawRotatedSquare(dst, gcx, gcy, half, sin, cos, 2, c)
	}
}

// drawRotatedSquare は中心 (cx, cy)、半辺 half、与えられた sin/cos の回転で
// 正方形を 4 本の線として描画する。
// sin/cos を引数化することで、同一アステロイド内のループで再計算を避ける。
func drawRotatedSquare(dst *ebiten.Image, cx, cy, half, sin, cos float64, strokeWidth float32, c color.Color) {
	locals := [4][2]float64{
		{-half, -half},
		{half, -half},
		{half, half},
		{-half, half},
	}
	var corners [4][2]float32
	for i, l := range locals {
		rx := cos*l[0] - sin*l[1]
		ry := sin*l[0] + cos*l[1]
		corners[i][0] = float32(cx + rx)
		corners[i][1] = float32(cy + ry)
	}
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		vector.StrokeLine(dst,
			corners[i][0], corners[i][1],
			corners[j][0], corners[j][1],
			strokeWidth, c, false)
	}
}
