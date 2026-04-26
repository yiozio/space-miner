package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/basicfont"
)

// defaultFace はテーマ非依存のテキスト描画用フォント。
// レトロ感を出すためビットマップフォントを採用し、scale で拡大して使う。
var defaultFace = text.NewGoXFace(basicfont.Face7x13)

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
