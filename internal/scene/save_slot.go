package scene

import (
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/save"
	"github.com/yiozio/space-miner/internal/ui"
)

// saveSlotMode はスロット選択画面の用途。
type saveSlotMode int

const (
	saveSlotModeSave saveSlotMode = iota
	saveSlotModeLoad
)

// SaveSlotPicker は SAVE / LOAD で 3 スロットから 1 つを選ぶシーン。
// 各スロットには保存時刻・累計プレイ時間・所持金・宙域名を表示する。
type SaveSlotPicker struct {
	mode    saveSlotMode
	saveCtx save.Context // mode = save の場合のみ使用
	metas   [save.SlotCount]*save.Meta
	cursor  int // 0..SlotCount-1
}

// NewSaveSlotForSave は現在の状態を書き出すスロットを選ぶシーンを返す。
func NewSaveSlotForSave(ctx save.Context) *SaveSlotPicker {
	s := &SaveSlotPicker{mode: saveSlotModeSave, saveCtx: ctx}
	s.refreshMetas()
	return s
}

// NewSaveSlotForLoad は読み出すスロットを選ぶシーンを返す。
// 空スロットは選択不可。
func NewSaveSlotForLoad() *SaveSlotPicker {
	s := &SaveSlotPicker{mode: saveSlotModeLoad}
	s.refreshMetas()
	// 既存セーブ中で最新のスロットにカーソルを合わせる
	if latest := save.LatestSlot(); latest >= 1 {
		s.cursor = latest - 1
	}
	return s
}

func (s *SaveSlotPicker) refreshMetas() {
	for i := 0; i < save.SlotCount; i++ {
		m, err := save.LoadMeta(i + 1)
		if err != nil {
			log.Printf("read slot %d meta: %v", i+1, err)
			s.metas[i] = nil
			continue
		}
		s.metas[i] = m
	}
}

func (s *SaveSlotPicker) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if s.cursor > 0 {
			s.cursor--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if s.cursor < save.SlotCount-1 {
			s.cursor++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		s.activate(d)
	}
	return nil
}

func (s *SaveSlotPicker) activate(d Director) {
	slot := s.cursor + 1
	switch s.mode {
	case saveSlotModeSave:
		// 既存スロットへの上書きは確認モーダルを挟む
		if s.metas[s.cursor] != nil {
			d.Push(NewConfirm(fmt.Sprintf("Overwrite slot %d?", slot), func(d Director, yes bool) {
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
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	// ヘッダ
	headerScale := 3.0
	header := "SAVE"
	if s.mode == saveSlotModeLoad {
		header = "LOAD"
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

	for i := 0; i < save.SlotCount; i++ {
		x := startX
		y := startY + float64(i)*(slotH+slotGap)
		s.drawSlot(dst, theme, x, y, slotW, slotH, i)
	}

	// フッタヒント
	hint := "[ Up/Down: Move    Enter: Select    Esc: Back ]"
	ui.DrawText(dst, hint, 20, float64(sh)-30, 1.4, theme.LineDim)
}

func (s *SaveSlotPicker) drawSlot(dst *ebiten.Image, theme *ui.Theme, x, y, w, h float64, idx int) {
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

	slotLabel := fmt.Sprintf("Slot %d", idx+1)
	ui.DrawText(dst, slotLabel, x+16, y+12, 1.6, theme.Line)

	meta := s.metas[idx]
	if meta == nil {
		// 空スロット表示
		emptyColor := theme.LineDim
		// LOAD では空スロットは選択不能であることを暗示
		ui.DrawText(dst, "(empty)", x+16, y+44, 1.4, emptyColor)
		return
	}

	// 1 行目: 保存時刻 + 経過時間
	savedAt := meta.SavedAt.Local().Format("2006-01-02 15:04")
	playtime := formatPlaytime(meta.Playtime)
	ui.DrawText(dst, fmt.Sprintf("%s   PLAY %s", savedAt, playtime), x+16, y+44, 1.3, theme.Line)

	// 2 行目: 所持金 + 宙域名
	mapName := meta.MapName
	if mapName == "" {
		mapName = "(deep space)"
	}
	ui.DrawText(dst, fmt.Sprintf("CR %d   %s", meta.Credits, mapName), x+16, y+68, 1.3, theme.LineDim)
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
