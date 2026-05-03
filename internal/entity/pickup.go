package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	pickupAttractRadius = 250.0
	pickupAttractAccel  = 0.45
	pickupCollectRadius = 18.0
	pickupDrag          = 0.96
	pickupLifeFrames    = 60 * 30 // 30秒
	pickupDrawSize      = 5
)

// PickupKind は Pickup の中身の種別。
type PickupKind int

const (
	PickupResource PickupKind = iota // Resource フィールドが有効
	PickupPart                       // PartID フィールドが有効
)

// Pickup は回収可能なドロップ（資源 / パーツ）。
// プレイヤー機の半径に入ると吸い寄せられ、接触で取得される。
type Pickup struct {
	X, Y     float64
	VX, VY   float64
	Kind     PickupKind
	Resource ResourceType // Kind == PickupResource のとき有効
	PartID   PartID       // Kind == PickupPart のとき有効
	Life     int
}

// NewPickup は座標 (x, y) に資源 r を生成する。
func NewPickup(x, y float64, r ResourceType) Pickup {
	return Pickup{
		X:        x,
		Y:        y,
		Kind:     PickupResource,
		Resource: r,
		Life:     pickupLifeFrames,
	}
}

// NewPartPickup は座標 (x, y) にパーツドロップを生成する（海賊撃破時の稀ドロップ用）。
func NewPartPickup(x, y float64, id PartID) Pickup {
	return Pickup{
		X:      x,
		Y:      y,
		Kind:   PickupPart,
		PartID: id,
		Life:   pickupLifeFrames,
	}
}

// Update は1フレーム分位置を更新する。プレイヤーに吸引され、接触すると true を返す。
func (p *Pickup) Update(playerX, playerY float64) (collected bool) {
	p.Life--
	dx := playerX - p.X
	dy := playerY - p.Y
	dist := math.Hypot(dx, dy)
	if dist < pickupCollectRadius {
		return true
	}
	if dist < pickupAttractRadius && dist > 0.001 {
		nx := dx / dist
		ny := dy / dist
		p.VX += nx * pickupAttractAccel
		p.VY += ny * pickupAttractAccel
	}
	p.VX *= pickupDrag
	p.VY *= pickupDrag
	p.X += p.VX
	p.Y += p.VY
	return false
}

// Draw は (sx, sy) を中心にひし形マーカーを描く。
// 資源は資源色、パーツはやや明るい黄色で描画して区別する。
func (p *Pickup) Draw(dst *ebiten.Image, sx, sy float64) {
	var c color.NRGBA
	switch p.Kind {
	case PickupPart:
		c = color.NRGBA{0xff, 0xe0, 0x60, 0xff}
	default:
		c = p.Resource.Info().Color
	}
	s := float32(pickupDrawSize)
	x, y := float32(sx), float32(sy)
	vector.StrokeLine(dst, x-s, y, x, y-s, 1, c, false)
	vector.StrokeLine(dst, x, y-s, x+s, y, 1, c, false)
	vector.StrokeLine(dst, x+s, y, x, y+s, 1, c, false)
	vector.StrokeLine(dst, x, y+s, x-s, y, 1, c, false)
}
