package entity

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/ui"
)

// PartKind はパーツ種別。
type PartKind int

const (
	PartCockpit PartKind = iota
	PartGun
	PartThruster
	PartFuel
	PartCargo
	PartArmor
	PartShield
	PartAutoAim
	PartWarp
)

// Part はグリッド配置されたパーツ。
// (GX, GY) はコックピットを原点 (0, 0) としたローカル座標。
// +y はビジュアル上の後方（ローカル -y が機体の進行方向）。
type Part struct {
	Kind   PartKind
	GX, GY int
}

// drawPart は1パーツを image 上の (x, y) を該当グリッド左上として描画する。
// 描画は theme.Line のみを使うレトロベクター風の単色線画。
func drawPart(dst *ebiten.Image, p Part, x, y float32, theme *ui.Theme) {
	g := float32(GridSize)
	inset := g * 0.12
	cx := x + g/2
	cy := y + g/2
	line := theme.Line

	switch p.Kind {
	case PartCockpit:
		// 上向き三角＋ベースで「コックピット」
		vector.StrokeLine(dst, cx, y+inset, x+inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+g-inset, x+g-inset, y+g-inset, 1, line, false)
	case PartGun:
		// ベースの矩形＋前方（-y）に伸びる砲身
		vector.StrokeRect(dst, x+inset, y+g/2-3, g-inset*2, 6, 1, line, false)
		vector.StrokeLine(dst, cx, y+g/2-3, cx, y, 2, line, false)
	case PartThruster:
		// エンジンブロック＋背面（+y 側）の小さな三角排気
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g*0.55, 1, line, false)
		vector.StrokeLine(dst, x+g*0.32, y+g*0.7, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*0.68, y+g*0.7, cx, y+g-inset, 1, line, false)
	case PartFuel:
		// 縦線で燃料タンク
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+g/3, y+inset, x+g/3, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*2/3, y+inset, x+g*2/3, y+g-inset, 1, line, false)
	case PartCargo:
		// 矩形＋X
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g-inset, y+inset, x+inset, y+g-inset, 1, line, false)
	case PartArmor:
		// 太枠
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 2, line, false)
	case PartShield:
		// 矩形＋十字
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset+2, cx, y+g-inset-2, 1, line, false)
		vector.StrokeLine(dst, x+inset+2, cy, x+g-inset-2, cy, 1, line, false)
	case PartAutoAim:
		// 矩形＋十字レチクル
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, cy, x+g-inset, cy, 1, line, false)
	case PartWarp:
		// 矩形＋内側に小矩形（簡易）
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeRect(dst, x+g*0.3, y+g*0.3, g*0.4, g*0.4, 1, line, false)
	}
}
