// Package i18n は UI 表示に使う多言語化文字列を集約する。
//
// 設計方針:
//   - すべての可視テキストは Strings 型のフィールドとして定義し、
//     言語ごとに値だけを差し替える（タイプセーフでキータイポしない）。
//   - データロジック (PartDef の数値・Pirate の挙動など) は i18n に持ち込まず、
//     表示用の Name/Desc/script 等のみを集約する。
//   - 言語切替は SetLang で動的に行い、現在値は S() で取得する。
package i18n

// Lang は対応言語の ID。
type Lang int

const (
	LangJA Lang = iota
	LangEN
)

// String は Lang のラベル文字列を返す（設定画面表示用）。
func (l Lang) String() string {
	switch l {
	case LangJA:
		return "日本語"
	case LangEN:
		return "English"
	}
	return "?"
}

// AllLangs は対応言語を順序付きで返す（設定画面の循環選択用）。
func AllLangs() []Lang { return []Lang{LangJA, LangEN} }

var (
	current   Lang = LangJA
	stringsJA      = newJA()
	stringsEN      = newEN()
)

// CurrentLang は現在選択されている言語を返す。
func CurrentLang() Lang { return current }

// SetLang は表示言語を切り替える。
func SetLang(l Lang) {
	switch l {
	case LangJA, LangEN:
		current = l
	}
}

// S は現在言語の Strings を返す。
// 呼び出し側は i18n.S().Menu.Continue のようにフィールドアクセスする。
func S() *Strings {
	switch current {
	case LangEN:
		return stringsEN
	default:
		return stringsJA
	}
}
