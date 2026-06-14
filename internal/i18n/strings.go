package i18n

import (
	"fmt"

	"github.com/yiozio/space-miner/internal/entity"
)

// PartName は PartID の表示名を返す。未登録なら空文字。
func PartName(id entity.PartID) string {
	if t, ok := S().Items.Parts[int(id)]; ok {
		return t.Name
	}
	return ""
}

// PartDesc は PartID の説明文を返す。未登録なら空文字。
func PartDesc(id entity.PartID) string {
	if t, ok := S().Items.Parts[int(id)]; ok {
		return t.Desc
	}
	return ""
}

// ResourceName は ResourceType の表示名を返す。
func ResourceName(t entity.ResourceType) string {
	if v, ok := S().Items.Resources[int(t)]; ok {
		return v.Name
	}
	return ""
}

// PiratePatternName は PiratePatternID の表示名を返す。
func PiratePatternName(id entity.PiratePatternID) string {
	if v, ok := S().Items.PiratePatterns[int(id)]; ok {
		return v.Name
	}
	return ""
}

// QuestDescription は Quest の本文を現在言語で返す。
// パラメータの数値・対象は Quest フィールドから取得し、表示書式は i18n.S().Tavern を使う。
func QuestDescription(q *entity.Quest) string {
	if q == nil {
		return ""
	}
	tv := S().Tavern
	switch q.Kind {
	case entity.QuestKindDelivery:
		return sprintf(tv.QuestDeliveryFmt, q.Amount, ResourceName(q.Resource))
	case entity.QuestKindBounty:
		return sprintf(tv.QuestBountyFmt, q.PirateTarget, q.MapName)
	}
	return "?"
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// Strings は表示テキストをカテゴリごとに分けて保持する。
// カテゴリは「画面・場面」と「UI 要素種別」「シナリオ要素 (アイテム/ダイアログ)」
// で大きく分け、各画面のスクリーン名・ボタン・ヒント・ヘッダ等をネストしたサブ struct に束ねる。
//
// パラメータを含む文字列は %d / %s 等のフォーマット指定子をそのまま埋めて、
// 呼び出し側で fmt.Sprintf(i18n.S().X.Y, args...) するスタイルにする。
type Strings struct {
	Common   CommonStrings
	Title    TitleStrings
	Menu     MenuStrings
	Save     SaveStrings
	Setting  SettingStrings
	GameOver GameOverStrings
	HUD      HUDStrings
	Worldmap WorldmapStrings
	StarMap  StarMapStrings
	Station  StationStrings
	Shop     ShopStrings
	Editor   EditorStrings
	Tavern   TavernStrings
	Items    ItemsStrings
	Dialog   DialogStrings
}

// CommonStrings は複数画面で共有する短いラベル。
type CommonStrings struct {
	Yes    string
	No     string
	OK     string
	Cancel string
	Back   string
	Empty  string // 空スロットの "(empty)"
}

// TitleStrings はタイトル画面のラベル。
type TitleStrings struct {
	Header   string // "SPACE MINER"
	Continue string
	NewGame  string
	Load     string
	Setting  string
	Quit     string
	Hint     string // フッタの操作ヒント
}

// MenuStrings はゲーム中のポーズメニュー。
type MenuStrings struct {
	Header      string
	Save        string
	Load        string
	Setting     string
	QuitToTitle string
	Hint        string
	QuitConfirm string // "Quit to title?"
}

// SaveStrings はセーブ/ロード画面。
type SaveStrings struct {
	HeaderSave       string
	HeaderLoad       string
	SlotLabel        string // "Slot %d"
	AutoSlotLabel    string // "Auto-save"
	ReadOnlySuffix   string // "(read-only)"
	OverwriteConfirm string // "Overwrite slot %d?"
	PlayPrefix       string // "PLAY %s"
	CreditsPrefix    string // "CR %d"
	DeepSpace        string // "(deep space)"
	Hint             string
}

// SettingStrings は設定画面。
type SettingStrings struct {
	Header   string
	Theme    string
	Language string
	Hint     string
}

// GameOverStrings はゲームオーバー画面。
type GameOverStrings struct {
	Header string
	Hint   string
}

// HUDStrings は探索画面の HUD。
type HUDStrings struct {
	CargoFmt    string // "CARGO %.0f/%.0f   CR %d"
	InvFmt      string // "%s %d   %s %d   %s %d" (素材ラベル + 数量を 3 種)
	SpeedPosFmt string // "SPEED %.2f   POS %.0f, %.0f"
	DockPrompt  string // "[ Space ] DOCK"
	// 操作ヒント要素
	HelpThrust   string // "Thrust"
	HelpRotate   string // "AD: Rotate"
	HelpBoost    string // "Shift: Boost"
	HelpFire     string // "Space: Fire"
	HelpDock     string // "Space: Dock"
	HelpFireDock string // "Space: Fire/Dock"
	HelpMap      string // "M: Map"
	HelpWarp     string // "N: Warp"
	HelpMenu     string // "Esc: Menu"
}

// WorldmapStrings はワールドマップ表示。
type WorldmapStrings struct {
	Header     string
	HeaderFmt  string // "WORLD MAP - %s" のように FullMap 名を併記する形
	Hint       string
	OutOfRange string // 区画外
	Hostile    string // 海賊エリアラベル
	Station    string // ステーションラベル
}

// StarMapStrings は恒星マップ。
type StarMapStrings struct {
	Header           string
	Hint             string
	WarpDriveMissing string // ワープドライブ未搭載時の注意
	WarpConfirmFmt   string // "Warp to %s?"
	NoWarpDrive      string
	NoTargets        string // 行先なし
	CurrentLocation  string // [ CURRENT LOCATION ]
	ZonesFmt         string // "ZONES %d"
	KindStar         string
	KindPlanet       string
	KindMoon         string
	SelectFmt        string // "> %s (%s)" 等の整形
}

// StationStrings はステーションメニュー全般で使う短いラベル。
type StationStrings struct {
	Header     string // "STATION"
	Repair     string
	Refuel     string
	Tavern     string
	Shop       string // "Parts Shop"
	Editor     string // "Ship Editor"
	Leave      string
	WelcomeFmt string // "WELCOME TO %s"
	Hint       string
	StatusFmt  string // "HP %d/%d   FUEL %d/%d   CREDITS %d"
}

// ShopStrings は装備ショップ。
type ShopStrings struct {
	Header        string // "SHOP"
	Buy           string
	Sell          string
	Hint          string
	NotEnoughCash string
	NoCargoSpace  string
	// 画面の各セクションラベル
	Inventory string
	Session   string
	Info      string
	// セッションサマリ
	BuyAmountFmt  string // "BUY %d cr"  購入額（売り戻した分は差し引く）
	SellAmountFmt string // "SELL %d cr" 購入していない所持品を売った額
	NetFmt        string // "NET %s%d"
	CreditsFmt    string // "CR %d"
	// アイテム情報
	BuyPriceFmt  string // "BUY %d cr"
	SellPriceFmt string // "SELL %d cr"
	WeightFmt    string // "WEIGHT %.1f"
	// 売却用の資源説明 (Resource Name 1 つを fmt 引数に取る)
	OreDescFmt string // "%s ore. Mining material."
	SpareFmt   string // "%s (spare)."
	// パーツ性能ステータス文字列群
	GunDmgCdFmt      string // "DMG %d   COOLDOWN %df"
	GunBulletSpdFmt  string // "BULLET SPD %.1f"
	GunStyleFmt      string // "STYLE %s%s" — 第二引数は impact suffix
	BulletStyleTrail string
	BulletStyleBall  string
	BulletStyleLaser string
	ImpactFXSuffix   string // " + IMPACT FX"
	ThrusterAccelFmt string // "ACCEL %.2f   MAX SPD %.1f"
	ThrusterBoostFmt string // "BOOST x%.1f   MAX %.1f"
	ThrusterFuelFmt  string // "FUEL/F %.2f"
	FuelCapFmt       string // "FUEL CAP %.0f"
	ArmorHPFmt       string // "HP +%d"
	ShieldHPFmt      string // "SHIELD HP +%d"
	ShieldRegenNote  string // "REGEN AFTER 2s NO DMG"
	CargoCapFmt      string // "CARGO CAP +%.0f"
	AutoAimRangeFmt  string // "RANGE %.0f   DPS %.1f"
	AutoAimNote      string // "BEAMS LAST-HIT ASTEROID"
	MineLayerNote    string // "BURSTS 6 WAYS AFTER ~1s"
	DroneNote        string // "ATTACKS NEAREST FOR ~10s"
}

// EditorStrings は機体エディタ。
type EditorStrings struct {
	Header       string
	Hint         string
	PartsHeader  string // パレット見出し "PARTS"
	CellLabel    string // "Cell: %s   %s"
	CellEmpty    string
	CursorPosFmt string // "Cursor (%d, %d)"
	// 機体性能パネル（グリッド左に表示）
	StatsHeader   string // 見出し "性能"
	StatFirepower string // "総火力"
	StatHull      string // "耐久値"
	StatShield    string // "シールド"
	StatSpeed     string // "速度"
	StatMax       string // "最高"
	StatBoost     string // "ブースト"
	DirForward    string // "前進"
	DirBackward   string // "後退"
	DirRight      string // "右方"
	DirLeft       string // "左方"
}

// TavernStrings は酒場 (クエスト掲示板)。
type TavernStrings struct {
	Header       string
	Subtitle     string // "Job Board"
	Accept       string
	Discard      string
	Refresh      string
	Hint         string
	NoQuest      string
	RewardFmt    string // "REWARD: %d cr"
	DiscardFmt   string // "DISCARD: %d cr"
	BonusPartFmt string // " + %s"
	ProgressFmt  string // "PROGRESS %d / %d"
	Ready        string // " + READY" 等
	KindDelivery string
	KindBounty   string
	// クエスト本文テンプレート
	QuestDeliveryFmt string // "Deliver %d %s"
	QuestBountyFmt   string // "Defeat %d pirates in %s"
	// 警告
	CargoFullWarn string
	// 完了通知
	CompletedFmt string
}

// ItemsStrings はゲーム内アイテム (パーツ・資源・海賊種別) の名称・説明。
type ItemsStrings struct {
	// Parts は PartID (int) → Name/Desc。整数キーで保持し、entity への循環依存を避ける。
	Parts map[int]ItemText
	// Resources は ResourceType (int) → Name のみ。
	Resources map[int]ItemText
	// PiratePatterns は PiratePatternID (int) → Name のみ。
	PiratePatterns map[int]ItemText
}

// ItemText は名前と説明を束ねる。Desc は不要なら空文字。
type ItemText struct {
	Name string
	Desc string
}

// DialogStrings は NPC ダイアログのスクリプト群と関連 UI 文字列。
//
// 個別ノードの Text を直接持たせる代わりに、シーン名 → 行の連番リストとして保持する。
// dialog パッケージは i18n.S().Dialog.X を参照して、ローカライズ済みの行を組み立てる。
type DialogStrings struct {
	// Opening はゲーム開始時のオープニング各行。
	Opening []string
	// StationFirstVisit はステーション初訪問時のスクリプト (ステーション名 → 行)。
	StationFirstVisit map[string][]string
	// Characters はキャラクター ID → 表示名。
	Characters map[string]string
	// シーン UI ラベル (会話シーン共通)
	OpeningHeader  string // "PROLOGUE"
	HintNext       string // "[ Enter / Space: Next   Esc: Skip ]"
	HintChoiceMove string // "[ Up/Down: Move   Enter: Select ]"
}
