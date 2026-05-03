package scene

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

// star はワールド座標を持つ単点。座標はレイヤーのタイル内 [0, tileSize)。
type star struct {
	x, y       float64
	brightness float64
}

// starLayer はパララックス（奥行き）係数とタイルサイズを持つ星のレイヤー。
// parallax = 1.0 でカメラと等速、< 1.0 で遠景としてゆっくり流れる。
type starLayer struct {
	stars    []star
	tileSize float64
	parallax float64
}

// starfield は複数のレイヤーをタイル状に敷き詰めて描画する。
// タイルはカメラに応じて上下左右へ無限に繰り返されるため、どこへ移動しても星が途切れない。
type starfield struct {
	layers []starLayer
}

// newStarLayer は1レイヤー分の星をタイル内にランダム配置する。
func newStarLayer(rng *rand.Rand, count int, tileSize, parallax, minBright, maxBright float64) starLayer {
	l := starLayer{tileSize: tileSize, parallax: parallax, stars: make([]star, count)}
	for i := range l.stars {
		l.stars[i] = star{
			x:          rng.Float64() * tileSize,
			y:          rng.Float64() * tileSize,
			brightness: minBright + rng.Float64()*(maxBright-minBright),
		}
	}
	return l
}

// 共有パララックス係数。
// 惑星バックドロップ（exploration.go の drawCelestialBackdrop）も同じ値を使う。
const (
	farStarParallax  = 0.0  // 最も遠い星: 動かない
	nearStarParallax = 0.10 // 近景星 + 惑星バックドロップ
	nearPlanetParallax = 0.35 // 現在居るマップの惑星/衛星
)

// newStarfield は遠景・近景の2レイヤー構成で生成する。
// 遠景は完全静止、近景はゆっくり流れる。惑星バックドロップは近景と同じパララックス。
func newStarfield(seed int64) *starfield {
	rng := rand.New(rand.NewSource(seed))
	return &starfield{
		layers: []starLayer{
			newStarLayer(rng, 420, 1500, farStarParallax, 0.18, 0.5), // 遠景（多数・暗い・静止）
			newStarLayer(rng, 70, 1000, nearStarParallax, 0.5, 1.0),  // 近景（少数・明るい・微妙に流れる）
		},
	}
}

// draw は各レイヤーをカメラ位置に応じてタイル展開して描画する。
func (sf *starfield) draw(dst *ebiten.Image, camX, camY float64, theme *ui.Theme) {
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	cx := float64(sw) / 2
	cy := float64(sh) / 2

	for _, layer := range sf.layers {
		// パララックスを反映した「実効カメラ」位置
		ecx := camX * layer.parallax
		ecy := camY * layer.parallax

		// 画面内のワールド範囲（実効カメラ基準）
		leftW := ecx - cx
		topW := ecy - cy
		rightW := ecx + cx
		botW := ecy + cy

		for _, st := range layer.stars {
			// 各星についてタイル繰返しのうち画面に入るオフセット範囲を求める
			kMinX := int(math.Ceil((leftW - st.x) / layer.tileSize))
			kMaxX := int(math.Floor((rightW - st.x) / layer.tileSize))
			kMinY := int(math.Ceil((topW - st.y) / layer.tileSize))
			kMaxY := int(math.Floor((botW - st.y) / layer.tileSize))

			c := theme.Line
			c.A = uint8(float64(c.A) * st.brightness)

			for ky := kMinY; ky <= kMaxY; ky++ {
				for kx := kMinX; kx <= kMaxX; kx++ {
					wx := st.x + float64(kx)*layer.tileSize
					wy := st.y + float64(ky)*layer.tileSize
					sx := wx - ecx + cx
					sy := wy - ecy + cy
					vector.DrawFilledRect(dst, float32(sx), float32(sy), 1, 1, c, false)
				}
			}
		}
	}
}
