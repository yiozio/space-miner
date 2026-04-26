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

// PartName はパーツ種別の表示名（UI 用）を返す。
func PartName(kind PartKind) string {
	switch kind {
	case PartCockpit:
		return "Cockpit"
	case PartGun:
		return "Gun"
	case PartThruster:
		return "Thruster"
	case PartFuel:
		return "Fuel"
	case PartCargo:
		return "Cargo"
	case PartArmor:
		return "Armor"
	case PartShield:
		return "Shield"
	case PartAutoAim:
		return "Auto-Aim"
	case PartWarp:
		return "Warp"
	}
	return "Unknown"
}

// AllPlaceablePartKinds はエディタのパレットに並べる種別を返す。
// Cockpit は必ず原点に1つだけ存在する前提のため除外する。
func AllPlaceablePartKinds() []PartKind {
	return []PartKind{
		PartGun, PartThruster, PartFuel, PartCargo,
		PartArmor, PartShield, PartAutoAim, PartWarp,
	}
}

// DrawPart は1パーツを image 上の (x, y) を該当グリッド左上として描画する。
// cellSize はグリッド一辺の論理ピクセル数で、エディタのような拡大表示にも対応する。
// 描画は theme.Line のみを使うレトロベクター風の単色線画。
func DrawPart(dst *ebiten.Image, p Part, x, y, cellSize float32, theme *ui.Theme) {
	g := cellSize
	inset := g * 0.12
	cx := x + g/2
	cy := y + g/2
	line := theme.Line

	switch p.Kind {
	case PartCockpit:
		vector.StrokeLine(dst, cx, y+inset, x+inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+g-inset, x+g-inset, y+g-inset, 1, line, false)
	case PartGun:
		barrel := g * 0.18
		vector.StrokeRect(dst, x+inset, cy-barrel/2, g-inset*2, barrel, 1, line, false)
		vector.StrokeLine(dst, cx, cy-barrel/2, cx, y, 2, line, false)
	case PartThruster:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g*0.55, 1, line, false)
		vector.StrokeLine(dst, x+g*0.32, y+g*0.7, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*0.68, y+g*0.7, cx, y+g-inset, 1, line, false)
	case PartFuel:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+g/3, y+inset, x+g/3, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*2/3, y+inset, x+g*2/3, y+g-inset, 1, line, false)
	case PartCargo:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g-inset, y+inset, x+inset, y+g-inset, 1, line, false)
	case PartArmor:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 2, line, false)
	case PartShield:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		crossInset := g * 0.20
		vector.StrokeLine(dst, cx, y+crossInset, cx, y+g-crossInset, 1, line, false)
		vector.StrokeLine(dst, x+crossInset, cy, x+g-crossInset, cy, 1, line, false)
	case PartAutoAim:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, cy, x+g-inset, cy, 1, line, false)
	case PartWarp:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeRect(dst, x+g*0.3, y+g*0.3, g*0.4, g*0.4, 1, line, false)
	}
}
