package scene

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/ui"
)

// ConfirmCallback は確認モーダルでユーザーが選択を確定したときに呼ばれる。
// 呼び出し時点で Confirm シーンは既に Pop 済み。yes が true なら肯定。
type ConfirmCallback func(d Director, yes bool)

// Confirm は中央に表示する Yes/No 確認モーダル。
// 直前のシーンの上に半透明オーバーレイ＋ボックスを重ねて表示する。
type Confirm struct {
	message  string
	menu     *ui.Menu
	onResult ConfirmCallback
}

// NewConfirm はメッセージとコールバックから確認モーダルを生成する。
// 既定カーソルは安全側の "No"。
func NewConfirm(message string, onResult ConfirmCallback) *Confirm {
	return &Confirm{
		message: message,
		menu: &ui.Menu{
			Items: []*ui.MenuItem{
				{Label: "No", Enabled: true},
				{Label: "Yes", Enabled: true},
			},
		},
		onResult: onResult,
	}
}

func (c *Confirm) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Pop()
		c.onResult(d, false)
		return nil
	}
	r := c.menu.Update()
	if !r.Activated {
		return nil
	}
	yes := c.menu.Items[c.menu.Cursor].Label == "Yes"
	d.Pop()
	c.onResult(d, yes)
	return nil
}

func (c *Confirm) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 黒半透明オーバーレイ
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 200}, false)

	// モーダルボックス（中央寄せ）
	boxW, boxH := float32(540), float32(240)
	boxX := (float32(sw) - boxW) / 2
	boxY := (float32(sh) - boxH) / 2
	vector.DrawFilledRect(dst, boxX, boxY, boxW, boxH, theme.Background, false)
	vector.StrokeRect(dst, boxX, boxY, boxW, boxH, 1, theme.Line, false)

	// メッセージ
	msgScale := 1.8
	mw, _ := ui.MeasureText(c.message, msgScale)
	ui.DrawText(dst, c.message,
		float64(boxX)+(float64(boxW)-mw)/2, float64(boxY)+50,
		msgScale, theme.Line)

	// 選択肢
	menuScale := 2.0
	maxW := c.menu.MaxLabelWidth(menuScale)
	mx := float64(boxX) + (float64(boxW)-maxW)/2
	my := float64(boxY) + 120
	c.menu.Draw(dst, theme, mx, my, menuScale)
}
