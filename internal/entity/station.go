package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

const (
	StationBodyRadius = 80
	StationDockRange  = 60 // ドック中心からこの距離以内なら接岸可能
	stationDockOffset = StationBodyRadius + 50
)

// Station は宇宙ステーション。当たり判定はなく、背景として存在するのみ。
// 本体の東側にドック ⊃ を持ち、自機が近づくとアクション可能になる。
// Name は所属する FullMap 名と一致させ、初回入船ダイアログの判別キーとして用いる。
type Station struct {
	Name       string
	X, Y       float64
	pulseFrame int
}

// NewStation は (x, y) に名前付きステーションを生成する。
func NewStation(name string, x, y float64) *Station {
	return &Station{Name: name, X: x, Y: y}
}

// DockX, DockY はドック中心のワールド座標。
func (s *Station) DockX() float64 { return s.X + stationDockOffset }
func (s *Station) DockY() float64 { return s.Y }

// IsPlayerInDock は (px, py) がドック判定範囲内にあるかを返す。
func (s *Station) IsPlayerInDock(px, py float64) bool {
	dx := px - s.DockX()
	dy := py - s.DockY()
	return math.Hypot(dx, dy) < StationDockRange
}

// Update はパルス用フレームカウンタを進める。
func (s *Station) Update() { s.pulseFrame++ }

// Draw は (sx, sy) をステーション中心としてベクター線画で描画する。
// ドック部分はライン色を太く、内側にパルスする補助色を重ねて目立たせる。
func (s *Station) Draw(dst *ebiten.Image, sx, sy float64, theme *ui.Theme) {
	bodyR := float32(StationBodyRadius)
	innerR := bodyR * 0.55
	cx, cy := float32(sx), float32(sy)

	// 本体 (六角)
	drawPolygon(dst, cx, cy, bodyR, 6, 0, 2, theme.Line)
	// 内側 (回転を変えた六角、薄色)
	drawPolygon(dst, cx, cy, innerR, 6, math.Pi/6, 1, theme.LineDim)
	// 中央コアマーカー
	vector.StrokeRect(dst, cx-3, cy-3, 6, 6, 1, theme.LineDim, false)

	// 連結アーム (本体 → ドック)
	armOffsetY := float32(14)
	dockX := cx + float32(stationDockOffset)
	dockY := cy
	vector.StrokeLine(dst, cx+bodyR*0.7, dockY-armOffsetY, dockX-17, dockY-armOffsetY, 1, theme.Line, false)
	vector.StrokeLine(dst, cx+bodyR*0.7, dockY+armOffsetY, dockX-17, dockY+armOffsetY, 1, theme.Line, false)

	// ドックブラケット ⊃ (左側オープン)
	bw := float32(34)
	bh := float32(48)
	bx := dockX - bw/2
	by := dockY - bh/2
	vector.StrokeLine(dst, bx, by, bx+bw, by, 3, theme.Line, false)
	vector.StrokeLine(dst, bx+bw, by, bx+bw, by+bh, 3, theme.Line, false)
	vector.StrokeLine(dst, bx+bw, by+bh, bx, by+bh, 3, theme.Line, false)

	// ドック内側のパルス枠
	pulse := 0.5 + 0.5*math.Sin(float64(s.pulseFrame)*0.08)
	pad := float32(4) + float32(pulse*6)
	c := theme.Line
	c.A = uint8(80 + pulse*120)
	vector.StrokeRect(dst, bx+pad, by+pad, bw-pad*2, bh-pad*2, 1, c, false)

	// 入口を示す ▶
	axCenter := bx
	axW := float32(12)
	axH := float32(10)
	vector.StrokeLine(dst, axCenter-axW, dockY-axH, axCenter, dockY, 2, theme.Line, false)
	vector.StrokeLine(dst, axCenter-axW, dockY+axH, axCenter, dockY, 2, theme.Line, false)
}

// drawPolygon は中心 (cx, cy)、半径 r、頂点数 n の正多角形を描画する。
// rotation は最初の頂点の角度（ラジアン）。
func drawPolygon(dst *ebiten.Image, cx, cy, r float32, n int, rotation float64, strokeWidth float32, c color.Color) {
	if n < 3 {
		return
	}
	prevX := cx + float32(math.Cos(rotation))*r
	prevY := cy + float32(math.Sin(rotation))*r
	for i := 1; i <= n; i++ {
		ang := rotation + float64(i)*2*math.Pi/float64(n)
		nx := cx + float32(math.Cos(ang))*r
		ny := cy + float32(math.Sin(ang))*r
		vector.StrokeLine(dst, prevX, prevY, nx, ny, strokeWidth, c, false)
		prevX, prevY = nx, ny
	}
}
