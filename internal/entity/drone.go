package entity

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// droneLifeFrames は設置から消滅までのフレーム数（約10秒）。
	droneLifeFrames = 600
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
// 攻撃対象の探索・ダメージ適用はシーン側が担い、Drone 自身は位置・寿命・所属・
// 性能パラメータ（射程・DPS）と端数ダメージの蓄積のみを保持する。
type Drone struct {
	X, Y    float64
	Life    int     // 残存フレーム（0 で消滅）
	Range   float64 // 攻撃射程（px）
	DPS     float64 // 毎秒ダメージ
	Hostile bool    // true なら敵機設置（自機を狙う）
	dmgAcc  float64 // 端数ダメージの蓄積（整数化できるまで持ち越す）
}

// Color はドローンの所属に応じた描画色を返す。
func (d *Drone) Color() color.NRGBA {
	if d.Hostile {
		return droneHostileColor
	}
	return droneColor
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
