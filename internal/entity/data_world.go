package entity

// data_world.go は初期ステージの World 定義データ。
// 新しい全体マップ（FullMap）や素材ゾーンを追加するときはここを編集する。
// 型・API（World / FullMap / ResourceZone / SpawnCap 等）は world.go を参照。

// 起点ステーションの座標。起点 FullMap の中心としても用いる。
const (
	DefaultStationX = 300.0
	DefaultStationY = -250.0
)

// 全体マップ（FullMap）の標準サイズ。中心 ± 半幅・半高。
// 通常巡航で端から端まで数分の広さ（半幅 30000px）。
const (
	DefaultFullMapHalfW = 30000.0
	DefaultFullMapHalfH = 30000.0
)

// DefaultWorld は初期ステージのワールドを返す。
// 起点ステーションを中心とする FullMap を 1 つ持ち、内部に 4 つの素材ゾーンを配置する。
// 区画外には小惑星は生成されない（将来別の FullMap をこの World に追加する）。
func DefaultWorld() *World {
	return &World{
		Maps: []FullMap{
			{
				Name: "Starter Sector",
				CX:   DefaultStationX, CY: DefaultStationY,
				HalfW: DefaultFullMapHalfW, HalfH: DefaultFullMapHalfH,
				Zones: []ResourceZone{
					// 起点近くの鉄ゾーン x2
					{
						CX: 5000, CY: -800, Radius: 4500, MaxAsteroids: 12,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 1},
						},
					},
					{
						CX: -5000, CY: 1800, Radius: 4500, MaxAsteroids: 10,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 1},
						},
					},
					// 中盤: 鉄+氷の混合ゾーン
					{
						CX: 9000, CY: 8000, Radius: 7000, MaxAsteroids: 14,
						Mix: []ResourceWeight{
							{Resource: ResourceIron, Weight: 2},
							{Resource: ResourceIce, Weight: 1},
						},
					},
					// 区画の遠い角: 青銅+氷の混合ゾーン
					{
						CX: -25000, CY: 25000, Radius: 9000, MaxAsteroids: 16,
						Mix: []ResourceWeight{
							{Resource: ResourceBronze, Weight: 2},
							{Resource: ResourceIce, Weight: 1},
						},
					},
				},
			},
		},
	}
}
