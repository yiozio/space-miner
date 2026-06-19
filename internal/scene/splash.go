package scene

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	assetimage "github.com/yiozio/space-miner/internal/asset/image"
	"github.com/yiozio/space-miner/internal/asset/sound"
)

// splash.go はゲーム起動時に表示するスプラッシュ画面。
// 重い惑星アセットの展開と音声（効果音 PCM 合成・BGM デコード）の先行読み込みを行い、
// その両方が完了するまでアニメーションをループ再生してから、タイトルへ遷移する。
// これでタイトル表示時に BGM を即再生できる（ブラウザでの音楽再生ラグを解消）。
// TODO: 後でここを開発者ロゴ表示に差し替える（演出はこのシーンを置き換えるだけでよい）。

// splashCycleFrames はつなぎアニメ 1 サイクルのフレーム数（60fps で 0.7 秒）。
const splashCycleFrames = 42

// Splash は起動直後のつなぎ画面。
type Splash struct {
	frame int
	stars *starfield
}

// NewSplash はスプラッシュ画面を生成し、待機中に惑星アセットと音声の読み込みを先行開始する。
func NewSplash() *Splash {
	assetimage.PreloadPlanet()
	go sound.Preload() // 効果音 PCM 合成と BGM フルデコードを別ゴルーチンで先行
	return &Splash{stars: newStarfield(1)}
}

func (s *Splash) Update(d Director) error {
	s.frame++
	// 惑星アセットと音声の両方の読み込みが完了したらタイトルへ。
	if assetimage.PlanetReady() && sound.AudioReady() {
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
	// 読み込み完了まで待つため、つなぎアニメは 0..1 を繰り返しループさせる。
	p := math.Mod(float64(s.frame)/float64(splashCycleFrames), 1.0)

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
