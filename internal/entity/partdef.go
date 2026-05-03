package entity

// partdef.go はパーツバリアントの型・レジストリの実装のみを置く。
// 個別バリアントの定義データ（性能・名前・価格）は data_parts.go を参照。

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
	// 弾の見た目関連。
	// GunBulletStyle: BulletStyleTrail（既定）/ Ball / Laser
	// GunBulletWidth: ライン太さ or Ball 半径（0 ならスタイル既定値）
	// GunBulletImpact: 命中時の着弾エフェクト（小さな円が広がる）有無
	GunBulletStyle  BulletStyle
	GunBulletWidth  float64
	GunBulletImpact bool

	// Thruster 用ステータス。Kind != PartThruster の def では未使用。
	// 通常時は Accel と MaxSpeed を全スラスタで合算。
	// ブースト時は Accel に BoostAccelMul（全スラスタ中の最大）を掛け、最大速度は BoostMaxSpeed を合算。
	// BoostFuelCost は全スラスタ分を合算して毎フレーム消費する。
	ThrustAccel         float64
	ThrustMaxSpeed      float64
	ThrustBoostAccelMul float64
	ThrustBoostMaxSpeed float64
	ThrustBoostFuelCost float64

	// Fuel 用ステータス。Kind != PartFuel の def では未使用。
	// 全 Fuel パーツの FuelCapacity を合算して MaxFuel になる。
	// Fuel パーツが 0 個なら MaxFuel = 0（ブースト不可）。
	FuelCapacity float64

	// Armor 用ステータス。Kind != PartArmor の def では未使用。
	// 全 Armor パーツの ArmorHP を合算して MaxHP に加算される（基本 HP に上乗せ）。
	ArmorHP int

	// Shield 用ステータス。Kind != PartShield の def では未使用。
	// 全 Shield パーツの ShieldHP を合算して MaxShieldHP になる。
	// 被弾はシールドが先に吸収し、無ダメージが一定時間続くと自動回復する。
	ShieldHP int

	// 所持重量（カーゴ）系ステータス。
	// CargoCapacity は搭載時に MaxCargo に加算される積載上限。
	// 通常は Cockpit と Cargo パーツが提供する。
	// Weight はそのパーツをスペアパーツとして所持する際の単位重量。
	CargoCapacity float64
	Weight        float64

	// AutoAim 用ステータス。Kind != PartAutoAim の def では未使用。
	// AutoAimRange は対象グリッドまでの最大射程（パーツワールド位置からの距離、px）。
	// AutoAimDPS は射程内のときに毎秒与えるダメージ。
	// 複数の AutoAim パーツが同じターゲットを射程内に捉えていれば DPS が合算される。
	AutoAimRange float64
	AutoAimDPS   float64
}

var (
	partDefs     = map[PartID]*PartDef{}
	partDefOrder []PartID // 安定した列挙順
)

// registerPartDef は data_parts.go の init() から呼ばれる登録用 API。
func registerPartDef(d *PartDef) {
	if _, exists := partDefs[d.ID]; exists {
		panic("duplicate PartID registration")
	}
	partDefs[d.ID] = d
	partDefOrder = append(partDefOrder, d.ID)
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
