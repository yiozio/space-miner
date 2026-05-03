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
	registerPartDef(&PartDef{
		ID: PartIDCockpit, Kind: PartCockpit,
		Name: "Cockpit", Desc: "Pilot seat. Required.",
		Price: 0,
	})

	// --- Gun ---
	registerPartDef(&PartDef{
		ID: PartIDGunMkI, Kind: PartGun,
		Name: "Gun Mk-I", Desc: "Standard forward gun.",
		Price:          80,
		GunDamage:      1,
		GunCooldown:    12,
		GunBulletSpeed: 12.0,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunMkII, Kind: PartGun,
		Name: "Gun Mk-II", Desc: "Heavy gun. High damage, slow rate.",
		Price:          220,
		GunDamage:      3,
		GunCooldown:    24,
		GunBulletSpeed: 10.0,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunRapid, Kind: PartGun,
		Name: "Rapid Gun", Desc: "Light gun. Fast rate, low damage.",
		Price:          140,
		GunDamage:      1,
		GunCooldown:    6,
		GunBulletSpeed: 14.0,
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
	})

	// --- ユーティリティ系（現状は単一バリアント） ---
	registerPartDef(&PartDef{ID: PartIDFuelStd, Kind: PartFuel, Name: "Fuel Tank", Desc: "Auxiliary fuel tank.", Price: 70})
	registerPartDef(&PartDef{ID: PartIDCargoStd, Kind: PartCargo, Name: "Cargo", Desc: "Resource storage.", Price: 60})
	registerPartDef(&PartDef{ID: PartIDArmorStd, Kind: PartArmor, Name: "Armor", Desc: "Hardened plating.", Price: 100})
	registerPartDef(&PartDef{ID: PartIDShieldStd, Kind: PartShield, Name: "Shield", Desc: "Shield generator.", Price: 150})
	registerPartDef(&PartDef{ID: PartIDAutoAimStd, Kind: PartAutoAim, Name: "Auto-Aim", Desc: "Auto-targets nearby asteroids.", Price: 250})
	registerPartDef(&PartDef{ID: PartIDWarpStd, Kind: PartWarp, Name: "Warp", Desc: "Warp drive.", Price: 400})
}
