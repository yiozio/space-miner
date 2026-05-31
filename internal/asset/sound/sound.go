// Package sound は埋め込み音声アセットへのアクセスと
// 再生用の audio.Context／デコード済み PCM を提供する。
package sound

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

//go:embed burner.ogg
var burnerOgg []byte

type Sound struct {
	pcm    []byte
	frames int64
}

// sampleRate は audio.Context と再サンプル後の PCM の共通サンプリングレート。
const sampleRate = 48000

// vorbis.DecodeF32 が返す PCM は 32bit float ステレオのインターリーブ形式
// （[L0, R0, L1, R1, ...]）で、1 フレーム = 4 byte * 2ch = 8 byte。
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
// 初回呼び出しで全アセットの PCM デコードも一度だけ行う。
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

	var burnerPCM, burnerFrames = decodeOGG(burnerOgg)
	burnerSound = Sound{pcm: burnerPCM, frames: burnerFrames}
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

// decodeOGG は OGG を 32bit float ステレオ PCM へ展開し、
// バイト列とフレーム数を返す。原音のサンプリングレートが Context と異なる場合は
// sampleRate へリサンプルして揃える（vorbis.DecodeF32 はリサンプルしないため）。
// デコード失敗はアセット異常なので致命扱い。
func decodeOGG(b []byte) ([]byte, int64) {
	s, err := vorbis.DecodeF32(bytes.NewReader(b))
	if err != nil {
		log.Fatalf("sound: decode ogg: %v", err)
	}
	pcm, err := io.ReadAll(s)
	if err != nil {
		log.Fatalf("sound: read pcm: %v", err)
	}
	if s.SampleRate() != sampleRate {
		pcm = resampleF32(pcm, s.SampleRate(), sampleRate)
	}
	frames := int64(len(pcm) / bytesPerFrame)
	return pcm, frames
}

// resampleF32 は 32bit float ステレオ PCM を srcRate から dstRate へ線形補間で
// リサンプルする。効果音用途では線形補間で十分。
func resampleF32(src []byte, srcRate, dstRate int) []byte {
	if srcRate == dstRate || len(src) == 0 {
		return src
	}
	srcFrames := len(src) / bytesPerFrame
	dstFrames := int(int64(srcFrames) * int64(dstRate) / int64(srcRate))
	dst := make([]byte, dstFrames*bytesPerFrame)
	for i := range dstFrames {
		// 出力フレーム i に対応する入力位置（連続値）と前後フレーム・補間係数。
		pos := float64(i) * float64(srcRate) / float64(dstRate)
		i0 := int(pos)
		i1 := min(i0+1, srcFrames-1)
		frac := float32(pos - float64(i0))
		for ch := range 2 {
			v0 := math.Float32frombits(binary.LittleEndian.Uint32(src[i0*bytesPerFrame+ch*4:]))
			v1 := math.Float32frombits(binary.LittleEndian.Uint32(src[i1*bytesPerFrame+ch*4:]))
			v := v0 + (v1-v0)*frac
			binary.LittleEndian.PutUint32(dst[i*bytesPerFrame+ch*4:], math.Float32bits(v))
		}
	}
	return dst
}
