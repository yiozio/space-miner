package scene

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assetimage "github.com/yiozio/space-miner/internal/asset/image"
	"github.com/yiozio/space-miner/internal/asset/logo"
	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/dialog"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/save"
	"github.com/yiozio/space-miner/internal/ui"
)

// Title はスタート画面シーン。
type Title struct {
	menu *ui.Menu
	bg   *titleBackground
}

const (
	titleItemContinue = iota
	titleItemNewGame
	titleItemLoad
	titleItemSetting
	titleItemQuit
)

// NewTitle は新しい Title シーンを返す。
// Continue はセーブが 1 つ以上あれば最新セーブをロード、Load はスロット選択を開く。
func NewTitle() *Title {
	hasSave := save.AnyExists()
	cursor := titleItemNewGame
	if hasSave {
		cursor = titleItemContinue
	}
	tl := &Title{
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Enabled: hasSave},
				{Enabled: true},
				{Enabled: hasSave},
				{Enabled: true},
				{Enabled: true},
			},
			Cursor: cursor,
		},
		bg: newTitleBackground(),
	}
	tl.applyLabels()
	sound.PlayTitleBGM() // タイトルの宇宙環境音（ゲーム開始時に StopBGM で停止）
	// オープニング表示中に惑星 GIF の展開をバックグラウンドで進める。
	// 完了するまでメニューは出さない（その間にゲーム本編へ入らせない）。
	assetimage.PreloadPlanet()
	return tl
}

// applyLabels はメニュー項目のラベルを現在言語で再設定する。
// 設定画面で言語を切り替えた後でもタイトルへ戻った瞬間に反映できるよう、
// Update 冒頭でも呼び直している。
func (t *Title) applyLabels() {
	s := i18n.S().Title
	t.menu.Items[titleItemContinue].Label = s.Continue
	t.menu.Items[titleItemNewGame].Label = s.NewGame
	t.menu.Items[titleItemLoad].Label = s.Load
	t.menu.Items[titleItemSetting].Label = s.Setting
	t.menu.Items[titleItemQuit].Label = s.Quit
}

func (t *Title) Update(d Director) error {
	t.applyLabels()
	sound.TickTitleBGM()
	t.bg.update()
	// 惑星アセットの展開が終わるまではメニューを操作させない（非表示）。
	if !assetimage.PlanetReady() {
		return nil
	}
	r := t.menu.Update()
	if !r.Activated {
		return nil
	}
	switch t.menu.Cursor {
	case titleItemContinue:
		slot := save.LatestSlot()
		if slot < 0 {
			return nil
		}
		res, err := save.Load(slot)
		if err != nil {
			log.Printf("continue: load slot %d: %v", slot, err)
			return nil
		}
		d.Replace(NewExplorationFromPlayer(res.Player, res.Playtime))
	case titleItemNewGame:
		d.Replace(NewExploration())
		// オープニング: 探索シーンの上に Push、閉じるとゲーム本編へ
		d.Push(NewOpeningScene(dialog.Opening()))
	case titleItemLoad:
		if !save.AnyExists() {
			return nil
		}
		d.Push(NewSaveSlotForLoad())
	case titleItemSetting:
		d.Push(NewSettings(d.Theme()))
	case titleItemQuit:
		d.Quit()
	}
	return nil
}

func (t *Title) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)
	t.bg.draw(dst, theme)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	s := i18n.S().Title

	// タイトルロゴ（SVG をベクター描画）。横幅は画面の 6 割を上限に収める。
	lw, lh := logo.Size()
	logoW := math.Min(float64(sw)*0.6, 720)
	logoScale := logoW / lw
	logoH := lh * logoScale
	logoX := (float64(sw)-logoW)/2 - 12
	// メニュー5項目 + 下部ヒントが収まるよう上寄せ（0.14 だと最下段が画面外に出る）
	logoY := float64(sh) * 0.05
	logo.Draw(dst, logoX, logoY, logoScale, theme.Line)

	// ロゴ上に自機を表すコックピット三角形と、その重心で明滅する光点を描く。
	drawTitleShipEmblem(dst, float64(sw)/2-12, logoY+90, theme)

	menuScale := 2.0
	my := logoY + logoH + 60
	if assetimage.PlanetReady() {
		// メニュー表示（アセット準備完了後）。
		maxW := t.menu.MaxLabelWidth(menuScale)
		mx := (float64(sw) - maxW) / 2
		t.menu.Draw(dst, theme, mx, my, menuScale)
		ui.DrawText(dst, s.Hint, 20, float64(sh)-30, 1.5, theme.LineDim)
	} else {
		// 準備中はメニューの代わりに読み込み表示。
		lw2, _ := ui.MeasureText(s.Loading, menuScale)
		ui.DrawText(dst, s.Loading, (float64(sw)-lw2)/2, my, menuScale, theme.LineDim)
	}
}

// drawTitleShipEmblem は (cx, bottomY) を底辺中央として、自機コックピット三角形
// （頂点が上・底辺が下）を描き、その重心に明滅する光点を重ねる。
// ゲーム内のコックピットパーツ（part.go の PartCockpit）と同じ比率で描く。
func drawTitleShipEmblem(dst *ebiten.Image, cx, bottomY float64, theme *ui.Theme) {
	const g = 112.0 // 三角形の外接セルサイズ
	const inset = g * 0.12
	// 光点の大きさ（調整用）。lightSize=基準の直径、lightFlicker=明滅の揺れ幅。
	const lightSize = 15.0
	const lightFlicker = 2.0
	x := cx - g/2    // セル左上
	y := bottomY - g // セル上端（底辺が bottomY に来るよう逆算）
	apexX, apexY := cx, y+inset
	lX, lY := x+inset, y+g-inset
	rX, rY := x+g-inset, y+g-inset
	vector.StrokeLine(dst, float32(apexX), float32(apexY), float32(lX), float32(lY), 2, theme.Line, true)
	vector.StrokeLine(dst, float32(apexX), float32(apexY), float32(rX), float32(rY), 2, theme.Line, true)
	vector.StrokeLine(dst, float32(lX), float32(lY), float32(rX), float32(rY), 2, theme.Line, true)
	// 重心（頂点 y 比 0.12, 0.88, 0.88 の平均）で明滅する光点。
	cyCentroid := y + g*(0.12+0.88+0.88)/3.0
	drawTrailLightSized(dst, cx, cyCentroid, lightSize, lightFlicker, theme.Line)
}
