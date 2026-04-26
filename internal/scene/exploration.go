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
	// 開始時の小惑星はプレイヤー周辺に出さない。最大形状半径＋自機半径＋余白を確保。
	asteroidMinDist          = 450.0
	asteroidMaxDist          = 1100.0
	asteroidMinSize          = 4
	asteroidMaxSize          = 10
	minimapScale             = 0.06
	collisionDamageThreshold = 1.0 // この相対速度未満ではダメージなし
	collisionDamageFactor    = 3.0 // 相対速度1あたりのダメージ
	collisionRestitution     = 0.6 // バウンスのエネルギー保持率
)

// Exploration は探索画面シーン。
// プレイヤー機を中心に俯瞰描画し、小惑星・弾・資源ピックアップ・ステーションを管理する。
type Exploration struct {
	player     *entity.Player
	cameraX    float64
	cameraY    float64
	starfield  *starfield
	asteroids  []*entity.Asteroid
	bullets    []entity.Bullet
	pickups    []entity.Pickup
	stations   []*entity.Station
	activeDock *entity.Station // 現在ドック範囲内のステーション。nil なら接岸不可
}

// NewExploration は新しい探索シーンを生成し、初期小惑星とステーションを配置する。
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
	// 起点近くに 1 基の宇宙ステーションを配置（自機開始位置から見えやすい距離）
	e.stations = append(e.stations, entity.NewStation(300, -250))
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

	// 自機 ⇄ 小惑星の衝突解決（押し戻し＋反射＋ダメージ）
	e.handlePlayerAsteroidCollisions()

	// ステーションのパルス更新とドック近接判定
	e.activeDock = nil
	for _, s := range e.stations {
		s.Update()
		if e.activeDock == nil && s.IsPlayerInDock(e.player.X, e.player.Y) {
			e.activeDock = s
		}
	}

	// ドック中に Space 押下: 発射ではなくステーションメニューを開く
	if e.activeDock != nil && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		e.player.ThrustState = entity.ThrustOff
		d.Push(NewStationMenu(e.player))
		return nil
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

// handlePlayerAsteroidCollisions は自機の各パーツと各小惑星グリッドを
// 円-円判定し、重なりを解消（押し戻し）、相対速度を反射、衝突相対速度に応じて
// プレイヤーへダメージを与える。小惑星側は質量∞扱いで影響を受けない。
func (e *Exploration) handlePlayerAsteroidCollisions() {
	p := e.player
	g := float64(entity.GridSize)
	sumR := g // パーツ半径(g/2) + グリッド半径(g/2)

	// 自機パーツのワールド位置を一度算出（角度はループ中変わらず、位置はその場で加算）
	sSin, sCos := math.Sin(p.Angle), math.Cos(p.Angle)
	type partOffset struct{ ox, oy float64 }
	offsets := make([]partOffset, len(p.Parts))
	for i, part := range p.Parts {
		lx := float64(part.GX) * g
		ly := float64(part.GY) * g
		// 船体描画と同じ R(angle + π/2) ローカル→ワールド変換
		offsets[i] = partOffset{
			ox: -sSin*lx - sCos*ly,
			oy: sCos*lx - sSin*ly,
		}
	}

	for _, a := range e.asteroids {
		aSin, aCos := math.Sin(a.Angle), math.Cos(a.Angle)
		for i := range p.Parts {
			pcx := p.X + offsets[i].ox
			pcy := p.Y + offsets[i].oy

			for _, gr := range a.Grids {
				lgx := float64(gr.GX) * g
				lgy := float64(gr.GY) * g
				wgx := aCos*lgx - aSin*lgy
				wgy := aSin*lgx + aCos*lgy
				gcx := a.X + wgx
				gcy := a.Y + wgy

				dx := pcx - gcx
				dy := pcy - gcy
				dist := math.Hypot(dx, dy)
				if dist >= sumR {
					continue
				}
				if dist < 0.001 {
					dx, dy, dist = 1, 0, 1
				}
				nx := dx / dist
				ny := dy / dist
				overlap := sumR - dist

				// 重なりを解消（自機のみ動かす）
				p.X += nx * overlap
				p.Y += ny * overlap
				pcx += nx * overlap
				pcy += ny * overlap

				// 相対速度の法線成分（負なら自機が小惑星に向かっている）
				rvx := p.VX - a.VX
				rvy := p.VY - a.VY
				vNormal := rvx*nx + rvy*ny
				if vNormal >= 0 {
					continue
				}

				impactSpeed := -vNormal
				if impactSpeed > collisionDamageThreshold {
					dmg := int((impactSpeed - collisionDamageThreshold) * collisionDamageFactor)
					p.Damage(dmg)
				}

				// 法線成分のみ反射（接線成分はそのまま残す＝かすめ続けない）
				rvx -= (1 + collisionRestitution) * vNormal * nx
				rvy -= (1 + collisionRestitution) * vNormal * ny
				p.VX = a.VX + rvx
				p.VY = a.VY + rvy
			}
		}
	}
}

func (e *Exploration) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	dst.Fill(theme.Background)

	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	cx, cy := float64(sw)/2, float64(sh)/2

	e.starfield.draw(dst, e.cameraX, e.cameraY, theme)

	// 宇宙ステーション（背景扱い）
	for _, s := range e.stations {
		s.Draw(dst, s.X-e.cameraX+cx, s.Y-e.cameraY+cy, theme)
	}

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

	// プレイヤー（被弾無敵中は数フレームおきに点滅）
	psx := e.player.X - e.cameraX + cx
	psy := e.player.Y - e.cameraY + cy
	if e.player.InvulnTimer == 0 || (e.player.InvulnTimer/4)%2 == 0 {
		e.player.DrawAt(dst, psx, psy, theme)
	}

	// ドック近接プロンプト
	if e.activeDock != nil {
		prompt := "[ Space ] DOCK"
		promptScale := 1.6
		pw, _ := ui.MeasureText(prompt, promptScale)
		ui.DrawText(dst, prompt, psx-pw/2, psy+72, promptScale, theme.Line)
	}

	e.drawHUD(dst, theme, sw, sh)
}

func (e *Exploration) drawHUD(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	// ステータス
	ui.DrawText(dst,
		fmt.Sprintf("HP %d/%d   FUEL %d/%d   CR %d",
			e.player.HP, e.player.MaxHP,
			int(e.player.Fuel), int(e.player.MaxFuel),
			e.player.Credits),
		20, 20, 1.5, theme.Line)

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
	// ステーション（小さな四角で目立たせる）
	for _, s := range e.stations {
		dx := (s.X - e.cameraX) * minimapScale
		dy := (s.Y - e.cameraY) * minimapScale
		nx := mx + miniW/2 + float32(dx)
		ny := my + miniH/2 + float32(dy)
		if nx < mx || nx > mx+miniW || ny < my || ny > my+miniH {
			continue
		}
		vector.StrokeRect(dst, nx-3, ny-3, 6, 6, 1, theme.Line, false)
	}

	ui.DrawText(dst, "[ WASD: Move    Shift: Boost    Space: Fire / Dock    Esc: Menu ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}
