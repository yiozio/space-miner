package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

// ステーション施設画面（ショップ・機体エディタ・酒場）共通の人物画像サイズ。
// 3 画面のうちコンテンツ幅が最小のショップに合わせる。
const (
	stationPortraitW = float64(shopSideWidth)*2 + shopInfoW + shopColGap*2
	stationPortraitH = 200.0
)

// drawStationPortrait は施設の人物画像のプレースホルダ枠を画面中央に描く
// （実画像は後日差し替え）。y は枠の上端。
func drawStationPortrait(dst *ebiten.Image, theme *ui.Theme, label string, sw int, y float64) {
	x := (float64(sw) - stationPortraitW) / 2
	vector.FillRect(dst, float32(x), float32(y), float32(stationPortraitW), float32(stationPortraitH),
		color.NRGBA{0, 0, 0, 255}, false)
	vector.StrokeRect(dst, float32(x), float32(y), float32(stationPortraitW), float32(stationPortraitH),
		1, theme.Line, false)
	lw, lh := ui.MeasureText(label, 1.4)
	ui.DrawText(dst, label, x+(stationPortraitW-lw)/2, y+(stationPortraitH-lh)/2, 1.4, theme.LineDim)
}
