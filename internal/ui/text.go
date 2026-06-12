package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/yiozio/space-miner/internal/asset/font"
)

// baseTextSize は scale=1 のときのフォントサイズ（px）。
// 旧ビットマップフォント（12px）とサイズ感を揃えている。
const baseTextSize = 12

// face は scale に応じたサイズの描画フェイスを返す。
// TTF（ベクター）なので任意の倍率で輪郭が崩れない。
func face(scale float64) *text.GoTextFace {
	return &text.GoTextFace{Source: font.Source(), Size: baseTextSize * scale}
}

// DrawText は (x, y) を左上として s を描画する。
// scale は等倍（12px）からの拡大率。
func DrawText(dst *ebiten.Image, s string, x, y, scale float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face(scale), op)
}

// MeasureText は scale を考慮した描画サイズを返す。
func MeasureText(s string, scale float64) (w, h float64) {
	return text.Measure(s, face(scale), 0)
}
