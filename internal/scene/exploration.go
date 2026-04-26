package scene

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	startAsteroidCount = 8
	asteroidMinDist    = 250.0
	asteroidMaxDist    = 850.0
	asteroidMinSize    = 4
	asteroidMaxSize    = 10
	minimapScale       = 0.06
)

// Exploration は探索画面シーン。
// プレイヤー機を中心に俯瞰描画し、小惑星・弾・資源ピックアップを管理する。
type Exploration struct {
	player    *entity.Player
	cameraX   float64
	cameraY   float64
	starfield *starfield
	asteroids []*entity.Asteroid
	bullets   []entity.Bullet
	pickups   []entity.Pickup
}

// NewExploration は新しい探索シーンを生成し、初期小惑星をばら撒く。
func NewExploration() *Exploration {
	e := &Exploration{
		player:    entity.NewPlayerPebble(),
		starfield: newStarfield(1, 400, 4000),
	}
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < startAsteroidCount; i++ {
		ang := rng.Float64() * math.Pi * 2
		dist := asteroidMinDist + rng.Float64()*(asteroidMaxDist-asteroidMinDist)
		x := math.Cos(ang) * dist
		y := math.Sin(ang) * dist
		size := asteroidMinSize + rng.Intn(asteroidMaxSize-asteroidMinSize+1)
		e.asteroids = append(e.asteroids, entity.NewAsteroid(rng.Int63(), x, y, size))
	}
	return e
}

func (e *Exploration) Update(d Director) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// メニュー中はアフターバーナーが残らないよう推力状態をリセット
		e.player.ThrustState = entity.ThrustOff
		d.Push(NewMenu())
		return nil
	}

	e.player.Update()

	// 小惑星の浮遊・自転
	for _, a := range e.asteroids {
		a.Update()
	}

	// 発射（押しっぱなしでクールダウン許可分だけ発射）
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		e.bullets = append(e.bullets, e.player.Shoot()...)
	}

	// 弾の更新と寿命処理
	nb := 0
	for i := range e.bullets {
		b := &e.bullets[i]
		b.Update()
		if b.Alive() {
			e.bullets[nb] = *b
			nb++
		}
	}
	e.bullets = e.bullets[:nb]

	// 弾 vs 小惑星（衝突したら弾を消し、破壊グリッドからピックアップを生成）
	for i := len(e.bullets) - 1; i >= 0; i-- {
		bx, by := e.bullets[i].X, e.bullets[i].Y
		for _, a := range e.asteroids {
			absorbed, drops := a.Hit(bx, by)
			if !absorbed {
				continue
			}
			e.pickups = append(e.pickups, drops...)
			e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
			break
		}
	}

	// 空になった小惑星を除去
	na := 0
	for _, a := range e.asteroids {
		if !a.Empty() {
			e.asteroids[na] = a
			na++
		}
	}
	e.asteroids = e.asteroids[:na]

	// ピックアップの更新（吸引・回収・寿命切れ）
	np := 0
	for i := range e.pickups {
		p := &e.pickups[i]
		if p.Update(e.player.X, e.player.Y) {
			e.player.AddResource(p.Resource, 1)
			continue
		}
		if p.Life > 0 {
			e.pickups[np] = *p
			np++
		}
	}
	e.pickups = e.pickups[:np]

	// カメラ追従
	e.cameraX = e.player.X
	e.cameraY = e.player.Y
	return nil
}

func (e *Exploration) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	cx, cy := float64(sw)/2, float64(sh)/2

	e.starfield.draw(dst, e.cameraX, e.cameraY, theme)

	// 小惑星
	for _, a := range e.asteroids {
		a.Draw(dst, a.X-e.cameraX+cx, a.Y-e.cameraY+cy)
	}

	// ピックアップ
	for i := range e.pickups {
		p := &e.pickups[i]
		p.Draw(dst, p.X-e.cameraX+cx, p.Y-e.cameraY+cy)
	}

	// 弾（カメラ＝プレイヤーが動くので、見かけのトレイル方向にプレイヤー速度を渡す）
	for i := range e.bullets {
		b := &e.bullets[i]
		b.Draw(dst, b.X-e.cameraX+cx, b.Y-e.cameraY+cy, e.player.VX, e.player.VY, theme)
	}

	// プレイヤー
	e.player.DrawAt(dst, e.player.X-e.cameraX+cx, e.player.Y-e.cameraY+cy, theme)

	e.drawHUD(dst, theme, sw, sh)
}

func (e *Exploration) drawHUD(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	// ステータス（仮値）
	ui.DrawText(dst, "HP 100   SHIELD 100   FUEL 100", 20, 20, 1.5, theme.Line)

	// インベントリ
	inv := e.player.Inventory
	ui.DrawText(dst,
		fmt.Sprintf("IRON %d   CRYSTAL %d   ICE %d",
			inv[entity.ResourceIron], inv[entity.ResourceCrystal], inv[entity.ResourceIce]),
		20, 50, 1.5, theme.Line)

	// 速度・座標（デバッグ補助）
	speed := math.Hypot(e.player.VX, e.player.VY)
	ui.DrawText(dst,
		fmt.Sprintf("SPEED %.2f   POS %.0f, %.0f", speed, e.player.X, e.player.Y),
		20, 80, 1.2, theme.LineDim)

	// ミニマップ
	miniW, miniH := float32(180), float32(180)
	mx := float32(sw) - miniW - 20
	my := float32(sh) - miniH - 20
	vector.StrokeRect(dst, mx, my, miniW, miniH, 1, theme.Line, false)
	ui.DrawText(dst, "MINIMAP", float64(mx)+10, float64(my)+8, 1.2, theme.LineDim)
	// プレイヤー（中央点）
	vector.DrawFilledRect(dst, mx+miniW/2-1, my+miniH/2-1, 2, 2, theme.Line, false)
	// 小惑星
	for _, a := range e.asteroids {
		dx := (a.X - e.cameraX) * minimapScale
		dy := (a.Y - e.cameraY) * minimapScale
		nx := mx + miniW/2 + float32(dx)
		ny := my + miniH/2 + float32(dy)
		if nx < mx || nx > mx+miniW || ny < my || ny > my+miniH {
			continue
		}
		vector.DrawFilledRect(dst, nx-1, ny-1, 2, 2, theme.LineDim, false)
	}

	ui.DrawText(dst, "[ WASD: Move    Shift: Boost    Space: Fire    Esc: Menu ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}
