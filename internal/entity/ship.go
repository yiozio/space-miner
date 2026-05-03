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

// ThrustActiveDir は現在燃焼しているスラスタ方向。
// ThrustState != ThrustOff のときに参照する。
// 通常は ThrustDirForward。後ろ向きスラスタを稼働させているときは ThrustDirBackward。
type ThrustActiveDir int

const (
	ThrustActiveForward ThrustActiveDir = iota
	ThrustActiveBackward
)

// Ship はグリッド配置のパーツで構成された機体の基本型。
// プレイヤー機・敵機（海賊船）の双方が共有する。
type Ship struct {
	Parts           []Part
	X, Y            float64 // ワールド座標
	VX, VY          float64
	Angle           float64 // 機体の向き（ラジアン）。0 で +x（右）、CW 増加。
	ThrustState     ThrustState
	ThrustActiveDir ThrustActiveDir // 現在炎を出しているスラスタ方向（Forward/Backward）

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
		DrawPart(img, p.Kind(), x, y, float32(GridSize), theme, p.Rotation)
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

// thrustEmitters は推進炎を出すべきパーツ群を返す。
// ThrustActiveDir に一致する向きのスラスタのみ対象（前進中なら Forward、後進中なら Backward）。
// Thruster が 1 つも無い場合は Cockpit を非常用エミッタとして返す（前進専用）。
// 横向き（Sideways）のスラスタは推進に寄与しないため炎も出さない。
func (s *Ship) thrustEmitters() []Part {
	wanted := ThrustDirForward
	if s.ThrustActiveDir == ThrustActiveBackward {
		wanted = ThrustDirBackward
	}
	var out []Part
	hasThruster := false
	for _, p := range s.Parts {
		if p.Kind() != PartThruster {
			continue
		}
		hasThruster = true
		if p.ThrustDir() == wanted {
			out = append(out, p)
		}
	}
	if hasThruster {
		return out
	}
	// 非常用 Cockpit 推進: 前進方向のみ
	if wanted != ThrustDirForward {
		return nil
	}
	for _, p := range s.Parts {
		if p.Kind() == PartCockpit {
			return []Part{p}
		}
	}
	return nil
}

// drawAfterburners は各推進エミッタ（Thruster、または非常時の Cockpit）の後端から
// 各パーツの向き（Rotation）に応じた方向へ炎を描画する。
// パーツのローカル「後端」と「炎方向」は Rotation で 90° 単位に回転する。
func (s *Ship) drawAfterburners(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	sin, cos := math.Sin(s.Angle), math.Cos(s.Angle)
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

	// ローカル → ワールド変換: 画像と同じ R(angle + π/2)
	// 結果は (lx, ly) → (-sin*lx - cos*ly, cos*lx - sin*ly)
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}

	for _, p := range s.thrustEmitters() {
		r := ((p.Rotation % 4) + 4) % 4
		// パーツ中心のローカル位置
		cxL := float64(p.GX) * g
		cyL := float64(p.GY) * g
		// 回転前の「後端中心オフセット」と「後方ベクトル」: ローカル +y が後方。
		// パーツ rotation R を適用して、パーツ局所の +y 方向を 90° CW × R 回転する。
		// (0, 1) を CW 回転: 0:(0,1) 1:(1,0) 2:(0,-1) 3:(-1,0)
		rxL, ryL := 0.0, 0.0
		switch r {
		case 0:
			rxL, ryL = 0, 1
		case 1:
			rxL, ryL = 1, 0
		case 2:
			rxL, ryL = 0, -1
		case 3:
			rxL, ryL = -1, 0
		}
		// 後端中心位置（ローカル）= パーツ中心 + 後方ベクトル × g/2
		halfG := g / 2
		rearLx := cxL + rxL*halfG
		rearLy := cyL + ryL*halfG
		// ワールド座標へ変換（パーツ後端と後方単位ベクトル）
		wox, woy := toWorld(rearLx, rearLy)
		bx, by := toWorld(rxL, ryL)
		// 接線方向（炎幅）: (rxL, ryL) を CW 90° で (ryL, -rxL)
		tx, ty := toWorld(ryL, -rxL)

		baseX := sx + wox
		baseY := sy + woy

		jLen := length + (rand.Float64()*2-1)*jitter
		if jLen < length*0.4 {
			jLen = length * 0.4
		}

		tipX := baseX + bx*jLen
		tipY := baseY + by*jLen
		leftX := baseX + tx*halfWidth
		leftY := baseY + ty*halfWidth
		rightX := baseX - tx*halfWidth
		rightY := baseY - ty*halfWidth

		vector.StrokeLine(dst, float32(leftX), float32(leftY), float32(tipX), float32(tipY), lineWidth, line, false)
		vector.StrokeLine(dst, float32(rightX), float32(rightY), float32(tipX), float32(tipY), lineWidth, line, false)

		if s.ThrustState == ThrustBoost {
			// ブースト時はコア炎＋基底ラインを足して派手さを増す
			coreLen := jLen * (0.45 + rand.Float64()*0.15)
			coreTipX := baseX + bx*coreLen
			coreTipY := baseY + by*coreLen
			coreHalf := halfWidth * 0.5
			cLeftX := baseX + tx*coreHalf
			cLeftY := baseY + ty*coreHalf
			cRightX := baseX - tx*coreHalf
			cRightY := baseY - ty*coreHalf
			vector.StrokeLine(dst, float32(cLeftX), float32(cLeftY), float32(coreTipX), float32(coreTipY), 1, line, false)
			vector.StrokeLine(dst, float32(cRightX), float32(cRightY), float32(coreTipX), float32(coreTipY), 1, line, false)
			vector.StrokeLine(dst, float32(leftX), float32(leftY), float32(rightX), float32(rightY), 1, line, false)
		}
	}
}

// DrawShieldOutline は搭載パーツ群の外周（隣接パーツのない面）をテーマライン色で描画する。
// グリッドが隣接している面はスキップし、シルエットの輪郭だけが残る。
// シールド HP が 1 以上のときに毎フレーム呼び出す想定。
func (s *Ship) DrawShieldOutline(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	if len(s.Parts) == 0 {
		return
	}
	g := float64(GridSize)
	half := g / 2

	occupied := make(map[[2]int]bool, len(s.Parts))
	for _, p := range s.Parts {
		occupied[[2]int{p.GX, p.GY}] = true
	}

	// 船体描画と同じ R(angle + π/2) ローカル → ワールド変換
	sSin, sCos := math.Sin(s.Angle), math.Cos(s.Angle)
	rotate := func(lx, ly float64) (float32, float32) {
		wx := -sSin*lx - sCos*ly
		wy := sCos*lx - sSin*ly
		return float32(sx + wx), float32(sy + wy)
	}

	type edge struct {
		dx, dy         int
		ax, ay, bx, by float64
	}
	for _, p := range s.Parts {
		lx := float64(p.GX) * g
		ly := float64(p.GY) * g
		edges := [4]edge{
			{0, -1, lx - half, ly - half, lx + half, ly - half}, // top
			{1, 0, lx + half, ly - half, lx + half, ly + half},  // right
			{0, 1, lx - half, ly + half, lx + half, ly + half},  // bottom
			{-1, 0, lx - half, ly - half, lx - half, ly + half}, // left
		}
		for _, ed := range edges {
			if occupied[[2]int{p.GX + ed.dx, p.GY + ed.dy}] {
				continue
			}
			x1, y1 := rotate(ed.ax, ed.ay)
			x2, y2 := rotate(ed.bx, ed.by)
			vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, theme.Line, false)
		}
	}
}
