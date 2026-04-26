package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MenuItem はメニューの1項目。Values が nil なら通常項目（決定で選択）、
// 非 nil なら値選択項目（左右で値を切り替え）。
type MenuItem struct {
	Label      string
	Enabled    bool
	Values     []string
	ValueIndex int
}

// Menu は縦並びのメニュー。スタート/設定/メニュー画面で共通利用する。
type Menu struct {
	Items  []*MenuItem
	Cursor int
}

// MenuResult は1フレームでのメニュー操作結果。
type MenuResult struct {
	Activated    bool // 通常項目で決定キーが押された
	ValueChanged bool // 値選択項目で値が変更された
}

// Update はキー入力を処理しカーソル移動・値変更・決定を反映する。
func (m *Menu) Update() MenuResult {
	var r MenuResult
	if len(m.Items) == 0 {
		return r
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		m.moveCursor(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		m.moveCursor(1)
	}
	cur := m.Items[m.Cursor]
	if !cur.Enabled {
		return r
	}
	if len(cur.Values) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
			cur.ValueIndex = (cur.ValueIndex - 1 + len(cur.Values)) % len(cur.Values)
			r.ValueChanged = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
			cur.ValueIndex = (cur.ValueIndex + 1) % len(cur.Values)
			r.ValueChanged = true
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			r.Activated = true
		}
	}
	return r
}

// moveCursor はカーソルを delta 方向に動かし、
// 無効項目はスキップする（全項目無効なら現在位置のまま）。
func (m *Menu) moveCursor(delta int) {
	n := len(m.Items)
	for i := 0; i < n; i++ {
		m.Cursor = (m.Cursor + delta + n) % n
		if m.Items[m.Cursor].Enabled {
			return
		}
	}
}

// Draw はメニューを (x, y) を起点に縦並びで描画する。
// 中央寄せが必要な場合は呼び出し側で MeasureText を用いて x を計算する。
func (m *Menu) Draw(dst *ebiten.Image, t *Theme, x, y, scale float64) {
	lineHeight := 32.0 * scale
	paddingX := 12.0 * scale
	paddingY := 6.0 * scale
	for i, item := range m.Items {
		clr := t.Line
		if !item.Enabled {
			clr = t.LineDim
		}
		label := item.Label
		if len(item.Values) > 0 {
			label = label + "  < " + item.Values[item.ValueIndex] + " >"
		}
		ly := y + float64(i)*lineHeight
		DrawText(dst, label, x, ly, scale, clr)
		if i == m.Cursor {
			w, h := MeasureText(label, scale)
			vector.StrokeRect(dst,
				float32(x-paddingX), float32(ly-paddingY),
				float32(w+paddingX*2), float32(h+paddingY*2),
				1, t.Line, false)
		}
	}
}

// MaxLabelWidth は描画時の最大ラベル幅を返す（中央寄せ計算用）。
func (m *Menu) MaxLabelWidth(scale float64) float64 {
	var maxW float64
	for _, item := range m.Items {
		label := item.Label
		if len(item.Values) > 0 {
			label = label + "  < " + item.Values[item.ValueIndex] + " >"
		}
		w, _ := MeasureText(label, scale)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}
