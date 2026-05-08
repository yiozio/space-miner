package i18n

import "github.com/yiozio/space-miner/internal/entity"

// newJAItems は日本語のアイテム文字列セットを返す。
// 数値ステータスは entity/data_*.go 側に残し、表示文字列のみここで定義する。
func newJAItems() ItemsStrings {
	return ItemsStrings{
		Parts: map[int]ItemText{
			int(entity.PartIDCockpit):       {Name: "コックピット", Desc: "操縦席。必須。スラスタ未搭載時は最低限の推進を提供する。"},
			int(entity.PartIDGunStarter):    {Name: "スターターガン", Desc: "支給品の豆鉄砲。低威力・低連射。"},
			int(entity.PartIDGunMkI):        {Name: "ガン Mk-I", Desc: "標準的な前方ガン。"},
			int(entity.PartIDGunMkII):       {Name: "ガン Mk-II", Desc: "重砲。高威力だが発射間隔は長い。"},
			int(entity.PartIDGunRapid):      {Name: "ラピッドガン", Desc: "軽量ガン。連射速度が速いが威力は低い。"},
			int(entity.PartIDGunPlasma):     {Name: "プラズマキャノン", Desc: "低速プラズマ弾。高威力で着弾時に爆発する。"},
			int(entity.PartIDGunLaser):      {Name: "レーザーパルス", Desc: "瞬間命中レーザー。極めて高速だが威力は控えめ。"},
			int(entity.PartIDThrusterStd):   {Name: "スラスター", Desc: "標準推進機関。"},
			int(entity.PartIDThrusterLight): {Name: "ライトスラスター", Desc: "小型推進機。推力は控えめだが燃費が良い。"},
			int(entity.PartIDThrusterHeavy): {Name: "ヘビースラスター", Desc: "大出力推進機。推力大、燃料消費激しめ。"},
			int(entity.PartIDFuelStd):       {Name: "燃料タンク", Desc: "標準的な燃料タンク。"},
			int(entity.PartIDCargoStd):      {Name: "カーゴ", Desc: "資源用ストレージ。積載量を増やす。"},
			int(entity.PartIDArmorStd):      {Name: "アーマー", Desc: "強化装甲板。最大 HP を増やす。"},
			int(entity.PartIDShieldStd):     {Name: "シールド", Desc: "シールド発生器。被弾を吸収し、無被弾 2 秒経過後に再生する。"},
			int(entity.PartIDAutoAimStd):    {Name: "オートエイム", Desc: "最後に弾の当たった小惑星をビームで自動攻撃する (DOT)。"},
			int(entity.PartIDWarpStd):       {Name: "ワープドライブ", Desc: "ワープ機関。"},
		},
		Resources: map[int]ItemText{
			int(entity.ResourceIron):   {Name: "鉄"},
			int(entity.ResourceBronze): {Name: "青銅"},
			int(entity.ResourceIce):    {Name: "氷"},
		},
		PiratePatterns: map[int]ItemText{
			int(entity.PiratePatternScout):   {Name: "スカウト"},
			int(entity.PiratePatternBrawler): {Name: "ブロウラー"},
			int(entity.PiratePatternCruiser): {Name: "クルーザー"},
		},
	}
}
