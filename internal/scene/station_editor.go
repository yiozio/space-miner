package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	editorGridHalf = 3                    // -3..+3
	editorGridSize = editorGridHalf*2 + 1 // 7
	editorCellSize = 50.0
	editorCellGap  = 2.0
)

// StationEditor は宇宙船編集画面。
// 7×7 のグリッドにカーソルを置き、左パレットで選んだパーツを設置・取り外しできる。
// コックピットは原点 (0, 0) に固定（編集対象外）。スペアパーツは Player.PartsInventory。
// brushRotation は新規配置時に適用される向き（0..3、90° 単位 CW）。R キーで循環。
type StationEditor struct {
	player        *entity.Player
	cursorGX      int
	cursorGY      int
	palette       []*entity.PartDef // 配置可能な全 def（一覧表示順）
	paletteIx     int               // 現在選択中の palette インデックス
	brushRotation int               // 0..3
}

// NewStationEditor は編集シーンを生成する。
// 既定で在庫のあるパーツを選択、無ければ palette 先頭。
func NewStationEditor(p *entity.Player) *StationEditor {
	se := &StationEditor{
		player:  p,
		palette: entity.AllPlaceablePartDefs(),
	}
	for i, def := range se.palette {
		if p.PartsInventory[def.ID] > 0 {
			se.paletteIx = i
			break
		}
	}
	return se
}

// selectedDef は現在選択中の PartDef を返す。palette が空のときは nil。
func (se *StationEditor) selectedDef() *entity.PartDef {
	if len(se.palette) == 0 {
		return nil
	}
	return se.palette[se.paletteIx]
}

func (se *StationEditor) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}

	oldGX, oldGY, oldPal := se.cursorGX, se.cursorGY, se.paletteIx

	// カーソル移動
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if se.cursorGY > -editorGridHalf {
			se.cursorGY--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if se.cursorGY < editorGridHalf {
			se.cursorGY++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if se.cursorGX > -editorGridHalf {
			se.cursorGX--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if se.cursorGX < editorGridHalf {
			se.cursorGX++
		}
	}

	// Q/E でパレット前後送り（バリアント数が多くても全件アクセス可能）
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) && len(se.palette) > 0 {
		se.paletteIx = (se.paletteIx - 1 + len(se.palette)) % len(se.palette)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) && len(se.palette) > 0 {
		se.paletteIx = (se.paletteIx + 1) % len(se.palette)
	}
	// カーソルかパレット選択が動いたら移動音。
	if se.cursorGX != oldGX || se.cursorGY != oldGY || se.paletteIx != oldPal {
		sound.PlayMenuMove()
	}

	// 回転: R キー
	//   - カーソル上に配置済みパーツがあればそれを 90° 回転
	//   - 無ければブラシ回転（次回設置時の向き）を循環
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if i := se.partAtCursor(); i >= 0 {
			if se.player.Parts[i].Kind() != entity.PartCockpit {
				se.player.Parts[i].Rotation = (se.player.Parts[i].Rotation + 1) % 4
				se.player.OnPartsChanged()
			}
		} else {
			se.brushRotation = (se.brushRotation + 1) % 4
		}
	}

	// 配置
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if se.tryPlace() {
			sound.PlayMenuSelect()
		}
	}
	// 取り外し
	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if se.tryRemove() {
			sound.PlayMenuSelect()
		}
	}
	return nil
}

// partAtCursor はカーソル位置のパーツのインデックスを返す（無ければ -1）。
func (se *StationEditor) partAtCursor() int {
	for i, p := range se.player.Parts {
		if p.GX == se.cursorGX && p.GY == se.cursorGY {
			return i
		}
	}
	return -1
}

// tryPlace はカーソル位置が空かつ選択パーツの在庫があれば設置する。
// tryPlace は配置に成功したら true を返す。
func (se *StationEditor) tryPlace() bool {
	def := se.selectedDef()
	if def == nil {
		return false
	}
	if se.player.PartsInventory[def.ID] <= 0 {
		return false
	}
	if se.partAtCursor() >= 0 {
		return false
	}
	se.player.Parts = append(se.player.Parts, entity.Part{
		DefID:    def.ID,
		GX:       se.cursorGX,
		GY:       se.cursorGY,
		Rotation: se.brushRotation,
	})
	se.player.PartsInventory[def.ID]--
	se.player.OnPartsChanged()
	return true
}

// tryRemove はカーソル位置のパーツを取り外し、PartsInventory に戻す。
// コックピットは取り外せない。取り外しに成功したら true を返す。
func (se *StationEditor) tryRemove() bool {
	i := se.partAtCursor()
	if i < 0 {
		return false
	}
	p := se.player.Parts[i]
	if p.Kind() == entity.PartCockpit {
		return false
	}
	se.player.Parts = append(se.player.Parts[:i], se.player.Parts[i+1:]...)
	se.player.PartsInventory[p.DefID]++
	se.player.OnPartsChanged()
	return true
}

func (se *StationEditor) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	ed := i18n.S().Editor
	hw, _ := ui.MeasureText(ed.Header, headerScale)
	ui.DrawText(dst, ed.Header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	// レイアウト
	gridPx := float64(editorGridSize)*editorCellSize + float64(editorGridSize-1)*editorCellGap
	paletteW := 320.0
	gap := 60.0
	totalW := gridPx + gap + paletteW
	startX := (float64(sw) - totalW) / 2
	gridStartX := startX
	paletteX := gridStartX + gridPx + gap
	contentY := 100.0

	se.drawStats(dst, theme, 24, contentY) // グリッド左の空きに機体性能
	se.drawShipGrid(dst, theme, gridStartX, contentY)
	se.drawPalette(dst, theme, paletteX, contentY)
	se.drawCursorInfo(dst, theme, gridStartX, contentY+gridPx+24)

	ui.DrawText(dst, ed.Hint, 20, float64(sh)-30, 1.3, theme.LineDim)
}

// gridCellPos はグリッド (gx, gy) の左上スクリーン座標を返す。
func gridCellPos(originX, originY float64, gx, gy int) (float64, float64) {
	col := gx + editorGridHalf
	row := gy + editorGridHalf
	return originX + float64(col)*(editorCellSize+editorCellGap),
		originY + float64(row)*(editorCellSize+editorCellGap)
}

func (se *StationEditor) drawShipGrid(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	cs := float64(editorCellSize)
	// 背景セル
	for gy := -editorGridHalf; gy <= editorGridHalf; gy++ {
		for gx := -editorGridHalf; gx <= editorGridHalf; gx++ {
			cx, cy := gridCellPos(x, y, gx, gy)
			vector.StrokeRect(dst, float32(cx), float32(cy), float32(cs), float32(cs), 1, theme.LineDim, false)
			if gx == 0 && gy == 0 {
				// 原点（コックピット位置）を薄く強調
				vector.StrokeRect(dst, float32(cx+2), float32(cy+2), float32(cs-4), float32(cs-4), 1, theme.Line, false)
			}
		}
	}
	// 配置済みパーツ
	for _, part := range se.player.Parts {
		cx, cy := gridCellPos(x, y, part.GX, part.GY)
		entity.DrawPart(dst, part.Kind(), float32(cx), float32(cy), float32(cs), theme, part.Rotation)
	}
	// カーソル
	cx, cy := gridCellPos(x, y, se.cursorGX, se.cursorGY)
	vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
}

func (se *StationEditor) drawPalette(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ed := i18n.S().Editor
	ui.DrawText(dst, ed.PartsHeader, x, y, 1.6, theme.Line)
	lineY := y + 32
	for i, def := range se.palette {
		qty := se.player.PartsInventory[def.ID]
		clr := theme.Line
		if qty == 0 {
			clr = theme.LineDim
		}
		prefix := "  "
		if i == se.paletteIx {
			prefix = "> "
		}
		ui.DrawText(dst,
			fmt.Sprintf(ed.PaletteRowFmt, prefix, i18n.PartName(def.ID), qty),
			x, lineY, 1.3, clr)
		lineY += 24
	}
}

func (se *StationEditor) drawCursorInfo(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ed := i18n.S().Editor
	cursorText := fmt.Sprintf(ed.CursorPosFmt, se.cursorGX, se.cursorGY)
	ui.DrawText(dst, cursorText, x, y, 1.3, theme.LineDim)
	if i := se.partAtCursor(); i >= 0 {
		p := se.player.Parts[i]
		name := i18n.PartName(p.DefID)
		if name == "" {
			name = "?"
		}
		ui.DrawText(dst, fmt.Sprintf(ed.CellLabel, name, rotationLabel(p.Rotation)), x, y+22, 1.3, theme.Line)
	} else {
		ui.DrawText(dst, ed.CellEmpty, x, y+22, 1.3, theme.LineDim)
	}
	if def := se.selectedDef(); def != nil {
		ui.DrawText(dst,
			fmt.Sprintf(ed.BrushLabel,
				i18n.PartName(def.ID), se.player.PartsInventory[def.ID], rotationLabel(se.brushRotation)),
			x, y+44, 1.3, theme.Line)
	}
}

// drawStats はグリッド左に機体性能を表示する。使われていない項目は出さない。
func (se *StationEditor) drawStats(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ed := i18n.S().Editor
	st := se.player.Stats()
	const (
		scale = 1.3
		line  = 24.0
		valX  = 100.0 // 値の左端（ラベルと揃える）
	)
	ui.DrawText(dst, ed.StatsHeader, x, y, 1.6, theme.Line)
	cy := y + 32
	put := func(label, val string) {
		ui.DrawText(dst, label, x, cy, scale, theme.Line)
		ui.DrawText(dst, val, x+valX, cy, scale, theme.Line)
		cy += line
	}
	if st.TotalDPS > 0 {
		put(ed.StatFirepower, fmt.Sprintf("%.1f", st.TotalDPS))
	}
	put(ed.StatHull, fmt.Sprintf("%d", st.MaxHP))
	if st.MaxShield > 0 {
		put(ed.StatShield, fmt.Sprintf("%d", st.MaxShield))
	}

	// 速度（方向別）。ブースト列は燃料がある（ブースト可能な）ときだけ。
	boostable := st.MaxFuel > 0
	cy += 8
	hdr := ed.StatSpeed + " (" + ed.StatMax
	if boostable {
		hdr += " / " + ed.StatBoost
	}
	hdr += ")"
	ui.DrawText(dst, hdr, x, cy, scale, theme.LineDim)
	cy += line
	dir := func(label string, ds entity.DirSpeed) {
		if !ds.Active {
			return
		}
		val := fmt.Sprintf("%.1f", ds.Max)
		if boostable {
			val += fmt.Sprintf(" / %.1f", ds.Boost)
		}
		ui.DrawText(dst, label, x, cy, scale, theme.Line)
		ui.DrawText(dst, val, x+valX, cy, scale, theme.Line)
		cy += line
	}
	dir(ed.DirForward, st.Fwd)
	dir(ed.DirBackward, st.Bck)
	dir(ed.DirRight, st.Rgt)
	dir(ed.DirLeft, st.Lft)
}

// rotationLabel は回転値を人間可読な文字列にする。
func rotationLabel(r int) string {
	switch ((r % 4) + 4) % 4 {
	case 0:
		return "0°"
	case 1:
		return "90°"
	case 2:
		return "180°"
	case 3:
		return "270°"
	}
	return ""
}
