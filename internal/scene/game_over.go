package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

// GameOver は HP=0 になった時に Exploration の上にかぶせるシーン。
// Enter/Space でタイトルに戻る。
type GameOver struct{}

// NewGameOver はゲームオーバー画面を返す。
func NewGameOver() *GameOver { return &GameOver{} }

func (g *GameOver) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// GameOver を閉じてから直下の Exploration をタイトルに差し替える
		d.Pop()
		d.Replace(NewTitle())
		return nil
	}
	return nil
}

func (g *GameOver) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 背景の暗いオーバーレイ
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	// メインタイトル
	s := i18n.S().GameOver
	headerScale := 6.0
	hw, hh := ui.MeasureText(s.Header, headerScale)
	headerY := float64(sh)*0.36 - hh/2
	ui.DrawText(dst, s.Header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// 操作ヒント
	hw2, _ := ui.MeasureText(s.Hint, 1.5)
	ui.DrawText(dst, s.Hint, (float64(sw)-hw2)/2, headerY+hh+60, 1.5, theme.LineDim)
}
