package scene

import "github.com/yiozio/space-miner/internal/entity"

// data_shop.go はステーション店舗の在庫構成データ。
// 取扱パーツの並びや初期入荷数のチューニングはここで行う。
// シーン実装は station_shop.go を参照。

// shopStockIDs は初期店在庫として並べるパーツバリアント（PartID）。
// 並び順がそのまま店舗グリッドの並びになる。
var shopStockIDs = []entity.PartID{
	entity.PartIDGunMkI,
	entity.PartIDGunMkII,
	entity.PartIDGunRapid,
	entity.PartIDThrusterStd,
	entity.PartIDThrusterLight,
	entity.PartIDThrusterHeavy,
	entity.PartIDCargoStd,
	entity.PartIDArmorStd,
	entity.PartIDAutoAimStd,
	entity.PartIDFuelStd,
	entity.PartIDShieldStd,
}

// shopInitialQuantity は def の Kind に応じた初期入荷数。
// 高価・希少なものは少なめ、消耗系（弾薬枠等）は多めに置く想定。
func shopInitialQuantity(d *entity.PartDef) int {
	switch d.Kind {
	case entity.PartGun:
		return 4
	case entity.PartThruster:
		return 3
	case entity.PartFuel, entity.PartCargo:
		return 5
	case entity.PartAutoAim, entity.PartShield, entity.PartWarp:
		return 2
	}
	return 3
}
