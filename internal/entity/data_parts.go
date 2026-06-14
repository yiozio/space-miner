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
	PartIDGunPlasma
	PartIDGunLaser

	PartIDThrusterStd
	PartIDThrusterLight
	PartIDThrusterHeavy

	PartIDFuelStd
	PartIDCargoStd
	PartIDArmorStd
	PartIDShieldStd
	PartIDAutoAimStd
	PartIDWarpStd
	PartIDMineLayer
	PartIDDroneStd
	PartIDDroneGun
)

func init() {
	// Cockpit はパイロット座席で配置必須。
	// Thruster が 1 つも無いときの非常用推進として、最低限のスラスタ性能も持つ。
	// 通常の Thruster が搭載されている場合、この値は使われない（player.go thrusterStats 参照）。
	registerPartDef(&PartDef{
		ID: PartIDCockpit, Kind: PartCockpit,
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
		Price:           40,
		GunDamage:       1,
		GunCooldown:     20,
		GunBulletSpeed:  9.0,
		Weight:          2,
		GunBulletStyle:  BulletStyleTrail,
		GunBulletWidth:  1.5,
		GunBulletImpact: false,
		GunFireSound:    FireSoundBurst,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunMkI, Kind: PartGun,
		Price:           80,
		GunDamage:       1,
		GunCooldown:     12,
		GunBulletSpeed:  12.0,
		Weight:          3,
		GunBulletStyle:  BulletStyleTrail,
		GunBulletWidth:  2,
		GunBulletImpact: false,
		GunFireSound:    FireSoundBurst,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunMkII, Kind: PartGun,
		Price:           220,
		GunDamage:       3,
		GunCooldown:     24,
		GunBulletSpeed:  10.0,
		Weight:          5,
		GunBulletStyle:  BulletStyleTrail,
		GunBulletWidth:  3,
		GunBulletImpact: true,
		GunFireSound:    FireSoundBurst,
	})
	registerPartDef(&PartDef{
		ID: PartIDGunRapid, Kind: PartGun,
		Price:           140,
		GunDamage:       1,
		GunCooldown:     6,
		GunBulletSpeed:  14.0,
		Weight:          2,
		GunBulletStyle:  BulletStyleTrail,
		GunBulletWidth:  1.2,
		GunBulletImpact: false,
		GunFireSound:    FireSoundBurst,
	})
	// Plasma Cannon: 大きなボール弾、低速・高威力、着弾エフェクトあり
	registerPartDef(&PartDef{
		ID: PartIDGunPlasma, Kind: PartGun,
		Price:           320,
		GunDamage:       4,
		GunCooldown:     30,
		GunBulletSpeed:  8.0,
		Weight:          6,
		GunBulletStyle:  BulletStyleBall,
		GunBulletWidth:  6,
		GunBulletImpact: true,
		GunFireSound:    FireSoundZap,
	})
	// Laser Pulse: 細く長いビーム、高速、着弾エフェクトあり
	registerPartDef(&PartDef{
		ID: PartIDGunLaser, Kind: PartGun,
		Price:           260,
		GunDamage:       2,
		GunCooldown:     14,
		GunBulletSpeed:  22.0,
		Weight:          3,
		GunBulletStyle:  BulletStyleLaser,
		GunBulletWidth:  1.5,
		GunBulletImpact: true,
		GunFireSound:    FireSoundLaser,
	})

	// --- Thruster ---
	registerPartDef(&PartDef{
		ID: PartIDThrusterStd, Kind: PartThruster,
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
		Price:        70,
		FuelCapacity: 100,
		Weight:       3,
	})
	registerPartDef(&PartDef{
		ID: PartIDCargoStd, Kind: PartCargo,
		Price:         60,
		CargoCapacity: 50,
		Weight:        4,
	})
	registerPartDef(&PartDef{
		ID: PartIDArmorStd, Kind: PartArmor,
		Price:   100,
		ArmorHP: 25,
		Weight:  7,
	})
	registerPartDef(&PartDef{
		ID: PartIDShieldStd, Kind: PartShield,
		Price:    150,
		ShieldHP: 50,
		Weight:   5,
	})
	registerPartDef(&PartDef{
		ID: PartIDAutoAimStd, Kind: PartAutoAim,
		Price:        250,
		Weight:       3,
		AutoAimRange: 500,
		AutoAimDPS:   4,
	})
	registerPartDef(&PartDef{
		ID: PartIDWarpStd, Kind: PartWarp,
		Price:  400,
		Weight: 8,
	})

	// 機雷敷設パーツ: 発射時に機体位置へ機雷を設置する。
	// 機雷は約1秒後に 6 方向へ弾をばらまく（弾の威力・速度・見た目は Gun 系ステータスを流用）。
	// 設置間隔（GunCooldown）は長めにして連続設置を抑える。
	registerPartDef(&PartDef{
		ID: PartIDMineLayer, Kind: PartMineLayer,
		Price:           300,
		GunDamage:       2,
		GunCooldown:     90,
		GunBulletSpeed:  7.0,
		Weight:          5,
		GunBulletStyle:  BulletStyleBall,
		GunBulletWidth:  3,
		GunBulletImpact: true,
		GunFireSound:    FireSoundBurst,
	})

	// ドローンランチャー（ビーム型）: 発射時に自律攻撃ドローンを設置する。
	// ドローンは約10秒間、射程内で最も近い小惑星 or 海賊にビームで継続ダメージを与える。
	// 射程・DPS は AutoAim 系ステータスを流用し、設置間隔は GunCooldown を使う。
	registerPartDef(&PartDef{
		ID: PartIDDroneStd, Kind: PartDroneLauncher,
		Price:        350,
		Weight:       4,
		GunCooldown:  180, // 設置間隔 3 秒（寿命 10 秒なので複数同時稼働しうる）
		AutoAimRange: 400,
		AutoAimDPS:   6,
		GunFireSound: FireSoundZap,
	})

	// ドローンランチャー（弾型）: 設置したドローンが射程内の対象へ一定間隔で弾を撃つ。
	// 必中のビーム型と違い命中判定は通常弾と同じ（外しうる）が、着弾までの飛翔があり貫通しない。
	// 弾の威力・速度・見た目は Gun 系、発射間隔は DroneFireInterval を使う。
	registerPartDef(&PartDef{
		ID: PartIDDroneGun, Kind: PartDroneLauncher,
		Price:             320,
		Weight:            4,
		GunCooldown:       150, // 設置間隔 2.5 秒
		AutoAimRange:      380,
		GunDamage:         2,
		GunBulletSpeed:    8.0,
		GunBulletStyle:    BulletStyleTrail,
		GunBulletWidth:    2,
		GunBulletImpact:   false,
		DroneFireInterval: 30, // 0.5 秒ごとに 1 発
		GunFireSound:      FireSoundBurst,
	})
}
