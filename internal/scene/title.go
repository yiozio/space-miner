package scene

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
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
	return &Title{
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Label: "Continue", Enabled: hasSave},
				{Label: "New Game", Enabled: true},
				{Label: "Load", Enabled: hasSave},
				{Label: "Setting", Enabled: true},
				{Label: "Quit Game", Enabled: true},
			},
			Cursor: cursor,
		},
	}
}

func (t *Title) Update(d Director) error {
	r := t.menu.Update()
	if !r.Activated {
		return nil
	}
	switch t.menu.Cursor {
	case titleItemContinue:
		slot := save.LatestSlot()
		if slot == 0 {
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
	titleScale := 6.0
	title := "SPACE  MINER"
	tw, th := ui.MeasureText(title, titleScale)
	titleY := float64(sh) * 0.18
	ui.DrawText(dst, title, (float64(sw)-tw)/2, titleY, titleScale, theme.Line)

	menuScale := 2.0
	maxW := t.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := titleY + th + 80
	t.menu.Draw(dst, theme, mx, my, menuScale)

	hint := "[ Up/Down: Move    Enter: Select ]"
	ui.DrawText(dst, hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
