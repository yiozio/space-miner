package dialog

import (
	"image/color"

	"github.com/yiozio/space-miner/internal/i18n"
)

// data_characters.go は会話に登場するキャラクター定義。
// 表示時はアバター枠に Color の塗り + 簡易な顔シンボルを描画する。
// 新キャラを増やすときはここに登録する。

// AvatarStyle はキャラの顔シンボル種別。
// 同色でもスタイル違いで識別できるようにする。
type AvatarStyle int

const (
	AvatarStyleSmile AvatarStyle = iota
	AvatarStyleStern
	AvatarStyleGoggles
	AvatarStyleHelmet
)

// Character は話者として登場するキャラクター。
type Character struct {
	ID    string
	Name  string
	Color color.NRGBA
	Style AvatarStyle
}

// 表示属性 (色・スタイル) のみここで保持。表示名 (Name) は i18n から都度引く。
var characters = map[string]*Character{
	"dock_master_aurora": {
		ID:    "dock_master_aurora",
		Color: color.NRGBA{0xff, 0xb0, 0x60, 0xff},
		Style: AvatarStyleSmile,
	},
	"engineer_tinker": {
		ID:    "engineer_tinker",
		Color: color.NRGBA{0x80, 0xe0, 0xff, 0xff},
		Style: AvatarStyleGoggles,
	},
	"foreman_helix": {
		ID:    "foreman_helix",
		Color: color.NRGBA{0xc0, 0x80, 0x40, 0xff},
		Style: AvatarStyleHelmet,
	},
	"comms": {
		ID:    "comms",
		Color: color.NRGBA{0xa0, 0xa0, 0xa0, 0xff},
		Style: AvatarStyleStern,
	},
}

// CharacterByID はキャラクター定義を返す。未登録なら nil。
// 名前は i18n.S().Dialog.Characters から現在言語で取得する。
func CharacterByID(id string) *Character {
	c, ok := characters[id]
	if !ok {
		return nil
	}
	// Name フィールドを毎回更新して返す（呼び出し側はコピーを受け取り、
	// 後続の言語切替に追従する）
	out := *c
	if name, ok := i18n.S().Dialog.Characters[id]; ok {
		out.Name = name
	}
	return &out
}
