package entity

import "image/color"

// data_resources.go は採掘可能な資源の定義データ。
// 新しい資源を追加するときは、ResourceType 定数と resourceInfos エントリ、
// および売却単価 resourcePrices を同時に拡張する。
// 型・参照 API は resource.go を参照。

// ResourceType は採掘可能な資源の種別。
// 詳細は docs/GAME_DESIGN.md「採掘システム / 小惑星の構造」を参照。
type ResourceType int

const (
	ResourceIron ResourceType = iota
	ResourceBronze
	ResourceIce
	resourceCount
)

// resourceInfos は各資源の表示・HP・重量情報。
// Color はグリッド表示色（種別判別用）、MaxHP はグリッド毎の最大HP、Weight は単位重量。
var resourceInfos = [resourceCount]ResourceInfo{
	ResourceIron:   {Name: "IRON", Color: color.NRGBA{0xc8, 0xc8, 0xc8, 0xff}, MaxHP: 3, Weight: 1.0},
	ResourceBronze: {Name: "BRONZE", Color: color.NRGBA{0xcd, 0x7f, 0x32, 0xff}, MaxHP: 8, Weight: 1.2},
	ResourceIce:    {Name: "ICE", Color: color.NRGBA{0x80, 0xe0, 0xff, 0xff}, MaxHP: 2, Weight: 0.7},
}

// resourcePrices は資源 1 単位あたりの売買単価（クレジット）。
var resourcePrices = [resourceCount]int{
	ResourceIron:   5,
	ResourceBronze: 30,
	ResourceIce:    8,
}
