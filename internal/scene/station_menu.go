package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	stationItemRepair = iota
	stationItemRefuel
	stationItemTavern
	stationItemShop
	stationItemEdit
)

// StationMenu はステーションのハブ画面。
// 探索シーンの上にオーバーレイ表示し、Repair / Refuel / Tavern / Shop / Editor の入口を提供する。
type StationMenu struct {
	player      *entity.Player
	world       *entity.World
	stationName string
	menu        *ui.Menu
}

// NewStationMenu はメニューを生成する。
// world / stationName は Tavern などサブシーンに渡すための補助情報。
func NewStationMenu(p *entity.Player, world *entity.World, stationName string) *StationMenu {
	sm := &StationMenu{
		player:      p,
		world:       world,
		stationName: stationName,
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Enabled: true},
				{Enabled: true},
				{Enabled: true},
				{Enabled: true},
				{Enabled: true},
			},
		},
	}
	sm.applyLabels()
	return sm
}

// applyLabels は項目ラベルを現在言語で再設定する。
// 言語切替直後にもステーション画面に戻った瞬間に反映するため Update 冒頭でも呼ぶ。
func (sm *StationMenu) applyLabels() {
	st := i18n.S().Station
	sm.menu.Items[stationItemRepair].Label = st.Repair
	sm.menu.Items[stationItemRefuel].Label = st.Refuel
	sm.menu.Items[stationItemTavern].Label = st.Tavern
	sm.menu.Items[stationItemShop].Label = st.Shop
	sm.menu.Items[stationItemEdit].Label = st.Editor
}

func (sm *StationMenu) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}
	sm.applyLabels()
	r := sm.menu.Update()
	if !r.Activated {
		return nil
	}
	switch sm.menu.Cursor {
	case stationItemRepair:
		// 簡易: 即時に HP を全回復（コストなし）
		if sm.player.HP < sm.player.MaxHP {
			sm.player.HP = sm.player.MaxHP
		}
	case stationItemRefuel:
		if sm.player.Fuel < sm.player.MaxFuel {
			sm.player.Fuel = sm.player.MaxFuel
		}
	case stationItemTavern:
		d.Push(NewStationTavern(sm.player, sm.world, sm.stationName))
	case stationItemShop:
		d.Push(NewStationShop(sm.player))
	case stationItemEdit:
		d.Push(NewStationEditor(sm.player))
	}
	return nil
}

func (sm *StationMenu) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 黒半透明オーバーレイ
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 200}, false)

	// ヘッダ
	st := i18n.S().Station
	headerScale := 4.0
	hw, hh := ui.MeasureText(st.Header, headerScale)
	headerY := float64(sh) * 0.18
	ui.DrawText(dst, st.Header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// ステータスサマリ
	statusScale := 1.5
	status := fmt.Sprintf(st.StatusFmt,
		sm.player.HP, sm.player.MaxHP,
		int(sm.player.Fuel), int(sm.player.MaxFuel),
		sm.player.Credits)
	stw, _ := ui.MeasureText(status, statusScale)
	ui.DrawText(dst, status, (float64(sw)-stw)/2, headerY+hh+24, statusScale, theme.LineDim)

	// メニュー
	menuScale := 2.0
	maxW := sm.menu.MaxLabelWidth(menuScale)
	mx := (float64(sw) - maxW) / 2
	my := headerY + hh + 100
	sm.menu.Draw(dst, theme, mx, my, menuScale)

	// 操作ヒント
	ui.DrawText(dst, st.Hint, 20, float64(sh)-30, 1.5, theme.LineDim)
}
