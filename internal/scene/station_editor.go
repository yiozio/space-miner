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
	player   *entity.Player
	cursorGX int
	cursorGY int
	selected entity.PartKind
}

// NewStationEditor は編集シーンを生成する。
// 既定で在庫のあるパーツを選択、無ければ Gun。
func NewStationEditor(p *entity.Player) *StationEditor {
	se := &StationEditor{player: p, selected: entity.PartGun}
	for _, kind := range entity.AllPlaceablePartKinds() {
		if p.PartsInventory[kind] > 0 {
			se.selected = kind
			break
		}
	}
	return se
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

	// 数字 1-8 でパレット選択
	placeable := entity.AllPlaceablePartKinds()
	for i, kind := range placeable {
		if i >= 9 {
			break
		}
		if inpututil.IsKeyJustPressed(ebiten.Key1 + ebiten.Key(i)) {
			se.selected = kind
		}
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
	if se.player.PartsInventory[se.selected] <= 0 {
		return
	}
	if se.partAtCursor() >= 0 {
		return
	}
	se.player.Parts = append(se.player.Parts, entity.Part{
		Kind: se.selected,
		GX:   se.cursorGX,
		GY:   se.cursorGY,
	})
	se.player.PartsInventory[se.selected]--
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
	if p.Kind == entity.PartCockpit {
		return
	}
	se.player.Parts = append(se.player.Parts[:i], se.player.Parts[i+1:]...)
	se.player.PartsInventory[p.Kind]++
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
	paletteW := 280.0
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
		"[ WASD/Arrows: Move    1-8: Select Part    Space: Place    X: Remove    Esc: Back ]",
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
		entity.DrawPart(dst, part, float32(cx), float32(cy), float32(cs), theme)
	}
	// カーソル
	cx, cy := gridCellPos(x, y, se.cursorGX, se.cursorGY)
	vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
}

func (se *StationEditor) drawPalette(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ui.DrawText(dst, "PARTS", x, y, 1.6, theme.Line)
	lineY := y + 32
	for i, kind := range entity.AllPlaceablePartKinds() {
		qty := se.player.PartsInventory[kind]
		clr := theme.Line
		if qty == 0 {
			clr = theme.LineDim
		}
		prefix := "  "
		if kind == se.selected {
			prefix = "> "
		}
		ui.DrawText(dst,
			fmt.Sprintf("%s%d %-9s x%d", prefix, i+1, entity.PartName(kind), qty),
			x, lineY, 1.4, clr)
		lineY += 26
	}
}

func (se *StationEditor) drawCursorInfo(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	cursorText := fmt.Sprintf("Cursor (%d, %d)", se.cursorGX, se.cursorGY)
	ui.DrawText(dst, cursorText, x, y, 1.3, theme.LineDim)
	if i := se.partAtCursor(); i >= 0 {
		p := se.player.Parts[i]
		ui.DrawText(dst, "Cell: "+entity.PartName(p.Kind), x, y+22, 1.3, theme.Line)
	} else {
		ui.DrawText(dst, "Cell: Empty", x, y+22, 1.3, theme.LineDim)
	}
	ui.DrawText(dst,
		fmt.Sprintf("Selected: %s (x%d)",
			entity.PartName(se.selected),
			se.player.PartsInventory[se.selected]),
		x, y+44, 1.3, theme.Line)
}
