package entity

import "image/color"

// data_world.go は初期ステージの World 定義データ。
// 新しい全体マップ（FullMap）や素材ゾーン、天体を追加するときはここを編集する。
// 型・API（World / FullMap / ResourceZone / Celestial / SpawnCap 等）は world.go と celestial.go を参照。
//
// 設計上の約束:
// - 恒星 (Star) は常に世界座標 (0, 0) を中心にあると扱う。
// - 各 FullMap の CX/CY は世界座標であり、同時に恒星マップ上での表示位置でもある。
//   これにより「恒星マップで見たワープ方向」と「世界座標差で計算するワープ角度」が一致する。
// - FullMap 同士は重ならない（HalfW/H = 30000、間隔は最低でも 60000 以上）。

// 起点ステーションの座標。起点 FullMap (Aurora) の中心としても用いる。
const (
	DefaultStationX = 50000.0
	DefaultStationY = -10000.0
)

// 全体マップ（FullMap）の標準サイズ。中心 ± 半幅・半高。
// 通常巡航で端から端まで数分の広さ（半幅 30000px）。
const (
	DefaultFullMapHalfW = 30000.0
	DefaultFullMapHalfH = 30000.0
)

// DefaultWorld は初期ステージのワールドを返す。
// 単一恒星系「Sol-α」に、惑星 Aurora（起点）、その衛星 Tinker、別惑星 Helix の
// 3 つの FullMap を配置する。各区画は宇宙ステーションを中心に取り、
// 恒星と FullMap の世界座標位置がそのまま恒星マップの配置になる。
func DefaultWorld() *World {
	const (
		auroraX, auroraY = DefaultStationX, DefaultStationY // (50000, -10000)
		tinkerX, tinkerY = 130000.0, -10000.0
		helixX, helixY   = -30000.0, 80000.0
	)
	return &World{
		Star: Celestial{
			Name:   "Sol-α",
			Kind:   CelestialStar,
			Color:  color.NRGBA{0xff, 0xd0, 0x60, 0xff},
			Radius: 46,
		},
		Maps: []FullMap{
			{
				Name: "Aurora",
				CX:   auroraX, CY: auroraY,
				HalfW: DefaultFullMapHalfW, HalfH: DefaultFullMapHalfH,
				Body: Celestial{
					Name:   "Aurora",
					Kind:   CelestialPlanet,
					Color:  color.NRGBA{0x60, 0xa0, 0xff, 0xff},
					Radius: 28,
					// 起点ステーションの上に大きく青い惑星を背景表示
					BackdropRadius:  320,
					BackdropOffsetX: 0,
					BackdropOffsetY: -450,
				},
				Zones: []ResourceZone{
					// 起点近くの鉄ゾーン x2
					{
						CX: auroraX + 4700, CY: auroraY - 550, Radius: 4500, MaxAsteroids: 12,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 1},
						},
					},
					{
						CX: auroraX - 5300, CY: auroraY + 2050, Radius: 4500, MaxAsteroids: 10,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 1},
						},
					},
					// 中盤: 鉄+氷の混合ゾーン
					{
						CX: auroraX + 8700, CY: auroraY + 8250, Radius: 7000, MaxAsteroids: 14,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 2},
							{Resource: ResourceIce, Weight: 1},
						},
					},
					// 区画の遠い角: 青銅+氷の混合ゾーン
					{
						CX: auroraX - 25300, CY: auroraY + 25250, Radius: 9000, MaxAsteroids: 16,
						Mix: []ResourceWeight{
							{Resource: ResourceBronze, Weight: 2},
							{Resource: ResourceIce, Weight: 1},
						},
					},
				},
				PirateZones: []PirateZone{
					// 起点 FullMap には弱めの Scout 出没エリアを 1 つだけ
					{
						CX: auroraX - 18000, CY: auroraY - 14000, Radius: 6000, MaxPirates: 2,
						Patterns: []PiratePatternID{PiratePatternScout},
					},
				},
			},
			{
				Name: "Tinker",
				CX:   tinkerX, CY: tinkerY,
				HalfW: DefaultFullMapHalfW, HalfH: DefaultFullMapHalfH,
				Body: Celestial{
					Name:       "Tinker",
					Kind:       CelestialMoon,
					Color:      color.NRGBA{0xb0, 0xc8, 0xd8, 0xff},
					Radius:     14,
					ParentName: "Aurora",
					// 衛星らしくやや小さめ
					BackdropRadius:  180,
					BackdropOffsetX: 200,
					BackdropOffsetY: -350,
				},
				Zones: []ResourceZone{
					// 氷リッチな衛星
					{
						CX: tinkerX + 2000, CY: tinkerY - 1500, Radius: 5500, MaxAsteroids: 14,
						Mix: []ResourceWeight{
							{Resource: ResourceIce, Weight: 1},
						},
					},
					{
						CX: tinkerX - 4000, CY: tinkerY + 4000, Radius: 7000, MaxAsteroids: 12,
						Mix: []ResourceWeight{
							{Resource: ResourceIce, Weight: 3},
							{Resource: ResourceIron, Weight: 1},
						},
					},
				},
				PirateZones: []PirateZone{
					// 衛星周辺は Scout + Brawler 混在
					{
						CX: tinkerX + 11000, CY: tinkerY + 9000, Radius: 7000, MaxPirates: 3,
						Patterns: []PiratePatternID{PiratePatternScout, PiratePatternBrawler},
					},
				},
			},
			{
				Name: "Helix",
				CX:   helixX, CY: helixY,
				HalfW: DefaultFullMapHalfW, HalfH: DefaultFullMapHalfH,
				Body: Celestial{
					Name:   "Helix",
					Kind:   CelestialPlanet,
					Color:  color.NRGBA{0xe0, 0x90, 0x50, 0xff},
					Radius: 34,
					// 大型ガス惑星風の大きな背景
					BackdropRadius:  420,
					BackdropOffsetX: -300,
					BackdropOffsetY: -650,
				},
				Zones: []ResourceZone{
					// 青銅リッチな惑星
					{
						CX: helixX + 4000, CY: helixY + 3000, Radius: 8000, MaxAsteroids: 18,
						Mix: []ResourceWeight{
							{Resource: ResourceBronze, Weight: 1},
						},
					},
					{
						CX: helixX - 6000, CY: helixY - 5000, Radius: 7000, MaxAsteroids: 14,
						Mix: []ResourceWeight{
							{Resource: ResourceBronze, Weight: 2},
							{Resource: ResourceIron, Weight: 1},
						},
					},
				},
				PirateZones: []PirateZone{
					// 終盤宙域: Brawler + Cruiser
					{
						CX: helixX, CY: helixY + 14000, Radius: 8000, MaxPirates: 4,
						Patterns: []PiratePatternID{PiratePatternBrawler, PiratePatternCruiser},
					},
				},
			},
		},
	}
}
