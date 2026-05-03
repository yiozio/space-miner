package scene

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

// WorldMapView は最後に入った FullMap を画面いっぱいに俯瞰表示するシーン。
// スクロールせず、対象 FullMap 全域を画面に収まるようスケールする。
// ゾーン円・ステーション・自機現在位置を確認できる。
type WorldMapView struct {
	fmap        *entity.FullMap
	stations    []*entity.Station
	playerX     float64
	playerY     float64
	playerAngle float64 // 自機の向き（ワールド座標系。前方は (cos, sin)）
}

// NewWorldMapView は対象 FullMap・ステーション一覧・自機座標と向きを受け取り、
// 全体マップシーンを生成する。fmap が nil ならプレースホルダ表示。
func NewWorldMapView(fmap *entity.FullMap, stations []*entity.Station, px, py, angle float64) *WorldMapView {
	return &WorldMapView{fmap: fmap, stations: stations, playerX: px, playerY: py, playerAngle: angle}
}

func (w *WorldMapView) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyM) ||
		inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		d.Pop()
	}
	return nil
}

func (w *WorldMapView) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 下層シーンを覆い隠す不透明背景
	dst.Fill(theme.Background)

	// ヘッダ
	header := "WORLD MAP"
	headerScale := 2.5
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := 20.0
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	hint := "[ M / Tab / Esc: Close ]"
	hintY := float64(sh) - 30
	ui.DrawText(dst, hint, 20, hintY, 1.5, theme.LineDim)

	if w.fmap == nil {
		ui.DrawText(dst, "(NOT IN ANY FULL MAP)", 20, headerY+hh+20, 1.5, theme.LineDim)
		return
	}

	// マップ表示領域：ヘッダの下から操作ヒントの上まで、左右に余白
	margin := 40.0
	topY := headerY + hh + 20
	botY := hintY - 20
	areaW := float64(sw) - margin*2
	areaH := botY - topY
	mapW := w.fmap.HalfW * 2
	mapH := w.fmap.HalfH * 2
	scale := math.Min(areaW/mapW, areaH/mapH)

	cx := margin + areaW/2
	cy := topY + areaH/2
	toScreen := func(wx, wy float64) (float32, float32) {
		return float32(cx + (wx-w.fmap.CX)*scale), float32(cy + (wy-w.fmap.CY)*scale)
	}

	// FullMap の外枠
	rectW := float32(mapW * scale)
	rectH := float32(mapH * scale)
	rectX := float32(cx) - rectW/2
	rectY := float32(cy) - rectH/2
	vector.StrokeRect(dst, rectX, rectY, rectW, rectH, 1, theme.Line, false)

	// 中心十字（FullMap の中心マーク）
	ccx, ccy := toScreen(w.fmap.CX, w.fmap.CY)
	cross := float32(8)
	vector.StrokeLine(dst, ccx-cross, ccy, ccx+cross, ccy, 1, theme.LineDim, false)
	vector.StrokeLine(dst, ccx, ccy-cross, ccx, ccy+cross, 1, theme.LineDim, false)

	// 素材ゾーン
	for i := range w.fmap.Zones {
		z := &w.fmap.Zones[i]
		zsx, zsy := toScreen(z.CX, z.CY)
		zr := float32(z.Radius * scale)
		// 代表色（最大重みの素材色）でゾーン円を描く
		vector.StrokeCircle(dst, zsx, zsy, zr, 1, zoneDominantColor(z), false)
		// ゾーン名（素材を + で連結）。長辺が画面に収まるよう、円中心に重ねて表示
		label := zoneLabel(z)
		lw, lh := ui.MeasureText(label, 1.2)
		ui.DrawText(dst, label, float64(zsx)-lw/2, float64(zsy)-lh/2, 1.2, theme.Line)
		// 素材ごとの色サンプル（小さな塗り四角を円の下にライン状に並べる）
		drawZoneSwatches(dst, zsx, zsy+float32(lh)/2+4, z)
	}

	// ステーション（対象 FullMap 内のもののみ）
	for _, s := range w.stations {
		if !w.fmap.Contains(s.X, s.Y) {
			continue
		}
		sx, sy := toScreen(s.X, s.Y)
		drawDiamond(dst, sx, sy, 6, 1.5, theme.Line)
		ui.DrawText(dst, "STATION", float64(sx)+10, float64(sy)-6, 1.0, theme.LineDim)
	}

	// 自機現在位置（FullMap 内のときのみ）。向きを示す二等辺三角形で描画する
	if w.fmap.Contains(w.playerX, w.playerY) {
		px, py := toScreen(w.playerX, w.playerY)
		drawPlayerArrow(dst, px, py, w.playerAngle, theme.Line)
	}
}

// drawPlayerArrow は (cx, cy) を中心とする二等辺三角形を、
// angle 方向（前方 = (cos, sin)）を向くように描画する。
// マップ用の小さなマーカー。サイズは固定で、マップスケールに依存しない。
func drawPlayerArrow(dst *ebiten.Image, cx, cy float32, angle float64, c color.Color) {
	const tip = float32(9)  // 中心から先端までの距離
	const back = float32(6) // 中心から底辺までの距離
	const half = float32(5) // 底辺の半幅
	sin, cos := float32(math.Sin(angle)), float32(math.Cos(angle))
	// ローカル → 画面: (lx, ly) → (cx + lx*cos - ly*sin, cy + lx*sin + ly*cos)
	rotate := func(lx, ly float32) (float32, float32) {
		return cx + lx*cos - ly*sin, cy + lx*sin + ly*cos
	}
	tipX, tipY := rotate(tip, 0)
	leftX, leftY := rotate(-back, -half)
	rightX, rightY := rotate(-back, half)
	const sw = float32(1.5)
	vector.StrokeLine(dst, tipX, tipY, leftX, leftY, sw, c, false)
	vector.StrokeLine(dst, leftX, leftY, rightX, rightY, sw, c, false)
	vector.StrokeLine(dst, rightX, rightY, tipX, tipY, sw, c, false)
}

// zoneDominantColor はゾーンの代表色（重みが最大の素材色）を返す。
func zoneDominantColor(z *entity.ResourceZone) color.Color {
	var best entity.ResourceType
	bestW := -1.0
	for _, m := range z.Mix {
		if m.Weight > bestW {
			bestW = m.Weight
			best = m.Resource
		}
	}
	return best.Info().Color
}

// zoneLabel はゾーン名「IRON」「IRON+ICE」のように構成素材を列挙する。
func zoneLabel(z *entity.ResourceZone) string {
	s := ""
	for i, m := range z.Mix {
		if i > 0 {
			s += "+"
		}
		s += m.Resource.Info().Name
	}
	return s
}

// drawZoneSwatches は素材ごとの色サンプルを (cx, top) を上端中央として横一列に並べる。
func drawZoneSwatches(dst *ebiten.Image, cx, top float32, z *entity.ResourceZone) {
	const sz = float32(6)
	const gap = float32(2)
	total := float32(len(z.Mix))*sz + float32(len(z.Mix)-1)*gap
	x := cx - total/2
	for _, m := range z.Mix {
		vector.DrawFilledRect(dst, x, top, sz, sz, m.Resource.Info().Color, false)
		x += sz + gap
	}
}

// drawDiamond は中心 (cx, cy)・半径 r のひし形をライン描画する。
func drawDiamond(dst *ebiten.Image, cx, cy, r, sw float32, c color.Color) {
	vector.StrokeLine(dst, cx, cy-r, cx+r, cy, sw, c, false)
	vector.StrokeLine(dst, cx+r, cy, cx, cy+r, sw, c, false)
	vector.StrokeLine(dst, cx, cy+r, cx-r, cy, sw, c, false)
	vector.StrokeLine(dst, cx-r, cy, cx, cy-r, sw, c, false)
}
