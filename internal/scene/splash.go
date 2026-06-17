package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	assetimage "github.com/yiozio/space-miner/internal/asset/image"
)

// splash.go はゲーム起動時に短く表示するスプラッシュ画面。
// 重い惑星アセットの背景展開を待つつなぎとして簡単なアニメーションを再生し、
// 一定時間後にタイトルへ遷移する。
// TODO: 後でここを開発者ロゴ表示に差し替える（演出はこのシーンを置き換えるだけでよい）。

// splashDurationFrames はスプラッシュ表示時間（60fps で 0.7 秒）。
const splashDurationFrames = 42

// Splash は起動直後のつなぎ画面。
type Splash struct {
	frame int
	stars *starfield
}

// NewSplash はスプラッシュ画面を生成し、待機中に惑星アセットの展開を先行開始する。
func NewSplash() *Splash {
	assetimage.PreloadPlanet()
	return &Splash{stars: newStarfield(1)}
}

func (s *Splash) Update(d Director) error {
	s.frame++
	if s.frame >= splashDurationFrames {
		d.Replace(NewTitle())
	}
	return nil
}

func (s *Splash) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)
	s.stars.draw(dst, 0, 0, theme)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	cx, cy := float64(sw)/2, float64(sh)/2
	p := float64(s.frame) / float64(splashDurationFrames) // 0..1

	// つなぎアニメ: 中央から広がってフェードするリングを 2 つ時間差で出す。
	for i := 0; i < 2; i++ {
		ph := p - float64(i)*0.35
		if ph < 0 || ph > 1 {
			continue
		}
		r := float32(20 + ph*140)
		c := theme.Line
		c.A = uint8(255 * (1 - ph))
		vector.StrokeCircle(dst, float32(cx), float32(cy), r, 2, c, true)
	}
	// 中央で明滅する光点。
	drawTrailLightSized(dst, cx, cy, 12, 2, theme.Line)
}
