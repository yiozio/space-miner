package entity

import (
	"math"
	"math/rand"
)

// ResourceWeight は ResourceZone 内での素材抽選の重み。
type ResourceWeight struct {
	Resource ResourceType
	Weight   float64
}

// ResourceZone は中心 (CX, CY)・半径 Radius の円形範囲で、
// 中心ほど多く小惑星を湧かせる素材ゾーン。
// MaxAsteroids はゾーン中心での同時生成上限。中心からの距離に応じて線形に減衰する。
// Mix はそのゾーンで生成される小惑星の素材比率。1 小惑星 = 1 素材。
type ResourceZone struct {
	CX, CY       float64
	Radius       float64
	MaxAsteroids int
	Mix          []ResourceWeight
}

// FullMap は世界の一区画。中心 (CX, CY) ・半幅/半高 HalfW/HalfH の矩形領域で、
// 内部に複数のゾーンを持つ（典型的には宇宙ステーションを中心に取る）。
// Name は UI 表示用の宙域名。区画外には何も生成されない
// （ワープあるいは忍耐強く航行することで別区画に到達する想定）。
type FullMap struct {
	Name         string
	CX, CY       float64
	HalfW, HalfH float64
	Zones        []ResourceZone
}

// Contains は (x, y) がこの区画内にあるか返す。
func (m *FullMap) Contains(x, y float64) bool {
	return math.Abs(x-m.CX) <= m.HalfW && math.Abs(y-m.CY) <= m.HalfH
}

// World はだだっ広い宇宙全体。複数の FullMap を含み、World 自身には境界がない。
// 区画は遠く離れて点在し、その間は空虚。
type World struct {
	Maps []FullMap
}

// Containing は (x, y) を含む最初の FullMap を返す。なければ nil。
// 区画は重ならない前提で利用する。
func (w *World) Containing(x, y float64) *FullMap {
	for i := range w.Maps {
		if w.Maps[i].Contains(x, y) {
			return &w.Maps[i]
		}
	}
	return nil
}

// InBounds は (x, y) がいずれかの FullMap 内にあるか返す。
// 区画外では小惑星は生成されない。
func (w *World) InBounds(x, y float64) bool {
	return w.Containing(x, y) != nil
}

// SpawnCap は (px, py) を中心とした、現フレームの小惑星同時生成上限。
// (px, py) を含む FullMap がなければ 0。
func (w *World) SpawnCap(px, py float64) int {
	m := w.Containing(px, py)
	if m == nil {
		return 0
	}
	cap := 0.0
	for i := range m.Zones {
		z := &m.Zones[i]
		d := math.Hypot(px-z.CX, py-z.CY)
		if d >= z.Radius {
			continue
		}
		cap += float64(z.MaxAsteroids) * (1 - d/z.Radius)
	}
	return int(cap)
}

// PickResource は (x, y) を含む FullMap のゾーン群から、
// フォールオフ込みの重みを合算して素材を抽選する。
// 区画外・全ゾーン外なら ok=false。
func (w *World) PickResource(x, y float64, rng *rand.Rand) (ResourceType, bool) {
	m := w.Containing(x, y)
	if m == nil {
		return 0, false
	}
	var weights [resourceCount]float64
	total := 0.0
	for i := range m.Zones {
		z := &m.Zones[i]
		d := math.Hypot(x-z.CX, y-z.CY)
		if d >= z.Radius {
			continue
		}
		f := 1 - d/z.Radius
		for _, mw := range z.Mix {
			weights[mw.Resource] += mw.Weight * f
			total += mw.Weight * f
		}
	}
	if total <= 0 {
		return 0, false
	}
	r := rng.Float64() * total
	for i := 0; i < int(resourceCount); i++ {
		if weights[i] <= 0 {
			continue
		}
		r -= weights[i]
		if r <= 0 {
			return ResourceType(i), true
		}
	}
	return ResourceType(resourceCount - 1), true
}
