package scene

import (
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/save"
	"github.com/yiozio/space-miner/internal/ui"
)

// saveSlotMode はスロット選択画面の用途。
type saveSlotMode int

const (
	saveSlotModeSave saveSlotMode = iota
	saveSlotModeLoad
)

// SaveSlotPicker は SAVE / LOAD で全スロット（オート + 手動 3 つ）から 1 つを選ぶシーン。
// 各スロットには保存時刻・累計プレイ時間・所持金・宙域名を表示する。
// SAVE モードではオートセーブ枠は選択できない（手動上書き禁止）。
type SaveSlotPicker struct {
	mode    saveSlotMode
	saveCtx save.Context // mode = save の場合のみ使用
	slots   []int        // 表示順のスロット番号（AutoSlot, 1, 2, 3）
	metas   []*save.Meta // metas[i] は slots[i] のメタ情報
	cursor  int          // 0..len(slots)-1
}

// NewSaveSlotForSave は現在の状態を書き出すスロットを選ぶシーンを返す。
// オートセーブ枠は表示されるが選択不可。初期カーソルは最初の手動スロットに合わせる。
func NewSaveSlotForSave(ctx save.Context) *SaveSlotPicker {
	s := &SaveSlotPicker{mode: saveSlotModeSave, saveCtx: ctx, slots: save.AllSlots()}
	s.refreshMetas()
	s.cursor = s.firstManualIndex()
	return s
}

// NewSaveSlotForLoad は読み出すスロットを選ぶシーンを返す。
// 空スロットは選択不可。オートセーブも対象。
func NewSaveSlotForLoad() *SaveSlotPicker {
	s := &SaveSlotPicker{mode: saveSlotModeLoad, slots: save.AllSlots()}
	s.refreshMetas()
	// 既存セーブ中で最新のスロットにカーソルを合わせる
	if latest := save.LatestSlot(); latest >= 0 {
		for i, slot := range s.slots {
			if slot == latest {
				s.cursor = i
				break
			}
		}
	}
	return s
}

func (s *SaveSlotPicker) refreshMetas() {
	s.metas = make([]*save.Meta, len(s.slots))
	for i, slot := range s.slots {
		m, err := save.LoadMeta(slot)
		if err != nil {
			log.Printf("read slot %d meta: %v", slot, err)
			s.metas[i] = nil
			continue
		}
		s.metas[i] = m
	}
}

// firstManualIndex は手動スロット先頭のインデックスを返す（SAVE 用の初期カーソル位置）。
// 通常は 1（slots = [Auto, 1, 2, 3] なので index 1 が Slot 1）。
func (s *SaveSlotPicker) firstManualIndex() int {
	for i, slot := range s.slots {
		if !save.IsAutoSlot(slot) {
			return i
		}
	}
	return 0
}

func (s *SaveSlotPicker) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		s.moveCursor(-1)
		sound.PlayMenuMove()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		s.moveCursor(+1)
		sound.PlayMenuMove()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		sound.PlayMenuSelect()
		s.activate(d)
	}
	return nil
}

// moveCursor は dir (-1 上 / +1 下) 方向に最も近い「選択可能な」スロットへ移動する。
// SAVE モードではオートセーブ枠を、LOAD モードでは選択可能要件があるが
// ここでは表示移動だけを許し、確定は activate で取り扱う（空スロット表示のため）。
func (s *SaveSlotPicker) moveCursor(dir int) {
	for i := s.cursor + dir; i >= 0 && i < len(s.slots); i += dir {
		if s.cursorAllowed(i) {
			s.cursor = i
			return
		}
	}
}

// cursorAllowed はその index にカーソルを置いてよいか返す。
// SAVE モードではオートセーブ枠を除外する。LOAD モードではすべて許可（空表示も含む）。
func (s *SaveSlotPicker) cursorAllowed(idx int) bool {
	if idx < 0 || idx >= len(s.slots) {
		return false
	}
	if s.mode == saveSlotModeSave && save.IsAutoSlot(s.slots[idx]) {
		return false
	}
	return true
}

func (s *SaveSlotPicker) activate(d Director) {
	slot := s.slots[s.cursor]
	switch s.mode {
	case saveSlotModeSave:
		// 念のための防御: オートセーブ枠が選ばれた場合は確定不可（cursor 制御で来ないはず）
		if save.IsAutoSlot(slot) {
			return
		}
		// 既存スロットへの上書きは確認モーダルを挟む
		if s.metas[s.cursor] != nil {
			d.Push(NewConfirm(fmt.Sprintf(i18n.S().Save.OverwriteConfirm, slot), func(d Director, yes bool) {
				if !yes {
					return
				}
				s.doSave(d, slot)
			}))
			return
		}
		s.doSave(d, slot)
	case saveSlotModeLoad:
		if s.metas[s.cursor] == nil {
			return
		}
		r, err := save.Load(slot)
		if err != nil {
			log.Printf("load slot %d: %v", slot, err)
			return
		}
		// メニューや関連オーバーレイをすべて閉じてから探索シーンを差し替える
		d.Pop()
		d.Replace(NewExplorationFromPlayer(r.Player, r.Playtime))
	}
}

func (s *SaveSlotPicker) doSave(d Director, slot int) {
	if err := save.Save(slot, s.saveCtx); err != nil {
		log.Printf("save slot %d: %v", slot, err)
		return
	}
	s.refreshMetas()
	d.Pop()
}

func (s *SaveSlotPicker) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 背景の暗いオーバーレイ
	vector.FillRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	// ヘッダ
	st := i18n.S().Save
	headerScale := 3.0
	header := st.HeaderSave
	if s.mode == saveSlotModeLoad {
		header = st.HeaderLoad
	}
	hw, hh := ui.MeasureText(header, headerScale)
	headerY := 36.0
	ui.DrawText(dst, header, (float64(sw)-hw)/2, headerY, headerScale, theme.Line)

	// スロット行
	slotW := 640.0
	slotH := 96.0
	slotGap := 16.0
	startX := (float64(sw) - slotW) / 2
	startY := headerY + hh + 50

	for i := range s.slots {
		x := startX
		y := startY + float64(i)*(slotH+slotGap)
		s.drawSlot(dst, theme, x, y, slotW, slotH, i)
	}

	// フッタヒント
	ui.DrawText(dst, st.Hint, 20, float64(sh)-30, 1.4, theme.LineDim)
}

func (s *SaveSlotPicker) drawSlot(dst *ebiten.Image, theme *ui.Theme, x, y, w, h float64, idx int) {
	slot := s.slots[idx]
	isAuto := save.IsAutoSlot(slot)
	disabled := s.mode == saveSlotModeSave && isAuto

	focused := idx == s.cursor
	frameColor := theme.LineDim
	if focused {
		frameColor = theme.Line
	}
	stroke := float32(1)
	if focused {
		stroke = 2
	}
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), stroke, frameColor, false)

	labelColor := theme.Line
	bodyColor := theme.Line
	subColor := theme.LineDim
	if disabled {
		labelColor = theme.LineDim
		bodyColor = theme.LineDim
		subColor = theme.LineDim
	}

	st := i18n.S().Save
	slotLabel := fmt.Sprintf(st.SlotLabel, slot)
	if isAuto {
		slotLabel = st.AutoSlotLabel
	}
	if disabled {
		slotLabel += "  " + st.ReadOnlySuffix
	}
	ui.DrawText(dst, slotLabel, x+16, y+12, 1.6, labelColor)

	meta := s.metas[idx]
	if meta == nil {
		ui.DrawText(dst, i18n.S().Common.Empty, x+16, y+44, 1.4, theme.LineDim)
		return
	}

	// 1 行目: 保存時刻 + 経過時間
	savedAt := meta.SavedAt.Local().Format("2006-01-02 15:04")
	playtime := formatPlaytime(meta.Playtime)
	playtimeText := fmt.Sprintf(st.PlayPrefix, playtime)
	ui.DrawText(dst, savedAt+"   "+playtimeText, x+16, y+44, 1.3, bodyColor)

	// 2 行目: 所持金 + 宙域名
	mapName := meta.MapName
	if mapName == "" {
		mapName = st.DeepSpace
	}
	ui.DrawText(dst, fmt.Sprintf(st.CreditsPrefix, meta.Credits)+"   "+mapName, x+16, y+68, 1.3, subColor)
}

// formatPlaytime は秒数を "Hh MMm" 形式（>=1h）または "MMm SSs"（<1h）で表示する。
func formatPlaytime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	d := time.Duration(seconds * float64(time.Second))
	hours := int(d / time.Hour)
	mins := int((d % time.Hour) / time.Minute)
	secs := int((d % time.Minute) / time.Second)
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, mins)
	}
	return fmt.Sprintf("%dm %02ds", mins, secs)
}
