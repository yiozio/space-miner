package scene

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

// StarMap は恒星系全体を俯瞰し、ワープ先の宇宙ステーションを選ぶシーン。
// 恒星は常に世界座標 (0, 0) と扱い、各 FullMap は CX/CY で配置する。
// 衛星は親惑星から線で結び、現在地と選択先をハイライトする。
//
// canWarp が false（Warp パーツ未搭載）の場合、表示はそのまま行うが
// Enter による確定は無効化され、画面下に未搭載である旨を表示する。
type StarMap struct {
	world      *entity.World
	currentMap string                                      // プレイヤーの現在 FullMap 名（ハイライト用）
	targets    []*entity.FullMap                           // ワープ可能なステーション（FullMap ごと）
	cursor     int                                         // targets のインデックス
	canWarp    bool                                        // Warp パーツ搭載時のみ確定可
	onWarp     func(d Director, dest *entity.FullMap) bool // 確定時コールバック。true で Pop する
}

// NewStarMap は恒星マップシーンを生成する。
//   - world:        対象となる恒星系
//   - currentMap:   プレイヤーが現在いる FullMap 名（区画外なら空文字）
//   - canWarp:      Warp パーツ搭載状態。false ならワープ確定は無効
//   - onWarp:       目的地確定時に呼ばれる。返り値 true で StarMap を閉じる
func NewStarMap(world *entity.World, currentMap string, canWarp bool, onWarp func(d Director, dest *entity.FullMap) bool) *StarMap {
	s := &StarMap{
		world:      world,
		currentMap: currentMap,
		canWarp:    canWarp,
		onWarp:     onWarp,
	}
	for i := range world.Maps {
		s.targets = append(s.targets, &world.Maps[i])
	}
	// 現在地が targets に含まれていれば、その次のスロットを初期選択に
	for i, t := range s.targets {
		if t.Name == currentMap {
			s.cursor = (i + 1) % len(s.targets)
			break
		}
	}
	return s
}

func (s *StarMap) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsKeyJustPressed(ebiten.KeyN) ||
		inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		d.Pop()
		return nil
	}
	if len(s.targets) == 0 {
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) ||
		inpututil.IsKeyJustPressed(ebiten.KeyS) ||
		inpututil.IsKeyJustPressed(ebiten.KeyD) {
		s.cursor = (s.cursor + 1) % len(s.targets)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyW) ||
		inpututil.IsKeyJustPressed(ebiten.KeyA) {
		s.cursor = (s.cursor - 1 + len(s.targets)) % len(s.targets)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if !s.canWarp {
			return nil // Warp パーツ未搭載: 確定は無視
		}
		dest := s.targets[s.cursor]
		if dest.Name == s.currentMap {
			return nil // 同地ワープは無効
		}
		if s.onWarp != nil && s.onWarp(d, dest) {
			d.Pop()
		}
	}
	return nil
}

func (s *StarMap) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	dst.Fill(theme.Background)

	// ヘッダ
	header := "STAR MAP"
	headerScale := 2.5
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := 20.0
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	hint := "[ Arrow: Select   Enter: Warp   N / Tab / Esc: Close ]"
	hintY := float64(sh) - 30
	ui.DrawText(dst, hint, 20, hintY, 1.4, theme.LineDim)

	if len(s.targets) == 0 {
		ui.DrawText(dst, "(no destinations available)", 20, headerY+hh+20, 1.4, theme.LineDim)
		return
	}

	// 描画領域
	margin := 80.0
	areaTop := headerY + hh + 40
	areaBot := hintY - 80 // 下部に詳細パネルを置くため余白
	areaLeft := margin
	areaRight := float64(sw) - margin
	areaCX := (areaLeft + areaRight) / 2
	areaCY := (areaTop + areaBot) / 2

	// 表示対象は世界座標。恒星は (0, 0)、各 FullMap は CX/CY。
	type bodyPos struct {
		body *entity.Celestial
		x, y float64
	}
	bodies := []bodyPos{{body: &s.world.Star, x: 0, y: 0}}
	for i := range s.world.Maps {
		bodies = append(bodies, bodyPos{
			body: &s.world.Maps[i].Body,
			x:    s.world.Maps[i].CX,
			y:    s.world.Maps[i].CY,
		})
	}

	// 範囲を計算してスケール
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, b := range bodies {
		if b.x < minX {
			minX = b.x
		}
		if b.x > maxX {
			maxX = b.x
		}
		if b.y < minY {
			minY = b.y
		}
		if b.y > maxY {
			maxY = b.y
		}
	}
	pad := 20000.0
	minX -= pad
	maxX += pad
	minY -= pad
	maxY += pad

	areaW := areaRight - areaLeft
	areaH := areaBot - areaTop
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX <= 0 {
		rangeX = 1
	}
	if rangeY <= 0 {
		rangeY = 1
	}
	scale := math.Min(areaW/rangeX, areaH/rangeY)
	systemCX := (minX + maxX) / 2
	systemCY := (minY + maxY) / 2

	toScreen := func(wx, wy float64) (float32, float32) {
		return float32(areaCX + (wx-systemCX)*scale),
			float32(areaCY + (wy-systemCY)*scale)
	}

	// FullMap 名 → ワールド位置（親惑星の参照に使う）
	worldPosOf := func(name string) (float64, float64, bool) {
		if s.world.Star.Name == name {
			return 0, 0, true
		}
		for i := range s.world.Maps {
			if s.world.Maps[i].Name == name {
				return s.world.Maps[i].CX, s.world.Maps[i].CY, true
			}
		}
		return 0, 0, false
	}

	// 衛星 → 親惑星の補助線
	for i := range s.world.Maps {
		body := &s.world.Maps[i].Body
		if body.ParentName == "" {
			continue
		}
		px, py, ok := worldPosOf(body.ParentName)
		if !ok {
			continue
		}
		x1, y1 := toScreen(px, py)
		x2, y2 := toScreen(s.world.Maps[i].CX, s.world.Maps[i].CY)
		vector.StrokeLine(dst, x1, y1, x2, y2, 1, theme.LineDim, false)
	}

	// 恒星
	starX, starY := toScreen(0, 0)
	drawStarMapBody(dst, &s.world.Star, starX, starY, theme, false, false)

	// 各 FullMap の Body
	for i := range s.world.Maps {
		body := &s.world.Maps[i].Body
		bx, by := toScreen(s.world.Maps[i].CX, s.world.Maps[i].CY)
		isCurrent := s.world.Maps[i].Name == s.currentMap
		isSelected := i == s.cursor
		drawStarMapBody(dst, body, bx, by, theme, isCurrent, isSelected)
	}

	// 下部の詳細パネル: 選択中ターゲットの情報
	dest := s.targets[s.cursor]
	infoY := areaBot + 16
	labelColor := theme.Line
	if !s.canWarp {
		labelColor = theme.LineDim
	}
	label := fmt.Sprintf("> %s  (%s)", dest.Name, kindLabel(dest.Body.Kind))
	if dest.Name == s.currentMap {
		label += "   [ CURRENT LOCATION ]"
	}
	ui.DrawText(dst, label, margin, infoY, 1.6, labelColor)
	zonesNote := fmt.Sprintf("ZONES %d", len(dest.Zones))
	ui.DrawText(dst, zonesNote, margin, infoY+30, 1.2, theme.LineDim)
	if !s.canWarp {
		warning := "WARP DRIVE NOT INSTALLED"
		ww, _ := ui.MeasureText(warning, 1.4)
		ui.DrawText(dst, warning, float64(sw)-margin-ww, infoY, 1.4, theme.Line)
	}
}

// drawStarMapBody は天体を恒星マップ上に描画する。
// current = プレイヤー現在地、selected = カーソル選択中。
func drawStarMapBody(dst *ebiten.Image, body *entity.Celestial, sx, sy float32, theme *ui.Theme, current, selected bool) {
	r := float32(body.Radius)
	// 本体（半透明塗り）
	fill := body.Color
	fill.A = 140
	vector.DrawFilledCircle(dst, sx, sy, r, fill, true)
	vector.StrokeCircle(dst, sx, sy, r, 1.5, body.Color, true)

	// 現在地マーカー（点線風の二重円）
	if current {
		vector.StrokeCircle(dst, sx, sy, r+8, 1, theme.Line, true)
	}
	// 選択カーソル
	if selected {
		vector.StrokeCircle(dst, sx, sy, r+14, 2, theme.Line, true)
		// 4 隅にティック
		tick := float32(4)
		vector.StrokeLine(dst, sx-r-14-tick, sy, sx-r-14+tick, sy, 2, theme.Line, true)
		vector.StrokeLine(dst, sx+r+14-tick, sy, sx+r+14+tick, sy, 2, theme.Line, true)
		vector.StrokeLine(dst, sx, sy-r-14-tick, sx, sy-r-14+tick, 2, theme.Line, true)
		vector.StrokeLine(dst, sx, sy+r+14-tick, sx, sy+r+14+tick, 2, theme.Line, true)
	}

	// ラベル
	label := body.Name
	lw, _ := ui.MeasureText(label, 1.2)
	ui.DrawText(dst, label, float64(sx)-lw/2, float64(sy)+float64(r)+8, 1.2, theme.Line)
}

// kindLabel は天体種別の表示名を返す。
func kindLabel(k entity.CelestialKind) string {
	switch k {
	case entity.CelestialStar:
		return "STAR"
	case entity.CelestialPlanet:
		return "PLANET"
	case entity.CelestialMoon:
		return "MOON"
	}
	return "?"
}
