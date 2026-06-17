package entity

// shipbase.go は「機体ベース（土台）」の型・レジストリ・描画を提供する。
// プレイヤー機はベースの上に GridHalf×2+1 角のグリッドを持ち、その上にパーツを配置する。
// ベースは基礎 HP・基礎積載・非常用推進（スラスタ未搭載時のフォールバック）を提供し、
// コックピットの役割を内包する（コックピットは独立パーツではない）。
// 個別ベースの定義データは data_shipbases.go を参照（ADR 0005）。

import (
	"github.com/hajimehoshi/ebiten/v2"

	assetimage "github.com/yiozio/space-miner/internal/asset/image"
	"github.com/yiozio/space-miner/internal/ui"
)

// ShipBaseID は機体ベースの識別子。ゼロ値が初期ベースになるよう Pebble を先頭に置く。
type ShipBaseID int

const (
	ShipBasePebble ShipBaseID = iota
)

// ShipBaseDef は機体ベースの定義。グリッドサイズと基礎ステータス、非常推進性能を持つ。
// 将来サイズ違いのベースを追加するときはここに def を増やす（data_shipbases.go）。
type ShipBaseDef struct {
	ID       ShipBaseID
	GridHalf int // 配置グリッドの半径。3x3 なら 1（セルは -GridHalf..GridHalf）
	Price    int

	// 基礎ステータス（パーツの合算に加算される土台値）。
	BaseHP    int     // 基本 HP（Armor の ArmorHP がこれに上乗せ）
	BaseCargo float64 // 基本積載量（Cargo パーツがこれに上乗せ）

	// 非常用推進。Thruster を 1 つも積んでいないときだけ前進方向に使う
	// （旧コックピットの最低限スラスタ性能と同じ役割）。
	ThrustAccel         float64
	ThrustMaxSpeed      float64
	ThrustBoostAccelMul float64
	ThrustBoostMaxSpeed float64
	ThrustBoostFuelCost float64
}

// EmergencyThrust は非常用推進性能を PartDef 形に詰めて返す（thrusterStatsByDir で合算に使う）。
func (b *ShipBaseDef) EmergencyThrust() *PartDef {
	return &PartDef{
		Kind:                PartThruster,
		ThrustAccel:         b.ThrustAccel,
		ThrustMaxSpeed:      b.ThrustMaxSpeed,
		ThrustBoostAccelMul: b.ThrustBoostAccelMul,
		ThrustBoostMaxSpeed: b.ThrustBoostMaxSpeed,
		ThrustBoostFuelCost: b.ThrustBoostFuelCost,
	}
}

var (
	shipBaseDefs  = map[ShipBaseID]*ShipBaseDef{}
	shipBaseOrder []ShipBaseID
)

// registerShipBaseDef は data_shipbases.go の init() から呼ばれる登録用 API。
func registerShipBaseDef(d *ShipBaseDef) {
	if _, exists := shipBaseDefs[d.ID]; exists {
		panic("duplicate ShipBaseID registration")
	}
	shipBaseDefs[d.ID] = d
	shipBaseOrder = append(shipBaseOrder, d.ID)
}

// ShipBaseDefByID はレジストリから def を取得する。未登録 ID では Pebble を返す（安全側）。
func ShipBaseDefByID(id ShipBaseID) *ShipBaseDef {
	if d, ok := shipBaseDefs[id]; ok {
		return d
	}
	return shipBaseDefs[ShipBasePebble]
}

// shipHullExtent は gridHalf のベース船体の外形半幅・半高（px）を返す。
// グリッド（半径 (gridHalf+0.5)*cellSize）よりやや広く・縦に長く取り、機首を前に突き出す。
func shipHullExtent(gridHalf int, cellSize float64) (wHalf, hHalf float64) {
	ext := (float64(gridHalf) + 0.5) * cellSize
	return ext * 1.04, ext * 1.30
}

// DrawShipBase は (cx, cy) を 3x3 グリッド中心として、ベース船体スプライトを描画する。
// cellSize はグリッド 1 セルの論理ピクセル（探索は GridSize、エディタは拡大値）。
// スプライトはセル 16px を cellSize に拡大（ニアレスト補間）して敷く。
// 色分け（海賊機の赤など）は呼び出し側が ColorScale で機体全体に適用する。
func DrawShipBase(dst *ebiten.Image, cx, cy float64, gridHalf int, cellSize float64, _ *ui.Theme) {
	base := assetimage.ShipBase()
	scale := cellSize / float64(assetimage.CellSize)
	// スプライト内での 3x3 グリッド中心（パネル左上 + パネル数 × 16 / 2）。
	span := float64(2*gridHalf+1) * float64(assetimage.CellSize)
	gcx := float64(assetimage.ShipBaseGridX) + span/2
	gcy := float64(assetimage.ShipBaseGridY) + span/2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(cx-gcx*scale, cy-gcy*scale)
	dst.DrawImage(base, op)
}
