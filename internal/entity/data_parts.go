package entity

// data_parts.go は全パーツバリアントの定義データ。
// 新しいバリアント（高威力ガン、軽量スラスタ等）を追加するときは、
// 1) PartID 定数を追加し
// 2) init() の registerPartDef でステータスを記述する。
// 型・レジストリ実装は partdef.go を参照。

// PartID は具体的なパーツバリアントの識別子。
// PartKind が振る舞いカテゴリ（Gun / Thruster ...）を表すのに対し、
// PartID は同カテゴリ内の性能違い（Mk-I / Mk-II ...）まで含めて識別する。
type PartID int

const (
	PartIDCockpit PartID = iota

	PartIDGunStarter
	PartIDGunMkI
	PartIDGunMkII
	PartIDGunRapid

	PartIDThrusterStd
	PartIDThrusterLight
	PartIDThrusterHeavy

	PartIDFuelStd
	PartIDCargoStd
	PartIDArmorStd
	PartIDShieldStd
	PartIDAutoAimStd
	PartIDWarpStd
)

func init() {
	// Cockpit はパイロット座席で配置必須。
	// Thruster が 1 つも無いときの非常用推進として、最低限のスラスタ性能も持つ。
	// 通常の Thruster が搭載されている場合、この値は使われない（player.go thrusterStats 参照）。
	registerPartDef(&PartDef{
		ID: PartIDCockpit, Kind: PartCockpit,
		Name: "Cockpit", Desc: "Pilot seat. Required. Provides minimal thrust if no thrusters are installed.",
		Price:               0,
		ThrustAccel:         0.05,
		ThrustMaxSpeed:      2,
		ThrustBoostAccelMul: 1.4,
		ThrustBoostMaxSpeed: 5.0,
		ThrustBoostFuelCost: 0.10,
		// Cockpit は最低限の積載スペースを持つ（Cargo 搭載前の入門容量）。
		CargoCapacity: 15,
	})

	// --- Gun ---
	// Starter は最初期支給品。性能は最弱で、買い替え前提の入門用。
	registerPartDef(&PartDef{
		ID: PartIDGunStarter, Kind: PartGun,
		Name: "Starter Gun", Desc: "Factory-issue popgun. Low damage, slow rate.",
		Price:          40,
		GunDamage:      1,
		GunCooldown:    20,
		GunBulletSpeed: 9.0,
		Weight:         2,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunMkI, Kind: PartGun,
		Name: "Gun Mk-I", Desc: "Standard forward gun.",
		Price:          80,
		GunDamage:      1,
		GunCooldown:    12,
		GunBulletSpeed: 12.0,
		Weight:         3,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunMkII, Kind: PartGun,
		Name: "Gun Mk-II", Desc: "Heavy gun. High damage, slow rate.",
		Price:          220,
		GunDamage:      3,
		GunCooldown:    24,
		GunBulletSpeed: 10.0,
		Weight:         5,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunRapid, Kind: PartGun,
		Name: "Rapid Gun", Desc: "Light gun. Fast rate, low damage.",
		Price:          140,
		GunDamage:      1,
		GunCooldown:    6,
		GunBulletSpeed: 14.0,
		Weight:         2,
	})

	// --- Thruster ---
	registerPartDef(&PartDef{
		ID: PartIDThrusterStd, Kind: PartThruster,
		Name: "Thruster", Desc: "Standard engine.",
		Price:               120,
		ThrustAccel:         0.15,
		ThrustMaxSpeed:      8.0,
		ThrustBoostAccelMul: 2.5,
		ThrustBoostMaxSpeed: 14.0,
		ThrustBoostFuelCost: 0.30,
		Weight:              6,
	})
	registerPartDef(&PartDef{
		ID: PartIDThrusterLight, Kind: PartThruster,
		Name: "Light Thruster", Desc: "Compact engine. Lower thrust, fuel-efficient.",
		Price:               90,
		ThrustAccel:         0.10,
		ThrustMaxSpeed:      6.5,
		ThrustBoostAccelMul: 2.2,
		ThrustBoostMaxSpeed: 10.0,
		ThrustBoostFuelCost: 0.20,
		Weight:              4,
	})
	registerPartDef(&PartDef{
		ID: PartIDThrusterHeavy, Kind: PartThruster,
		Name: "Heavy Thruster", Desc: "High-output engine. Strong thrust, hungry.",
		Price:               220,
		ThrustAccel:         0.22,
		ThrustMaxSpeed:      10.0,
		ThrustBoostAccelMul: 2.8,
		ThrustBoostMaxSpeed: 18.0,
		ThrustBoostFuelCost: 0.45,
		Weight:              9,
	})

	// --- ユーティリティ系（現状は単一バリアント） ---
	registerPartDef(&PartDef{
		ID: PartIDFuelStd, Kind: PartFuel,
		Name: "Fuel Tank", Desc: "Standard fuel tank.",
		Price:        70,
		FuelCapacity: 100,
		Weight:       3,
	})
	registerPartDef(&PartDef{
		ID: PartIDCargoStd, Kind: PartCargo,
		Name: "Cargo", Desc: "Resource storage. Increases cargo capacity.",
		Price:         60,
		CargoCapacity: 50,
		Weight:        4,
	})
	registerPartDef(&PartDef{
		ID: PartIDArmorStd, Kind: PartArmor,
		Name: "Armor", Desc: "Hardened plating. Increases max HP.",
		Price:   100,
		ArmorHP: 25,
		Weight:  7,
	})
	registerPartDef(&PartDef{
		ID: PartIDShieldStd, Kind: PartShield,
		Name: "Shield", Desc: "Shield generator. Absorbs damage; regenerates after 2s without damage.",
		Price:    150,
		ShieldHP: 50,
		Weight:   5,
	})
	registerPartDef(&PartDef{
		ID: PartIDAutoAimStd, Kind: PartAutoAim,
		Name: "Auto-Aim", Desc: "Beams the last-hit asteroid grid by grid. Damage over time.",
		Price:        250,
		Weight:       3,
		AutoAimRange: 500,
		AutoAimDPS:   4,
	})
	registerPartDef(&PartDef{
		ID: PartIDWarpStd, Kind: PartWarp,
		Name: "Warp", Desc: "Warp drive.",
		Price:  400,
		Weight: 8,
	})
}
