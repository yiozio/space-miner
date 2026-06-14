package entity

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// droneLifeFrames は設置から消滅までのフレーム数（約10秒）。
	droneLifeFrames = 600
)

// DroneMode はドローンの攻撃方式。
type DroneMode int

const (
	DroneBeam   DroneMode = iota // ビーム（継続 DPS、必中）
	DroneBullet                  // 弾（一定間隔で対象へ射出、命中判定は通常弾と同じ）
)

// droneColor はプレイヤー設置ドローンのアクセント色（シアン系）。
// プレイヤー弾（テーマ色）や AutoAim ビーム（琥珀）と区別する。
var droneColor = color.NRGBA{0x40, 0xe0, 0xc0, 0xff}

// droneHostileColor は敵機（海賊）が設置したドローンのアクセント色（赤系）。
// 敵弾・敵機ラインと同系統で「敵性」を伝える。
var droneHostileColor = color.NRGBA{0xff, 0x60, 0x40, 0xff}

// Drone は発射時に設置される自律攻撃ドローン。寿命 droneLifeFrames が尽きると消滅する。
// Hostile=false（プレイヤー設置）は射程内で最も近い小惑星グリッドまたは海賊を、
// Hostile=true（敵機設置）は自機を狙う。設置後は移動しない。
// Mode で攻撃方式が変わる: DroneBeam は継続ダメージ（DPS）、DroneBullet は一定間隔で弾を射出する。
// 攻撃対象の探索・ダメージ適用・弾の登録はシーン側が担い、
// Drone 自身は位置・寿命・所属・性能パラメータと各種タイマーのみを保持する。
type Drone struct {
	X, Y    float64
	Life    int       // 残存フレーム（0 で消滅）
	Range   float64   // 攻撃射程（px）
	Hostile bool      // true なら敵機設置（自機を狙う）
	Mode    DroneMode // 攻撃方式（ビーム / 弾）

	// DroneBeam 用。
	DPS    float64 // 毎秒ダメージ
	dmgAcc float64 // 端数ダメージの蓄積（整数化できるまで持ち越す）

	// DroneBullet 用。
	BulletDamage    int
	BulletSpeed     float64
	BulletStyle     BulletStyle
	BulletWidth     float64
	BulletImpact    bool
	ExplosionRadius float64 // >0 なら弾が着弾時に範囲ダメージを与える
	FireInterval    int     // 弾の発射間隔（フレーム）
	fireTimer       int     // 次弾までの残りフレーム
}

// NewDroneFromDef は def の性能で (x, y) に設置するドローンを生成する。
// hostile は所属（true=敵機設置で自機を狙う）。def に弾速があれば弾モード、なければビームモード。
func NewDroneFromDef(def *PartDef, x, y float64, hostile bool) Drone {
	d := Drone{
		X: x, Y: y,
		Life:    droneLifeFrames,
		Range:   def.AutoAimRange,
		Hostile: hostile,
	}
	if def.GunBulletSpeed > 0 {
		d.Mode = DroneBullet
		d.BulletDamage = def.GunDamage
		d.BulletSpeed = def.GunBulletSpeed
		d.BulletStyle = def.GunBulletStyle
		d.BulletWidth = def.GunBulletWidth
		d.BulletImpact = def.GunBulletImpact
		d.ExplosionRadius = def.GunExplosionRadius
		d.FireInterval = def.DroneFireInterval
	} else {
		d.Mode = DroneBeam
		d.DPS = def.AutoAimDPS
	}
	return d
}

// Color はドローンの所属に応じた描画色を返す。
func (d *Drone) Color() color.NRGBA {
	if d.Hostile {
		return droneHostileColor
	}
	return droneColor
}

// Fire は弾モードの発射タイマーを 1 フレーム進め、発射すべきフレームなら true を返す。
// 射程内に対象があるフレームだけ呼ぶ想定。発射時は次弾までのタイマーを張り直す。
func (d *Drone) Fire() bool {
	if d.fireTimer > 0 {
		d.fireTimer--
		return false
	}
	d.fireTimer = d.FireInterval
	return true
}

// FireBullet は対象 (tx, ty) へ向かう弾を 1 発生成する（弾モード用）。
// Hostile はドローンの所属を引き継ぐ（敵機設置なら自機を撃つ敵弾になる）。
func (d *Drone) FireBullet(tx, ty float64) Bullet {
	ang := math.Atan2(ty-d.Y, tx-d.X)
	return Bullet{
		X: d.X, Y: d.Y,
		VX:              math.Cos(ang) * d.BulletSpeed,
		VY:              math.Sin(ang) * d.BulletSpeed,
		Life:            bulletLifeFrames,
		Damage:          d.BulletDamage,
		Hostile:         d.Hostile,
		Style:           d.BulletStyle,
		Width:           d.BulletWidth,
		ImpactFX:        d.BulletImpact,
		ExplosionRadius: d.ExplosionRadius,
	}
}

// Update は寿命を進め、消滅したら true を返す。
func (d *Drone) Update() (expired bool) {
	d.Life--
	return d.Life <= 0
}

// TickDamage は DPS を 1 フレーム分蓄積し、整数化できたぶんを返す（端数は持ち越す）。
// 射程内に対象があるフレームだけ呼ぶ想定。
func (d *Drone) TickDamage() int {
	d.dmgAcc += d.DPS / 60.0
	dmg := int(d.dmgAcc)
	d.dmgAcc -= float64(dmg)
	return dmg
}

// Draw はドローン本体を菱形＋中心点で描画する。寿命終盤は点滅して消滅が近いことを伝える。
// sx, sy は描画スクリーン座標。
func (d *Drone) Draw(dst *ebiten.Image, sx, sy float64) {
	// 残り 1 秒を切ったら点滅させる。
	if d.Life < 60 {
		period := d.Life/6 + 2
		if (d.Life/period)%2 != 0 {
			return // 消灯フェーズ
		}
	}
	c := d.Color()
	x := float32(sx)
	y := float32(sy)
	const r = 5
	// 菱形（45° 回転の四角形）の輪郭。
	vector.StrokeLine(dst, x, y-r, x+r, y, 1.5, c, true)
	vector.StrokeLine(dst, x+r, y, x, y+r, 1.5, c, true)
	vector.StrokeLine(dst, x, y+r, x-r, y, 1.5, c, true)
	vector.StrokeLine(dst, x-r, y, x, y-r, 1.5, c, true)
	vector.FillCircle(dst, x, y, 1.5, c, true)
}
