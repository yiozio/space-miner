package entity

// data_shipbases.go は全機体ベースの定義データ（ADR 0005: データと実装の分離）。
// 新しいベース（中型・大型など）を追加するときは ShipBaseID を増やし、
// init() の registerShipBaseDef でステータスを記述する。型・レジストリは shipbase.go を参照。

func init() {
	// Pebble: 初期ベース。3x3 グリッド。
	// 基礎 HP と非常推進は旧コックピットパーツの値を引き継ぐ（コックピット機能を内包）。
	registerShipBaseDef(&ShipBaseDef{
		ID:       ShipBasePebble,
		GridHalf: 1, // 3x3
		Price:    0,
		BaseHP:   PlayerHPDefault, // 100
		// 旧コックピット相当の最低限の積載・推進。
		BaseCargo:           15,
		ThrustAccel:         0.05,
		ThrustMaxSpeed:      2,
		ThrustBoostAccelMul: 1.4,
		ThrustBoostMaxSpeed: 5.0,
		ThrustBoostFuelCost: 0.10,
	})
}
