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
	editorCellSize = 50.0
	editorCellGap  = 2.0

	editorPaletteRows = 13 // パレットの同時表示行数（見出し込みでグリッドの高さに収まる行数）

	// 収まらない名前を選択中に流すマーキーの速度（1文字進むフレーム数）と
	// 先頭位置で一時停止するステップ数。
	editorMarqueeStepFrames = 12
	editorMarqueeRestSteps  = 4
)

// StationEditor は宇宙船編集画面。
// 7×7 のグリッドにカーソルを置き、左パレットで選んだパーツを設置・取り外しできる。
// コックピットは原点 (0, 0) に固定（編集対象外）。スペアパーツは Player.PartsInventory。
type StationEditor struct {
	player        *entity.Player
	cursorGX      int
	cursorGY      int
	palette       []*entity.PartDef // 在庫のある def（一覧表示順）
	paletteIx     int               // 現在選択中の palette インデックス
	paletteScroll int               // パレットの先頭表示行（スクロール位置）
	marqueeTick   int               // 選択行のマーキー用カウンタ（選択が変わると 0 に戻る）
	notice        string            // 一時的な警告メッセージ（スラスタ必須など）
	noticeTimer   int               // notice の残り表示フレーム
}

// noticeDurationFrames は警告メッセージの表示時間（60fps で約 2.5 秒）。
const noticeDurationFrames = 150

// NewStationEditor は編集シーンを生成する。
func NewStationEditor(p *entity.Player) *StationEditor {
	se := &StationEditor{player: p}
	se.refreshPalette()
	return se
}

// refreshPalette は在庫のあるパーツだけでパレットを作り直す。
// 設置・取外で在庫が増減したときに呼ぶ。直前に選択していた def が残っていれば
// 選択を維持し、一覧から消えた場合は同じ位置に留まるようクランプする。
func (se *StationEditor) refreshPalette() {
	var selID entity.PartID
	hasSel := false
	if def := se.selectedDef(); def != nil {
		selID, hasSel = def.ID, true
	}
	se.palette = se.palette[:0]
	for _, def := range entity.AllPlaceablePartDefs() {
		if se.player.PartsInventory[def.ID] > 0 {
			se.palette = append(se.palette, def)
		}
	}
	ix := -1
	for i, def := range se.palette {
		if hasSel && def.ID == selID {
			ix = i
			break
		}
	}
	if ix >= 0 {
		se.paletteIx = ix
	} else {
		// 選択していた def が消えた → 別の行を指すのでマーキーを先頭に戻す
		se.marqueeTick = 0
		if se.paletteIx >= len(se.palette) {
			se.paletteIx = max(len(se.palette)-1, 0)
		}
	}
	se.ensurePaletteVisible()
}

// ensurePaletteVisible は選択中の行が表示範囲に入るようスクロール位置を追従させる。
func (se *StationEditor) ensurePaletteVisible() {
	if se.paletteIx < se.paletteScroll {
		se.paletteScroll = se.paletteIx
	}
	if se.paletteIx >= se.paletteScroll+editorPaletteRows {
		se.paletteScroll = se.paletteIx - editorPaletteRows + 1
	}
}

// selectedDef は現在選択中の PartDef を返す。palette が空のときは nil。
func (se *StationEditor) selectedDef() *entity.PartDef {
	if len(se.palette) == 0 {
		return nil
	}
	return se.palette[se.paletteIx]
}

// gridHalf は機体ベースの配置グリッド半径（3x3 なら 1）。
func (se *StationEditor) gridHalf() int { return se.player.GridHalf() }

// gridSize はグリッド一辺のセル数（3x3 なら 3）。
func (se *StationEditor) gridSize() int { return se.gridHalf()*2 + 1 }

func (se *StationEditor) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}

	if se.noticeTimer > 0 {
		se.noticeTimer--
	}

	oldGX, oldGY, oldPal := se.cursorGX, se.cursorGY, se.paletteIx

	// カーソル移動
	half := se.gridHalf()
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if se.cursorGY > -half {
			se.cursorGY--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if se.cursorGY < half {
			se.cursorGY++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if se.cursorGX > -half {
			se.cursorGX--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if se.cursorGX < half {
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
	se.ensurePaletteVisible()
	// カーソルかパレット選択が動いたら移動音。
	if se.cursorGX != oldGX || se.cursorGY != oldGY || se.paletteIx != oldPal {
		sound.PlayMenuMove()
	}
	// マーキーは選択が変わったら先頭から流し直す
	if se.paletteIx != oldPal {
		se.marqueeTick = 0
	} else {
		se.marqueeTick++
	}

	// 回転: R キーでカーソル上の配置済みパーツを 90° 回転
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if i := se.partAtCursor(); i >= 0 {
			se.player.Parts[i].Rotation = (se.player.Parts[i].Rotation + 1) % 4
			se.player.OnPartsChanged()
		}
	}

	// 配置
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if se.tryPlace() {
			sound.PlayMenuSelect()
			se.refreshPalette()
		}
	}
	// 取り外し
	if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if se.tryRemove() {
			sound.PlayMenuSelect()
			se.refreshPalette()
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
		DefID: def.ID,
		GX:    se.cursorGX,
		GY:    se.cursorGY,
	})
	se.player.PartsInventory[def.ID]--
	se.player.OnPartsChanged()
	return true
}

// tryRemove はカーソル位置のパーツを取り外し、PartsInventory に戻す。
// 機体は最低 1 基スラスタを積む規約のため、最後のスラスタは取り外せない（警告を出す）。
// 取り外しに成功したら true を返す。
func (se *StationEditor) tryRemove() bool {
	i := se.partAtCursor()
	if i < 0 {
		return false
	}
	p := se.player.Parts[i]
	// 最後の 1 基のスラスタは外せない。
	if p.Kind() == entity.PartThruster && se.thrusterCount() <= 1 {
		se.notice = i18n.S().Editor.NeedThruster
		se.noticeTimer = noticeDurationFrames
		sound.PlayMenuCancel()
		return false
	}
	se.player.Parts = append(se.player.Parts[:i], se.player.Parts[i+1:]...)
	se.player.PartsInventory[p.DefID]++
	se.player.OnPartsChanged()
	return true
}

// thrusterCount は現在搭載しているスラスタの数を返す。
func (se *StationEditor) thrusterCount() int {
	n := 0
	for _, p := range se.player.Parts {
		if p.Kind() == entity.PartThruster {
			n++
		}
	}
	return n
}

func (se *StationEditor) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	vector.FillRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	ed := i18n.S().Editor
	hw, hh := ui.MeasureText(ed.Header, headerScale)
	ui.DrawText(dst, ed.Header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	// レイアウト: ショップと同様にヘッダー直下へ整備士画像を大きく表示し、
	// その下へ画像と同じ横幅で「性能 | グリッド | パーツ」の3カラムを収める。
	// グリッドは画面中央、左右カラムの外端は画像の端に揃える。
	gridPx := float64(se.gridSize())*editorCellSize + float64(se.gridSize()-1)*editorCellGap
	gap := 24.0
	gridStartX := (float64(sw) - gridPx) / 2
	statsX := (float64(sw) - stationPortraitW) / 2
	paletteX := gridStartX + gridPx + gap
	paletteW := statsX + stationPortraitW - paletteX

	portraitY := 24 + hh + 8
	drawStationPortrait(dst, theme, "MECHANIC", sw, portraitY)

	// 画像の下に3カラムを配置する
	contentY := portraitY + stationPortraitH + 36

	se.drawStats(dst, theme, statsX, contentY)
	// カーソル情報（2行）は左カラムの下部に置き、グリッド下端と下揃えにする
	se.drawCursorInfo(dst, theme, statsX, contentY+gridPx-44)
	se.drawShipGrid(dst, theme, gridStartX, contentY)
	se.drawPalette(dst, theme, paletteX, contentY, paletteW)

	ui.DrawText(dst, ed.Hint, 20, float64(sh)-30, 1.3, theme.LineDim)

	// 一時的な警告（スラスタ必須など）はグリッド直下中央に目立つ色で出す。
	if se.noticeTimer > 0 && se.notice != "" {
		const noticeScale = 1.6
		nw, _ := ui.MeasureText(se.notice, noticeScale)
		ny := contentY + gridPx + 10
		ui.DrawText(dst, se.notice, (float64(sw)-nw)/2, ny, noticeScale,
			color.NRGBA{0xff, 0x80, 0x60, 0xff})
	}
}

// gridCellPos はグリッド (gx, gy) の左上スクリーン座標を返す。
func gridCellPos(originX, originY float64, gx, gy, gridHalf int) (float64, float64) {
	col := gx + gridHalf
	row := gy + gridHalf
	return originX + float64(col)*(editorCellSize+editorCellGap),
		originY + float64(row)*(editorCellSize+editorCellGap)
}

func (se *StationEditor) drawShipGrid(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	cs := float64(editorCellSize)
	half := se.gridHalf()
	gridPx := float64(se.gridSize())*editorCellSize + float64(se.gridSize()-1)*editorCellGap
	// グリッド背面にベース船体を描く（中心 = グリッド中心 = 原点セル中心）
	entity.DrawShipBase(dst, x+gridPx/2, y+gridPx/2, half, cs, theme)
	// 背景セル
	for gy := -half; gy <= half; gy++ {
		for gx := -half; gx <= half; gx++ {
			cx, cy := gridCellPos(x, y, gx, gy, half)
			vector.StrokeRect(dst, float32(cx), float32(cy), float32(cs), float32(cs), 1, theme.LineDim, false)
		}
	}
	// 配置済みパーツ
	for _, part := range se.player.Parts {
		cx, cy := gridCellPos(x, y, part.GX, part.GY, half)
		entity.DrawPart(dst, part.Kind(), float32(cx), float32(cy), float32(cs), theme, part.Rotation)
	}
	// カーソル
	cx, cy := gridCellPos(x, y, se.cursorGX, se.cursorGY, half)
	vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
}

// drawPalette は在庫パーツの一覧を幅 w のカラムに描く。
func (se *StationEditor) drawPalette(dst *ebiten.Image, theme *ui.Theme, x, y, w float64) {
	ed := i18n.S().Editor
	ui.DrawText(dst, ed.PartsHeader, x, y, 1.6, theme.Line)
	lineY := y + 32
	if len(se.palette) == 0 {
		ui.DrawText(dst, i18n.S().Common.Empty, x, lineY, 1.3, theme.LineDim)
		return
	}
	// 各行: 名前は左揃え、個数は右揃え。収まらない名前は "…" で省略し、
	// 選択中の行だけマーキーで 1 文字ずつ左へ流す。
	const scale = 1.3
	rowRight := x + w - 12 // 右端はスクロールバーの分を空ける
	prefixW, _ := ui.MeasureText("> ", scale)
	end := min(se.paletteScroll+editorPaletteRows, len(se.palette))
	for i := se.paletteScroll; i < end; i++ {
		def := se.palette[i]
		qty := fmt.Sprintf("x%d", se.player.PartsInventory[def.ID])
		qw, _ := ui.MeasureText(qty, scale)
		ui.DrawText(dst, qty, rowRight-qw, lineY, scale, theme.Line)

		selected := i == se.paletteIx
		if selected {
			ui.DrawText(dst, "> ", x, lineY, scale, theme.Line)
		}
		nameX := x + prefixW
		nameW := rowRight - qw - 8 - nameX
		name := i18n.PartName(def.ID)
		if selected {
			name = marqueeText(name, se.marqueeTick, scale, nameW)
		}
		ui.DrawText(dst, fitTextEllipsis(name, scale, nameW), nameX, lineY, scale, theme.Line)
		lineY += 24
	}

	// 表示しきれないときは右端にスクロールバーを描く
	total := len(se.palette)
	if total <= editorPaletteRows {
		return
	}
	trackX := x + w - 4
	trackY := y + 32
	trackH := float64(editorPaletteRows) * 24
	vector.StrokeRect(dst, float32(trackX), float32(trackY), 4, float32(trackH), 1, theme.LineDim, false)
	thumbH := max(trackH*float64(editorPaletteRows)/float64(total), 12)
	thumbY := trackY + (trackH-thumbH)*float64(se.paletteScroll)/float64(total-editorPaletteRows)
	vector.FillRect(dst, float32(trackX), float32(thumbY), 4, float32(thumbH), theme.Line, false)
}

// fitTextEllipsis は s が maxW に収まるならそのまま返し、
// 収まらない場合は末尾に "…" を付けて収まる長さまで切り詰める。
func fitTextEllipsis(s string, scale, maxW float64) string {
	if w, _ := ui.MeasureText(s, scale); w <= maxW {
		return s
	}
	r := []rune(s)
	for n := len(r) - 1; n > 0; n-- {
		t := string(r[:n]) + "…"
		if w, _ := ui.MeasureText(t, scale); w <= maxW {
			return t
		}
	}
	return "…"
}

// marqueeText は maxW に収まらない s を tick に応じて 1 文字ずつ左へ流した
// 部分文字列を返す。先頭位置でしばらく停止し、末尾まで見えたら先頭へ戻る。
// 収まる場合は s をそのまま返す。
func marqueeText(s string, tick int, scale, maxW float64) string {
	if w, _ := ui.MeasureText(s, scale); w <= maxW {
		return s
	}
	r := []rune(s)
	// 末尾まで表示できる最小のシフト量
	maxShift := 0
	for k := range r {
		if w, _ := ui.MeasureText(string(r[k:]), scale); w <= maxW {
			maxShift = k
			break
		}
	}
	cycle := editorMarqueeRestSteps + maxShift + 1
	pos := (tick / editorMarqueeStepFrames) % cycle
	shift := max(pos-editorMarqueeRestSteps, 0)
	return string(r[shift:])
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
