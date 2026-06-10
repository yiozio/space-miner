package scene

import (
	"fmt"
	"image/color"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

// StationTavern は宇宙酒場シーン。
// 4 枠（2x2）の自動生成クエスト掲示板を表示し、各クエストに対して
// 「CLEAR（要件が満たされていれば実行可）」「DISCARD（破棄コスト支払いで再生成）」を行う。
// CLEAR / DISCARD のいずれもスロットを差し替えて新しい依頼が現れる。
type StationTavern struct {
	player      *entity.Player
	world       *entity.World
	stationName string
	rng         *rand.Rand
	cursor      int // 0..3（2x2: 行 = cursor/2, 列 = cursor%2）
}

// NewStationTavern は対象ステーションの掲示板を初期化（または読込）してシーンを返す。
func NewStationTavern(p *entity.Player, world *entity.World, stationName string) *StationTavern {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := &StationTavern{
		player:      p,
		world:       world,
		stationName: stationName,
		rng:         rng,
	}
	p.EnsureTavernBoard(stationName, world, rng)
	return s
}

func (s *StationTavern) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}
	// 2x2 グリッド移動。左右は矢印キーのみ（D は破棄に使うため WASD は割り当てない）
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if s.cursor >= 2 {
			s.cursor -= 2
			sound.PlayMenuMove()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if s.cursor < 2 {
			s.cursor += 2
			sound.PlayMenuMove()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		if s.cursor%2 == 1 {
			s.cursor--
			sound.PlayMenuMove()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		if s.cursor%2 == 0 {
			s.cursor++
			sound.PlayMenuMove()
		}
	}
	board := s.player.EnsureTavernBoard(s.stationName, s.world, s.rng)
	q := &board.Slots[s.cursor]

	// CLEAR: 要件達成 + 報酬受領可能ならクリア → スロット差し替え
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if !q.IsEmpty() && s.player.CanClearQuest(q) && s.player.CanReceiveReward(q) {
			s.player.ClearQuest(q)
			s.player.RegenerateSlot(s.stationName, s.cursor, s.world, s.rng)
			sound.PlayMenuSelect()
		}
	}

	// DISCARD: D キーで破棄コスト支払い → スロット差し替え
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if !q.IsEmpty() && s.player.Credits >= q.DiscardCost {
			s.player.Credits -= q.DiscardCost
			s.player.RegenerateSlot(s.stationName, s.cursor, s.world, s.rng)
			sound.PlayMenuCancel()
		}
	}
	return nil
}

func (s *StationTavern) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	// ヘッダ
	tv := i18n.S().Tavern
	headerScale := 3.0
	hw, hh := ui.MeasureText(tv.Header, headerScale)
	ui.DrawText(dst, tv.Header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)
	subtitle := s.stationName + " - " + tv.Subtitle
	stw, _ := ui.MeasureText(subtitle, 1.4)
	ui.DrawText(dst, subtitle, (float64(sw)-stw)/2, 24+hh+8, 1.4, theme.LineDim)

	// クレジット表示
	credits := fmt.Sprintf(i18n.S().Save.CreditsPrefix, s.player.Credits)
	cw, _ := ui.MeasureText(credits, 1.6)
	ui.DrawText(dst, credits, float64(sw)-cw-30, 24, 1.6, theme.Line)

	// レイアウト: ショップと同様にヘッダー直下へバーテンダー画像を大きく表示し、
	// その下に画像と同じ横幅でクエストカードを 2x2 で並べる
	cardH := 140.0
	cardGap := 20.0
	cardW := (stationPortraitW - cardGap) / 2
	startX := (float64(sw) - stationPortraitW) / 2

	portraitY := 24 + hh + 38 // サブタイトルの下
	drawStationPortrait(dst, theme, "BARTENDER", sw, portraitY)

	// クエストカード（2x2）
	board := s.player.EnsureTavernBoard(s.stationName, s.world, s.rng)
	startY := portraitY + stationPortraitH + 24
	for i := range board.Slots {
		x := startX + float64(i%2)*(cardW+cardGap)
		y := startY + float64(i/2)*(cardH+cardGap)
		s.drawCard(dst, theme, x, y, cardW, cardH, &board.Slots[i], i == s.cursor)
	}

	// 操作ヒント
	ui.DrawText(dst, tv.Hint, 20, float64(sh)-30, 1.4, theme.LineDim)
}

func (s *StationTavern) drawCard(dst *ebiten.Image, theme *ui.Theme, x, y, w, h float64, q *entity.Quest, focused bool) {
	frame := theme.LineDim
	stroke := float32(1)
	if focused {
		frame = theme.Line
		stroke = 2
	}
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), stroke, frame, false)

	tv := i18n.S().Tavern
	if q.IsEmpty() {
		ui.DrawText(dst, i18n.S().Common.Empty, x+16, y+(h-20)/2, 1.4, theme.LineDim)
		return
	}

	// 種別タグ
	kindLabel := tv.KindDelivery
	if q.Kind == entity.QuestKindBounty {
		kindLabel = tv.KindBounty
	}
	ui.DrawText(dst, kindLabel, x+16, y+12, 1.2, theme.LineDim)

	// 説明
	ui.DrawText(dst, i18n.QuestDescription(q), x+16, y+34, 1.6, theme.Line)

	// 進捗
	cur, target := s.player.QuestProgress(q)
	progress := fmt.Sprintf(tv.ProgressFmt, cur, target)
	progressColor := theme.LineDim
	if cur >= target {
		progressColor = theme.Line
		progress += "   " + tv.Ready
	}
	ui.DrawText(dst, progress, x+16, y+68, 1.3, progressColor)

	// 報酬
	rewardText := fmt.Sprintf(tv.RewardFmt, q.RewardCredits)
	if q.HasPartReward {
		if def := entity.PartDefByID(q.RewardPart); def != nil {
			rewardText += fmt.Sprintf(tv.BonusPartFmt, i18n.PartName(def.ID))
		}
	}
	ui.DrawText(dst, rewardText, x+16, y+92, 1.3, theme.Line)

	// 破棄コスト
	discardText := fmt.Sprintf(tv.DiscardFmt, q.DiscardCost)
	dw, _ := ui.MeasureText(discardText, 1.2)
	ui.DrawText(dst, discardText, x+w-dw-16, y+h-26, 1.2, theme.LineDim)

	// 状態（積載超過のため受領できない場合）
	if focused && cur >= target && !s.player.CanReceiveReward(q) {
		ui.DrawText(dst, tv.CargoFullWarn, x+16, y+h-26, 1.1, color.NRGBA{0xff, 0xa0, 0x80, 0xff})
	}
}
