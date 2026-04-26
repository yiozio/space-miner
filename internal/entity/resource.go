package entity

import "image/color"

// ResourceType は採掘可能な資源の種別。
// 詳細は docs/GAME_DESIGN.md「採掘システム / 小惑星の構造」を参照。
type ResourceType int

const (
	ResourceIron ResourceType = iota
	ResourceCrystal
	ResourceIce
	resourceCount
)

// ResourceInfo は資源種別ごとの視覚＆HP情報。
// Color はグリッド表示色（種別判別用）、MaxHP はグリッド毎の最大HP。
type ResourceInfo struct {
	Name  string
	Color color.NRGBA
	MaxHP int
}

var resourceInfos = [resourceCount]ResourceInfo{
	ResourceIron:    {Name: "IRON", Color: color.NRGBA{0xc8, 0xc8, 0xc8, 0xff}, MaxHP: 3},
	ResourceCrystal: {Name: "CRYSTAL", Color: color.NRGBA{0x80, 0x80, 0xff, 0xff}, MaxHP: 8},
	ResourceIce:     {Name: "ICE", Color: color.NRGBA{0x80, 0xe0, 0xff, 0xff}, MaxHP: 2},
}

// Info は ResourceType の表示・HP情報を返す。
func (r ResourceType) Info() ResourceInfo {
	return resourceInfos[r]
}

// AllResourceTypes は全資源種別を順に返す（HUD 表示や生成で使用）。
func AllResourceTypes() []ResourceType {
	return []ResourceType{ResourceIron, ResourceCrystal, ResourceIce}
}
