package entity

import "image/color"

// resource.go は資源種別の型・参照 API を置く。
// 個別の資源データ（名前・色・HP・価格）は data_resources.go を参照。

// ResourceInfo は資源種別ごとの視覚＆HP情報。
// Weight は所持時の単位重量（カーゴ計算用）。
type ResourceInfo struct {
	Name   string
	Color  color.NRGBA
	MaxHP  int
	Weight float64
}

// Info は ResourceType の表示・HP情報を返す。
func (r ResourceType) Info() ResourceInfo {
	return resourceInfos[r]
}

// Price は資源 1 単位あたりの売買単価を返す。
func (r ResourceType) Price() int {
	if r < 0 || int(r) >= len(resourcePrices) {
		return 0
	}
	return resourcePrices[r]
}

// AllResourceTypes は全資源種別を順に返す（HUD 表示や生成で使用）。
func AllResourceTypes() []ResourceType {
	return []ResourceType{ResourceIron, ResourceBronze, ResourceIce}
}
