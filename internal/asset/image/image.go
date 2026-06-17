// Package image はゲームの機体・パーツ描画に使うピクセル画像（スプライト）を
// 埋め込みで提供する。ベース船体シートとパーツシートの 2 枚を持ち、
// パーツシートは 16x16px のセルが 4x4 に並ぶ構成（配置は data 参照）。
package image

import (
	"bytes"
	_ "embed"
	stdimage "image"
	_ "image/png"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed spaceship_3_3_v2.png
var shipBasePNG []byte

//go:embed spaceship_parts.png
var partsPNG []byte

// CellSize はパーツ／ベースパネル 1 セルの元ピクセル数。
const CellSize = 16

// ShipBaseGridX, ShipBaseGridY はベース船体スプライト内で 3x3 パネルが始まる左上座標。
const (
	ShipBaseGridX = 2
	ShipBaseGridY = 8
)

var (
	loadOnce sync.Once
	shipBase *ebiten.Image
	parts    *ebiten.Image
)

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
