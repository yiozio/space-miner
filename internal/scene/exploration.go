package scene

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

// Exploration は探索画面シーン。
// プレイヤー機を中心に俯瞰描画し、星空背景・HUD・ミニマップ枠を持つ。
type Exploration struct {
	player    *entity.Player
	cameraX   float64
	cameraY   float64
	starfield *starfield
}

// NewExploration は新しい探索シーンを生成する。
func NewExploration() *Exploration {
	return &Exploration{
		player:    entity.NewPlayerPebble(),
		starfield: newStarfield(1, 400, 4000),
	}
}

func (e *Exploration) Update(d Director) error {
	// 暫定: メニュー画面（Phase 3）実装まで Esc はタイトル復帰
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		d.Replace(NewTitle())
		return nil
	}
	e.player.Update()
	// カメラはプレイヤーに追従
	e.cameraX = e.player.X
	e.cameraY = e.player.Y
	return nil
}

func (e *Exploration) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	// 背景
	e.starfield.draw(dst, e.cameraX, e.cameraY, theme)

	// プレイヤー機（ワールド→スクリーン変換）
	psx := e.player.X - e.cameraX + float64(sw)/2
	psy := e.player.Y - e.cameraY + float64(sh)/2
	e.player.DrawAt(dst, psx, psy, theme)

	e.drawHUD(dst, theme, sw, sh)
}

func (e *Exploration) drawHUD(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	// 仮 HUD（数値はプレースホルダ）
	ui.DrawText(dst, "HP 100   SHIELD 100   FUEL 100", 20, 20, 1.5, theme.Line)
	speed := math.Hypot(e.player.VX, e.player.VY)
	ui.DrawText(dst,
		fmt.Sprintf("SPEED %.2f   POS %.0f, %.0f", speed, e.player.X, e.player.Y),
		20, 50, 1.2, theme.LineDim)

	// ミニマップ枠（中身は後続フェーズで実装）
	miniW, miniH := float32(180), float32(180)
	mx := float32(sw) - miniW - 20
	my := float32(sh) - miniH - 20
	vector.StrokeRect(dst, mx, my, miniW, miniH, 1, theme.Line, false)
	ui.DrawText(dst, "MINIMAP", float64(mx)+10, float64(my)+8, 1.2, theme.LineDim)
	// プレイヤー位置を中央点で示す
	vector.DrawFilledRect(dst, mx+miniW/2-1, my+miniH/2-1, 2, 2, theme.Line, false)

	// 操作ヒント
	ui.DrawText(dst, "[ WASD/Arrows: Move    Shift: Boost    Esc: Title ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}
