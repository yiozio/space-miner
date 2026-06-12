// Package logo はタイトルロゴ（SVG 由来）を ebiten のベクター描画で表示する。
// SVG の単一 path を vector.Path へ変換し、nonzero 規則で塗りつぶす
// （穴あき文字の内側も正しく抜ける）。
package logo

import (
	_ "embed"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed space_miner_logo.svg
var svgData []byte

// viewBox のサイズ。スケール計算の基準（縦横比）として使う。
const (
	NativeW = 257.6192
	NativeH = 102.50299
)

var (
	once     sync.Once
	basePath vector.Path // viewBox 座標のままのパス（描画時に拡大・移動）
)

// build は SVG をパースしてパスをキャッシュする（初回のみ）。
func build() {
	parsePath(extractPathD(svgData), &basePath)
}

// Size はロゴの基準サイズ（viewBox）を返す。
func Size() (w, h float64) { return NativeW, NativeH }

// Draw は (x, y) を左上、scale 倍、col 色でロゴを描画する。
// キャッシュ済みパスを GeoM で拡大・移動して FillPath で塗る。
func Draw(dst *ebiten.Image, x, y, scale float64, col color.Color) {
	once.Do(build)

	var geoM ebiten.GeoM
	geoM.Scale(scale, scale)
	geoM.Translate(x, y)
	var p vector.Path
	p.AddPath(&basePath, &vector.AddPathOptions{GeoM: geoM})

	fillOp := &vector.FillOptions{FillRule: vector.FillRuleNonZero}
	drawOp := &vector.DrawPathOptions{AntiAlias: true}
	drawOp.ColorScale.ScaleWithColor(col)
	vector.FillPath(dst, &p, fillOp, drawOp)
}
