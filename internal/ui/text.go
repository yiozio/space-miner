package ui

import (
	"image/color"

	"github.com/hajimehoshi/bitmapfont/v4"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// defaultFace はテーマ非依存のテキスト描画用フォント。
// CJK (日本語) も含む 12px ビットマップフォント。scale で拡大して使う。
var defaultFace = text.NewGoXFace(bitmapfont.Face)

// DrawText は (x, y) を左上として s を描画する。
// scale は等倍からの拡大率。
func DrawText(dst *ebiten.Image, s string, x, y, scale float64, c color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, defaultFace, op)
}

// MeasureText は scale を考慮した描画サイズを返す。
func MeasureText(s string, scale float64) (w, h float64) {
	w, h = text.Measure(s, defaultFace, 0)
	return w * scale, h * scale
}
