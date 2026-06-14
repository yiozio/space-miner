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

// ThrustActiveDir は現在燃焼しているスラスタ方向のビットフラグ。
// ThrustState != ThrustOff のときに参照する。
// 前後/左右を同時稼働できるよう OR で結合される（例: 前進 + 右ストラフ）。
type ThrustActiveDir int

const (
	ThrustActiveForward  ThrustActiveDir = 1 << iota // 1: 前向きスラスタ稼働中
	ThrustActiveBackward                             // 2: 後ろ向きスラスタ稼働中
	ThrustActiveLeft                                 // 4: 左向きスラスタ稼働中
	ThrustActiveRight                                // 8: 右向きスラスタ稼働中
)

// Ship はグリッド配置のパーツで構成された機体の基本型。
// プレイヤー機・敵機（海賊船）の双方が共有する。
type Ship struct {
	Parts           []Part
	BaseID          ShipBaseID // 機体ベース（土台）。グリッドサイズ・基礎ステータスを決める。海賊機では未使用。
	X, Y            float64    // ワールド座標
	VX, VY          float64
	Angle           float64 // 機体の向き（ラジアン）。0 で +x（右）、CW 増加。
	ThrustState     ThrustState
	ThrustActiveDir ThrustActiveDir // 現在炎を出しているスラスタ方向（Forward/Backward）

	// Trail は直近の通過位置（古い順、ワールド座標）。描画側が尾を引く軌跡に使う。
	Trail []TrailPoint

	// 描画キャッシュ。テーマ変更時に再生成する。
	cachedTheme  *ui.Theme
	image        *ebiten.Image
	imageOffsetX float64 // 画像内のコックピット中心 X
	imageOffsetY float64 // 画像内のコックピット中心 Y
}

// cockpitPivotFracY はコックピット三角形（頂点が上・底辺が下）の重心の、
// セル上端からの y 比率。回転中心・軌跡・光点をこの重心に合わせる。
// 三角形の頂点 y 比は (0.12, 0.88, 0.88)（drawPartRaw の inset=0.12 に対応）。
const cockpitPivotFracY = (0.12 + 0.88 + 0.88) / 3.0

// partPivotShiftY は回転中心（コックピット重心）がセル中心より前寄りにあることによる
// ローカル y の補正量。パーツ中心のローカル y は GY*GridSize - partPivotShiftY となる。
// セル基準（GY*GridSize）のまま扱うと、機体描画に対して後方へこの分だけズレる。
const partPivotShiftY = GridSize * (cockpitPivotFracY - 0.5)

// PartLocalCenter はパーツ (gx, gy) の機体ローカル座標における中心位置を返す。
// 機体描画（ensureImage の imageOffset）と同じ基準なので、発射位置・衝突判定・
// エフェクト基点など「描画された機体に合わせたい」計算は必ずこれを使うこと。
func PartLocalCenter(gx, gy int) (lx, ly float64) {
	return float64(gx) * GridSize, float64(gy)*GridSize - partPivotShiftY
}

// TrailPoint は軌跡の1点（ワールド座標）。
type TrailPoint struct{ X, Y float64 }

// 軌跡の保持点数と、点を追加する最小移動距離（px）。
// 距離で間引くことで、停止中に同じ点が溜まらず、見た目の密度が一定になる。
// 尾の長さは「直近 shipTrailFrames フレーム分の移動距離」を目安に速度へ比例させ、
// 低速時に長い尾が引きずられるのを防ぐ。
const (
	shipTrailMax     = 30
	shipTrailSpacing = 8.0
	shipTrailFrames  = 30 // 尾が表す移動時間の目安（60fps で 0.5 秒）
)

// trailTargetLen は現在速度に応じた軌跡の目標点数を返す。
func (s *Ship) trailTargetLen() int {
	speed := math.Hypot(s.VX, s.VY)
	return min(int(speed*shipTrailFrames/shipTrailSpacing), shipTrailMax)
}

// PushTrail は前回の点から十分動いていれば現在位置を軌跡に追加する。
// 速度に応じた目標点数を超えた分は古い点から捨てる（減速すると尾が縮む）。
// Player.Update / Pirate.Update など、機体を動かした後に毎フレーム呼ぶ。
func (s *Ship) PushTrail() {
	target := s.trailTargetLen()
	if n := len(s.Trail); n > 0 {
		dx := s.X - s.Trail[n-1].X
		dy := s.Y - s.Trail[n-1].Y
		if dx*dx+dy*dy >= shipTrailSpacing*shipTrailSpacing {
			s.Trail = append(s.Trail, TrailPoint{s.X, s.Y})
		}
	} else if target > 0 {
		s.Trail = append(s.Trail, TrailPoint{s.X, s.Y})
	}
	if len(s.Trail) > target {
		copy(s.Trail, s.Trail[len(s.Trail)-target:])
		s.Trail = s.Trail[:target]
	}
}

// ClearTrail は軌跡を消す。ワープなどの瞬間移動で尾が伸びるのを防ぐ。
func (s *Ship) ClearTrail() { s.Trail = s.Trail[:0] }

// InvalidateImage は船体画像のキャッシュを破棄する。
// パーツ構成が変わった（編集された）場合に呼ぶ。
func (s *Ship) InvalidateImage() {
	s.cachedTheme = nil
	s.image = nil
}

// GridHalf は機体ベースの配置グリッド半径を返す（3x3 なら 1）。
func (s *Ship) GridHalf() int { return ShipBaseDefByID(s.BaseID).GridHalf }

// HullRadius はベース船体の下方向（尾）への外接半径を返す。バイタルバーの位置決めに使う。
func (s *Ship) HullRadius() float64 {
	_, hHalf := shipHullExtent(s.GridHalf(), float64(GridSize))
	return hHalf
}

// TrailLightOffsets はベース船体の左右に尖った先端（光点の発生位置）の、
// ピボットからのワールド座標オフセットを 2 つ返す（右先端, 左先端の順）。
// 軌跡・光点を機体の向きに合わせて配置するために使う。
func (s *Ship) TrailLightOffsets() [2][2]float64 {
	sin, cos := math.Sin(s.Angle), math.Cos(s.Angle)
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	wHalf, hHalf := shipHullExtent(s.GridHalf(), float64(GridSize))
	// ハル輪郭の左右中央（最も横に張り出した尖り）。ハル中心はピボットより
	// partPivotShiftY だけ前方にあるので、その分 y を補正する。
	ly := hHalf*0.12 - partPivotShiftY
	rx, ry := toWorld(wHalf, ly)
	lx, lyy := toWorld(-wHalf, ly)
	return [2][2]float64{{rx, ry}, {lx, lyy}}
}

// ensureImage はテーマが変わっていれば船体画像を作り直す。
// ベース船体を敷いた上に、固定グリッド（±GridHalf）のセルへパーツを重ねて 1 枚の画像にする。
func (s *Ship) ensureImage(theme *ui.Theme) {
	if s.cachedTheme == theme && s.image != nil {
		return
	}
	gridHalf := s.GridHalf()
	g := float64(GridSize)
	_, hHalf := shipHullExtent(gridHalf, g)
	// 画像はベース船体とグリッド全体（角セル含む）を収める正方形。
	imgHalf := math.Ceil(math.Max(hHalf, (float64(gridHalf)+0.5)*g) + 6)
	size := int(imgHalf * 2)
	center := float64(size) / 2
	img := ebiten.NewImage(size, size)
	// ベース船体を先に敷き、その上にパーツを重ねる。
	DrawShipBase(img, center, center, gridHalf, g, theme)
	for _, p := range s.Parts {
		x := center + float64(p.GX)*g - g/2
		y := center + float64(p.GY)*g - g/2
		DrawPart(img, p.Kind(), float32(x), float32(y), float32(GridSize), theme, p.Rotation)
	}
	s.image = img
	// ピボット（回転中心）は従来どおり原点セルのコックピット重心位置に保つ。
	// これで PartLocalCenter・当たり判定・発射計算は変更不要。
	s.imageOffsetX = center
	s.imageOffsetY = center + partPivotShiftY
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
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-s.imageOffsetX, -s.imageOffsetY)
	// 画像はローカル -y を前方として描かれているので、Angle が 0（=+x）のとき
	// 画像を CW に π/2 回す必要がある。
	op.GeoM.Rotate(s.Angle + math.Pi/2)
	op.GeoM.Translate(sx, sy)
	dst.DrawImage(s.image, op)
	// アフターバーナーは不透明なベース船体に隠れないよう、船体画像の手前に描く。
	if s.ThrustState != ThrustOff {
		s.drawAfterburners(dst, sx, sy, theme)
	}
}

// thrustEmitters は推進炎を出すべきパーツ群を返す。
// ThrustActiveDir のビットに対応する向きのスラスタを集める（前後と左右を同時に含みうる）。
// Thruster が 1 つも無い場合は Cockpit を非常用エミッタとして返す（前進方向のみ）。
func (s *Ship) thrustEmitters() []Part {
	dirActive := func(d ThrustDir) bool {
		switch d {
		case ThrustDirForward:
			return s.ThrustActiveDir&ThrustActiveForward != 0
		case ThrustDirBackward:
			return s.ThrustActiveDir&ThrustActiveBackward != 0
		case ThrustDirLeft:
			return s.ThrustActiveDir&ThrustActiveLeft != 0
		case ThrustDirRight:
			return s.ThrustActiveDir&ThrustActiveRight != 0
		}
		return false
	}
	var out []Part
	hasThruster := false
	for _, p := range s.Parts {
		if p.Kind() != PartThruster {
			continue
		}
		hasThruster = true
		if dirActive(p.ThrustDir()) {
			out = append(out, p)
		}
	}
	if hasThruster {
		return out
	}
	// 非常用推進: 前進方向のみ。
	if s.ThrustActiveDir&ThrustActiveForward == 0 {
		return nil
	}
	// 海賊機はコックピットパーツを非常用エミッタにする。
	// プレイヤー機（コックピット非搭載）の非常推進炎は drawAfterburners が
	// 船体後端から別途描く（ここでは何も返さない）。
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
	boost := s.ThrustState == ThrustBoost

	// ローカル → ワールド変換: 画像と同じ R(angle + π/2)
	// 結果は (lx, ly) → (-sin*lx - cos*ly, cos*lx - sin*ly)
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}

	// emitFlame は基点 (baseX, baseY)（スクリーン）から後方単位ベクトル (bx, by)・
	// 接線単位ベクトル (tx, ty) の向きへ炎を 1 つ描く。
	emitFlame := func(baseX, baseY, bx, by, tx, ty float64) {
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

		if boost {
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

	hasThruster := false
	for _, p := range s.Parts {
		if p.Kind() == PartThruster {
			hasThruster = true
			break
		}
	}

	for _, p := range s.thrustEmitters() {
		r := ((p.Rotation % 4) + 4) % 4
		// パーツ中心のローカル位置
		cxL, cyL := PartLocalCenter(p.GX, p.GY)
		// 回転前の「後端中心オフセット」と「後方ベクトル」: ローカル +y が後方。
		// 画像は ebiten の GeoM.Rotate(R*π/2) で回転するため、ローカル +y は
		// (x, y) → (-y, x) に従って回る。これは視覚的に CW 90°×R 回転に相当する。
		// (0, 1) → R=0:(0,1) R=1:(-1,0) R=2:(0,-1) R=3:(1,0)
		rxL, ryL := 0.0, 0.0
		switch r {
		case 0:
			rxL, ryL = 0, 1
		case 1:
			rxL, ryL = -1, 0
		case 2:
			rxL, ryL = 0, -1
		case 3:
			rxL, ryL = 1, 0
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
		emitFlame(sx+wox, sy+woy, bx, by, tx, ty)
	}

	// スラスタ未搭載で前進中は、ベースの非常推進炎を船体後端（ハル尾部）中央から出す。
	// 原点から出すと船体に埋もれるため、ハル後端まで基点を下げる。
	if !hasThruster && s.ThrustActiveDir&ThrustActiveForward != 0 {
		_, hHalf := shipHullExtent(s.GridHalf(), g)
		wox, woy := toWorld(0, hHalf*0.92) // ローカル +y がハル後端方向
		bx, by := toWorld(0, 1)            // 後方単位ベクトル
		tx, ty := toWorld(1, 0)            // 接線単位ベクトル
		emitFlame(sx+wox, sy+woy, bx, by, tx, ty)
	}
}

// shieldExpand はシールド輪郭をベース船体外形より広げる倍率。
const shieldExpand = 1.12

// DrawShieldOutline はシールド展開中、ベース船体の外周を一回り大きく囲む輪郭線を描く。
// 船体シルエットと相似のリングを theme.Line で 1 本足す形で「張られている」ことを示す。
// シールド HP が 1 以上のときに毎フレーム呼び出す想定。
func (s *Ship) DrawShieldOutline(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	sin, cos := math.Sin(s.Angle), math.Cos(s.Angle)
	// 船体描画と同じ R(angle + π/2) ローカル → ワールド変換。
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}
	wHalf, hHalf := shipHullExtent(s.GridHalf(), float64(GridSize))
	// ハル中心基準の外形ポリゴンを少し拡大して取る。ハル中心はピボットより
	// partPivotShiftY だけ前方にあるので、ローカル y をその分補正してから回す。
	pts := shipHullPolygon(0, 0, wHalf*shieldExpand, hHalf*shieldExpand)
	n := len(pts)
	scr := make([][2]float32, n)
	for i, p := range pts {
		wx, wy := toWorld(p[0], p[1]-partPivotShiftY)
		scr[i] = [2]float32{float32(sx + wx), float32(sy + wy)}
	}
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		vector.StrokeLine(dst, scr[i][0], scr[i][1], scr[j][0], scr[j][1], 1.5, theme.Line, true)
	}
}
