package sound

import (
	"math"
	"time"
)

// ui.go はメニュー操作の効果音（ワンショット）を手続き的に合成して提供する。
//
//   - メニュー移動音（Move）  : 短く高い正弦波の「ピッ」。
//   - メニュー決定音（Select）: 低めの正弦波を指数減衰させた柔らかい「ポロン」。
//
// 再生は他の効果音と同じ fire-and-forget（playOneShot）。

// 合成済み PCM（初期化時に一度だけ構築）。
var (
	menuMovePCM   []byte
	menuSelectPCM []byte
	menuCancelPCM []byte
)

// 再生音量（0.0〜1.0）。
const (
	menuMoveVol   = 0.1
	menuSelectVol = 0.1
	menuCancelVol = 0.1
)

// メニュー移動音（ピッ）の合成パラメータ。短い正弦波にアタック／リリース。
const (
	menuMoveDur     = 60 * time.Millisecond
	menuMoveFreq    = 1400.0 // 高め
	menuMoveAttack  = 4 * time.Millisecond
	menuMoveRelease = 14 * time.Millisecond
)

// メニュー決定音（ポロン）の合成パラメータ。低めの正弦波＋2倍音を指数減衰。
// キャンセル音は同じ形でさらに低い基音を使う。
const (
	menuSelectDur     = 260 * time.Millisecond
	menuSelectFreq    = 440.0 // 決定音の基音
	menuCancelFreq    = 300.0 // キャンセル音の基音（決定音より低め）
	menuSelectHarmAmp = 0.3   // 2 倍音の混合量（プラックの厚み）
	menuSelectDecay   = 90 * time.Millisecond
	menuSelectAttack  = 3 * time.Millisecond
	menuSelectRelease = 20 * time.Millisecond
)

// buildUISounds は全 UI 効果音 PCM を生成する。buildSfx から呼ばれる。
func buildUISounds() {
	menuMovePCM = genMenuMovePCM()
	menuSelectPCM = genPluckPCM(menuSelectFreq)
	menuCancelPCM = genPluckPCM(menuCancelFreq)
}

// PlayMenuMove は移動音、PlayMenuSelect は決定音、PlayMenuCancel はキャンセル音
// （決定音より低い）を再生する。
func PlayMenuMove()   { playOneShot(menuMovePCM, menuMoveVol) }
func PlayMenuSelect() { playOneShot(menuSelectPCM, menuSelectVol) }
func PlayMenuCancel() { playOneShot(menuCancelPCM, menuCancelVol) }

// genMenuMovePCM は短い正弦波にアタック／リリースを掛けた「ピッ」を作る。
func genMenuMovePCM() []byte {
	n := frameCount(menuMoveDur)
	mono := make([]float32, n)
	step := 2 * math.Pi * menuMoveFreq / sampleRate
	atk := frameCount(menuMoveAttack)
	rel := frameCount(menuMoveRelease)
	for i := range n {
		s := float32(math.Sin(step * float64(i)))
		env := float32(1)
		switch {
		case i < atk:
			env = float32(i) / float32(atk)
		case i >= n-rel:
			env = float32(n-1-i) / float32(rel)
		}
		mono[i] = s * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genPluckPCM は基音 freq の正弦波＋2倍音を指数減衰させた柔らかい「ポロン」を作る。
// 決定音・キャンセル音で基音だけ変えて共用する。
func genPluckPCM(freq float64) []byte {
	n := frameCount(menuSelectDur)
	mono := make([]float32, n)
	step := 2 * math.Pi * freq / sampleRate
	tau := float64(menuSelectDecay) / float64(time.Second)
	atk := frameCount(menuSelectAttack)
	rel := frameCount(menuSelectRelease)
	for i := range n {
		ph := step * float64(i)
		s := float32(math.Sin(ph)) + menuSelectHarmAmp*float32(math.Sin(2*ph))
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		if i < atk {
			env *= float32(i) / float32(atk)
		}
		if i >= n-rel {
			env *= float32(n-1-i) / float32(rel)
		}
		mono[i] = s * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}
