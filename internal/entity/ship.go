package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	assetimage "github.com/yiozio/space-miner/internal/asset/image"
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
	BaseID          ShipBaseID // 機体ベース（土台）。グリッドサイズ・基礎ステータスを決める。プレイヤー・海賊機の双方で使う。
	X, Y            float64    // ワールド座標
	VX, VY          float64
	Angle           float64 // 機体の向き（ラジアン）。0 で +x（右）、CW 増加。
	ThrustState     ThrustState
	ThrustActiveDir ThrustActiveDir // 現在炎を出しているスラスタ方向（Forward/Backward）

	// Trail は直近の通過位置（古い順、ワールド座標）。描画側が尾を引く軌跡に使う。
	Trail []TrailPoint

	// LineColor はライン描画色の上書き。ゼロ値（A==0）なら theme.Line を使う。
	// 海賊機など、テーマと別色で機体を描き分けるために設定する。
	// スプライト描画では色乗算（ColorScale）として機体全体に適用する。
	LineColor color.NRGBA

	// 推進炎アニメーション状態（TickThrustAnim が毎フレーム更新）。
	AnimTick      int             // 炎フレーム送り用の汎用カウンタ
	lastActiveDir ThrustActiveDir // 直近で炎を出していた方向（消火フレーム表示に使う）
	flameOffTimer int             // >0 の間、消火フレーム（炎消）を表示する残フレーム

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

// HullColliders はベース船体を近似する衝突円を、ピボット基準のローカル座標で返す
// （各要素は {lx, ly, radius}）。当たり判定をベース船体の外形に合わせるために使う。
// 中心線上に数個の円を並べ、各 y 位置でのハル半幅を半径とする（縦長のカプセル状近似）。
func (s *Ship) HullColliders() [][3]float64 {
	wHalf, hHalf := shipHullExtent(s.GridHalf(), float64(GridSize))
	// t: ハル縦位置（-1=機首, +1=機尾）, wf: その位置のハル半幅 / wHalf
	samples := [...]struct{ t, wf float64 }{
		{-0.55, 0.50},
		{0.12, 1.00},
		{0.50, 0.80},
		{0.85, 0.55},
	}
	out := make([][3]float64, len(samples))
	for i, smp := range samples {
		// ハル中心はピボットより partPivotShiftY だけ前方にあるので y を補正する。
		out[i] = [3]float64{0, smp.t*hHalf - partPivotShiftY, smp.wf * wHalf}
	}
	return out
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
	// 画像はベース船体スプライト（回転を含む）を余裕をもって収める正方形。
	// グリッド中心からスプライト四隅までの最大距離 = 回転に耐える半径。
	bw, bh := assetimage.ShipBaseSize()
	scale := g / float64(assetimage.CellSize)
	span := float64(2*gridHalf+1) * float64(assetimage.CellSize)
	gcx := float64(assetimage.ShipBaseGridX) + span/2
	gcy := float64(assetimage.ShipBaseGridY) + span/2
	maxd := 0.0
	for _, c := range [4][2]float64{{0, 0}, {float64(bw), 0}, {0, float64(bh)}, {float64(bw), float64(bh)}} {
		if d := math.Hypot(c[0]-gcx, c[1]-gcy); d > maxd {
			maxd = d
		}
	}
	imgHalf := math.Ceil(math.Max(maxd*scale, (float64(gridHalf)+0.5)*g) + 4)
	size := int(imgHalf * 2)
	center := float64(size) / 2
	img := ebiten.NewImage(size, size)
	// ベース船体スプライトを先に敷き、その上にパーツスプライトを重ねる。
	// スラスタはアイドルのセルで焼き込み、点火スプライト・炎は drawThrust が手前に重ねる。
	DrawShipBase(img, center, center, gridHalf, g, theme)
	for _, p := range s.Parts {
		x := center + float64(p.GX)*g - g/2
		y := center + float64(p.GY)*g - g/2
		DrawPart(img, p.Kind(), float32(x), float32(y), float32(GridSize), theme, p.Rotation)
	}
	s.image = img
	// ピボット（回転中心）は従来どおり原点セルの中心（=グリッド中心）から
	// partPivotShiftY だけ後方に置く。PartLocalCenter・当たり判定・発射計算は変更不要。
	s.imageOffsetX = center
	s.imageOffsetY = center + partPivotShiftY
	s.cachedTheme = theme
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
	// 海賊機などは LineColor を色乗算として機体全体に適用する（赤系の色分け）。
	if s.LineColor.A != 0 {
		op.ColorScale.ScaleWithColor(s.LineColor)
	}
	dst.DrawImage(s.image, op)
	// 推進炎・点火スラスタは不透明なベース船体に隠れないよう、船体画像の手前に描く。
	// 点火中は炎アニメ、停止直後は消火フレームを一瞬表示する。
	if s.ThrustState != ThrustOff {
		s.drawThrust(dst, sx, sy, false)
	} else if s.flameOffTimer > 0 {
		s.drawThrust(dst, sx, sy, true)
	}
}

// 炎アニメーションの調整値（px は元画像＝16x16 基準）。
const (
	flameHoldFrames     = 3 // 1 フレームを何 tick 表示するか（小→中→大→中の送り速度）
	flameOffFrames      = 6 // 停止直後に消火フレーム（炎消）を出す長さ
	flameAttachPx       = 3 // 炎を噴射口へ食い込ませる量（半透明エッジ 2px + さらに 1px）
	defaultThrustDropPx = 7 // 非常用（デフォルト）スラスタの炎を下げる量
)

// flameAnimCol は AnimTick から炎フレームのシート列（炎大=0/中=1/小=2）を返す。
// 点火アニメは 小→中→大→中 のループ（= 体感 小中大中小中大…）。
func flameAnimCol(tick int) int {
	seq := [4]int{2, 1, 0, 1} // 小, 中, 大, 中
	return seq[(tick/flameHoldFrames)%4]
}

// TickThrustAnim は炎アニメ状態を 1 フレーム進める（Player/Pirate の Update から毎フレーム呼ぶ）。
// 点火中は炎送りカウンタを進め、直近の噴射方向を記録し消火タイマーを張り直す。
// 停止中は消火タイマーを減らし、0 になるまで消火フレームを描けるようにする。
func (s *Ship) TickThrustAnim() {
	s.AnimTick++
	if s.ThrustState != ThrustOff {
		s.lastActiveDir = s.ThrustActiveDir
		s.flameOffTimer = flameOffFrames
	} else if s.flameOffTimer > 0 {
		s.flameOffTimer--
	}
}

// thrustEmittersFor は dir のビットに対応する向きのスラスタ（炎を出すパーツ）を返す。
// Thruster が 1 つも無い場合は Cockpit を非常用エミッタとして返す（前進方向のみ）。
// プレイヤー機（コックピット非搭載）の非常推進は drawThrust が船体後端から別途描く。
func (s *Ship) thrustEmittersFor(dir ThrustActiveDir) []Part {
	dirActive := func(d ThrustDir) bool {
		switch d {
		case ThrustDirForward:
			return dir&ThrustActiveForward != 0
		case ThrustDirBackward:
			return dir&ThrustActiveBackward != 0
		case ThrustDirLeft:
			return dir&ThrustActiveLeft != 0
		case ThrustDirRight:
			return dir&ThrustActiveRight != 0
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
	if dir&ThrustActiveForward == 0 {
		return nil
	}
	for _, p := range s.Parts {
		if p.Kind() == PartCockpit {
			return []Part{p}
		}
	}
	return nil
}

// drawThrust は点火中のスラスタへ点火スプライトと炎スプライトを重ねて描く。
// off=true のときは停止直後の消火フレーム（炎消）を直近の噴射方向に一瞬描く。
// 炎は元画像の上 2px が半透明なので、後方へ 2px ぶん食い込ませて噴射口に密着させる。
func (s *Ship) drawThrust(dst *ebiten.Image, sx, sy float64, off bool) {
	g := float64(GridSize)
	scale := g / float64(assetimage.CellSize)
	overlap := float64(flameAttachPx) * scale // 噴射口への食い込み（後方へ詰める）
	sin, cos := math.Sin(s.Angle), math.Cos(s.Angle)
	toWorld := func(lx, ly float64) (float64, float64) {
		return -sin*lx - cos*ly, cos*lx - sin*ly
	}

	dir := s.ThrustActiveDir
	flameCol := flameAnimCol(s.AnimTick)
	flameScale := 1.0
	if s.ThrustState == ThrustBoost {
		flameScale = 1.4
	}
	drawFiring := true
	if off {
		dir = s.lastActiveDir
		flameCol = 3 // 炎消
		flameScale = 1.0
		drawFiring = false
	}
	if dir == 0 {
		return
	}
	flameImg := assetimage.Cell(flameCol, 0)
	firingImg := assetimage.Cell(2, 1) // スラスタ点火セル

	hasThruster := false
	for _, p := range s.Parts {
		if p.Kind() == PartThruster {
			hasThruster = true
			break
		}
	}

	for _, p := range s.thrustEmittersFor(dir) {
		r := ((p.Rotation % 4) + 4) % 4
		cxL, cyL := PartLocalCenter(p.GX, p.GY)
		// 後方単位ベクトル（ローカル）: (0,1) を CW 90°×R 回転（drawCellSprite と同じ規約）。
		var rxL, ryL float64
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
		rot := s.Angle + math.Pi/2 + float64(r)*(math.Pi/2)
		// 点火スラスタをアイドルの上に重ねる（スラスタパーツのときだけ）。
		if drawFiring && p.Kind() == PartThruster {
			wox, woy := toWorld(cxL, cyL)
			s.drawWorldSprite(dst, firingImg, sx+wox, sy+woy, g, rot)
		}
		// 炎: パーツ中心から 1 セル後方、さらに overlap ぶん噴射口へ詰める。
		fLx := cxL + rxL*(g-overlap)
		fLy := cyL + ryL*(g-overlap)
		wox, woy := toWorld(fLx, fLy)
		s.drawWorldSprite(dst, flameImg, sx+wox, sy+woy, g*flameScale, rot)
	}

	// スラスタ未搭載で前進中は、ベース内蔵スラスタ（底面中央）の非常推進炎を出す。
	// ベース絵の噴射口がやや下にあるため defaultThrustDropPx ぶん下げる。
	if !hasThruster && dir&ThrustActiveForward != 0 {
		_, hHalf := shipHullExtent(s.GridHalf(), g)
		wox, woy := toWorld(0, hHalf*0.92-overlap+float64(defaultThrustDropPx)*scale)
		s.drawWorldSprite(dst, flameImg, sx+wox, sy+woy, g*flameScale, s.Angle+math.Pi/2)
	}
}

// drawWorldSprite は 16x16 セル画像を cellSize に拡大し、スクリーン (scx, scy) を中心に
// rot（ラジアン CW）回転して描く。海賊機の色乗算（LineColor）も適用する。
func (s *Ship) drawWorldSprite(dst, sub *ebiten.Image, scx, scy, cellSize, rot float64) {
	scale := cellSize / float64(assetimage.CellSize)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(assetimage.CellSize)/2, -float64(assetimage.CellSize)/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Rotate(rot)
	op.GeoM.Translate(scx, scy)
	if s.LineColor.A != 0 {
		op.ColorScale.ScaleWithColor(s.LineColor)
	}
	dst.DrawImage(sub, op)
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
