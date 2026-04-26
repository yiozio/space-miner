package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yiozio/space-miner/internal/ui"
)

// Settings は設定画面シーン。スタート画面とメニュー画面の双方から開く。
type Settings struct {
	menu *ui.Menu
}

const (
	settingsItemTheme = iota
	settingsItemBack
)

// NewSettings は現在のテーマを反映した状態で Settings シーンを返す。
func NewSettings(current *ui.Theme) *Settings {
	names := make([]string, len(ui.Themes))
	for i, t := range ui.Themes {
		names[i] = t.Name
	}
	return &Settings{
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Label: "Theme", Enabled: true, Values: names, ValueIndex: ui.ThemeIndex(current)},
				{Label: "Back", Enabled: true},
			},
		},
	}
}

func (s *Settings) Update(d Director) error {
	// Esc で即時に戻れるようにする（Back 項目と同等）
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}
	r := s.menu.Update()
	if r.ValueChanged && s.menu.Cursor == settingsItemTheme {
		idx := s.menu.Items[settingsItemTheme].ValueIndex
		d.SetTheme(ui.Themes[idx])
	}
	if r.Activated && s.menu.Cursor == settingsItemBack {
		d.Pop()
	}
	return nil
}

func (s *Settings) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	headerScale := 4.0
	header := "SETTINGS"
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := float64(sh) * 0.18
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	menuScale := 2.0
	maxW := s.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := headerY + hh + 80
	s.menu.Draw(dst, theme, mx, my, menuScale)

	hint := "[ Up/Down: Move    Left/Right: Change    Enter/Esc: Back ]"
	ui.DrawText(dst, hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
