package scene

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yiozio/space-miner/internal/dialog"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/save"
	"github.com/yiozio/space-miner/internal/ui"
)

// Title はスタート画面シーン。
type Title struct {
	menu *ui.Menu
}

const (
	titleItemContinue = iota
	titleItemNewGame
	titleItemLoad
	titleItemSetting
	titleItemQuit
)

// NewTitle は新しい Title シーンを返す。
// Continue はセーブが 1 つ以上あれば最新セーブをロード、Load はスロット選択を開く。
func NewTitle() *Title {
	hasSave := save.AnyExists()
	cursor := titleItemNewGame
	if hasSave {
		cursor = titleItemContinue
	}
	tl := &Title{
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Enabled: hasSave},
				{Enabled: true},
				{Enabled: hasSave},
				{Enabled: true},
				{Enabled: true},
			},
			Cursor: cursor,
		},
	}
	tl.applyLabels()
	return tl
}

// applyLabels はメニュー項目のラベルを現在言語で再設定する。
// 設定画面で言語を切り替えた後でもタイトルへ戻った瞬間に反映できるよう、
// Update 冒頭でも呼び直している。
func (t *Title) applyLabels() {
	s := i18n.S().Title
	t.menu.Items[titleItemContinue].Label = s.Continue
	t.menu.Items[titleItemNewGame].Label = s.NewGame
	t.menu.Items[titleItemLoad].Label = s.Load
	t.menu.Items[titleItemSetting].Label = s.Setting
	t.menu.Items[titleItemQuit].Label = s.Quit
}

func (t *Title) Update(d Director) error {
	t.applyLabels()
	r := t.menu.Update()
	if !r.Activated {
		return nil
	}
	switch t.menu.Cursor {
	case titleItemContinue:
		slot := save.LatestSlot()
		if slot < 0 {
			return nil
		}
		res, err := save.Load(slot)
		if err != nil {
			log.Printf("continue: load slot %d: %v", slot, err)
			return nil
		}
		d.Replace(NewExplorationFromPlayer(res.Player, res.Playtime))
	case titleItemNewGame:
		d.Replace(NewExploration())
		// オープニング: 探索シーンの上に Push、閉じるとゲーム本編へ
		d.Push(NewOpeningScene(dialog.Opening()))
	case titleItemLoad:
		if !save.AnyExists() {
			return nil
		}
		d.Push(NewSaveSlotForLoad())
	case titleItemSetting:
		d.Push(NewSettings(d.Theme()))
	case titleItemQuit:
		d.Quit()
	}
	return nil
}

func (t *Title) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	s := i18n.S().Title
	titleScale := 6.0
	tw, th := ui.MeasureText(s.Header, titleScale)
	titleY := float64(sh) * 0.18
	ui.DrawText(dst, s.Header, (float64(sw)-tw)/2, titleY, titleScale, theme.Line)

	menuScale := 2.0
	maxW := t.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := titleY + th + 80
	t.menu.Draw(dst, theme, mx, my, menuScale)

	ui.DrawText(dst, s.Hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
