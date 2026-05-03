package dialog

import "image/color"

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

var characters = map[string]*Character{
	"dock_master_aurora": {
		ID:    "dock_master_aurora",
		Name:  "Dockmaster Vex",
		Color: color.NRGBA{0xff, 0xb0, 0x60, 0xff},
		Style: AvatarStyleSmile,
	},
	"engineer_tinker": {
		ID:    "engineer_tinker",
		Name:  "Engineer Solis",
		Color: color.NRGBA{0x80, 0xe0, 0xff, 0xff},
		Style: AvatarStyleGoggles,
	},
	"foreman_helix": {
		ID:    "foreman_helix",
		Name:  "Foreman Halberg",
		Color: color.NRGBA{0xc0, 0x80, 0x40, 0xff},
		Style: AvatarStyleHelmet,
	},
	"comms": {
		ID:    "comms",
		Name:  "Comms",
		Color: color.NRGBA{0xa0, 0xa0, 0xa0, 0xff},
		Style: AvatarStyleStern,
	},
}

// CharacterByID はキャラクター定義を返す。未登録なら nil。
func CharacterByID(id string) *Character {
	return characters[id]
}
