package entity

// data_pirates.go は海賊機のパターン定義集。
// 新しい海賊バリアントを増やすときはここに登録する。
// 行動 AI 自体は pirate.go の Update を参照。

// PiratePatternID は海賊バリアントの識別子。
type PiratePatternID int

const (
	PiratePatternScout PiratePatternID = iota
	PiratePatternBrawler
	PiratePatternCruiser
)

// PiratePattern は海賊機 1 種の定義。
//
// 機体構成 (Parts) と AI パラメータ、撃破時のドロップ条件を持つ。
// AI パラメータの意味:
//   - TurnSpeed:    1 フレームあたりの最大旋回（rad/frame）
//   - ThrustAccel:  追跡時の加速度（px/frame²）
//   - MaxSpeed:     速度上限（px/frame）
//   - PreferredDist: プレイヤーから保ちたい距離（これより遠ければ追跡）
//   - FireRange:    発射可能な距離（px）
//
// ドロップ:
//   - DropCreditsMin..DropCreditsMax の範囲で credits を必ず付与
//   - PartDropRate の確率で PartDrops からランダムに 1 つパーツ pickup を出す
type PiratePattern struct {
	ID    PiratePatternID
	Name  string
	Parts []Part
	MaxHP int

	TurnSpeed     float64
	ThrustAccel   float64
	MaxSpeed      float64
	PreferredDist float64
	FireRange     float64

	DropCreditsMin int
	DropCreditsMax int
	PartDropRate   float64
	PartDrops      []PartID
}

var piratePatterns = map[PiratePatternID]*PiratePattern{
	PiratePatternScout: {
		ID:   PiratePatternScout,
		Name: "Scout",
		Parts: []Part{
			{DefID: PartIDCockpit, GX: 0, GY: 0},
			{DefID: PartIDGunStarter, GX: 0, GY: -1},
			{DefID: PartIDThrusterLight, GX: 0, GY: 1},
		},
		MaxHP:          18,
		TurnSpeed:      0.05,
		ThrustAccel:    0.10,
		MaxSpeed:       4.5,
		PreferredDist:  340,
		FireRange:      520,
		DropCreditsMin: 25,
		DropCreditsMax: 55,
		PartDropRate:   0.02,
		PartDrops:      []PartID{PartIDGunStarter, PartIDThrusterLight},
	},
	PiratePatternBrawler: {
		ID:   PiratePatternBrawler,
		Name: "Brawler",
		Parts: []Part{
			{DefID: PartIDCockpit, GX: 0, GY: 0},
			{DefID: PartIDGunMkI, GX: -1, GY: 0},
			{DefID: PartIDGunMkI, GX: 1, GY: 0},
			{DefID: PartIDArmorStd, GX: 0, GY: -1},
			{DefID: PartIDThrusterStd, GX: 0, GY: 1},
		},
		MaxHP:          45,
		TurnSpeed:      0.04,
		ThrustAccel:    0.12,
		MaxSpeed:       5.0,
		PreferredDist:  300,
		FireRange:      560,
		DropCreditsMin: 70,
		DropCreditsMax: 130,
		PartDropRate:   0.05,
		PartDrops:      []PartID{PartIDGunMkI, PartIDArmorStd, PartIDThrusterStd},
	},
	PiratePatternCruiser: {
		ID:   PiratePatternCruiser,
		Name: "Cruiser",
		Parts: []Part{
			{DefID: PartIDCockpit, GX: 0, GY: 0},
			{DefID: PartIDGunMkII, GX: -1, GY: -1},
			{DefID: PartIDGunMkII, GX: 1, GY: -1},
			{DefID: PartIDArmorStd, GX: -1, GY: 0},
			{DefID: PartIDArmorStd, GX: 1, GY: 0},
			{DefID: PartIDThrusterHeavy, GX: 0, GY: 1},
		},
		MaxHP:          95,
		TurnSpeed:      0.03,
		ThrustAccel:    0.10,
		MaxSpeed:       4.2,
		PreferredDist:  360,
		FireRange:      620,
		DropCreditsMin: 150,
		DropCreditsMax: 280,
		PartDropRate:   0.10,
		PartDrops:      []PartID{PartIDGunMkII, PartIDThrusterHeavy, PartIDArmorStd, PartIDShieldStd},
	},
}

// PiratePatternByID はパターン定義を返す。未登録なら nil。
func PiratePatternByID(id PiratePatternID) *PiratePattern {
	return piratePatterns[id]
}
