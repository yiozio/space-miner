package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

// StationEditor は宇宙船編集画面。
// 現状は船体プレビューのみのプレースホルダ。配置編集は後続フェーズで実装。
type StationEditor struct {
	player *entity.Player
}

func NewStationEditor(p *entity.Player) *StationEditor {
	return &StationEditor{player: p}
}

func (se *StationEditor) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
	}
	return nil
}

func (se *StationEditor) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	header := "SHIP EDITOR"
	hw, _ := ui.MeasureText(header, headerScale)
	ui.DrawText(dst, header, (float64(sw)-hw)/2, 30, headerScale, theme.Line)

	// 現在の船体プレビュー（中央に固定描画。実際の角度は無視して上向き表示）
	originalAngle := se.player.Angle
	se.player.Angle = -1.5707963267948966 // -π/2 で上向き
	se.player.DrawAt(dst, float64(sw)/2, float64(sh)/2, theme)
	se.player.Angle = originalAngle

	msg := "(Layout editing coming soon)"
	mw, _ := ui.MeasureText(msg, 1.6)
	ui.DrawText(dst, msg, (float64(sw)-mw)/2, float64(sh)-90, 1.6, theme.LineDim)

	ui.DrawText(dst, "[ Esc: Back ]", 20, float64(sh)-30, 1.5, theme.LineDim)
}
