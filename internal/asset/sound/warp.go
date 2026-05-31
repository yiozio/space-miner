package sound

import (
	"math"
	"math/rand"
	"time"
)

// warp.go はワープ演出の効果音（ワンショット）を手続き的に合成して提供する。
//
// ワープ演出の前半（テレポートまで＝45 フレーム≒0.75 秒 @60fps）は、ローパスを掛けた
// ホワイトノイズを徐々に大きくして緊張感を高める。
// テレポートの瞬間（中点）には、アタック強めのローパスノイズと、低→さらに低へ下降
// する正弦波を鳴らす。

// 合成済み PCM（初期化時に一度だけ構築）。
var (
	warpPCM     []byte
	warpJumpPCM []byte
)

// ワープ前半音（上昇＝徐々に大きくなるローパスノイズ）の合成パラメータ。
const (
	warpVol      = 0.35
	warpDur      = 750 * time.Millisecond // ワープ前半（テレポートまで＝45f@60fps）
	warpNoiseCut = 2000.0                 // ノイズのローパス・カットオフ [Hz]
	warpRelease  = 15 * time.Millisecond  // 末尾の短いリリース（クリック防止）
)

// ワープ確定（テレポート→完了）の後半音の合成パラメータ。
// アタック強めのローパスノイズ（先頭最大→速い減衰）と、低→さらに低へ下降する
// 正弦波（前半の正弦波とは繋げない独立した音）を重ねる。
const (
	warpJumpVol = 0.5
	warpJumpDur = 750 * time.Millisecond // ワープ後半（テレポート→完了＝45f@60fps）

	// アタックのローパスノイズ。
	warpJumpCut        = 600.0 // ローパス・カットオフ [Hz]
	warpJumpNoiseAmp   = 0.7   // ノイズの混合量
	warpJumpNoiseDecay = 90 * time.Millisecond

	// 後半の下降正弦波（低→さらに低）。
	warpJumpSineHi    = 120.0 // 開始周波数 [Hz]（低め）
	warpJumpSineLo    = 40.0  // 終了周波数 [Hz]（さらに低く）
	warpJumpSineAmp   = 0.6   // 正弦波の混合量
	warpJumpSineDecay = 350 * time.Millisecond

	warpJumpRelease = 40 * time.Millisecond
)

// buildWarpSounds はワープ音 PCM を生成する。buildSfx から呼ばれる。
func buildWarpSounds() {
	warpPCM = genWarpPCM()
	warpJumpPCM = genWarpJumpPCM()
}

// PlayWarp は前半の上昇音、PlayWarpJump はテレポート時のノイズ音を再生する。
func PlayWarp()     { playOneShot(warpPCM, warpVol) }
func PlayWarpJump() { playOneShot(warpJumpPCM, warpJumpVol) }

// genWarpPCM はローパスを掛けたホワイトノイズを徐々に大きくしていく前半の上昇音を作る。
// 音量は 0→最大のスウェルで緊張感を高め、末尾だけ短いリリースで締める。
func genWarpPCM() []byte {
	n := frameCount(warpDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(37))
	alpha := lowpassAlpha(warpNoiseCut)
	rel := frameCount(warpRelease)
	var lp float32
	for i := range n {
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		// 徐々に大きく（0→最大のスウェル）。末尾だけ短いリリース。
		env := float32(i) / float32(n-1)
		if i >= n-rel {
			env *= float32(n-1-i) / float32(rel)
		}
		mono[i] = lp * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genWarpJumpPCM はワープ後半音を作る。アタック強めのローパスノイズ（先頭最大→速い
// 指数減衰）と、低→さらに低へ指数で下降する正弦波（ゆるやかに減衰）を重ね、末尾だけ
// リリースで締める。立ち上がりはフェードしない（鋭い当たり）。
func genWarpJumpPCM() []byte {
	n := frameCount(warpJumpDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(41))
	alpha := lowpassAlpha(warpJumpCut)
	noiseTau := float64(warpJumpNoiseDecay) / float64(time.Second)
	sineTau := float64(warpJumpSineDecay) / float64(time.Second)
	rel := frameCount(warpJumpRelease)
	var lp float32
	var phase float64 // 0〜1 に正規化した正弦位相
	for i := range n {
		t := float64(i) / float64(n-1)
		// アタックのローパスノイズ（先頭最大→速い減衰）。
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		noise := warpJumpNoiseAmp * lp * float32(math.Exp(-float64(i)/sampleRate/noiseTau))
		// 低→さらに低へ下降する正弦波（ゆるやかに減衰）。
		freq := warpJumpSineHi * math.Pow(warpJumpSineLo/warpJumpSineHi, t)
		phase += freq / sampleRate
		phase -= math.Floor(phase)
		sine := warpJumpSineAmp * float32(math.Sin(2*math.Pi*phase)) *
			float32(math.Exp(-float64(i)/sampleRate/sineTau))
		// 共通の末尾リリース。
		env := float32(1)
		if i >= n-rel {
			env = float32(n-1-i) / float32(rel)
		}
		mono[i] = (noise + sine) * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}
