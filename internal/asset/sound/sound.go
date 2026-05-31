// Package sound は再生用の audio.Context と、手続き的に合成した効果音 PCM
// （回転音・バーナー音）を提供する。
package sound

import (
	"encoding/binary"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Sound struct {
	pcm    []byte
	frames int64
}

// sampleRate は audio.Context と合成 PCM の共通サンプリングレート。
const sampleRate = 48000

// 合成 PCM は 32bit float ステレオのインターリーブ形式（[L0, R0, L1, R1, ...]）で、
// 1 フレーム = 4 byte * 2ch = 8 byte。
const bytesPerFrame = 8

var (
	initOnce sync.Once
	ctx      *audio.Context

	rotateSound         Sound
	rotateReversedSound Sound

	// rotateSustainPCM は回転音のループ用サステイン断片（loopHigh→loopLow→loopHigh）。
	// 両端・中央が同一サンプルのため継ぎ目なくループできる。初期化時に一度だけ構築。
	rotateSustainPCM []byte

	burnerSound Sound // ロケットバーナー音の原音
)

// Context は audio.Context を遅延初期化して返す。
// 初回呼び出しで全効果音の PCM 合成も一度だけ行う。
func Context() *audio.Context {
	initOnce.Do(initAudio)
	return ctx
}

func initAudio() {
	ctx = audio.NewContext(sampleRate)
	var rotatePCM, rotateFrames = generateRotatePCM()
	var rotateReversedPCM = reversePCM(rotatePCM)
	rotateSound = Sound{pcm: rotatePCM, frames: rotateFrames}
	rotateReversedSound = Sound{pcm: rotateReversedPCM, frames: rotateFrames}

	// サステイン = 逆再生(loopHigh→loopLow) + 順再生(loopLow→loopHigh)。継ぎ目
	// （loopLow）とループ折り返し（loopHigh）が同一サンプルになり無音クリックが出ない。
	rev := reverseClip(loopLow, loopHigh)
	fwd := rotateSound.forwardClip(loopLow, loopHigh)
	rotateSustainPCM = make([]byte, 0, len(rev)+len(fwd))
	rotateSustainPCM = append(rotateSustainPCM, rev...)
	rotateSustainPCM = append(rotateSustainPCM, fwd...)

	var burnerPCM, burnerFrames = generateBurnerPCM()
	burnerSound = Sound{pcm: burnerPCM, frames: burnerFrames}

	buildSfx()
}

// 回転音の合成パラメータ。OGG を読み込む代わりに正弦波＋ホワイトノイズを
// 手続き的に生成して原音とする。音色はここの値で調整する。
const (
	rotateGenDur   = 4 * time.Second // 生成する原音長（loopHigh 以上であること）
	rotateFreq     = 440.0           // 正弦波の周波数 [Hz]
	rotateSineAmp  = 0.6             // 正弦波の振幅（0.0〜1.0）
	rotateNoiseAmp = 0.4             // ホワイトノイズの振幅（0.0〜1.0）
)

// generateRotatePCM は回転音の原音を合成する。正弦波にホワイトノイズを混ぜた
// 32bit float ステレオ PCM（先頭からの順再生用）とフレーム数を返す。
// ノイズはチャンネル独立に与えてステレオ感を出す。逆再生してもノイズ・正弦波
// ともに性質が変わらないため、既存の逆再生ロジックはそのまま使える。
func generateRotatePCM() ([]byte, int64) {
	frames := int(int64(rotateGenDur) * sampleRate / int64(time.Second))
	pcm := make([]byte, frames*bytesPerFrame)
	rng := rand.New(rand.NewSource(1)) // 再現性のため固定シード
	step := 2 * math.Pi * rotateFreq / sampleRate
	for f := range frames {
		sine := rotateSineAmp * float32(math.Sin(step*float64(f)))
		for ch := range 2 {
			noise := rotateNoiseAmp * (rng.Float32()*2 - 1)
			v := max(-1, min(sine+noise, 1)) // クリッピング防止
			binary.LittleEndian.PutUint32(pcm[f*bytesPerFrame+ch*4:], math.Float32bits(v))
		}
	}
	return pcm, int64(frames)
}

// バーナー音の合成パラメータ。低音のホワイトノイズ（白色雑音に一極ローパスを
// 掛けて高域を削ったもの）を生成し、重いごう音にする。音色はここの値で調整する。
const (
	burnerGenDur    = 11 * time.Second // 生成する原音長（burnerIntroEnd 以上であること）
	burnerNoiseCut  = 200.0            // ローパスのカットオフ周波数 [Hz]（低いほど重い）
	burnerNoisePeak = 0.9              // 正規化後のピーク振幅（0.0〜1.0）
)

// generateBurnerPCM は低音のホワイトノイズを合成する。白色雑音を一極ローパスで
// 濾過して高域を落とし、ローパスで小さくなった振幅をピーク基準で正規化する。
// ノイズ・フィルタ状態はチャンネル独立に持たせてステレオ感を出す。
func generateBurnerPCM() ([]byte, int64) {
	frames := int(int64(burnerGenDur) * sampleRate / int64(time.Second))
	rng := rand.New(rand.NewSource(2)) // 再現性のため固定シード

	// 一極ローパス係数 alpha = wc/(wc+1)（wc = 2π·fc/fs）。
	wc := 2 * math.Pi * burnerNoiseCut / sampleRate
	alpha := float32(wc / (wc + 1))

	samples := make([]float32, frames*2)
	var lp [2]float32 // チャンネルごとのフィルタ状態
	var peak float32
	for f := range frames {
		for ch := range 2 {
			white := rng.Float32()*2 - 1
			lp[ch] += alpha * (white - lp[ch])
			samples[f*2+ch] = lp[ch]
			if a := float32(math.Abs(float64(lp[ch]))); a > peak {
				peak = a
			}
		}
	}

	// ローパスで下がった振幅をピーク基準で burnerNoisePeak まで持ち上げる。
	gain := float32(1)
	if peak > 0 {
		gain = burnerNoisePeak / peak
	}
	pcm := make([]byte, frames*bytesPerFrame)
	for i, v := range samples {
		off := (i/2)*bytesPerFrame + (i%2)*4
		binary.LittleEndian.PutUint32(pcm[off:], math.Float32bits(v*gain))
	}
	return pcm, int64(frames)
}

// reversePCM は 32bit float ステレオ PCM をフレーム単位で時間反転する。
// 1 フレーム（8 byte = L 4byte + R 4byte）を単位に末尾から並べ直すので、
// フレーム内の L/R 並びは保たれる（= そのまま逆再生になる）。
func reversePCM(src []byte) []byte {
	n := len(src) / bytesPerFrame
	dst := make([]byte, len(src))
	for i := range n {
		copy(dst[i*bytesPerFrame:(i+1)*bytesPerFrame], src[(n-1-i)*bytesPerFrame:(n-i)*bytesPerFrame])
	}
	return dst
}

// frameAt は原音先頭からの時刻 d に対応するフレーム位置を返す。
// [0, s.frames] にクランプする。
func (s Sound) frameAt(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	f := int64(d) * sampleRate / int64(time.Second)
	return min(f, s.frames)
}

// forwardClip は原音区間 [a, b] を順再生するための PCM 断片。
func (s Sound) forwardClip(a, b time.Duration) []byte {
	fa, fb := s.frameAt(a), s.frameAt(b)
	fb = max(fb, fa)
	return s.pcm[fa*bytesPerFrame : fb*bytesPerFrame]
}

// reverseClip は回転原音の区間 [a, b]（a < b）を b→a の向きに鳴らすための
// 逆再生バッファ断片。原音フレーム k は逆再生バッファのフレーム n-k に対応する。
func reverseClip(a, b time.Duration) []byte {
	n := rotateSound.frames
	fa, fb := rotateSound.frameAt(a), rotateSound.frameAt(b)
	fb = max(fb, fa)
	return rotateReversedSound.pcm[(n-fb)*bytesPerFrame : (n-fa)*bytesPerFrame]
}
