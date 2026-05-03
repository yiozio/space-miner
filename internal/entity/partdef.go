package entity

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

// PartDef はパーツバリアントの定義。
// 同じ Kind を持つ複数の def を作り、性能（ステータス）を変えることでバリアントを表現する。
// 描画は Kind に依存し、バリアント間で共通の見た目を使う。
type PartDef struct {
	ID    PartID
	Kind  PartKind
	Name  string
	Desc  string
	Price int

	// Gun 用ステータス。Kind != PartGun の def では未使用。
	GunDamage      int
	GunCooldown    int     // フレーム単位（60fps）
	GunBulletSpeed float64 // px/frame

	// Thruster 用ステータス。Kind != PartThruster の def では未使用。
	// 通常時は Accel と MaxSpeed を全スラスタで合算。
	// ブースト時は Accel に BoostAccelMul（全スラスタ中の最大）を掛け、最大速度は BoostMaxSpeed を合算。
	// BoostFuelCost は全スラスタ分を合算して毎フレーム消費する。
	ThrustAccel         float64
	ThrustMaxSpeed      float64
	ThrustBoostAccelMul float64
	ThrustBoostMaxSpeed float64
	ThrustBoostFuelCost float64
}

var (
	partDefs      = map[PartID]*PartDef{}
	partDefOrder  []PartID // 安定した列挙順
	cockpitDefIDs []PartID
)

func registerPartDef(d *PartDef) {
	if _, exists := partDefs[d.ID]; exists {
		panic("duplicate PartID registration")
	}
	partDefs[d.ID] = d
	partDefOrder = append(partDefOrder, d.ID)
	if d.Kind == PartCockpit {
		cockpitDefIDs = append(cockpitDefIDs, d.ID)
	}
}

func init() {
	registerPartDef(&PartDef{
		ID: PartIDCockpit, Kind: PartCockpit,
		Name: "Cockpit", Desc: "Pilot seat. Required.",
		Price: 0,
	})

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

	registerPartDef(&PartDef{ID: PartIDFuelStd, Kind: PartFuel, Name: "Fuel Tank", Desc: "Auxiliary fuel tank.", Price: 70})
	registerPartDef(&PartDef{ID: PartIDCargoStd, Kind: PartCargo, Name: "Cargo", Desc: "Resource storage.", Price: 60})
	registerPartDef(&PartDef{ID: PartIDArmorStd, Kind: PartArmor, Name: "Armor", Desc: "Hardened plating.", Price: 100})
	registerPartDef(&PartDef{ID: PartIDShieldStd, Kind: PartShield, Name: "Shield", Desc: "Shield generator.", Price: 150})
	registerPartDef(&PartDef{ID: PartIDAutoAimStd, Kind: PartAutoAim, Name: "Auto-Aim", Desc: "Auto-targets nearby asteroids.", Price: 250})
	registerPartDef(&PartDef{ID: PartIDWarpStd, Kind: PartWarp, Name: "Warp", Desc: "Warp drive.", Price: 400})
}

// PartDefByID はレジストリから def を取得する。未登録 ID では nil を返す。
func PartDefByID(id PartID) *PartDef { return partDefs[id] }

// AllPartDefs は登録順に全ての def を返す。
func AllPartDefs() []*PartDef {
	out := make([]*PartDef, 0, len(partDefOrder))
	for _, id := range partDefOrder {
		out = append(out, partDefs[id])
	}
	return out
}

// AllPlaceablePartDefs は編集パレット・店在庫向けに、
// プレイヤーが配置・売買できる全 def を登録順に返す（Cockpit を除く）。
func AllPlaceablePartDefs() []*PartDef {
	out := make([]*PartDef, 0, len(partDefOrder))
	for _, id := range partDefOrder {
		d := partDefs[id]
		if d.Kind == PartCockpit {
			continue
		}
		out = append(out, d)
	}
	return out
}
