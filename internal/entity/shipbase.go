package entity

// shipbase.go は「機体ベース（土台）」の型・レジストリ・描画を提供する。
// プレイヤー機はベースの上に GridHalf×2+1 角のグリッドを持ち、その上にパーツを配置する。
// ベースは基礎 HP・基礎積載・非常用推進（スラスタ未搭載時のフォールバック）を提供し、
// コックピットの役割を内包する（コックピットは独立パーツではない）。
// 個別ベースの定義データは data_shipbases.go を参照（ADR 0005）。

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

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

// shipHullPolygon は中心 (cx, cy) を基準にしたベース船体の輪郭頂点を返す（時計回り、ローカル -y が前方）。
func shipHullPolygon(cx, cy float64, wHalf, hHalf float64) [][2]float64 {
	return [][2]float64{
		{cx, cy - hHalf},                  // 機首
		{cx + wHalf*0.55, cy - hHalf*0.5}, // 右肩
		{cx + wHalf, cy + hHalf*0.12},     // 右舷中央
		{cx + wHalf*0.7, cy + hHalf*0.7},  // 右腰
		{cx + wHalf*0.42, cy + hHalf},     // 右尾
		{cx - wHalf*0.42, cy + hHalf},     // 左尾
		{cx - wHalf*0.7, cy + hHalf*0.7},  // 左腰
		{cx - wHalf, cy + hHalf*0.12},     // 左舷中央
		{cx - wHalf*0.55, cy - hHalf*0.5}, // 左肩
	}
}

// DrawShipBase は (cx, cy) を中心としたベース船体をワイヤーフレームで描画する。
// 背景色で塗りつぶして船体シルエット（星空を遮る）を作り、theme.Line で輪郭、
// theme.LineDim で内部ディテール（中心線・補強材）を描く。
// cellSize はグリッド 1 セルの論理ピクセル（探索は GridSize、エディタは拡大値）。
func DrawShipBase(dst *ebiten.Image, cx, cy float64, gridHalf int, cellSize float64, theme *ui.Theme) {
	wHalf, hHalf := shipHullExtent(gridHalf, cellSize)
	pts := shipHullPolygon(cx, cy, wHalf, hHalf)

	// 不透明な背景色で塗り、船体のシルエットを作る（後ろの星空を遮る）。
	var path vector.Path
	path.MoveTo(float32(pts[0][0]), float32(pts[0][1]))
	for _, p := range pts[1:] {
		path.LineTo(float32(p[0]), float32(p[1]))
	}
	path.Close()
	fill := theme.Background
	fill.A = 255
	fop := &vector.FillOptions{}
	dop := &vector.DrawPathOptions{AntiAlias: true}
	dop.ColorScale.ScaleWithColor(fill)
	vector.FillPath(dst, &path, fop, dop)

	// 内部ディテール（薄い色）: 中心線と 2 本の補強材。輪郭の前に描いて下に敷く。
	dim := theme.LineDim
	detailW := float32(math.Max(1, cellSize*0.03))
	vector.StrokeLine(dst, float32(cx), float32(cy-hHalf*0.55), float32(cx), float32(cy+hHalf*0.85), detailW, dim, true)
	vector.StrokeLine(dst, float32(cx-wHalf*0.6), float32(cy+hHalf*0.1), float32(cx+wHalf*0.6), float32(cy+hHalf*0.1), detailW, dim, true)
	vector.StrokeLine(dst, float32(cx-wHalf*0.45), float32(cy+hHalf*0.55), float32(cx+wHalf*0.45), float32(cy+hHalf*0.55), detailW, dim, true)

	// 輪郭（明るい色）。
	lineW := float32(math.Max(1.5, cellSize*0.045))
	for i := range pts {
		j := (i + 1) % len(pts)
		vector.StrokeLine(dst,
			float32(pts[i][0]), float32(pts[i][1]),
			float32(pts[j][0]), float32(pts[j][1]),
			lineW, theme.Line, true)
	}
}
