package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	menuItemSave = iota
	menuItemLoad
	menuItemSetting
	menuItemQuitTitle
)

// Menu はゲーム中のポーズメニューシーン。
// 直前のシーン（Exploration 等）の上に黒半透明オーバーレイを重ね、その上に表示する。
type Menu struct {
	menu *ui.Menu
}

// NewMenu は新しい Menu シーンを返す。
func NewMenu() *Menu {
	return &Menu{
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Label: "Save", Enabled: false}, // セーブ機能は後続フェーズで実装
				{Label: "Load", Enabled: false}, // セーブ機能は後続フェーズで実装
				{Label: "Setting", Enabled: true},
				{Label: "Quit To Title", Enabled: true},
			},
			Cursor: menuItemSetting,
		},
	}
}

func (m *Menu) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}
	r := m.menu.Update()
	if !r.Activated {
		return nil
	}
	switch m.menu.Cursor {
	case menuItemSetting:
		d.Push(NewSettings(d.Theme()))
	case menuItemQuitTitle:
		d.Push(NewConfirm("Quit to title?", func(d Director, yes bool) {
			if !yes {
				return
			}
			d.Pop()               // メニューを閉じる
			d.Replace(NewTitle()) // 直下のシーン（Exploration 等）をタイトルに置き換え
		}))
	}
	return nil
}

func (m *Menu) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 黒半透明オーバーレイ
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 180}, false)

	// ヘッダ
	headerScale := 4.0
	header := "MENU"
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := float64(sh) * 0.22
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// メニュー
	menuScale := 2.0
	maxW := m.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := headerY + hh + 80
	m.menu.Draw(dst, theme, mx, my, menuScale)

	// 操作ヒント
	ui.DrawText(dst, "[ Up/Down: Move    Enter: Select    Esc: Close ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}
