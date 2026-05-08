package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/save"
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
	menu    *ui.Menu
	saveCtx save.Context
}

// NewMenu はセーブ用コンテキスト（プレイヤー / プレイ時間 / 宙域名）を受け取り、
// 新しい Menu シーンを返す。
// Save / Load 項目は 3 スロットの選択画面 (SaveSlotPicker) を開く。
// Load はいずれかのスロットにセーブが存在するときだけ有効。
func NewMenu(ctx save.Context) *Menu {
	m := &Menu{
		saveCtx: ctx,
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Enabled: true},
				{Enabled: save.AnyExists()},
				{Enabled: true},
				{Enabled: true},
			},
			Cursor: menuItemSetting,
		},
	}
	m.applyLabels()
	return m
}

// applyLabels はメニュー項目のラベルを現在言語で再設定する。
// 言語切替直後にも反映できるよう Update 冒頭でも呼ぶ。
func (m *Menu) applyLabels() {
	s := i18n.S().Menu
	m.menu.Items[menuItemSave].Label = s.Save
	m.menu.Items[menuItemLoad].Label = s.Load
	m.menu.Items[menuItemSetting].Label = s.Setting
	m.menu.Items[menuItemQuitTitle].Label = s.QuitToTitle
}

func (m *Menu) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}
	// スロット選択画面から戻ってきた場合に Load の有効状態を毎フレーム再評価する。
	// 早期 return で見落とすことが無いよう、メニュー入力処理より前に行う。
	m.applyLabels()
	m.menu.Items[menuItemLoad].Enabled = save.AnyExists()
	r := m.menu.Update()
	if !r.Activated {
		return nil
	}
	switch m.menu.Cursor {
	case menuItemSave:
		d.Push(NewSaveSlotForSave(m.saveCtx))
	case menuItemLoad:
		if !save.AnyExists() {
			return nil
		}
		d.Push(NewSaveSlotForLoad())
	case menuItemSetting:
		d.Push(NewSettings(d.Theme()))
	case menuItemQuitTitle:
		d.Push(NewConfirm(i18n.S().Menu.QuitConfirm, func(d Director, yes bool) {
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
	s := i18n.S().Menu
	headerScale := 4.0
	hw, hh := ui.MeasureText(s.Header, headerScale)
	headerY := float64(sh) * 0.22
	ui.DrawText(dst, s.Header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// メニュー
	menuScale := 2.0
	maxW := m.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := headerY + hh + 80
	m.menu.Draw(dst, theme, mx, my, menuScale)

	// 操作ヒント
	ui.DrawText(dst, s.Hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
