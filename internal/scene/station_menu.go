package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	stationItemRepair = iota
	stationItemRefuel
	stationItemShop
	stationItemEdit
)

// StationMenu はステーションのハブ画面。
// 探索シーンの上にオーバーレイ表示し、Repair / Refuel / Shop / Editor の入口を提供する。
type StationMenu struct {
	player *entity.Player
	menu   *ui.Menu
}

// NewStationMenu はメニューを生成する。
func NewStationMenu(p *entity.Player) *StationMenu {
	return &StationMenu{
		player: p,
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Label: "Repair", Enabled: true},
				{Label: "Refuel", Enabled: true},
				{Label: "Parts Shop", Enabled: true},
				{Label: "Ship Editor", Enabled: true},
			},
		},
	}
}

func (sm *StationMenu) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}
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
	headerScale := 4.0
	header := "STATION"
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := float64(sh) * 0.18
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// ステータスサマリ
	statusScale := 1.5
	status := fmt.Sprintf("HP %d/%d   FUEL %d/%d   CREDITS %d",
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
	ui.DrawText(dst, "[ Up/Down: Move    Enter: Select    Esc: Leave Station ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}
