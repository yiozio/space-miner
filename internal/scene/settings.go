package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

// Settings は設定画面シーン。スタート画面とメニュー画面の双方から開く。
// テーマ切替と言語切替を提供する。言語変更時はラベル文字列を全て再構築する。
type Settings struct {
	menu *ui.Menu
}

const (
	settingsItemTheme = iota
	settingsItemLanguage
	settingsItemBack
)

// NewSettings は現在のテーマを反映した状態で Settings シーンを返す。
func NewSettings(current *ui.Theme) *Settings {
	s := &Settings{}
	s.rebuildMenu(current)
	return s
}

// rebuildMenu は i18n の現在言語に基づいて menu Items を作り直す。
// 言語切替直後にも呼び、ラベルと値リストを最新の言語に揃える。
func (s *Settings) rebuildMenu(currentTheme *ui.Theme) {
	themeNames := make([]string, len(ui.Themes))
	for i, t := range ui.Themes {
		themeNames[i] = t.Name
	}
	langs := i18n.AllLangs()
	langNames := make([]string, len(langs))
	langIdx := 0
	for i, l := range langs {
		langNames[i] = l.String()
		if l == i18n.CurrentLang() {
			langIdx = i
		}
	}
	str := i18n.S()
	s.menu = &ui.Menu{
		Items: []*ui.MenuItem{
			{Label: str.Setting.Theme, Enabled: true, Values: themeNames, ValueIndex: ui.ThemeIndex(currentTheme)},
			{Label: str.Setting.Language, Enabled: true, Values: langNames, ValueIndex: langIdx},
			{Label: str.Common.Back, Enabled: true},
		},
	}
}

func (s *Settings) Update(d Director) error {
	// Esc で即時に戻れるようにする（Back 項目と同等）
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}
	r := s.menu.Update()
	if r.ValueChanged {
		switch s.menu.Cursor {
		case settingsItemTheme:
			idx := s.menu.Items[settingsItemTheme].ValueIndex
			d.SetTheme(ui.Themes[idx])
		case settingsItemLanguage:
			idx := s.menu.Items[settingsItemLanguage].ValueIndex
			langs := i18n.AllLangs()
			if idx >= 0 && idx < len(langs) {
				i18n.SetLang(langs[idx])
				// ラベルが言語に依存するため即時再構築
				s.rebuildMenu(d.Theme())
				s.menu.Cursor = settingsItemLanguage
			}
		}
	}
	if r.Activated && s.menu.Cursor == settingsItemBack {
		d.Pop()
	}
	return nil
}

func (s *Settings) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	str := i18n.S().Setting
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	headerScale := 4.0
	hw, hh := ui.MeasureText(str.Header, headerScale)
	headerY := float64(sh) * 0.18
	ui.DrawText(dst, str.Header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	menuScale := 2.0
	maxW := s.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := headerY + hh + 80
	s.menu.Draw(dst, theme, mx, my, menuScale)

	ui.DrawText(dst, str.Hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
