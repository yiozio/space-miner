package entity

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AsteroidGrid は小惑星を構成する1グリッド。
type AsteroidGrid struct {
	GX, GY   int
	Resource ResourceType
	HP       int
}

// Asteroid はワールド座標 (X, Y) を中心とした、共通グリッドサイズの集合体。
// 各グリッドは独自の HP を持ち、破壊されると Pickup として落ちる。
type Asteroid struct {
	X, Y  float64
	Grids []AsteroidGrid
}

// NewAsteroid は (x, y) を中心とする size マス分の小惑星を生成する。
// 形状はランダムウォーク的にグリッドを連結し、各グリッドの資源種別もランダム。
func NewAsteroid(seed int64, x, y float64, size int) *Asteroid {
	rng := rand.New(rand.NewSource(seed))
	cells := map[[2]int]bool{{0, 0}: true}
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(cells) < size {
		// 既存セルからランダムに伸ばす
		var keys [][2]int
		for k := range cells {
			keys = append(keys, k)
		}
		base := keys[rng.Intn(len(keys))]
		d := dirs[rng.Intn(4)]
		next := [2]int{base[0] + d[0], base[1] + d[1]}
		cells[next] = true
	}
	a := &Asteroid{X: x, Y: y}
	types := AllResourceTypes()
	for k := range cells {
		r := types[rng.Intn(len(types))]
		a.Grids = append(a.Grids, AsteroidGrid{
			GX: k[0], GY: k[1],
			Resource: r,
			HP:       r.Info().MaxHP,
		})
	}
	return a
}

// Hit はワールド座標 (bx, by) が小惑星のいずれかのグリッドに含まれていれば
// そのグリッドに 1 ダメージ与え、絶命したグリッドからは Pickup を返す。
// absorbed が true なら呼び出し側で弾を消す想定。
func (a *Asteroid) Hit(bx, by float64) (absorbed bool, pickups []Pickup) {
	g := float64(GridSize)
	for i := 0; i < len(a.Grids); i++ {
		gr := &a.Grids[i]
		cx := a.X + float64(gr.GX)*g
		cy := a.Y + float64(gr.GY)*g
		if bx >= cx-g/2 && bx < cx+g/2 && by >= cy-g/2 && by < cy+g/2 {
			gr.HP -= bulletDamage
			if gr.HP <= 0 {
				pickups = append(pickups, NewPickup(cx, cy, gr.Resource))
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
// 各グリッドは資源色で輪郭描画。HP が減るほどアルファが下がり「もろさ」を表現。
func (a *Asteroid) Draw(dst *ebiten.Image, sx, sy float64) {
	g := float32(GridSize)
	for _, gr := range a.Grids {
		info := gr.Resource.Info()
		cx := float32(sx) + float32(gr.GX)*g
		cy := float32(sy) + float32(gr.GY)*g
		c := info.Color
		if gr.HP < info.MaxHP {
			ratio := float64(gr.HP) / float64(info.MaxHP)
			c.A = uint8(80 + 175*ratio)
		}
		vector.StrokeRect(dst, cx-g/2, cy-g/2, g, g, 2, c, false)
	}
}
