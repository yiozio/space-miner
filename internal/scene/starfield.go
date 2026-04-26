package scene

import (
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/ui"
)

// star はワールド座標を持つ単点。
type star struct {
	x, y       float64
	brightness float64
}

// starfield は固定シードで生成された星の集合。
// パララックスなしで、カメラ位置に応じて見えている範囲だけ描画する。
type starfield struct {
	stars []star
}

// newStarfield は area×area の正方領域に n 個の星をばら撒く。
func newStarfield(seed int64, n int, area float64) *starfield {
	rng := rand.New(rand.NewSource(seed))
	sf := &starfield{stars: make([]star, n)}
	for i := range sf.stars {
		sf.stars[i] = star{
			x:          rng.Float64()*area*2 - area,
			y:          rng.Float64()*area*2 - area,
			brightness: 0.3 + rng.Float64()*0.7,
		}
	}
	return sf
}

// draw はカメラ中心 (camX, camY) を画面中央としてスター群を描画する。
func (sf *starfield) draw(dst *ebiten.Image, camX, camY float64, theme *ui.Theme) {
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	cx := float64(sw) / 2
	cy := float64(sh) / 2
	for _, st := range sf.stars {
		sx := st.x - camX + cx
		sy := st.y - camY + cy
		if sx < 0 || sx >= float64(sw) || sy < 0 || sy >= float64(sh) {
			continue
		}
		c := theme.Line
		c.A = uint8(float64(c.A) * st.brightness)
		vector.DrawFilledRect(dst, float32(sx), float32(sy), 1, 1, c, false)
	}
}
