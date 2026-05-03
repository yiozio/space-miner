package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/entity"
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
type StationEditor struct {
	player    *entity.Player
	cursorGX  int
	cursorGY  int
	palette   []*entity.PartDef // 配置可能な全 def（一覧表示順）
	paletteIx int               // 現在選択中の palette インデックス
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
		d.Pop()
		return nil
	}

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

	// 数字 1-9 でパレット先頭9個を直接選択
	for i := 0; i < 9 && i < len(se.palette); i++ {
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			se.paletteIx = i
		}
	}
	// Q/E でパレット前後送り（バリアント数が多くても全件アクセス可能）
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) && len(se.palette) > 0 {
		se.paletteIx = (se.paletteIx - 1 + len(se.palette)) % len(se.palette)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) && len(se.palette) > 0 {
		se.paletteIx = (se.paletteIx + 1) % len(se.palette)
	}

	// 配置
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		se.tryPlace()
	}
	// 取り外し
	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		se.tryRemove()
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
func (se *StationEditor) tryPlace() {
	def := se.selectedDef()
	if def == nil {
		return
	}
	if se.player.PartsInventory[def.ID] <= 0 {
		return
	}
	if se.partAtCursor() >= 0 {
		return
	}
	se.player.Parts = append(se.player.Parts, entity.Part{
		DefID: def.ID,
		GX:    se.cursorGX,
		GY:    se.cursorGY,
	})
	se.player.PartsInventory[def.ID]--
	se.player.Ship.InvalidateImage()
}

// tryRemove はカーソル位置のパーツを取り外し、PartsInventory に戻す。
// コックピットは取り外せない。
func (se *StationEditor) tryRemove() {
	i := se.partAtCursor()
	if i < 0 {
		return
	}
	p := se.player.Parts[i]
	if p.Kind() == entity.PartCockpit {
		return
	}
	se.player.Parts = append(se.player.Parts[:i], se.player.Parts[i+1:]...)
	se.player.PartsInventory[p.DefID]++
	se.player.Ship.InvalidateImage()
}

func (se *StationEditor) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	header := "SHIP EDITOR"
	hw, _ := ui.MeasureText(header, headerScale)
	ui.DrawText(dst, header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	// レイアウト
	gridPx := float64(editorGridSize)*editorCellSize + float64(editorGridSize-1)*editorCellGap
	paletteW := 320.0
	gap := 60.0
	totalW := gridPx + gap + paletteW
	startX := (float64(sw) - totalW) / 2
	gridStartX := startX
	paletteX := gridStartX + gridPx + gap
	contentY := 100.0

	se.drawShipGrid(dst, theme, gridStartX, contentY)
	se.drawPalette(dst, theme, paletteX, contentY)
	se.drawCursorInfo(dst, theme, gridStartX, contentY+gridPx+24)

	ui.DrawText(dst,
		"[ WASD/Arrows: Move    1-9: Quick Select    Q/E: Cycle Palette    Space: Place    X: Remove    Esc: Back ]",
		20, float64(sh)-30, 1.3, theme.LineDim)
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
		entity.DrawPart(dst, part.Kind(), float32(cx), float32(cy), float32(cs), theme)
	}
	// カーソル
	cx, cy := gridCellPos(x, y, se.cursorGX, se.cursorGY)
	vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
}

func (se *StationEditor) drawPalette(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ui.DrawText(dst, "PARTS", x, y, 1.6, theme.Line)
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
		// 先頭9件は数字キーで直接選択可能
		idxLabel := "  "
		if i < 9 {
			idxLabel = fmt.Sprintf("%d ", i+1)
		}
		ui.DrawText(dst,
			fmt.Sprintf("%s%s%-15s x%d", prefix, idxLabel, def.Name, qty),
			x, lineY, 1.3, clr)
		lineY += 24
	}
}

func (se *StationEditor) drawCursorInfo(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	cursorText := fmt.Sprintf("Cursor (%d, %d)", se.cursorGX, se.cursorGY)
	ui.DrawText(dst, cursorText, x, y, 1.3, theme.LineDim)
	if i := se.partAtCursor(); i >= 0 {
		p := se.player.Parts[i]
		name := "?"
		if d := p.Def(); d != nil {
			name = d.Name
		}
		ui.DrawText(dst, "Cell: "+name, x, y+22, 1.3, theme.Line)
	} else {
		ui.DrawText(dst, "Cell: Empty", x, y+22, 1.3, theme.LineDim)
	}
	if def := se.selectedDef(); def != nil {
		ui.DrawText(dst,
			fmt.Sprintf("Selected: %s (x%d)", def.Name, se.player.PartsInventory[def.ID]),
			x, y+44, 1.3, theme.Line)
	}
}
