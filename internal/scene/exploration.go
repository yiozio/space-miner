package scene

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assetimage "github.com/yiozio/space-miner/internal/asset/image"
	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/dialog"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/i18n"
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
	mines      []entity.Mine
	drones     []entity.Drone
	droneBeams []droneBeam // 当該フレームに発射中のドローン攻撃ビーム（描画用）
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

	// 海賊
	pirates        []*entity.Pirate
	pirateSpawnRng *rand.Rand

	// 着弾エフェクト
	impacts []entity.Impact
	// 爆発エフェクト（海賊撃墜時など）
	explosions []entity.Explosion
	// 瞬間命中レーザーのビーム可視化（数フレーム描画）
	beams []entity.Beam

	// プレイ時間（秒）。ロード時は保存値から再開、新規ゲームは 0。
	// 60fps 想定で毎フレーム 1/60 ずつ加算する。
	playtime float64

	// ワープ進行状態。warpTimer > 0 の間は通常の Update をスキップしてアニメ専用に。
	warpTimer int
	warpDest  *entity.FullMap

	// 回転音制御。A/D 押下でフェードイン → 持続ループ → 終端でフェードアウト。
	rotationSound *sound.RotationSound
	// バーナー音制御。推進中はイントロ→ループ再生、停止で再生終了。
	burnerSound *sound.BurnerSound

	// たまに鳴らす「ポポポ」ビープの残フレームと専用乱数（演出用、ゲーム性に無影響）。
	beepTimer int
	beepRng   *rand.Rand
}

const (
	// ワープアニメ全体のフレーム数（60fps で 1.5 秒）。
	warpDuration = 90
	// たまに鳴らすビープの出現間隔（フレーム）。60fps で約 10〜25 秒。
	beepIntervalMinFrames = 600
	beepIntervalMaxFrames = 1500
	// 海賊の1フレームあたり出現確率。上限未満でもこの確率でしか湧かないため、
	// 倒した直後に矢継ぎ早に出ず、平均 ~1.7 秒間隔（@60fps）で増えていく。
	pirateSpawnChancePerFrame = 0.01
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

// droneBeam はドローン → 対象のビーム描画情報。hostile で色を切り替える。
type droneBeam struct {
	fromX, fromY float64
	toX, toY     float64
	hostile      bool
}

// NewExploration は新規ゲーム用の探索シーンを生成する（Pebble 初期構成、playtime=0）。
func NewExploration() *Exploration {
	return NewExplorationFromPlayer(entity.NewPlayerPebble(), 0)
}

// NewExplorationFromPlayer は指定の Player（セーブから復元したものなど）と
// 累計プレイ時間で探索シーンを生成する。World 定義は固定で entity.DefaultWorld()。
// 小惑星はゾーン定義に従って実行時に逐次スポーンされるため、開始時には生成しない。
func NewExplorationFromPlayer(p *entity.Player, playtime float64) *Exploration {
	sound.StopBGM() // タイトル BGM を止めてゲーム本編へ
	e := &Exploration{
		player:         p,
		starfield:      newStarfield(1),
		world:          entity.DefaultWorld(),
		spawnRng:       rand.New(rand.NewSource(2)),
		pirateSpawnRng: rand.New(rand.NewSource(3)),
		autoAimGridIdx: -1,
		playtime:       playtime,
		rotationSound:  sound.NewRotationSound(),
		burnerSound:    sound.NewBurnerSound(),
		beepRng:        rand.New(rand.NewSource(4)),
	}
	e.beepTimer = beepIntervalMinFrames + e.beepRng.Intn(beepIntervalMaxFrames-beepIntervalMinFrames)
	// 各 FullMap の中心にステーションを配置（恒星マップ／ワープ先選択でも参照される）
	for i := range e.world.Maps {
		m := &e.world.Maps[i]
		e.stations = append(e.stations, entity.NewStation(m.Name, m.CX, m.CY))
	}
	// 開始時点でいる FullMap を記録。区画外なら最寄マップを採用し、
	// lastMap が nil にならないようにする（区画外でも最後／最寄のマップを表示できる）。
	e.lastMap = e.world.Containing(e.player.X, e.player.Y)
	if e.lastMap == nil {
		e.lastMap = e.world.NearestMap(e.player.X, e.player.Y)
	}
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
	sSin, sCos := math.Sin(p.Angle), math.Cos(p.Angle)
	dpsSum := 0.0
	for _, part := range p.Parts {
		d := part.Def()
		if d == nil || d.Kind != entity.PartAutoAim {
			continue
		}
		lx, ly := entity.PartLocalCenter(part.GX, part.GY)
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
		sound.PlayAsteroidBreak()
		if len(a.Grids) == 0 {
			e.autoAimTarget = nil
			e.autoAimGridIdx = -1
			return
		}
		e.autoAimGridIdx = a.NearestGridIdx(e.player.X, e.player.Y)
	}
}

// updateDrones は設置ドローンの寿命を進め、生存ドローンごとに最寄の対象へ攻撃させる。
// 寿命切れのドローンは取り除く。描画用に droneBeams を毎フレーム書き換える。
func (e *Exploration) updateDrones() {
	e.droneBeams = e.droneBeams[:0]
	nd := 0
	for i := range e.drones {
		d := &e.drones[i]
		if d.Update() {
			continue // 寿命切れで消滅
		}
		e.droneAttack(d)
		e.drones[nd] = *d
		nd++
	}
	e.drones = e.drones[:nd]
}

// droneAttack は 1 体のドローンの攻撃を 1 フレーム分処理する。
// Hostile（敵機設置）なら自機を、そうでなければ射程内で最も近い小惑星 or 海賊を狙い、
// DPS 分の継続ダメージを与える。攻撃したフレームはビーム描画情報を droneBeams に追加する。
func (e *Exploration) droneAttack(d *entity.Drone) {
	// 敵機が設置したドローンは自機を狙う。
	if d.Hostile {
		if math.Hypot(e.player.X-d.X, e.player.Y-d.Y) > d.Range {
			return // 射程外
		}
		if d.Mode == entity.DroneBullet {
			// 弾型: 自機へ弾を撃つ（命中判定は敵弾と同じ経路）。
			if d.Fire() {
				e.bullets = append(e.bullets, d.FireBullet(e.player.X, e.player.Y))
			}
			return
		}
		// ビーム型: 必中の継続ダメージ。
		e.droneBeams = append(e.droneBeams, droneBeam{
			fromX: d.X, fromY: d.Y, toX: e.player.X, toY: e.player.Y, hostile: true,
		})
		if dmg := d.TickDamage(); dmg > 0 {
			e.playDamageSound()
			e.player.Damage(dmg)
		}
		return
	}

	bestDist := d.Range
	var targetAst *entity.Asteroid
	targetGrid := -1
	var targetPirate *entity.Pirate
	var tx, ty float64

	// 最寄の小惑星グリッド
	for _, a := range e.asteroids {
		idx := a.NearestGridIdx(d.X, d.Y)
		if idx < 0 {
			continue
		}
		gx, gy, ok := a.GridWorldPos(idx)
		if !ok {
			continue
		}
		if dist := math.Hypot(gx-d.X, gy-d.Y); dist < bestDist {
			bestDist = dist
			targetAst, targetGrid, tx, ty = a, idx, gx, gy
			targetPirate = nil
		}
	}
	// 最寄の海賊（小惑星より近ければ優先）
	for _, pr := range e.pirates {
		if pr.HP <= 0 {
			continue
		}
		if dist := math.Hypot(pr.X-d.X, pr.Y-d.Y); dist < bestDist {
			bestDist = dist
			targetPirate, tx, ty = pr, pr.X, pr.Y
			targetAst, targetGrid = nil, -1
		}
	}

	if targetAst == nil && targetPirate == nil {
		return // 射程内に対象なし
	}

	if d.Mode == entity.DroneBullet {
		// 弾型: 対象の現在位置へ弾を撃つ（命中判定は通常弾と同じ経路）。
		if d.Fire() {
			e.bullets = append(e.bullets, d.FireBullet(tx, ty))
		}
		return
	}

	// ビーム型: 描画情報（ダメージの有無に関わらず狙っている間は描く）＋必中の継続ダメージ
	e.droneBeams = append(e.droneBeams, droneBeam{fromX: d.X, fromY: d.Y, toX: tx, toY: ty})

	dmg := d.TickDamage()
	if dmg <= 0 {
		return
	}
	if targetPirate != nil {
		targetPirate.TakeHit(dmg) // 撃破処理は cullPiratesAndDrop が担う
		return
	}
	destroyed, pk, ok := targetAst.HitGrid(targetGrid, dmg)
	if ok && destroyed {
		e.pickups = append(e.pickups, pk)
		sound.PlayAsteroidBreak()
	}
}

// trySpawnPirate は現フレームの海賊出現上限に達していなければ、
// ミニマップ外のリング上で PirateZone 内の点を選んで海賊を 1 体追加する。
func (e *Exploration) trySpawnPirate() {
	cap := e.world.PirateSpawnCap(e.player.X, e.player.Y)
	if len(e.pirates) >= cap {
		return
	}
	// 上限未満でも確率で間引く（倒した直後に矢継ぎ早に湧かないように）。
	if e.pirateSpawnRng.Float64() >= pirateSpawnChancePerFrame {
		return
	}
	first := len(e.pirates) == 0 // 0 体→出現の瞬間だけ警告音を鳴らす
	for tries := 0; tries < 8; tries++ {
		ang := e.pirateSpawnRng.Float64() * math.Pi * 2
		dist := asteroidSpawnRingMin + e.pirateSpawnRng.Float64()*(asteroidSpawnRingMax-asteroidSpawnRingMin)
		x := e.player.X + math.Cos(ang)*dist
		y := e.player.Y + math.Sin(ang)*dist
		if !e.world.InBounds(x, y) {
			continue
		}
		patternID, ok := e.world.PickPiratePattern(x, y, e.pirateSpawnRng)
		if !ok {
			continue
		}
		def := entity.PiratePatternByID(patternID)
		if def == nil {
			continue
		}
		e.pirates = append(e.pirates, entity.NewPirate(x, y, e.player.X, e.player.Y, def))
		if first {
			sound.PlayWarning()
		}
		return
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

// playDamageSound はシールドの有無で被ダメージ音を出し分ける。
// シールドが残っていれば「ダンッ」、無ければより低く少し長い破裂音を鳴らす。
// Damage 適用でシールド値が減る前に呼ぶこと。
func (e *Exploration) playDamageSound() {
	if e.player.ShieldHP > 0 {
		sound.PlayDamage()
	} else {
		sound.PlayDamageBurst()
	}
}

// playFireSound は発射音の種類に対応する効果音を再生する。
func playFireSound(s entity.GunFireSound) {
	switch s {
	case entity.FireSoundBurst:
		sound.PlayFireBurst()
	case entity.FireSoundZap:
		sound.PlayFireZap()
	case entity.FireSoundLaser:
		sound.PlayFireLaser()
	}
}

// isRotationKeyPressed は Player.Update と同じ判定で回転入力中かを返す。
// 回転音の同期にだけ用いる。
func isRotationKeyPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyA) ||
		ebiten.IsKeyPressed(ebiten.KeyArrowLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyD) ||
		ebiten.IsKeyPressed(ebiten.KeyArrowRight)
}

func (e *Exploration) Update(d Director) error {
	// HP 0 ならゲームオーバー画面を被せ、以降の処理はスキップする。
	// GameOver が最上位にいる間 Exploration.Update は呼ばれないため二重 Push にならない。
	if e.player.HP <= 0 {
		e.player.ThrustState = entity.ThrustOff
		e.rotationSound.Stop()
		e.burnerSound.Stop()
		sound.StopBGM()
		d.Push(NewGameOver())
		return nil
	}

	// ワープ中は専用アニメだけ進め、入力はすべて無視。
	// 回転音を流用し、前半は回転開始(intro)、後半（テレポート以降）は回転終了(outro)を流す。
	if e.warpTimer > 0 {
		e.burnerSound.Stop()
		e.rotationSound.Update(e.warpTimer > warpDuration/2)
		e.tickWarp()
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		// メニュー中はアフターバーナーが残らないよう推力状態をリセット
		e.player.ThrustState = entity.ThrustOff
		e.rotationSound.Stop()
		e.burnerSound.Stop()
		sound.StopBGM()
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
		e.rotationSound.Stop()
		e.burnerSound.Stop()
		sound.StopBGM()
		d.Push(NewWorldMapView(e.lastMap, e.stations, e.player.X, e.player.Y, e.player.Angle))
		return nil
	}

	// 恒星マップ → ワープ（Warp パーツ未搭載でも閲覧は可。確定は搭載時のみ）
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		e.player.ThrustState = entity.ThrustOff
		e.rotationSound.Stop()
		e.burnerSound.Stop()
		sound.StopBGM()
		canWarp := e.player.HasWarpDrive()
		current := e.CurrentMapName()
		d.Push(NewStarMap(e.world, current, canWarp, func(d Director, dest *entity.FullMap) bool {
			e.startWarp(dest)
			return true
		}))
		return nil
	}

	e.player.Update()
	e.player.PushTrail()
	e.rotationSound.Update(isRotationKeyPressed())
	e.burnerSound.Update(e.player.ThrustState != entity.ThrustOff, e.player.ThrustState == entity.ThrustBoost)
	sound.PlayGameBGM() // 探索中はゲーム BGM をループ（メニュー/ステーションでは StopBGM）
	// たまに「ポポポ」ビープ（残響付き）を鳴らす。探索アクティブ時のみ進む。
	if e.beepTimer <= 0 {
		sound.PlayBeeps()
		e.beepTimer = beepIntervalMinFrames + e.beepRng.Intn(beepIntervalMaxFrames-beepIntervalMinFrames)
	} else {
		e.beepTimer--
	}
	e.playtime += 1.0 / 60.0 // ebitengine 既定 TPS（60）想定の累計プレイ時間

	// 現在いる FullMap を更新（区画外なら直前の値を保持）
	if m := e.world.Containing(e.player.X, e.player.Y); m != nil {
		e.lastMap = m
	}

	// ゾーンに応じた小惑星のスポーン（フレームあたり最大 1 体）
	e.trySpawnAsteroid()
	e.trySpawnPirate()

	// 小惑星の浮遊・自転
	for _, a := range e.asteroids {
		a.Update()
	}

	// 海賊 AI: 各機が旋回・追跡・発射し、敵弾・レーザー要求・設置ドローンを処理する
	for _, pr := range e.pirates {
		bullets, lasers, drones := pr.Update(e.player.X, e.player.Y)
		pr.PushTrail()
		if len(bullets) > 0 {
			e.bullets = append(e.bullets, bullets...)
		}
		for _, l := range lasers {
			e.fireLaser(l)
		}
		if len(drones) > 0 {
			e.drones = append(e.drones, drones...)
		}
	}

	// 自機 ⇄ 小惑星の衝突解決（押し戻し＋反射＋ダメージ）
	e.handlePlayerAsteroidCollisions()

	// 自機 ⇄ 海賊の衝突解決（押し戻し＋反射＋相互ダメージ）
	e.handlePlayerPirateCollisions()

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
		e.rotationSound.Stop()
		e.burnerSound.Stop()
		sound.StopBGM()
		stationName := e.activeDock.Name
		// 初回入船時は専用スクリプトを上に重ねる（閉じるとステーションメニューに戻る）
		firstVisit := !e.player.VisitedStations[stationName]
		if firstVisit {
			if e.player.VisitedStations == nil {
				e.player.VisitedStations = make(map[string]bool)
			}
			e.player.VisitedStations[stationName] = true
		}
		// 入場時オートセーブ（VisitedStations 更新後に取り、初回会話状態も保存に含める）
		if err := save.Save(save.AutoSlot, save.Context{
			Player:   e.player,
			Playtime: e.playtime,
			MapName:  stationName,
		}); err != nil {
			log.Printf("auto-save on dock %s: %v", stationName, err)
		}
		d.Push(NewStationMenu(e.player, e.world, stationName))
		if firstVisit {
			if script := dialog.ScriptForStation(stationName); script != nil {
				d.Push(NewDialogScene(script))
			}
		}
		return nil
	}

	// 発射（押しっぱなしでクールダウン許可分だけ発射）
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		bullets, lasers, mines, drones, fireSounds := e.player.Shoot()
		if len(bullets) > 0 {
			e.bullets = append(e.bullets, bullets...)
		}
		if len(mines) > 0 {
			e.mines = append(e.mines, mines...)
		}
		if len(drones) > 0 {
			e.drones = append(e.drones, drones...)
		}
		for _, l := range lasers {
			e.fireLaser(l)
		}
		for _, s := range fireSounds {
			playFireSound(s)
		}
	}

	// 着弾エフェクトの更新（寿命切れを除去）
	{
		ni := 0
		for i := range e.impacts {
			eff := &e.impacts[i]
			if eff.Update() {
				continue
			}
			e.impacts[ni] = *eff
			ni++
		}
		e.impacts = e.impacts[:ni]
	}
	// 爆発エフェクトの更新（寿命切れを除去）
	{
		ni := 0
		for i := range e.explosions {
			eff := &e.explosions[i]
			if eff.Update() {
				continue
			}
			e.explosions[ni] = *eff
			ni++
		}
		e.explosions = e.explosions[:ni]
	}
	// レーザービームの更新（寿命切れを除去）
	{
		nb := 0
		for i := range e.beams {
			beam := &e.beams[i]
			if beam.Update() {
				continue
			}
			e.beams[nb] = *beam
			nb++
		}
		e.beams = e.beams[:nb]
	}

	// 機雷の更新（信管を進め、起爆したら 6 方向へ弾を放ち、エフェクトと音を出す）
	{
		nm := 0
		for i := range e.mines {
			m := &e.mines[i]
			if m.Update() {
				e.bullets = append(e.bullets, m.Detonate()...)
				e.spawnImpact(m.X, m.Y, false)
				sound.PlayExplosion()
				continue
			}
			e.mines[nm] = *m
			nm++
		}
		e.mines = e.mines[:nm]
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

	// 爆発弾の着弾処理（範囲ダメージ）。通常の単体命中ループより前に処理する。
	e.handleExplosiveBullets()

	// 弾 vs 小惑星（衝突したら弾を消し、破壊グリッドからピックアップを生成）
	// 命中時はその小惑星を AutoAim のターゲットに設定する。プレイヤー弾と敵弾の両方が小惑星を割る。
	for i := len(e.bullets) - 1; i >= 0; i-- {
		b := &e.bullets[i]
		for _, a := range e.asteroids {
			absorbed, drops := a.Hit(b.X, b.Y, b.Damage)
			if !absorbed {
				continue
			}
			e.pickups = append(e.pickups, drops...)
			hostile := b.Hostile
			impact := b.ImpactFX
			bx, by := b.X, b.Y
			e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
			sound.PlayHit()
			if len(drops) > 0 {
				sound.PlayAsteroidBreak()
			}
			// AutoAim 対象更新はプレイヤー弾の命中時のみ
			if !hostile && e.autoAimTarget != a {
				e.autoAimTarget = a
				e.autoAimGridIdx = -1
				e.autoAimDmgAcc = 0
			}
			if impact {
				e.spawnImpact(bx, by, hostile)
			}
			break
		}
	}

	// プレイヤー弾 vs 海賊
	e.handlePlayerBulletsHitPirates()

	// 敵弾 vs 自機
	e.handleHostileBulletsHitPlayer()

	// 設置ドローンの更新（最寄の小惑星 or 海賊へ継続ダメージ。寿命切れは消滅）
	e.updateDrones()

	// 撃破された海賊を除去し、credits とパーツを落とす（カル距離超過の海賊も除去）
	e.cullPiratesAndDrop()

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
			accepted := false
			switch p.Kind {
			case entity.PickupResource:
				accepted = e.player.AddResource(p.Resource, 1)
			case entity.PickupPart:
				accepted = e.player.AddSparePart(p.PartID, 1)
			}
			if accepted {
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

// handlePlayerAsteroidCollisions は自機の当たり判定矩形（OBB）と各小惑星グリッド（円）を
// 判定し、重なりを解消（押し戻し）、相対速度を反射、衝突相対速度に応じて
// プレイヤーへダメージを与える。小惑星側は質量∞扱いで影響を受けない。
func (e *Exploration) handlePlayerAsteroidCollisions() {
	p := e.player
	g := float64(entity.GridSize)
	gridR := g / 2 // 小惑星グリッドの半径

	for _, a := range e.asteroids {
		aSin, aCos := math.Sin(a.Angle), math.Cos(a.Angle)
		for _, gr := range a.Grids {
			lgx := float64(gr.GX) * g
			lgy := float64(gr.GY) * g
			gcx := a.X + (aCos*lgx - aSin*lgy)
			gcy := a.Y + (aSin*lgx + aCos*lgy)

			nx, ny, depth, hit := p.HullCircleHit(gcx, gcy, gridR)
			if !hit {
				continue
			}
			// 重なりを解消（自機のみ動かす）。nx,ny は自機を小惑星から離す向き。
			p.X += nx * depth
			p.Y += ny * depth

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
				if dmg > 0 {
					e.playDamageSound()
					p.Damage(dmg)
				}
			}

			// 法線成分のみ反射（接線成分はそのまま残す＝かすめ続けない）
			rvx -= (1 + collisionRestitution) * vNormal * nx
			rvy -= (1 + collisionRestitution) * vNormal * ny
			p.VX = a.VX + rvx
			p.VY = a.VY + rvy
		}
	}
}

// handlePlayerPirateCollisions は自機パーツと海賊機パーツの円-円判定を行い、
// 重なりを双方半分ずつ押し戻し、相対法線速度に反発係数を掛けて両機にインパルスを与え、
// 衝突相対速度に応じて両者にダメージを与える（自機・海賊の質量を等しいと仮定）。
func (e *Exploration) handlePlayerPirateCollisions() {
	if len(e.pirates) == 0 {
		return
	}
	p := e.player
	g := float64(entity.GridSize)
	prR := g / 2 // 海賊パーツの半径

	// 自機の当たり判定矩形（OBB）と、海賊の各パーツ（円）を判定する。
	for _, pr := range e.pirates {
		if pr.HP <= 0 {
			continue
		}
		prSin, prCos := math.Sin(pr.Angle), math.Cos(pr.Angle)
		for _, prPart := range pr.Parts {
			lx2, ly2 := entity.PartLocalCenter(prPart.GX, prPart.GY)
			prCX := pr.X + (-prSin*lx2 - prCos*ly2)
			prCY := pr.Y + (prCos*lx2 - prSin*ly2)

			nx, ny, depth, hit := p.HullCircleHit(prCX, prCY, prR)
			if !hit {
				continue
			}

			// 双方を半分ずつ押し戻す（等質量）。nx,ny は自機を海賊から離す向き。
			push := depth / 2
			p.X += nx * push
			p.Y += ny * push
			pr.X -= nx * push
			pr.Y -= ny * push

			// 相対速度の法線成分（負なら接近中）
			rvx := p.VX - pr.VX
			rvy := p.VY - pr.VY
			vNormal := rvx*nx + rvy*ny
			if vNormal >= 0 {
				continue
			}

			impactSpeed := -vNormal
			if impactSpeed > collisionDamageThreshold {
				dmg := int((impactSpeed - collisionDamageThreshold) * collisionDamageFactor)
				if dmg > 0 {
					e.playDamageSound()
					p.Damage(dmg)
					pr.TakeHit(dmg)
				}
			}

			// 等質量・反発係数 e の弾性衝突インパルス
			j := -(1 + collisionRestitution) * vNormal / 2
			p.VX += j * nx
			p.VY += j * ny
			pr.VX -= j * nx
			pr.VY -= j * ny
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
		drawCelestialBackdrop(dst, &m.Body, m.CX, m.CY, e.cameraX, e.cameraY, cx, cy, e.playtime)
	}

	// 宇宙ステーション（背景扱い）
	for _, s := range e.stations {
		s.Draw(dst, s.X-e.cameraX+cx, s.Y-e.cameraY+cy, theme)
	}

	// 小惑星
	for _, a := range e.asteroids {
		a.Draw(dst, a.X-e.cameraX+cx, a.Y-e.cameraY+cy)
	}

	// 海賊（赤アクセントの船体 + 識別リング）。軌跡を船体の下に描く。
	for _, pr := range e.pirates {
		psx, psy := pr.X-e.cameraX+cx, pr.Y-e.cameraY+cy
		drawShipTrail(dst, pr.Trail, -e.cameraX+cx, -e.cameraY+cy, pirateTrailColor)
		pr.DrawAt(dst, psx, psy, theme)
		drawTrailLight(dst, psx, psy, pirateTrailColor)
	}

	// ピックアップ
	for i := range e.pickups {
		p := &e.pickups[i]
		p.Draw(dst, p.X-e.cameraX+cx, p.Y-e.cameraY+cy)
	}

	// プレイヤー機本体。弾・ビーム・着弾/爆発・AutoAim ビームより先に描き、
	// それらを不透明なベース船体の手前に出す。軌跡・光点は左右端・画像下端付近から引く。
	psx := e.player.X - e.cameraX + cx
	psy := e.player.Y - e.cameraY + cy
	e.player.DrawAt(dst, psx, psy, theme)
	for _, off := range e.player.TrailLightOffsets() {
		drawShipTrail(dst, e.player.Trail, -e.cameraX+cx+off[0], -e.cameraY+cy+off[1], theme.Line)
		drawTrailLight(dst, psx+off[0], psy+off[1], theme.Line)
	}
	// シールドが 1 以上なら、外周（隣接面以外）を描画
	if e.player.ShieldHP > 0 {
		e.player.Ship.DrawShieldOutline(dst, psx, psy, theme)
	}

	// 設置中の機雷（弾の下に描画）
	for i := range e.mines {
		m := &e.mines[i]
		m.Draw(dst, m.X-e.cameraX+cx, m.Y-e.cameraY+cy, theme)
	}

	// 設置中のドローン
	for i := range e.drones {
		d := &e.drones[i]
		d.Draw(dst, d.X-e.cameraX+cx, d.Y-e.cameraY+cy)
	}

	// 弾（カメラ＝プレイヤーが動くので、見かけのトレイル方向にプレイヤー速度を渡す）
	for i := range e.bullets {
		b := &e.bullets[i]
		b.Draw(dst, b.X-e.cameraX+cx, b.Y-e.cameraY+cy, e.player.VX, e.player.VY, theme)
	}

	// レーザービーム（弾の上、着弾エフェクトの下）
	for i := range e.beams {
		bm := &e.beams[i]
		x1 := bm.X1 - e.cameraX + cx
		y1 := bm.Y1 - e.cameraY + cy
		x2 := bm.X2 - e.cameraX + cx
		y2 := bm.Y2 - e.cameraY + cy
		bm.DrawScreen(dst, x1, y1, x2, y2)
	}

	// 着弾エフェクト
	for i := range e.impacts {
		eff := &e.impacts[i]
		eff.Draw(dst, eff.X-e.cameraX+cx, eff.Y-e.cameraY+cy)
	}
	// 爆発エフェクト
	for i := range e.explosions {
		eff := &e.explosions[i]
		eff.Draw(dst, eff.X-e.cameraX+cx, eff.Y-e.cameraY+cy)
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

	// ドローン攻撃ビーム（ドローン → 対象）。味方はシアン系、敵機設置は赤系で描き分ける。
	droneBeamColor := color.NRGBA{0x40, 0xe0, 0xc0, 0xff}
	droneHostileBeamColor := color.NRGBA{0xff, 0x60, 0x40, 0xff}
	for _, b := range e.droneBeams {
		x1 := float32(b.fromX - e.cameraX + cx)
		y1 := float32(b.fromY - e.cameraY + cy)
		x2 := float32(b.toX - e.cameraX + cx)
		y2 := float32(b.toY - e.cameraY + cy)
		c := droneBeamColor
		if b.hostile {
			c = droneHostileBeamColor
		}
		vector.StrokeLine(dst, x1, y1, x2, y2, 1.5, c, false)
	}

	// HP / シールドバーを船体下に描画（パーツに被らない位置）。
	// 機体本体・軌跡・光点・シールドは弾より前のブロックで描画済み。
	e.drawPlayerVitalBars(dst, theme, psx, psy)

	// ドック近接プロンプト
	if e.activeDock != nil {
		prompt := i18n.S().HUD.DockPrompt
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
	hud := i18n.S().HUD
	// ステータス: HP / シールド / FUEL は自機下部のバーで表示するためここから除外。
	// 積荷・所持金とインベントリは右上に右寄せで表示する。
	right := float64(sw) - 20
	statusLine := fmt.Sprintf(hud.CargoFmt,
		e.player.CargoLoad(), e.player.MaxCargo,
		e.player.Credits)
	w, _ := ui.MeasureText(statusLine, 1.5)
	ui.DrawText(dst, statusLine, right-w, 20, 1.5, theme.Line)

	// インベントリ
	inv := e.player.Inventory
	invLine := fmt.Sprintf(hud.InvFmt,
		i18n.ResourceName(entity.ResourceIron), inv[entity.ResourceIron],
		i18n.ResourceName(entity.ResourceBronze), inv[entity.ResourceBronze],
		i18n.ResourceName(entity.ResourceIce), inv[entity.ResourceIce])
	w, _ = ui.MeasureText(invLine, 1.5)
	ui.DrawText(dst, invLine, right-w, 50, 1.5, theme.Line)

	// ミニマップ
	miniW, miniH := float32(180), float32(180)
	mx := float32(sw) - miniW - 20
	my := float32(sh) - miniH - 20

	// 速度・座標（デバッグ補助）: ミニマップの直上に右寄せで表示する
	speed := math.Hypot(e.player.VX, e.player.VY)
	speedLine := fmt.Sprintf(hud.SpeedPosFmt, speed, e.player.X, e.player.Y)
	w, h := ui.MeasureText(speedLine, 1.2)
	ui.DrawText(dst, speedLine, right-w, float64(my)-h-6, 1.2, theme.LineDim)
	// 不透明の黒背景で星空・小惑星を覆う
	vector.FillRect(dst, mx, my, miniW, miniH, color.NRGBA{0, 0, 0, 255}, false)
	// 敵（生存中の海賊）がいる間は縁を太く赤くして警戒を示す
	borderW, borderColor := float32(1), theme.Line
	for _, pr := range e.pirates {
		if pr.HP > 0 {
			borderW, borderColor = 3, color.NRGBA{0xff, 0x60, 0x40, 0xff}
			break
		}
	}
	vector.StrokeRect(dst, mx, my, miniW, miniH, borderW, borderColor, false)
	// プレイヤー（中央点）
	vector.FillRect(dst, mx+miniW/2-1, my+miniH/2-1, 2, 2, theme.Line, false)
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
		vector.FillRect(dst, nx-1, ny-1, 2, 2, c, false)
	}
	// 海賊（範囲内は赤い小点 / 範囲外は縁の内側に方向マーカー「<」）
	mcx, mcy := mx+miniW/2, my+miniH/2
	for _, pr := range e.pirates {
		nx := mcx + float32((pr.X-e.cameraX)*minimapScale)
		ny := mcy + float32((pr.Y-e.cameraY)*minimapScale)
		if nx >= mx && nx <= mx+miniW && ny >= my && ny <= my+miniH {
			vector.FillRect(dst, nx-1, ny-1, 3, 3, color.NRGBA{0xff, 0x60, 0x40, 0xff}, false)
			continue
		}
		drawMinimapMarker(dst, mcx, mcy, mx, my, miniW, miniH, nx, ny, color.NRGBA{0xff, 0x60, 0x40, 0xff})
	}

	// ステーション（範囲内は小さな四角 / 範囲外は縁の内側に方向マーカー「<」）。
	// 表示は現在いるワールドマップ（FullMap）内のステーションに限定する。
	for _, s := range e.stations {
		if e.lastMap == nil || !e.lastMap.Contains(s.X, s.Y) {
			continue
		}
		nx := mcx + float32((s.X-e.cameraX)*minimapScale)
		ny := mcy + float32((s.Y-e.cameraY)*minimapScale)
		if nx >= mx && nx <= mx+miniW && ny >= my && ny <= my+miniH {
			vector.StrokeRect(dst, nx-3, ny-3, 6, 6, 1, theme.Line, false)
			continue
		}
		drawMinimapMarker(dst, mcx, mcy, mx, my, miniW, miniH, nx, ny, theme.Line)
	}

	ui.DrawText(dst, e.buildControlsHelp(), 20, float64(sh)-30, 1.5, theme.LineDim)
}

// buildControlsHelp は現在のプレイヤー状態で実際に使えるキー操作だけを並べた
// 1 行のヘルプ文字列を返す。スラスタは方向ごとに装備の有無で QWES を取捨し、
// ブースト・射撃・ドック・ワープも条件を満たすときだけ表示する。
func (e *Exploration) buildControlsHelp() string {
	hasFwd, hasBck, hasLft, hasRgt := false, false, false, false
	hasThruster := false
	hasGun := false
	for _, p := range e.player.Parts {
		switch p.Kind() {
		case entity.PartThruster:
			hasThruster = true
			switch p.ThrustDir() {
			case entity.ThrustDirForward:
				hasFwd = true
			case entity.ThrustDirBackward:
				hasBck = true
			case entity.ThrustDirLeft:
				hasLft = true
			case entity.ThrustDirRight:
				hasRgt = true
			}
		case entity.PartGun, entity.PartMineLayer, entity.PartDroneLauncher:
			hasGun = true
		}
	}
	// スラスタ未搭載のときは Cockpit が前向きにフォールバックする
	if !hasThruster {
		hasFwd = true
	}

	var parts []string
	thrustKeys := ""
	if hasLft {
		thrustKeys += "Q"
	}
	if hasFwd {
		thrustKeys += "W"
	}
	if hasRgt {
		thrustKeys += "E"
	}
	if hasBck {
		thrustKeys += "S"
	}
	hud := i18n.S().HUD
	if thrustKeys != "" {
		parts = append(parts, thrustKeys+": "+hud.HelpThrust)
	}
	parts = append(parts, hud.HelpRotate)
	if e.player.MaxFuel > 0 {
		parts = append(parts, hud.HelpBoost)
	}
	switch {
	case hasGun && e.activeDock != nil:
		parts = append(parts, hud.HelpFireDock)
	case hasGun:
		parts = append(parts, hud.HelpFire)
	case e.activeDock != nil:
		parts = append(parts, hud.HelpDock)
	}
	parts = append(parts, hud.HelpMap)
	if e.player.HasWarpDrive() {
		parts = append(parts, hud.HelpWarp)
	}
	parts = append(parts, hud.HelpMenu)
	return "[ " + strings.Join(parts, "    ") + " ]"
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
	sound.PlayWarp()
}

// tickWarp は warpTimer > 0 の間、ワープアニメを 1 フレーム進める。
// 中点フレームで実際の座標移動と一時状態のクリアを行う。
func (e *Exploration) tickWarp() {
	e.warpTimer--

	if e.warpTimer == warpDuration/2 {
		sound.PlayWarpJump()
		dest := e.warpDest
		if dest != nil {
			e.player.X = dest.CX
			e.player.Y = dest.CY
			e.player.VX = 0
			e.player.VY = 0
			e.player.ClearTrail() // テレポートで軌跡が伸びないよう消す
			e.lastMap = dest
			// ワープ前の局所状態（小惑星・ピックアップ・弾・機雷・ドローン・海賊・着弾・爆発・自動照準）は持ち越さない
			e.asteroids = e.asteroids[:0]
			e.pickups = e.pickups[:0]
			e.bullets = e.bullets[:0]
			e.mines = e.mines[:0]
			e.drones = e.drones[:0]
			e.droneBeams = e.droneBeams[:0]
			e.pirates = e.pirates[:0]
			e.impacts = e.impacts[:0]
			e.explosions = e.explosions[:0]
			e.beams = e.beams[:0]
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
	vector.FillRect(dst, 0, 0, float32(sw), float32(sh), color.NRGBA{255, 255, 255, alpha}, false)
}

// drawCelestialBackdrop は天体を FullMap 中心 (mapCX, mapCY) を anchor として描画する。
// BackdropRadius が 0 の場合は何もしない。
// 設計どおりの位置に見え、自機が動いても少しだけしか流れないようにする。
// 不透明な惑星らしさを出すため、暗いベースの上に光源側（左上）へオフセットした
// 明部とハイライトを重ね、輪郭はやや暗めにして遠景の球体感を演出する。
// pirateTrailColor は海賊船の軌跡色（船体と同じ赤系）。
var pirateTrailColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}

const (
	shipTrailAlpha = 0.45 // 軌跡の最大不透明度（新しい点ほどこれに近づく）
	shipTrailWidth = 2.0
)

// drawShipTrail は船の軌跡を、古いほど薄くなる線分列として描く。
// (offX, offY) はワールド座標→スクリーン座標の加算オフセット。
func drawShipTrail(dst *ebiten.Image, trail []entity.TrailPoint, offX, offY float64, c color.NRGBA) {
	n := len(trail)
	if n < 2 {
		return
	}
	for i := 1; i < n; i++ {
		a, b := trail[i-1], trail[i]
		frac := float64(i) / float64(n-1) // 新しいほど 1 に近い＝濃い
		cc := c
		cc.A = uint8(float64(c.A) * frac * shipTrailAlpha)
		vector.StrokeLine(dst,
			float32(a.X+offX), float32(a.Y+offY),
			float32(b.X+offX), float32(b.Y+offY),
			shipTrailWidth, cc, true)
	}
}

// drawMinimapEnemyMarker はミニマップ外の敵を、縁の内側に「<」状のシェブロン
// （敵方向を指す矢じり）で方向表示する。(tx, ty) は敵のミニマップ投影位置（範囲外）。
func drawMinimapMarker(dst *ebiten.Image, mcx, mcy, mx, my, w, h, tx, ty float32, c color.NRGBA) {
	dirX, dirY := tx-mcx, ty-mcy
	d := float32(math.Hypot(float64(dirX), float64(dirY)))
	if d == 0 {
		return
	}
	ux, uy := dirX/d, dirY/d // 中心→対象 の単位ベクトル
	const inset, size = 8.0, 5.0
	// マーカー位置は縁の内側にクランプ
	px := max(mx+inset, min(tx, mx+w-inset))
	py := max(my+inset, min(ty, my+h-inset))
	// 対象方向を指す矢じり（先端＋左右の翼）
	perpX, perpY := -uy, ux
	tipX, tipY := px+ux*size, py+uy*size
	vector.StrokeLine(dst, tipX, tipY, tipX-ux*size+perpX*size, tipY-uy*size+perpY*size, 1.5, c, true)
	vector.StrokeLine(dst, tipX, tipY, tipX-ux*size-perpX*size, tipY-uy*size-perpY*size, 1.5, c, true)
}

// drawTrailLight は軌跡の発生点（機体中心）に、幅 4〜6px で明滅する光点を描く。
func drawTrailLight(dst *ebiten.Image, sx, sy float64, c color.NRGBA) {
	drawTrailLightSized(dst, sx, sy, 5, 1, c) // 基準幅 5px ±1（=4〜6px）
}

// drawTrailLightSized は幅を指定して明滅する光点を描く。
// base は基準の直径、flicker は毎フレームの揺れ幅（直径 base±flicker でまたたく）。
func drawTrailLightSized(dst *ebiten.Image, sx, sy, base, flicker float64, c color.NRGBA) {
	w := base + (rand.Float64()*2-1)*flicker
	lit := scaleColor(c, 1.3)
	lit.A = 255
	vector.FillCircle(dst, float32(sx), float32(sy), float32(w/2), lit, true)
}

// planetSpinTurnsPerSec は惑星の自転速度（毎秒の周回率）。小さいほどゆっくり回る。
const planetSpinTurnsPerSec = 0.02

// planetCloudSpeed は雲アニメ（GIF）の再生速度倍率。1.0 で GIF 本来の速さ。
const planetCloudSpeed = 0.5

func drawCelestialBackdrop(dst *ebiten.Image, body *entity.Celestial,
	mapCX, mapCY, cameraX, cameraY, screenCX, screenCY, playtime float64) {
	if body == nil || body.BackdropRadius <= 0 {
		return
	}
	// anchor = FullMap 中心。anchor 上では設計オフセットそのまま見える。
	// 自機が動いた分だけ (anchor - camera) * P を加える（P が小さいほど動かない）。
	const p = nearPlanetParallax
	sx := float32(body.BackdropOffsetX + (mapCX-cameraX)*p + screenCX)
	sy := float32(body.BackdropOffsetY + (mapCY-cameraY)*p + screenCY)
	r := float32(body.BackdropRadius)
	// 視界外なら早期スキップ（半径分のマージン込み）
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()
	if sx+r < 0 || sx-r > float32(sw) || sy+r < 0 || sy-r > float32(sh) {
		return
	}
	// 惑星はテクスチャを貼った自転する立体球として描く（シェーダ）。失敗時は平面描画へ続行。
	if body.Kind == entity.CelestialPlanet {
		// 自転位相は 1 周で正規化（長時間プレイでも float32 精度を保つ）。
		spin := math.Mod(playtime*planetSpinTurnsPerSec, 1.0)
		// 雲のアニメーションは GIF を planetCloudSpeed 倍速で再生（控えめにゆっくり）。
		tex := assetimage.Planet3rdFrameAt(playtime * planetCloudSpeed)
		if drawPlanetSphere(dst, tex, float64(sx), float64(sy), float64(r), spin) {
			return
		}
	}
	// 暗側ベース（不透明）。ここが影の側として残る。
	dark := scaleColor(body.Color, 0.9)
	dark.A = 255
	vector.FillCircle(dst, sx, sy, r, dark, true)
	// 明側: 元の色を光源（左上）方向にオフセットしてやや小さく描く。
	// 半径と offset の合計が r 以下になるよう調整して輪郭の外にはみ出さない。
	lit := body.Color
	lit.A = 255
	vector.FillCircle(dst, sx-r*0.15, sy-r*0.15, r*0.78, lit, true)
	// ハイライト: 光源直接光のように小さく明るい点。
	hi := scaleColor(body.Color, 1.35)
	hi.A = 220
	vector.FillCircle(dst, sx-r*0.42, sy-r*0.42, r*0.2, hi, true)
	// 輪郭は本体色をやや暗くして縁取り。遠景の球体らしくシャープすぎないように。
	rim := scaleColor(body.Color, 0.7)
	rim.A = 255
	vector.StrokeCircle(dst, sx, sy, r, 1.0, rim, true)
}

// scaleColor は NRGBA の RGB 各成分を s 倍する。s>1 はクランプ、A は保持。
func scaleColor(c color.NRGBA, s float64) color.NRGBA {
	clamp := func(v float64) uint8 {
		switch {
		case v < 0:
			return 0
		case v > 255:
			return 255
		}
		return uint8(v)
	}
	return color.NRGBA{
		R: clamp(float64(c.R) * s),
		G: clamp(float64(c.G) * s),
		B: clamp(float64(c.B) * s),
		A: c.A,
	}
}

// pirateInsideHull は (x, y) が海賊の当たり判定矩形（OBB）内かを判定する。
func pirateInsideHull(pr *entity.Pirate, x, y float64) bool {
	return pr.HullContains(x, y)
}

// pirateHullWithin は (x, y) が海賊の当たり判定矩形から距離 radius 以内にあるかを返す（爆発の波及等）。
func pirateHullWithin(pr *entity.Pirate, x, y, radius float64) bool {
	return pr.HullWithin(x, y, radius)
}

// playerHullWithin は (x, y) が自機の当たり判定矩形から距離 radius 以内にあるかを返す。
func (e *Exploration) playerHullWithin(x, y, radius float64) bool {
	return e.player.HullWithin(x, y, radius)
}

// handleExplosiveBullets は爆発弾（ExplosionRadius>0）の着弾を専用処理する。
// 通常の単体命中ループより前に走り、対象（味方弾→小惑星/海賊、敵弾→自機）に
// 直撃した爆発弾を取り除いて detonateBullet で範囲ダメージ＋爆発エフェクトを出す。
// 直撃しなかった爆発弾はそのまま飛翔を続ける（寿命切れでは爆発しない）。
func (e *Exploration) handleExplosiveBullets() {
	for i := len(e.bullets) - 1; i >= 0; i-- {
		b := &e.bullets[i]
		if b.ExplosionRadius <= 0 {
			continue
		}
		hx, hy := b.X, b.Y
		// 小惑星は敵味方どちらの弾でも起爆トリガになる。
		hit := false
		for _, a := range e.asteroids {
			if a.ContainsPoint(hx, hy) {
				hit = true
				break
			}
		}
		if !hit {
			if b.Hostile {
				hit = e.playerHullWithin(hx, hy, 0)
			} else {
				for _, pr := range e.pirates {
					if pr.HP > 0 && pirateInsideHull(pr, hx, hy) {
						hit = true
						break
					}
				}
			}
		}
		if !hit {
			continue
		}
		bb := *b
		e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
		e.detonateBullet(&bb, hx, hy)
	}
}

// detonateBullet は爆発弾の着弾処理。着弾点 (hx, hy) を中心に b.ExplosionRadius 内の
// 対象へ b.Damage を与え、爆発エフェクトと音を出す。
// b.Hostile に応じて対象が変わる（敵弾→自機、味方弾→小惑星と海賊）。
func (e *Exploration) detonateBullet(b *entity.Bullet, hx, hy float64) {
	r := b.ExplosionRadius
	dmg := b.Damage
	// 小惑星は敵味方どちらの弾でも砕ける。命中小惑星はプレイヤー弾のときだけ AutoAim 対象にする。
	for _, a := range e.asteroids {
		pks, anyHit := a.HitRadius(hx, hy, r, dmg)
		if !anyHit {
			continue
		}
		e.pickups = append(e.pickups, pks...)
		if len(pks) > 0 {
			sound.PlayAsteroidBreak()
		}
		if !b.Hostile && e.autoAimTarget != a {
			e.autoAimTarget = a
			e.autoAimGridIdx = -1
			e.autoAimDmgAcc = 0
		}
	}
	if b.Hostile {
		// 敵弾の爆発は自機を巻き込む。
		if e.playerHullWithin(hx, hy, r) {
			e.playDamageSound()
			e.player.Damage(dmg)
		}
	} else {
		// 味方弾の爆発は範囲内の海賊を巻き込む（撃破処理は cullPiratesAndDrop が担う）。
		for _, pr := range e.pirates {
			if pr.HP <= 0 {
				continue
			}
			if pirateHullWithin(pr, hx, hy, r) {
				pr.TakeHit(dmg)
			}
		}
	}
	c := color.NRGBA{0xff, 0xa0, 0x40, 0xff}
	if b.Hostile {
		c = color.NRGBA{0xff, 0x60, 0x40, 0xff}
	}
	e.explosions = append(e.explosions, entity.NewExplosion(hx, hy, c, e.spawnRng))
	sound.PlayExplosion()
}

// handlePlayerBulletsHitPirates はプレイヤー弾と海賊機体のハル外形判定を行い、
// 命中した弾を消費して該当海賊にダメージを与える（敵弾・既消滅弾はスキップ）。
func (e *Exploration) handlePlayerBulletsHitPirates() {
	for i := len(e.bullets) - 1; i >= 0; i-- {
		b := &e.bullets[i]
		if b.Hostile {
			continue
		}
		for _, pr := range e.pirates {
			if pr.HP <= 0 {
				continue
			}
			if !pirateInsideHull(pr, b.X, b.Y) {
				continue
			}
			pr.TakeHit(b.Damage)
			impact := b.ImpactFX
			bx, by := b.X, b.Y
			e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
			sound.PlayHit()
			if impact {
				e.spawnImpact(bx, by, false)
			}
			break
		}
	}
}

// handleHostileBulletsHitPlayer は敵弾とプレイヤー機体の判定を行い、
// 命中した弾を消費してダメージを与える。判定は自機の当たり判定矩形（OBB）で行う。
func (e *Exploration) handleHostileBulletsHitPlayer() {
	for i := len(e.bullets) - 1; i >= 0; i-- {
		b := &e.bullets[i]
		if !b.Hostile {
			continue
		}
		if !e.player.HullContains(b.X, b.Y) {
			continue
		}
		e.playDamageSound()
		e.player.Damage(b.Damage)
		impact := b.ImpactFX
		bx, by := b.X, b.Y
		e.bullets = append(e.bullets[:i], e.bullets[i+1:]...)
		if impact {
			e.spawnImpact(bx, by, true)
		}
	}
}

// drawPlayerVitalBars は自機の下に HP / シールド / Fuel バーを縦に並べて描画する。
// バーの位置は船体パーツのバウンディング半径（円相当）+ 余白で、
// 機体の向きに依らずパーツと被らないようにする。
// シールドは MaxShieldHP > 0 のとき、Fuel は MaxFuel > 0 のときのみ表示。
func (e *Exploration) drawPlayerVitalBars(dst *ebiten.Image, theme *ui.Theme, psx, psy float64) {
	// 船体半径: 各パーツのセル中心 + g/2 の最大距離。ベース船体の外接半径も下限にする。
	g := float64(entity.GridSize)
	radius := e.player.HullRadius()
	for _, part := range e.player.Parts {
		dx, dy := entity.PartLocalCenter(part.GX, part.GY)
		d := math.Hypot(dx, dy) + g/2
		if d > radius {
			radius = d
		}
	}
	const (
		barW      = 80.0
		barH      = 6.0
		barGap    = 4.0
		barMargin = 18.0
	)
	x0 := psx - barW/2
	y := psy + radius + barMargin

	// HP バー（赤）
	drawVitalBar(dst, x0, y, barW, barH,
		float64(e.player.HP)/float64(maxIntOr1(e.player.MaxHP)),
		color.NRGBA{0xff, 0x60, 0x60, 0xff}, theme.LineDim)
	y += barH + barGap

	// シールドバー（シアン、MaxShieldHP > 0 のとき）
	if e.player.MaxShieldHP > 0 {
		drawVitalBar(dst, x0, y, barW, barH,
			float64(e.player.ShieldHP)/float64(e.player.MaxShieldHP),
			color.NRGBA{0x60, 0xc0, 0xff, 0xff}, theme.LineDim)
		y += barH + barGap
	}

	// Fuel バー（黄、MaxFuel > 0 のとき）。常に最下段。
	if e.player.MaxFuel > 0 {
		drawVitalBar(dst, x0, y, barW, barH,
			e.player.Fuel/e.player.MaxFuel,
			color.NRGBA{0xff, 0xe0, 0x60, 0xff}, theme.LineDim)
	}
}

// drawVitalBar は HP/シールド共通のバー描画。fill は 0..1 のクランプ済み比率。
func drawVitalBar(dst *ebiten.Image, x, y, w, h float64, fill float64, fillColor, frameColor color.NRGBA) {
	if fill < 0 {
		fill = 0
	}
	if fill > 1 {
		fill = 1
	}
	// 背景（薄い塗り）
	bg := frameColor
	bg.A = 80
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), bg, false)
	// フィル
	if fill > 0 {
		vector.FillRect(dst, float32(x), float32(y), float32(w*fill), float32(h), fillColor, false)
	}
	// 枠
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), 1, frameColor, false)
}

// maxIntOr1 は 0 を 1 に置換して 0 除算を防ぐ。
func maxIntOr1(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

// spawnImpact は (x, y) に着弾エフェクトを追加する。hostile=true なら赤色、false ならテーマライン色相当。
func (e *Exploration) spawnImpact(x, y float64, hostile bool) {
	c := color.NRGBA{0xff, 0xc0, 0x40, 0xff}
	if hostile {
		c = color.NRGBA{0xff, 0x60, 0x40, 0xff}
	}
	e.impacts = append(e.impacts, entity.NewImpact(x, y, c))
}

// fireLaser は瞬間命中レーザーを処理する。
// レイの最も近いヒット（小惑星グリッドまたは船体）にダメージを与え、
// ビーム可視化を appended する。命中対象が無ければ Range の終端までビームを描く。
func (e *Exploration) fireLaser(l entity.LaserShot) {
	bestT := l.Range
	hit := false
	hitX, hitY := l.X+l.DX*l.Range, l.Y+l.DY*l.Range

	// 小惑星グリッドへの命中（プレイヤー弾・敵弾の両方）
	g := float64(entity.GridSize)
	gridR := g / 2
	type gridHit struct {
		a   *entity.Asteroid
		idx int
	}
	var asteroidHit *gridHit
	for _, a := range e.asteroids {
		aSin, aCos := math.Sin(a.Angle), math.Cos(a.Angle)
		for i, gr := range a.Grids {
			lgx := float64(gr.GX) * g
			lgy := float64(gr.GY) * g
			gcx := a.X + aCos*lgx - aSin*lgy
			gcy := a.Y + aSin*lgx + aCos*lgy
			if t, ok := raySphereHit(l.X, l.Y, l.DX, l.DY, gcx, gcy, gridR, bestT); ok {
				bestT = t
				hit = true
				hitX = l.X + l.DX*t
				hitY = l.Y + l.DY*t
				asteroidHit = &gridHit{a: a, idx: i}
			}
		}
	}

	// 船体への命中
	pirateIdx := -1
	playerHit := false
	if !l.Hostile {
		// プレイヤーが撃ったレーザー → 海賊にダメージ。当たり判定矩形（OBB）で最近接を取る。
		for i, pr := range e.pirates {
			if pr.HP <= 0 {
				continue
			}
			if t, ok := pr.HullRayHit(l.X, l.Y, l.DX, l.DY, bestT); ok {
				bestT = t
				hit = true
				hitX = l.X + l.DX*t
				hitY = l.Y + l.DY*t
				pirateIdx = i
				asteroidHit = nil
			}
		}
	} else {
		// 敵レーザー → プレイヤーにダメージ（当たり判定矩形 OBB）
		if t, ok := e.player.HullRayHit(l.X, l.Y, l.DX, l.DY, bestT); ok {
			bestT = t
			hit = true
			hitX = l.X + l.DX*t
			hitY = l.Y + l.DY*t
			playerHit = true
			asteroidHit = nil
			pirateIdx = -1
		}
	}

	// ダメージ適用
	if asteroidHit != nil {
		destroyed, pk, _ := asteroidHit.a.HitGrid(asteroidHit.idx, l.Damage)
		if destroyed {
			e.pickups = append(e.pickups, pk)
			sound.PlayAsteroidBreak()
		}
	} else if pirateIdx >= 0 {
		e.pirates[pirateIdx].TakeHit(l.Damage)
		// プレイヤー弾命中時は AutoAim ターゲット更新は行わない（レーザーは別系統）
	} else if playerHit {
		e.playDamageSound()
		e.player.Damage(l.Damage)
	}

	// ビーム可視化
	beamColor := color.NRGBA{0xff, 0xc0, 0x40, 0xff}
	if l.Hostile {
		beamColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}
	}
	e.beams = append(e.beams, entity.NewBeam(l.X, l.Y, hitX, hitY, l.Width, beamColor))

	// 着弾エフェクト
	if hit && l.ImpactFX {
		e.spawnImpact(hitX, hitY, l.Hostile)
	}
}

// raySphereHit はレイと円の交差判定。最初のヒット t を返す（0 ≤ t ≤ maxT）。
// origin (ox, oy)、単位方向 (dx, dy)、円中心 (cx, cy) 半径 r。
func raySphereHit(ox, oy, dx, dy, cx, cy, r, maxT float64) (float64, bool) {
	fx := ox - cx
	fy := oy - cy
	b := fx*dx + fy*dy
	c := fx*fx + fy*fy - r*r
	disc := b*b - c
	if disc < 0 {
		return 0, false
	}
	sd := math.Sqrt(disc)
	t1 := -b - sd
	if t1 >= 0 && t1 <= maxT {
		return t1, true
	}
	t2 := -b + sd
	if t2 >= 0 && t2 <= maxT {
		return t2, true
	}
	return 0, false
}

// cullPiratesAndDrop は撃破された海賊・カル距離超過の海賊を除去する。
// 撃破時は credits を加算し、PartDropRate に従って稀にパーツ pickup を生成する。
// 加えて Bounty クエストの進捗用に PiratesKilledByMap を更新する（撃破した海賊が居た FullMap）。
func (e *Exploration) cullPiratesAndDrop() {
	n := 0
	for _, pr := range e.pirates {
		if pr.HP <= 0 {
			e.dropPirateLoot(pr)
			// 撃墜時の爆発エフェクトと爆発音
			explosionColor := color.NRGBA{0xff, 0x80, 0x40, 0xff}
			e.explosions = append(e.explosions,
				entity.NewExplosion(pr.X, pr.Y, explosionColor, e.spawnRng))
			sound.PlayExplosion()
			if pm := e.world.Containing(pr.X, pr.Y); pm != nil {
				if e.player.PiratesKilledByMap == nil {
					e.player.PiratesKilledByMap = make(map[string]int)
				}
				e.player.PiratesKilledByMap[pm.Name]++
			}
			continue
		}
		if math.Hypot(pr.X-e.player.X, pr.Y-e.player.Y) > asteroidCullDist {
			continue
		}
		e.pirates[n] = pr
		n++
	}
	e.pirates = e.pirates[:n]
}

// dropPirateLoot は撃破海賊からのドロップを生成する。
// credits は自機に直接加算し、稀にパーツ pickup を物理空間に落とす。
func (e *Exploration) dropPirateLoot(pr *entity.Pirate) {
	pat := pr.Pattern
	if pat == nil {
		return
	}
	min := pat.DropCreditsMin
	max := pat.DropCreditsMax
	if max < min {
		max = min
	}
	credits := min
	if max > min {
		credits += e.spawnRng.Intn(max - min + 1)
	}
	e.player.Credits += credits

	if len(pat.PartDrops) > 0 && e.spawnRng.Float64() < pat.PartDropRate {
		id := pat.PartDrops[e.spawnRng.Intn(len(pat.PartDrops))]
		e.pickups = append(e.pickups, entity.NewPartPickup(pr.X, pr.Y, id))
	}
}
