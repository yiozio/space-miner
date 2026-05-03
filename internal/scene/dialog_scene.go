package scene

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/dialog"
	"github.com/yiozio/space-miner/internal/ui"
)

// dialogStyle は DialogScene の見た目モード。
type dialogStyle int

const (
	dialogStyleOverlay    dialogStyle = iota // 通常: 下のシーンの上に暗いオーバーレイ
	dialogStyleFullScreen                    // オープニング: 全画面背景 + アバター無し
)

const (
	dialogRevealCharsPerFrame = 0.7 // タイプライタ速度（約 42 文字/秒 @60fps）
	dialogTextScale           = 1.4
	dialogChoiceScale         = 1.4
	dialogNameScale           = 1.3
)

// DialogScene は dialog.Script を再生するシーン。
// 通常モードは画面下部にダイアログ枠（左にアバター、右にテキスト）を表示。
// 全画面モードはオープニング用で、ベクター背景の上にナレーションを表示する。
type DialogScene struct {
	script    *dialog.Script
	current   string  // 現ノード ID。"" なら次フレームで Pop。
	cursor    int     // 選択肢のカーソル位置
	revealAcc float64 // 表示済み文字数（フラクション付き）
	style     dialogStyle
}

// NewDialogScene は通常のダイアログ（暗いオーバーレイ + アバター付き）を生成する。
func NewDialogScene(script *dialog.Script) *DialogScene {
	return &DialogScene{
		script:  script,
		current: script.Start,
		style:   dialogStyleOverlay,
	}
}

// NewOpeningScene は全画面ダイアログ（ベクター背景 + アバターなしのナレーション）を生成する。
func NewOpeningScene(script *dialog.Script) *DialogScene {
	return &DialogScene{
		script:  script,
		current: script.Start,
		style:   dialogStyleFullScreen,
	}
}

func (s *DialogScene) currentNode() (dialog.Node, bool) {
	return s.script.Node(s.current)
}

func (s *DialogScene) textFullyRevealed() bool {
	n, ok := s.currentNode()
	if !ok {
		return true
	}
	return int(s.revealAcc) >= len(n.Text)
}

func (s *DialogScene) Update(d Director) error {
	if s.current == "" {
		d.Pop()
		return nil
	}
	n, ok := s.currentNode()
	if !ok {
		d.Pop()
		return nil
	}

	if !s.textFullyRevealed() {
		s.revealAcc += dialogRevealCharsPerFrame
		if s.revealAcc > float64(len(n.Text)) {
			s.revealAcc = float64(len(n.Text))
		}
	}

	// Esc は会話全体スキップ
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		return nil
	}

	advance := inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeySpace)

	// テキストが途中なら advance で全文表示にショートカット
	if advance && !s.textFullyRevealed() {
		s.revealAcc = float64(len(n.Text))
		return nil
	}

	if !s.textFullyRevealed() {
		return nil
	}

	if len(n.Choices) > 0 {
		// 選択肢あり: 上下で選択
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
			if s.cursor > 0 {
				s.cursor--
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
			if s.cursor < len(n.Choices)-1 {
				s.cursor++
			}
		}
		if advance {
			c := n.Choices[s.cursor]
			s.gotoNode(c.Next)
		}
		return nil
	}

	// 選択肢なし: advance で Next（空なら終了）
	if advance {
		s.gotoNode(n.Next)
	}
	return nil
}

func (s *DialogScene) gotoNode(id string) {
	if id == "" {
		s.current = "" // 次フレームで Pop
		return
	}
	if _, ok := s.script.Node(id); !ok {
		s.current = ""
		return
	}
	s.current = id
	s.cursor = 0
	s.revealAcc = 0
}

func (s *DialogScene) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	if s.style == dialogStyleFullScreen {
		drawOpeningBackground(dst, theme, sw, sh)
	} else {
		// 暗いオーバーレイ
		vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
			color.NRGBA{0, 0, 0, 180}, false)
	}

	s.drawDialogBox(dst, theme, sw, sh)
}

// drawDialogBox は画面下部のダイアログ枠（アバター + テキスト + 選択肢）を描く。
func (s *DialogScene) drawDialogBox(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	n, ok := s.currentNode()
	if !ok {
		return
	}

	const (
		boxMargin = 32.0
		boxHeight = 200.0
		avatarSz  = 144.0
		innerPad  = 16.0
	)
	boxX := boxMargin
	boxY := float64(sh) - boxMargin - boxHeight
	boxW := float64(sw) - boxMargin*2

	// 枠
	vector.DrawFilledRect(dst, float32(boxX), float32(boxY),
		float32(boxW), float32(boxHeight), color.NRGBA{0, 0, 0, 230}, false)
	vector.StrokeRect(dst, float32(boxX), float32(boxY),
		float32(boxW), float32(boxHeight), 2, theme.Line, false)

	// アバター（Speaker が空なら省略）
	textX := boxX + innerPad
	textW := boxW - innerPad*2
	speaker := dialog.CharacterByID(n.Speaker)
	if s.style != dialogStyleFullScreen && speaker != nil {
		ax := boxX + innerPad
		ay := boxY + (boxHeight-avatarSz)/2
		drawAvatar(dst, theme, speaker, ax, ay, avatarSz)
		textX = ax + avatarSz + innerPad
		textW = boxX + boxW - innerPad - textX
	}

	// 話者名（あれば、テキストの上）
	textY := boxY + innerPad
	if speaker != nil {
		ui.DrawText(dst, speaker.Name, textX, textY, dialogNameScale, speaker.Color)
		textY += 24
	}

	// 本文（タイプライタ）
	revealedRunes := []rune(n.Text)
	revealed := int(s.revealAcc)
	if revealed > len(revealedRunes) {
		revealed = len(revealedRunes)
	}
	bodyText := string(revealedRunes[:revealed])
	for _, line := range wrapText(bodyText, textW, dialogTextScale) {
		ui.DrawText(dst, line, textX, textY, dialogTextScale, theme.Line)
		_, lh := ui.MeasureText("Ag", dialogTextScale)
		textY += lh + 4
	}

	// 選択肢（フル表示後のみ）
	if s.textFullyRevealed() && len(n.Choices) > 0 {
		choiceY := boxY + boxHeight - innerPad - float64(len(n.Choices))*24
		for i, c := range n.Choices {
			prefix := "  "
			clr := theme.LineDim
			if i == s.cursor {
				prefix = "> "
				clr = theme.Line
			}
			ui.DrawText(dst, prefix+c.Label, textX, choiceY, dialogChoiceScale, clr)
			choiceY += 24
		}
	}

	// 進行ヒント
	if s.textFullyRevealed() && len(n.Choices) == 0 {
		hint := "[ Enter / Space: Next   Esc: Skip ]"
		hw, _ := ui.MeasureText(hint, 1.0)
		ui.DrawText(dst, hint, boxX+boxW-innerPad-hw, boxY+boxHeight-innerPad-12,
			1.0, theme.LineDim)
	} else if len(n.Choices) > 0 && s.textFullyRevealed() {
		hint := "[ Up/Down: Move   Enter: Select ]"
		hw, _ := ui.MeasureText(hint, 1.0)
		ui.DrawText(dst, hint, boxX+boxW-innerPad-hw, boxY+innerPad-2,
			1.0, theme.LineDim)
	}
}

// drawOpeningBackground は全画面用の装飾背景を描く（恒星系シルエット風）。
func drawOpeningBackground(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	dst.Fill(color.NRGBA{0, 0, 0, 255})

	cx := float32(sw) / 2
	cy := float32(sh)/2 - 50
	bodyR := float32(180)

	fill := theme.Line
	fill.A = 26
	vector.DrawFilledCircle(dst, cx, cy, bodyR, fill, true)
	vector.StrokeCircle(dst, cx, cy, bodyR, 2, theme.Line, true)

	for _, r := range []float32{bodyR + 60, bodyR + 130, bodyR + 220} {
		vector.StrokeCircle(dst, cx, cy, r, 1, theme.LineDim, true)
	}

	title := "PROLOGUE"
	tScale := 4.0
	tw, _ := ui.MeasureText(title, tScale)
	ui.DrawText(dst, title, float64(cx)-tw/2, 56, tScale, theme.Line)
}

// drawAvatar はアバター枠 + キャラクター固有の簡易顔シンボルを描く。
func drawAvatar(dst *ebiten.Image, theme *ui.Theme, ch *dialog.Character, x, y, size float64) {
	// 枠
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(size), float32(size),
		color.NRGBA{0, 0, 0, 255}, false)
	vector.StrokeRect(dst, float32(x), float32(y), float32(size), float32(size),
		1, theme.Line, false)

	cx := float32(x + size/2)
	cy := float32(y + size/2)
	headR := float32(size * 0.34)

	// 頭（キャラ色の半透明塗り + 縁線）
	fill := ch.Color
	fill.A = 160
	vector.DrawFilledCircle(dst, cx, cy, headR, fill, true)
	vector.StrokeCircle(dst, cx, cy, headR, 1.5, ch.Color, true)

	// 目（共通: 2 つの小さな塗り円）
	eyeOffsetX := headR * 0.4
	eyeOffsetY := -headR * 0.15
	eyeR := float32(2)
	vector.DrawFilledCircle(dst, cx-eyeOffsetX, cy+eyeOffsetY, eyeR, theme.Line, true)
	vector.DrawFilledCircle(dst, cx+eyeOffsetX, cy+eyeOffsetY, eyeR, theme.Line, true)

	// 口（スタイル別）
	mouthY := cy + headR*0.45
	mouthHalfW := headR * 0.35
	switch ch.Style {
	case dialog.AvatarStyleSmile:
		// やや弧（簡易: 2 線で V を逆さ向きに）
		vector.StrokeLine(dst, cx-mouthHalfW, mouthY-2, cx, mouthY+3, 2, theme.Line, true)
		vector.StrokeLine(dst, cx, mouthY+3, cx+mouthHalfW, mouthY-2, 2, theme.Line, true)
	case dialog.AvatarStyleStern:
		vector.StrokeLine(dst, cx-mouthHalfW, mouthY, cx+mouthHalfW, mouthY, 2, theme.Line, true)
	case dialog.AvatarStyleGoggles:
		// 目に水平線を重ねてゴーグル感
		gy := cy + eyeOffsetY
		vector.StrokeLine(dst, cx-headR*0.7, gy, cx+headR*0.7, gy, 2, theme.Line, true)
		vector.StrokeLine(dst, cx-mouthHalfW, mouthY, cx+mouthHalfW, mouthY, 2, theme.Line, true)
	case dialog.AvatarStyleHelmet:
		// 上に弧（ヘルメット縁）
		vector.StrokeLine(dst, cx-headR, cy-headR*0.1, cx-headR, cy-headR, 2, theme.Line, true)
		vector.StrokeLine(dst, cx-headR, cy-headR, cx+headR, cy-headR, 2, theme.Line, true)
		vector.StrokeLine(dst, cx+headR, cy-headR, cx+headR, cy-headR*0.1, 2, theme.Line, true)
		vector.StrokeLine(dst, cx-mouthHalfW, mouthY, cx+mouthHalfW, mouthY, 2, theme.Line, true)
	}
}

// wrapText はテキストを最大幅に収まるように単語境界で折り返した行配列を返す。
func wrapText(s string, maxWidth, scale float64) []string {
	if s == "" {
		return []string{""}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{s}
	}
	var lines []string
	current := ""
	for _, w := range words {
		candidate := current
		if candidate != "" {
			candidate += " " + w
		} else {
			candidate = w
		}
		cw, _ := ui.MeasureText(candidate, scale)
		if cw > maxWidth && current != "" {
			lines = append(lines, current)
			current = w
		} else {
			current = candidate
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
