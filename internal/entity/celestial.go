package entity

import "image/color"

// CelestialKind は天体の種別。
type CelestialKind int

const (
	CelestialStar CelestialKind = iota
	CelestialPlanet
	CelestialMoon
)

// Celestial は恒星系の天体（恒星・惑星・衛星）。
// 同じ実体を恒星マップ表示と探索シーンの背景描画で共用する。
//
// 恒星マップ上の位置は FullMap.CX/CY をそのまま用いる（恒星は常に世界座標 0,0 と扱う）。
// これにより「恒星マップで見たワープ方向」と「実際の世界座標差で計算するワープ角度」が
// 同じになる。
type Celestial struct {
	Name  string
	Kind  CelestialKind
	Color color.NRGBA

	// Radius は恒星マップ表示用の論理半径（マップスケールに乗る）。
	Radius float64
	// 衛星の場合の親惑星名。星・惑星では空文字。
	// 恒星マップで親子関係を補助線として描画する。
	ParentName string

	// 探索シーン背景用 ------------------------------------------------
	// BackdropRadius が 0 なら背景描画なし。
	BackdropRadius float64
	// FullMap 中心からの位置オフセット（ワールド座標）。
	BackdropOffsetX float64
	BackdropOffsetY float64
}
