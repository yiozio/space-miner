// Package logo はタイトルロゴ（SVG 由来）を ebiten のベクター描画で表示する。
// SVG の単一 path を vector.Path へ変換し、nonzero 規則で塗りつぶす
// （穴あき文字の内側も正しく抜ける）。
package logo

import (
	_ "embed"
	"image"
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
	once      sync.Once
	baseVerts []ebiten.Vertex // viewBox 座標のままの塗り頂点（描画時に拡大・移動）
	indices   []uint16

	// DrawTriangles 用の 1px 白テクスチャ。
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() { whiteImage.Fill(color.White) }

// build は SVG をパースして三角形分割し、頂点をキャッシュする（初回のみ）。
func build() {
	var p vector.Path
	parsePath(extractPathD(svgData), &p)
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
	}
	baseVerts, indices = vs, is
}

// Size はロゴの基準サイズ（viewBox）を返す。
func Size() (w, h float64) { return NativeW, NativeH }

// Draw は (x, y) を左上、scale 倍、col 色でロゴを描画する。
// キャッシュ済み頂点を複製して座標と色だけ差し替えるため毎フレーム呼んでよい。
func Draw(dst *ebiten.Image, x, y, scale float64, col color.Color) {
	once.Do(build)

	cr, cg, cb, ca := col.RGBA()
	fr, fg, fb, fa := float32(cr)/0xffff, float32(cg)/0xffff, float32(cb)/0xffff, float32(ca)/0xffff
	sx, sy := float32(x), float32(y)
	sc := float32(scale)

	vs := make([]ebiten.Vertex, len(baseVerts))
	for i := range baseVerts {
		v := baseVerts[i]
		v.DstX = sx + v.DstX*sc
		v.DstY = sy + v.DstY*sc
		v.ColorR, v.ColorG, v.ColorB, v.ColorA = fr, fg, fb, fa
		vs[i] = v
	}
	op := &ebiten.DrawTrianglesOptions{FillRule: ebiten.FillRuleNonZero, AntiAlias: true}
	dst.DrawTriangles(vs, indices, whiteSubImage, op)
}
