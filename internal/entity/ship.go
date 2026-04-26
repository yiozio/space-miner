package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yiozio/space-miner/internal/ui"
)

// Ship はグリッド配置のパーツで構成された機体の基本型。
// プレイヤー機・敵機（海賊船）の双方が共有する。
type Ship struct {
	Parts  []Part
	X, Y   float64 // ワールド座標
	VX, VY float64
	Angle  float64 // 機体の向き（ラジアン）。0 で +x（右）、CW 増加。

	// 描画キャッシュ。テーマ変更時に再生成する。
	cachedTheme  *ui.Theme
	image        *ebiten.Image
	imageOffsetX float64 // 画像内のコックピット中心 X
	imageOffsetY float64 // 画像内のコックピット中心 Y
}

// ensureImage はテーマが変わっていれば船体画像を作り直す。
func (s *Ship) ensureImage(theme *ui.Theme) {
	if s.cachedTheme == theme && s.image != nil {
		return
	}
	minGX, minGY, maxGX, maxGY := s.bounds()
	w := (maxGX - minGX + 1) * GridSize
	h := (maxGY - minGY + 1) * GridSize
	img := ebiten.NewImage(w, h)
	for _, p := range s.Parts {
		x := float32((p.GX - minGX) * GridSize)
		y := float32((p.GY - minGY) * GridSize)
		drawPart(img, p, x, y, theme)
	}
	s.image = img
	// コックピット (GX=0, GY=0) のセル中心を回転中心とする
	s.imageOffsetX = float64(-minGX*GridSize) + float64(GridSize)/2
	s.imageOffsetY = float64(-minGY*GridSize) + float64(GridSize)/2
	s.cachedTheme = theme
}

// bounds はパーツ配置のグリッド境界を返す。
func (s *Ship) bounds() (minGX, minGY, maxGX, maxGY int) {
	if len(s.Parts) == 0 {
		return
	}
	minGX, maxGX = s.Parts[0].GX, s.Parts[0].GX
	minGY, maxGY = s.Parts[0].GY, s.Parts[0].GY
	for _, p := range s.Parts[1:] {
		if p.GX < minGX {
			minGX = p.GX
		}
		if p.GX > maxGX {
			maxGX = p.GX
		}
		if p.GY < minGY {
			minGY = p.GY
		}
		if p.GY > maxGY {
			maxGY = p.GY
		}
	}
	return
}

// DrawAt はスクリーン座標 (sx, sy) を機体中心として船体を描画する。
func (s *Ship) DrawAt(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	s.ensureImage(theme)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.imageOffsetX, -s.imageOffsetY)
	// 画像はローカル -y を前方として描かれているので、Angle が 0（=+x）のとき
	// 画像を CW に π/2 回す必要がある。
	op.GeoM.Rotate(s.Angle + math.Pi/2)
	op.GeoM.Translate(sx, sy)
	dst.DrawImage(s.image, op)
}
