package scene

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/save"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	asteroidMinSize = 4
	asteroidMaxSize = 10
	// ミニマップが映す半径は 180/2/0.06=1500px だが、対角線方向の角は ~2120px まで届く。
	// 角にも掛からないよう、生成リングはこれより外側で取る。
	asteroidSpawnRingMin = 2200.0
	asteroidSpawnRingMax = 3000.0
	// プレイヤーから十分離れた小惑星は破棄し、再生成に任せる。
	asteroidCullDist         = 4000.0
	minimapScale             = 0.06
	collisionDamageThreshold = 1.0 // この相対速度未満ではダメージなし
	collisionDamageFactor    = 3.0 // 相対速度1あたりのダメージ
	collisionRestitution     = 0.6 // バウンスのエネルギー保持率
	// ゾーン中心へ寄せる初速。漂流上限の範囲内に収まる小さな値。
	asteroidInboundDrift = 0.2
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
	world      *entity.World
	spawnRng   *rand.Rand
	lastMap    *entity.FullMap // 最後に入った FullMap。区画外でも保持し、全体マップ表示の対象とする

	// AutoAim 制御: 最後に弾が当たった小惑星をターゲットとし、
	// 各 AutoAim パーツが射程内なら毎フレーム DPS を合算してダメージを与える。
	autoAimTarget  *entity.Asteroid
	autoAimGridIdx int
	autoAimDmgAcc  float64
	autoAimBeams   []autoAimBeam // 当該フレームに発射中のビーム（描画用）

	// プレイ時間（秒）。ロード時は保存値から再開、新規ゲームは 0。
	// 60fps 想定で毎フレーム 1/60 ずつ加算する。
	playtime float64

	// ワープ進行状態。warpTimer > 0 の間は通常の Update をスキップしてアニメ専用に。
	warpTimer int
	warpDest  *entity.FullMap
}

const (
	// ワープアニメ全体のフレーム数（60fps で 1.5 秒）。
	warpDuration = 90
)

// CurrentMapName は現在いる FullMap 名を返す（区画外なら空文字）。
func (e *Exploration) CurrentMapName() string {
	if e.lastMap == nil {
		return ""
	}
	return e.lastMap.Name
}

// Playtime は累計プレイ時間（秒）を返す。
func (e *Exploration) Playtime() float64 { return e.playtime }

// Player はメニュー等から現在のプレイヤーを参照するためのアクセサ。
func (e *Exploration) Player() *entity.Player { return e.player }

// autoAimBeam は1パーツ → 対象グリッドのビーム描画情報。
type autoAimBeam struct {
	fromX, fromY float64
	toX, toY     float64
}

// NewExploration は新規ゲーム用の探索シーンを生成する（Pebble 初期構成、playtime=0）。
func NewExploration() *Exploration {
	return NewExplorationFromPlayer(entity.NewPlayerPebble(), 0)
}

// NewExplorationFromPlayer は指定の Player（セーブから復元したものなど）と
// 累計プレイ時間で探索シーンを生成する。World 定義は固定で entity.DefaultWorld()。
// 小惑星はゾーン定義に従って実行時に逐次スポーンされるため、開始時には生成しない。
func NewExplorationFromPlayer(p *entity.Player, playtime float64) *Exploration {
	e := &Exploration{
		player:         p,
		starfield:      newStarfield(1),
		world:          entity.DefaultWorld(),
		spawnRng:       rand.New(rand.NewSource(2)),
		autoAimGridIdx: -1,
		playtime:       playtime,
	}
	// 各 FullMap の中心にステーションを配置（恒星マップ／ワープ先選択でも参照される）
	for i := range e.world.Maps {
		m := &e.world.Maps[i]
		e.stations = append(e.stations, entity.NewStation(m.CX, m.CY))
	}
	// 開始時点でいる FullMap を記録（区画外なら nil）
	e.lastMap = e.world.Containing(e.player.X, e.player.Y)
	return e
}

// applyAutoAim は autoAimTarget に対して AutoAim パーツの継続ダメージを 1 フレーム分適用する。
// 各パーツのワールド位置から対象グリッドまで AutoAimRange 以内ならビームを発射し、
// 全パーツの DPS を合算して dmgAcc に蓄積、>= 1 で整数ダメージとしてグリッドに与える。
// グリッドが破壊されたら同小惑星の最寄グリッドに再ターゲットする。
// 描画用に autoAimBeams を毎フレーム書き換える。
func (e *Exploration) applyAutoAim() {
	e.autoAimBeams = e.autoAimBeams[:0]
	a := e.autoAimTarget
	if a == nil || len(a.Grids) == 0 {
		e.autoAimTarget = nil
		return
	}
	// 対象グリッドの選択（無効なら最寄を選ぶ）
	if e.autoAimGridIdx < 0 || e.autoAimGridIdx >= len(a.Grids) {
		e.autoAimGridIdx = a.NearestGridIdx(e.player.X, e.player.Y)
		if e.autoAimGridIdx < 0 {
			e.autoAimTarget = nil
			return
		}
	}
	gx, gy, ok := a.GridWorldPos(e.autoAimGridIdx)
	if !ok {
		e.autoAimTarget = nil
		return
	}

	// 各 AutoAim パーツのワールド位置と射程チェック
	p := e.player
	g := float64(entity.GridSize)
	sSin, sCos := math.Sin(p.Angle), math.Cos(p.Angle)
	dpsSum := 0.0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != entity.PartAutoAim {
			continue
		}
		lx := float64(part.GX) * g
		ly := float64(part.GY) * g
		// 船体描画と同じ R(angle + π/2) ローカル → ワールド変換
		px := p.X + (-sSin*lx - sCos*ly)
		py := p.Y + (sCos*lx - sSin*ly)
		if math.Hypot(gx-px, gy-py) > d.AutoAimRange {
			continue
		}
		dpsSum += d.AutoAimDPS
		e.autoAimBeams = append(e.autoAimBeams, autoAimBeam{
			fromX: px, fromY: py, toX: gx, toY: gy,
		})
	}
	if dpsSum <= 0 {
		return
	}

	e.autoAimDmgAcc += dpsSum / 60.0
	if e.autoAimDmgAcc < 1 {
		return
	}
	dmg := int(e.autoAimDmgAcc)
	e.autoAimDmgAcc -= float64(dmg)
	destroyed, pk, hitOk := a.HitGrid(e.autoAimGridIdx, dmg)
	if !hitOk {
		e.autoAimGridIdx = -1
		return
	}
	if destroyed {
		e.pickups = append(e.pickups, pk)
		if len(a.Grids) == 0 {
			e.autoAimTarget = nil
			e.autoAimGridIdx = -1
			return
		}
		e.autoAimGridIdx = a.NearestGridIdx(e.player.X, e.player.Y)
	}
}

// trySpawnAsteroid は現フレームの生成上限に達していなければ、
// ミニマップ外のリング上で全体マップ内・ゾーン内の点を選んで小惑星を 1 つ追加する。
// 生成位置で重なるゾーンの素材重みを合算して 1 素材を抽選する。
func (e *Exploration) trySpawnAsteroid() {
	cap := e.world.SpawnCap(e.player.X, e.player.Y)
	if len(e.asteroids) >= cap {
		return
	}
	for tries := 0; tries < 8; tries++ {
		ang := e.spawnRng.Float64() * math.Pi * 2
		dist := asteroidSpawnRingMin + e.spawnRng.Float64()*(asteroidSpawnRingMax-asteroidSpawnRingMin)
		x := e.player.X + math.Cos(ang)*dist
		y := e.player.Y + math.Sin(ang)*dist
		if !e.world.InBounds(x, y) {
			continue
		}
		res, ok := e.world.PickResource(x, y, e.spawnRng)
		if !ok {
			continue
		}
		size := asteroidMinSize + e.spawnRng.Intn(asteroidMaxSize-asteroidMinSize+1)
		a := entity.NewAsteroid(e.spawnRng.Int63(), x, y, size, res)
		// 生成直後にプレイヤー方向へ軽く寄せ、ミニマップ内に流入してくる挙動を作る
		toX, toY := e.player.X-x, e.player.Y-y
		if d := math.Hypot(toX, toY); d > 0 {
			a.VX += (toX / d) * asteroidInboundDrift
			a.VY += (toY / d) * asteroidInboundDrift
		}
		e.asteroids = append(e.asteroids, a)
		return
	}
}

func (e *Exploration) Update(d Director) error {
	// ワープ中は専用アニメだけ進め、入力はすべて無視
	if e.warpTimer > 0 {
		e.tickWarp()
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// メニュー中はアフターバーナーが残らないよう推力状態をリセット
		e.player.ThrustState = entity.ThrustOff
		d.Push(NewMenu(save.Context{
			Player:   e.player,
			Playtime: e.playtime,
			MapName:  e.CurrentMapName(),
		}))
		return nil
	}

	// 全体マップ（最後に入った FullMap を確認）
	if inpututil.IsKeyJustPressed(ebiten.KeyM) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		e.player.ThrustState = entity.ThrustOff
		d.Push(NewWorldMapView(e.lastMap, e.stations, e.player.X, e.player.Y, e.player.Angle))
		return nil
	}

	// 恒星マップ → ワープ
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		e.player.ThrustState = entity.ThrustOff
		current := e.CurrentMapName()
		d.Push(NewStarMap(e.world, current, func(d Director, dest *entity.FullMap) bool {
			e.startWarp(dest)
			return true
		}))
		return nil
	}

	e.player.Update()
	e.playtime += 1.0 / 60.0 // ebitengine 既定 TPS（60）想定の累計プレイ時間

	// 現在いる FullMap を更新（区画外なら直前の値を保持）
	if m := e.world.Containing(e.player.X, e.player.Y); m != nil {
		e.lastMap = m
	}

	// ゾーンに応じた小惑星のスポーン（フレームあたり最大 1 体）
	e.trySpawnAsteroid()

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
	// 命中時はその小惑星を AutoAim のターゲットに設定する。
	for i := len(e.bullets) - 1; i >= 0; i-- {
		b := &e.bullets[i]
		for _, a := range e.asteroids {
			absorbed, drops := a.Hit(b.X, b.Y, b.Damage)
			if !absorbed {
				continue
			}
			e.pickups = append(e.pickups, drops...)
			e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
			if e.autoAimTarget != a {
				e.autoAimTarget = a
				e.autoAimGridIdx = -1 // 次フレームで最寄グリッドを選び直す
				e.autoAimDmgAcc = 0
			}
			break
		}
	}

	// AutoAim パーツによる継続ダメージ（ビームの描画情報も生成）
	e.applyAutoAim()

	// 空・遠方の小惑星を除去（遠方は再生成に任せ、ミニマップ外で滞留させない）
	na := 0
	for _, a := range e.asteroids {
		if a.Empty() {
			continue
		}
		if math.Hypot(a.X-e.player.X, a.Y-e.player.Y) > asteroidCullDist {
			continue
		}
		e.asteroids[na] = a
		na++
	}
	e.asteroids = e.asteroids[:na]

	// AutoAim ターゲットがリストから消えていたらクリア（破壊済 or カル済）
	if e.autoAimTarget != nil {
		found := false
		for _, a := range e.asteroids {
			if a == e.autoAimTarget {
				found = true
				break
			}
		}
		if !found {
			e.autoAimTarget = nil
			e.autoAimGridIdx = -1
			e.autoAimDmgAcc = 0
		}
	}

	// ピックアップの更新（吸引・回収・寿命切れ）
	// 積載超過なら回収を拒否し、自機から外側へ少し弾いて吸引ループを抜ける。
	np := 0
	for i := range e.pickups {
		p := &e.pickups[i]
		if p.Update(e.player.X, e.player.Y) {
			if e.player.AddResource(p.Resource, 1) {
				continue
			}
			dx := p.X - e.player.X
			dy := p.Y - e.player.Y
			d := math.Hypot(dx, dy)
			if d < 0.001 {
				dx, dy, d = 1, 0, 1
			}
			const rejectPush = 6.0
			p.VX += dx / d * rejectPush
			p.VY += dy / d * rejectPush
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

	// 背景天体: 各 FullMap の Body を、星空の手前・ステーションの奥に大きく描画する。
	// 区画外（lastMap=nil）では描かない。プレイ可能領域 ±30000 内なら近隣区画の Body も視野に入る。
	for i := range e.world.Maps {
		m := &e.world.Maps[i]
		drawCelestialBackdrop(dst, &m.Body, m.CX, m.CY, e.cameraX, e.cameraY, cx, cy)
	}

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

	// AutoAim ビーム（パーツ → 対象グリッド）
	beamColor := color.NRGBA{0xff, 0xc0, 0x40, 0xff}
	for _, b := range e.autoAimBeams {
		x1 := float32(b.fromX - e.cameraX + cx)
		y1 := float32(b.fromY - e.cameraY + cy)
		x2 := float32(b.toX - e.cameraX + cx)
		y2 := float32(b.toY - e.cameraY + cy)
		vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, beamColor, false)
	}

	// プレイヤー（被弾無敵中は数フレームおきに点滅）
	psx := e.player.X - e.cameraX + cx
	psy := e.player.Y - e.cameraY + cy
	if e.player.InvulnTimer == 0 || (e.player.InvulnTimer/4)%2 == 0 {
		e.player.DrawAt(dst, psx, psy, theme)
		// シールドが 1 以上なら、外周（隣接面以外）を点滅描画
		if e.player.ShieldHP > 0 {
			e.player.Ship.DrawShieldOutline(dst, psx, psy, theme)
		}
	}

	// ドック近接プロンプト
	if e.activeDock != nil {
		prompt := "[ Space ] DOCK"
		promptScale := 1.6
		pw, _ := ui.MeasureText(prompt, promptScale)
		ui.DrawText(dst, prompt, psx-pw/2, psy+72, promptScale, theme.Line)
	}

	// ワープ中: 線流れ + ホワイトアウトを最前面に重ね、HUD を非表示にする
	if e.warpTimer > 0 {
		e.drawWarpOverlay(dst, theme, sw, sh)
		return
	}

	e.drawHUD(dst, theme, sw, sh)
}

func (e *Exploration) drawHUD(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	// ステータス（シールドは MaxShieldHP > 0 のときだけ表示）
	statusLine := fmt.Sprintf("HP %d/%d", e.player.HP, e.player.MaxHP)
	if e.player.MaxShieldHP > 0 {
		statusLine += fmt.Sprintf("   SH %d/%d", e.player.ShieldHP, e.player.MaxShieldHP)
	}
	statusLine += fmt.Sprintf("   FUEL %d/%d   CARGO %.0f/%.0f   CR %d",
		int(e.player.Fuel), int(e.player.MaxFuel),
		e.player.CargoLoad(), e.player.MaxCargo,
		e.player.Credits)
	ui.DrawText(dst, statusLine, 20, 20, 1.5, theme.Line)

	// インベントリ
	inv := e.player.Inventory
	ui.DrawText(dst,
		fmt.Sprintf("IRON %d   BRONZE %d   ICE %d",
			inv[entity.ResourceIron], inv[entity.ResourceBronze], inv[entity.ResourceIce]),
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
	// 不透明の黒背景で星空・小惑星を覆う
	vector.DrawFilledRect(dst, mx, my, miniW, miniH, color.NRGBA{0, 0, 0, 255}, false)
	vector.StrokeRect(dst, mx, my, miniW, miniH, 1, theme.Line, false)
	// プレイヤー（中央点）
	vector.DrawFilledRect(dst, mx+miniW/2-1, my+miniH/2-1, 2, 2, theme.Line, false)
	// 小惑星（1 個 = 1 素材で構成されているので、先頭グリッドの素材色で描画）
	for _, a := range e.asteroids {
		if len(a.Grids) == 0 {
			continue
		}
		dx := (a.X - e.cameraX) * minimapScale
		dy := (a.Y - e.cameraY) * minimapScale
		nx := mx + miniW/2 + float32(dx)
		ny := my + miniH/2 + float32(dy)
		if nx < mx || nx > mx+miniW || ny < my || ny > my+miniH {
			continue
		}
		c := a.Grids[0].Resource.Info().Color
		vector.DrawFilledRect(dst, nx-1, ny-1, 2, 2, c, false)
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

	ui.DrawText(dst, "[ WASD: Move    Shift: Boost    Space: Fire/Dock    M: Map    N: Warp    Esc: Menu ]",
		20, float64(sh)-30, 1.5, theme.LineDim)
}

// startWarp はワープ確定時に呼ばれる。自機を目的地方向に向け、速度をリセットして
// アニメ用タイマーを起動する。実際のテレポートはアニメ中点で行う。
//
// 機首角度は「現在の FullMap 中心 → 目的地 FullMap 中心」の世界座標差から計算する。
// 恒星マップは同じ世界座標を使ってレイアウトしているため、星マップで見た方向と一致する。
func (e *Exploration) startWarp(dest *entity.FullMap) {
	if dest == nil {
		return
	}
	var dx, dy float64
	if e.lastMap != nil {
		dx = dest.CX - e.lastMap.CX
		dy = dest.CY - e.lastMap.CY
	} else {
		// 区画外: 現在地 → 目的地 で代替
		dx = dest.CX - e.player.X
		dy = dest.CY - e.player.Y
	}
	if dx != 0 || dy != 0 {
		e.player.Angle = math.Atan2(dy, dx)
	}
	e.player.VX = 0
	e.player.VY = 0
	e.player.ThrustState = entity.ThrustOff
	e.warpDest = dest
	e.warpTimer = warpDuration
}

// tickWarp は warpTimer > 0 の間、ワープアニメを 1 フレーム進める。
// 中点フレームで実際の座標移動と一時状態のクリアを行う。
func (e *Exploration) tickWarp() {
	e.warpTimer--

	if e.warpTimer == warpDuration/2 {
		dest := e.warpDest
		if dest != nil {
			e.player.X = dest.CX
			e.player.Y = dest.CY
			e.player.VX = 0
			e.player.VY = 0
			e.lastMap = dest
			// ワープ前の局所状態（小惑星・ピックアップ・弾・自動照準）は持ち越さない
			e.asteroids = e.asteroids[:0]
			e.pickups = e.pickups[:0]
			e.bullets = e.bullets[:0]
			e.autoAimTarget = nil
			e.autoAimGridIdx = -1
			e.autoAimDmgAcc = 0
		}
	}

	e.cameraX = e.player.X
	e.cameraY = e.player.Y

	if e.warpTimer == 0 {
		e.warpDest = nil
	}
}

// drawWarpOverlay はワープアニメの線流れ + ホワイトアウトを描画する。
// 各線は始点 (cx, cy) から機首方向（前方）に streakLen だけ伸ばす。
// これで線の向きが自機の向きと一致する。ホワイトアウトは中点でピーク。
func (e *Exploration) drawWarpOverlay(dst *ebiten.Image, theme *ui.Theme, sw, sh int) {
	progress := float64(warpDuration-e.warpTimer) / float64(warpDuration)
	pulse := math.Sin(progress * math.Pi) // 0 -> 1 -> 0

	// 機首方向ベクトル（前方）
	streakLen := 60.0 + 280.0*pulse
	forwardX := math.Cos(e.player.Angle) * streakLen
	forwardY := math.Sin(e.player.Angle) * streakLen

	// 決定論的に線をばらまく（フレーム番号からシード）
	rng := rand.New(rand.NewSource(int64(warpDuration-e.warpTimer) * 7919))
	const numStreaks = 70
	for i := 0; i < numStreaks; i++ {
		cx := rng.Float64() * float64(sw)
		cy := rng.Float64() * float64(sh)
		x1 := float32(cx)
		y1 := float32(cy)
		x2 := float32(cx + forwardX)
		y2 := float32(cy + forwardY)
		vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, theme.Line, false)
	}

	// ホワイトアウト（sin で中点ピーク）
	alpha := uint8(pulse * 255)
	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh), color.NRGBA{255, 255, 255, alpha}, false)
}

// drawCelestialBackdrop は天体を (mapCX, mapCY) + Backdrop オフセットの位置に
// 円として描画する。BackdropRadius が 0 の場合は何もしない。
// 描画は半透明の塗り + 縁線で「背景にある巨大な球体」感を出す。
func drawCelestialBackdrop(dst *ebiten.Image, body *entity.Celestial,
	mapCX, mapCY, cameraX, cameraY, screenCX, screenCY float64) {
	if body == nil || body.BackdropRadius <= 0 {
		return
	}
	wx := mapCX + body.BackdropOffsetX
	wy := mapCY + body.BackdropOffsetY
	sx := float32(wx - cameraX + screenCX)
	sy := float32(wy - cameraY + screenCY)
	r := float32(body.BackdropRadius)
	// 視界外なら早期スキップ（半径分のマージン込み）
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	if sx+r < 0 || sx-r > float32(sw) || sy+r < 0 || sy-r > float32(sh) {
		return
	}
	// 本体: 半透明塗り
	fill := body.Color
	fill.A = 110
	vector.DrawFilledCircle(dst, sx, sy, r, fill, true)
	// 輪郭: 不透明
	vector.StrokeCircle(dst, sx, sy, r, 1.5, body.Color, true)
}
