package entity

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/ui"
)

// ThrustState は推力描画用の状態。Player/Enemy が毎フレーム設定する。
type ThrustState int

const (
	ThrustOff ThrustState = iota
	ThrustOn
	ThrustBoost
)

// Ship はグリッド配置のパーツで構成された機体の基本型。
// プレイヤー機・敵機（海賊船）の双方が共有する。
type Ship struct {
	Parts       []Part
	X, Y        float64 // ワールド座標
	VX, VY      float64
	Angle       float64 // 機体の向き（ラジアン）。0 で +x（右）、CW 増加。
	ThrustState ThrustState

	// 描画キャッシュ。テーマ変更時に再生成する。
	cachedTheme  *ui.Theme
	image        *ebiten.Image
	imageOffsetX float64 // 画像内のコックピット中心 X
	imageOffsetY float64 // 画像内のコックピット中心 Y
}

// InvalidateImage は船体画像のキャッシュを破棄する。
// パーツ構成が変わった（編集された）場合に呼ぶ。
func (s *Ship) InvalidateImage() {
	s.cachedTheme = nil
	s.image = nil
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
		DrawPart(img, p.Kind(), x, y, float32(GridSize), theme)
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
// ThrustState に応じて Thruster パーツの後方にアフターバーナーを重ねる。
func (s *Ship) DrawAt(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	s.ensureImage(theme)
	// アフターバーナーは船体の下に敷くと末端が船体線で隠れるので、先に描画して
	// その上に船体画像を重ねる。
	if s.ThrustState != ThrustOff {
		s.drawAfterburners(dst, sx, sy, theme)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.imageOffsetX, -s.imageOffsetY)
	// 画像はローカル -y を前方として描かれているので、Angle が 0（=+x）のとき
	// 画像を CW に π/2 回す必要がある。
	op.GeoM.Rotate(s.Angle + math.Pi/2)
	op.GeoM.Translate(sx, sy)
	dst.DrawImage(s.image, op)
}

// drawAfterburners は各 Thruster パーツの後端から後方に向けて炎を描画する。
// 後方 = ワールド座標における前進方向の逆。
func (s *Ship) drawAfterburners(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	sin, cos := math.Sin(s.Angle), math.Cos(s.Angle)
	bx, by := -cos, -sin // 後方ベクトル
	px, py := -sin, cos  // 後方に直交する炎幅方向

	g := float64(GridSize)

	var length, halfWidth, jitter float64
	var lineWidth float32
	if s.ThrustState == ThrustBoost {
		length = g * 1.5
		halfWidth = g * 0.32
		jitter = g * 0.35
		lineWidth = 2
	} else {
		length = g * 0.55
		halfWidth = g * 0.16
		jitter = g * 0.12
		lineWidth = 1
	}
	line := theme.Line

	for _, p := range s.Parts {
		if p.Kind() != PartThruster {
			continue
		}
		// パーツの後端中心（ローカル）。ローカル +y が後方。
		lx := float64(p.GX) * g
		rearLy := float64(p.GY)*g + g/2
		// ローカル → ワールド: 画像と同じ R(angle + π/2) を適用
		// R(angle + π/2) * (lx, rearLy)
		//   = (-sin*lx - cos*rearLy, cos*lx - sin*rearLy)
		wox := -sin*lx - cos*rearLy
		woy := cos*lx - sin*rearLy
		baseX := sx + wox
		baseY := sy + woy

		jLen := length + (rand.Float64()*2-1)*jitter
		if jLen < length*0.4 {
			jLen = length * 0.4
		}

		tipX := baseX + bx*jLen
		tipY := baseY + by*jLen
		leftX := baseX + px*halfWidth
		leftY := baseY + py*halfWidth
		rightX := baseX - px*halfWidth
		rightY := baseY - py*halfWidth

		vector.StrokeLine(dst, float32(leftX), float32(leftY), float32(tipX), float32(tipY), lineWidth, line, false)
		vector.StrokeLine(dst, float32(rightX), float32(rightY), float32(tipX), float32(tipY), lineWidth, line, false)

		if s.ThrustState == ThrustBoost {
			// ブースト時はコア炎＋基底ラインを足して派手さを増す
			coreLen := jLen * (0.45 + rand.Float64()*0.15)
			coreTipX := baseX + bx*coreLen
			coreTipY := baseY + by*coreLen
			coreHalf := halfWidth * 0.5
			cLeftX := baseX + px*coreHalf
			cLeftY := baseY + py*coreHalf
			cRightX := baseX - px*coreHalf
			cRightY := baseY - py*coreHalf
			vector.StrokeLine(dst, float32(cLeftX), float32(cLeftY), float32(coreTipX), float32(coreTipY), 1, line, false)
			vector.StrokeLine(dst, float32(cRightX), float32(cRightY), float32(coreTipX), float32(coreTipY), 1, line, false)
			vector.StrokeLine(dst, float32(leftX), float32(leftY), float32(rightX), float32(rightY), 1, line, false)
		}
	}
}
