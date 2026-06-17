// Package image はゲームの機体・パーツ描画に使うピクセル画像（スプライト）を
// 埋め込みで提供する。ベース船体シートとパーツシートの 2 枚を持ち、
// パーツシートは 16x16px のセルが 4x4 に並ぶ構成（配置は data 参照）。
package image

import (
	"bytes"
	_ "embed"
	stdimage "image"
	"image/draw"
	"image/gif"
	_ "image/png"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed spaceship_3_3_v2.png
var shipBasePNG []byte

//go:embed spaceship_parts.png
var partsPNG []byte

//go:embed 3rd_planet.gif
var planet3rdGIF []byte

// CellSize はパーツ／ベースパネル 1 セルの元ピクセル数。
const CellSize = 16

// ShipBaseGridX, ShipBaseGridY はベース船体スプライト内で 3x3 パネルが始まる左上座標。
const (
	ShipBaseGridX = 2
	ShipBaseGridY = 8
)

// ShipBaseW, ShipBaseH はベース船体スプライトの元ピクセルサイズ（PNG と一致、テストで検証）。
const (
	ShipBaseW = 52
	ShipBaseH = 61
)

var (
	loadOnce sync.Once
	shipBase *ebiten.Image
	parts    *ebiten.Image

	planetAnimOnce sync.Once
	planetAnim     *planetAnimData
)

// planetAnimData は 3rd_planet.gif を展開した惑星アニメーション（正距円筒テクスチャの連番）。
// frames は各フレーム（2x 縮小済み）、cum は各フレーム終了時刻の累積秒、total は1ループ秒。
type planetAnimData struct {
	frames []*ebiten.Image
	cum    []float64
	total  float64
}

// planetTexMaxW は惑星テクスチャの目標最大幅（px）。これを超えるソースは 2x 縮小を繰り返す。
const planetTexMaxW = 800

// loadPlanetAnim は GIF を初回のみ展開する。任意のフレーム数・サイズ・disposal に対応する。
// 各フレームを直前の合成結果へ重ね（GIF の disposal 規約を反映）、幅が大きい場合は
// planetTexMaxW 以下まで 2x ボックス縮小して ebiten 画像にする。
func loadPlanetAnim() *planetAnimData {
	planetAnimOnce.Do(func() {
		g, err := gif.DecodeAll(bytes.NewReader(planet3rdGIF))
		if err != nil {
			panic("asset/image: 3rd_planet.gif のデコードに失敗: " + err.Error())
		}
		canvas := stdimage.NewRGBA(stdimage.Rect(0, 0, g.Config.Width, g.Config.Height))
		var prev *stdimage.RGBA // disposal=Previous 用の直前スナップショット
		a := &planetAnimData{}
		t := 0.0
		for i, fr := range g.Image {
			disposal := byte(gif.DisposalNone)
			if i < len(g.Disposal) {
				disposal = g.Disposal[i]
			}
			if disposal == gif.DisposalPrevious {
				prev = cloneRGBA(canvas)
			}
			// 直前の合成結果に重ねる（変更画素のみ／部分フレームでも正しく繋がる）。
			draw.Draw(canvas, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
			// NewImageFromImage は画素をコピーするので、以降 canvas を書き換えても影響しない。
			a.frames = append(a.frames, ebiten.NewImageFromImage(shrinkToWidth(canvas, planetTexMaxW)))
			d := float64(g.Delay[i]) / 100.0
			if d <= 0 {
				d = 0.1
			}
			t += d
			a.cum = append(a.cum, t)
			// 次フレームへ向けた disposal 処理。
			switch disposal {
			case gif.DisposalBackground:
				clearRect(canvas, fr.Bounds())
			case gif.DisposalPrevious:
				if prev != nil {
					copy(canvas.Pix, prev.Pix)
				}
			}
		}
		a.total = t
		planetAnim = a
	})
	return planetAnim
}

// shrinkToWidth は幅が maxW を超える間 2x 縮小を繰り返す。縮小不要なら src をそのまま返す
// （呼び出し側の NewImageFromImage が画素をコピーするため共有で問題ない）。
func shrinkToWidth(src *stdimage.RGBA, maxW int) *stdimage.RGBA {
	img := src
	for img.Rect.Dx() > maxW && img.Rect.Dx() >= 2 && img.Rect.Dy() >= 2 {
		img = downscale2x(img)
	}
	return img
}

// cloneRGBA は RGBA の複製を返す。
func cloneRGBA(s *stdimage.RGBA) *stdimage.RGBA {
	d := stdimage.NewRGBA(s.Rect)
	copy(d.Pix, s.Pix)
	return d
}

// clearRect は矩形範囲を透明にする（disposal=Background 用）。
func clearRect(img *stdimage.RGBA, r stdimage.Rectangle) {
	r = r.Intersect(img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		o := img.PixOffset(r.Min.X, y)
		for x := 0; x < r.Dx()*4; x++ {
			img.Pix[o+x] = 0
		}
	}
}

// downscale2x は RGBA を 2x2 ボックス平均で半分に縮小する（奇数サイズは末端1pxを捨てる）。
func downscale2x(src *stdimage.RGBA) *stdimage.RGBA {
	w, h := src.Rect.Dx()/2, src.Rect.Dy()/2
	dst := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b, a int
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					i := src.PixOffset(src.Rect.Min.X+2*x+dx, src.Rect.Min.Y+2*y+dy)
					r += int(src.Pix[i])
					g += int(src.Pix[i+1])
					b += int(src.Pix[i+2])
					a += int(src.Pix[i+3])
				}
			}
			o := dst.PixOffset(x, y)
			dst.Pix[o] = uint8(r / 4)
			dst.Pix[o+1] = uint8(g / 4)
			dst.Pix[o+2] = uint8(b / 4)
			dst.Pix[o+3] = uint8(a / 4)
		}
	}
	return dst
}

// Planet3rdFrameAt は時刻 t（秒）に対応する惑星アニメフレームを返す（ループ）。
// フレームが無ければ nil。
func Planet3rdFrameAt(t float64) *ebiten.Image {
	a := loadPlanetAnim()
	if a.total <= 0 || len(a.frames) == 0 {
		return nil
	}
	tt := math.Mod(t, a.total)
	if tt < 0 {
		tt += a.total
	}
	for i, c := range a.cum {
		if tt < c {
			return a.frames[i]
		}
	}
	return a.frames[len(a.frames)-1]
}

// load は初回アクセス時に PNG をデコードして ebiten 画像にする（描画スレッドから呼ばれる想定）。
func load() {
	loadOnce.Do(func() {
		shipBase = mustDecode(shipBasePNG)
		parts = mustDecode(partsPNG)
	})
}

func mustDecode(b []byte) *ebiten.Image {
	img, _, err := stdimage.Decode(bytes.NewReader(b))
	if err != nil {
		// 埋め込み PNG が壊れている場合のみ（ビルド時に確定）。
		panic("asset/image: 埋め込み PNG のデコードに失敗: " + err.Error())
	}
	return ebiten.NewImageFromImage(img)
}

// ShipBase はベース船体スプライト全体を返す。
func ShipBase() *ebiten.Image {
	load()
	return shipBase
}

// ShipBaseSize はベース船体スプライトの元ピクセルサイズ（幅・高さ）を返す。
func ShipBaseSize() (w, h int) {
	load()
	b := shipBase.Bounds()
	return b.Dx(), b.Dy()
}

// Cell はパーツシートの (col, row) セル（16x16）のサブ画像を返す。
func Cell(col, row int) *ebiten.Image {
	load()
	r := stdimage.Rect(col*CellSize, row*CellSize, (col+1)*CellSize, (row+1)*CellSize)
	return parts.SubImage(r).(*ebiten.Image)
}
