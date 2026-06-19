package scene

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	assetimage "github.com/yiozio/space-miner/internal/asset/image"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/ui"
)

// title_bg.go はタイトル画面の動く背景を提供する。
// 左上に青い惑星＋星々、ランダムな小惑星が回転しながら漂い、たまに海賊船が横切る。

// タイトル背景の画面サイズ（ゲームの固定解像度）。
const (
	titleScreenW = 1280
	titleScreenH = 720
)

// 右上に置く青い惑星。ロゴ右端（x≈988）と重ならないよう、
// 右端に寄せて一部を画面外に出す（左端 x=1000 で 12px のマージン）。
var titlePlanetColor = color.NRGBA{0x60, 0xa0, 0xff, 0xff}

const (
	titlePlanetX = titleScreenW - 130
	titlePlanetY = 150
	titlePlanetR = 150
)

const titleAsteroidCount = 6

// titleBackground はタイトル画面の動く背景（星・惑星・漂う小惑星・たまに通る海賊船）。
type titleBackground struct {
	rng         *rand.Rand
	starfield   *starfield
	asteroids   []*entity.Asteroid
	pirate      *entity.Pirate // nil なら不在
	pirateTimer int            // 次の海賊出現までの残フレーム
	t           float64        // 経過秒（惑星の自転・雲アニメ用）
}

func newTitleBackground() *titleBackground {
	rng := rand.New(rand.NewSource(1))
	bg := &titleBackground{
		rng:         rng,
		starfield:   newStarfield(1),
		pirateTimer: 180 + rng.Intn(360),
	}
	for range titleAsteroidCount {
		bg.asteroids = append(bg.asteroids, bg.newAsteroid())
	}
	return bg
}

// newAsteroid はランダムな位置・自転・ゆるい漂流速度の小惑星を作る。
func (bg *titleBackground) newAsteroid() *entity.Asteroid {
	x := bg.rng.Float64() * titleScreenW
	y := bg.rng.Float64() * titleScreenH
	size := 2 + bg.rng.Intn(5) // 2..6 セル
	res := entity.AllResourceTypes()[bg.rng.Intn(3)]
	a := entity.NewAsteroid(bg.rng.Int63(), x, y, size, res)
	// 漂流速度と自転を背景向けに上書き（ゆっくり漂って回る）。
	ang := bg.rng.Float64() * 2 * math.Pi
	sp := 0.3 + bg.rng.Float64()*0.6
	a.VX = math.Cos(ang) * sp
	a.VY = math.Sin(ang) * sp
	a.AngularVel = (bg.rng.Float64()*2 - 1) * 0.01
	return a
}

func (bg *titleBackground) update() {
	bg.t += 1.0 / 60.0 // 惑星の自転・雲アニメを進める
	for _, a := range bg.asteroids {
		a.Update()
		titleWrap(&a.X, &a.Y)
	}
	if bg.pirate != nil {
		bg.pirate.X += bg.pirate.VX
		bg.pirate.Y += bg.pirate.VY
		bg.pirate.PushTrail()
		if titleOffscreen(bg.pirate.X, bg.pirate.Y, 140) {
			bg.pirate = nil
			bg.pirateTimer = 360 + bg.rng.Intn(600) // 6〜16 秒後にまた出す
		}
		return
	}
	bg.pirateTimer--
	if bg.pirateTimer <= 0 {
		bg.spawnPirate()
	}
}

// spawnPirate は画面の左右どちらかの外から、反対側へ横切る海賊船を出す。
func (bg *titleBackground) spawnPirate() {
	ids := []entity.PiratePatternID{
		entity.PiratePatternScout, entity.PiratePatternBrawler, entity.PiratePatternCruiser,
	}
	pat := entity.PiratePatternByID(ids[bg.rng.Intn(len(ids))])
	speed := 1.5 + bg.rng.Float64()*1.5
	y := 80 + bg.rng.Float64()*(titleScreenH-160)
	var x, vx float64
	if bg.rng.Intn(2) == 0 {
		x, vx = -120, speed
	} else {
		x, vx = titleScreenW+120, -speed
	}
	vy := (bg.rng.Float64()*2 - 1) * 0.4
	// NewPirate は (playerX,playerY) の方を向くので、進行方向の遠点を渡して機首を合わせる。
	p := entity.NewPirate(x, y, x+vx*1000, y+vy*1000, pat)
	p.VX, p.VY = vx, vy
	bg.pirate = p
}

func (bg *titleBackground) draw(dst *ebiten.Image, theme *ui.Theme) {
	bg.starfield.draw(dst, 0, 0, theme)
	// 惑星は本編と同じ Aurora の自転テクスチャ球（端ほど濃い青白の大気つき）で描く。
	// アセット未準備のときだけ従来の青い惑星でつなぐ（通常はスプラッシュ中に準備完了）。
	tex := assetimage.Planet3rdFrameAt(bg.t * planetCloudSpeed)
	// その場自転（視点固定）: 経度方向の回転のみ。光源は固定（orbitLight=false）。
	rot := rotY(math.Mod(bg.t*planetSpinTurnsPerSec, 1.0) * 2 * math.Pi)
	atmo := planetAtmosphere{strength: 0.8, color: [3]float32{0.6, 0.8, 1.0}, outer: 1.07}
	if !drawPlanetSphere(dst, tex, titlePlanetX, titlePlanetY, titlePlanetR, rot, atmo, false) {
		drawTitlePlanet(dst, titlePlanetX, titlePlanetY, titlePlanetR, titlePlanetColor)
	}
	for _, a := range bg.asteroids {
		a.Draw(dst, a.X, a.Y)
	}
	if bg.pirate != nil {
		drawShipTrail(dst, bg.pirate.Trail, 0, 0, pirateTrailColor)
		bg.pirate.DrawAt(dst, bg.pirate.X, bg.pirate.Y, theme)
		drawTrailLight(dst, bg.pirate.X, bg.pirate.Y, pirateTrailColor)
	}
}

// titleWrap は画面外（余白込み）に出た座標を反対側へ回り込ませる（漂流を継続させる）。
func titleWrap(x, y *float64) {
	const m = 80
	switch {
	case *x < -m:
		*x = titleScreenW + m
	case *x > titleScreenW+m:
		*x = -m
	}
	switch {
	case *y < -m:
		*y = titleScreenH + m
	case *y > titleScreenH+m:
		*y = -m
	}
}

func titleOffscreen(x, y, m float64) bool {
	return x < -m || x > titleScreenW+m || y < -m || y > titleScreenH+m
}

// drawTitlePlanet は drawCelestialBackdrop と同じ陰影で惑星を1つ描く（左上の青い惑星用）。
func drawTitlePlanet(dst *ebiten.Image, cx, cy, r float32, base color.NRGBA) {
	dark := scaleColor(base, 0.9)
	dark.A = 255
	vector.FillCircle(dst, cx, cy, r, dark, true)
	lit := base
	lit.A = 255
	vector.FillCircle(dst, cx-r*0.15, cy-r*0.15, r*0.78, lit, true)
	hi := scaleColor(base, 1.35)
	hi.A = 220
	vector.FillCircle(dst, cx-r*0.42, cy-r*0.42, r*0.2, hi, true)
	rim := scaleColor(base, 0.7)
	rim.A = 255
	vector.StrokeCircle(dst, cx, cy, r, 1.0, rim, true)
}
