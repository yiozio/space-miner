package ui

import "image/color"

// Theme は配色プリセット。背景・ライン・補助色の3色で構成する。
// 描画コードは色を直書きせず、必ず Theme 経由で取得する。
type Theme struct {
	Name       string
	Background color.NRGBA
	Line       color.NRGBA
	LineDim    color.NRGBA
}

var (
	ThemeBlack = &Theme{
		Name:       "Black",
		Background: color.NRGBA{0x00, 0x00, 0x00, 0xFF},
		Line:       color.NRGBA{0x40, 0xF0, 0x40, 0xFF},
		LineDim:    color.NRGBA{0x18, 0x60, 0x18, 0xFF},
	}
	ThemeNavy = &Theme{
		Name:       "Navy",
		Background: color.NRGBA{0x06, 0x0A, 0x28, 0xFF},
		Line:       color.NRGBA{0x40, 0xE0, 0xFF, 0xFF},
		LineDim:    color.NRGBA{0x20, 0x60, 0x90, 0xFF},
	}
	ThemeDarkGreen = &Theme{
		Name:       "DarkGreen",
		Background: color.NRGBA{0x04, 0x1A, 0x0C, 0xFF},
		Line:       color.NRGBA{0xFF, 0xC0, 0x40, 0xFF},
		LineDim:    color.NRGBA{0x80, 0x60, 0x20, 0xFF},
	}
)

// Themes は設定画面で切り替え可能なプリセット一覧。
var Themes = []*Theme{ThemeBlack, ThemeNavy, ThemeDarkGreen}

// ThemeByName は名前からテーマを取得する。未知の名前は ThemeBlack を返す。
func ThemeByName(name string) *Theme {
	for _, t := range Themes {
		if t.Name == name {
			return t
		}
	}
	return ThemeBlack
}

// ThemeIndex は与えられたテーマの Themes 内インデックスを返す。
// 見つからなければ 0 を返す。
func ThemeIndex(t *Theme) int {
	for i, x := range Themes {
		if x == t {
			return i
		}
	}
	return 0
}
