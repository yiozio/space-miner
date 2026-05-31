// Package sound は埋め込み音声アセットへのアクセスと
// 再生用の audio.Context／デコード済み PCM を提供する。
package sound

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

/* 14s 32bit(16bit x 2ch(stereo)) */
//go:embed rotate.ogg
var rotateOgg []byte

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

	// rotateSustainPCM は回転音のループ用サステイン断片（原音 5s→4s→5s）。
	// 両端・中央が同一サンプルのため継ぎ目なくループできる。初期化時に一度だけ構築。
	rotateSustainPCM []byte
)

// Context は audio.Context を遅延初期化して返す。
// 初回呼び出しで全アセットの PCM デコードも一度だけ行う。
func Context() *audio.Context {
	initOnce.Do(initAudio)
	return ctx
}

func initAudio() {
	ctx = audio.NewContext(sampleRate)
	var rotatePCM, rotateFrames = decodeOGG(rotateOgg)
	var rotateReversedPCM = reversePCM(rotatePCM)
	rotateSound = Sound{pcm: rotatePCM, frames: rotateFrames}
	rotateReversedSound = Sound{pcm: rotateReversedPCM, frames: rotateFrames}

	// サステイン = 逆再生(5s→4s) + 順再生(4s→5s)。継ぎ目（4s）と
	// ループ折り返し（5s）が同一サンプルになるので無音クリックが出ない。
	rev := reverseClip(loopLow, loopHigh)
	fwd := forwardClip(loopLow, loopHigh)
	rotateSustainPCM = make([]byte, 0, len(rev)+len(fwd))
	rotateSustainPCM = append(rotateSustainPCM, rev...)
	rotateSustainPCM = append(rotateSustainPCM, fwd...)
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
// [0, rotateSound.frames] にクランプする。
func frameAt(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	f := int64(d) * sampleRate / int64(time.Second)
	return min(f, rotateSound.frames)
}

// forwardClip は原音区間 [a, b] を順再生するための PCM 断片。
func forwardClip(a, b time.Duration) []byte {
	fa, fb := frameAt(a), frameAt(b)
	fb = max(fb, fa)
	return rotateSound.pcm[fa*bytesPerFrame : fb*bytesPerFrame]
}

// reverseClip は原音区間 [a, b]（a < b）を b→a の向きに鳴らすための
// 逆再生バッファ断片。原音フレーム k は逆再生バッファのフレーム n-k に対応する。
func reverseClip(a, b time.Duration) []byte {
	n := rotateSound.frames
	fa, fb := frameAt(a), frameAt(b)
	fb = max(fb, fa)
	return rotateReversedSound.pcm[(n-fb)*bytesPerFrame : (n-fa)*bytesPerFrame]
}

// decodeOGG は OGG を 32bit float ステレオ PCM へ展開し、
// バイト列とフレーム数を返す。デコード失敗はアセット異常なので致命扱い。
func decodeOGG(b []byte) ([]byte, int64) {
	s, err := vorbis.DecodeF32(bytes.NewReader(b))
	if err != nil {
		log.Fatalf("sound: decode ogg: %v", err)
	}
	pcm, err := io.ReadAll(s)
	if err != nil {
		log.Fatalf("sound: read pcm: %v", err)
	}
	frames := s.Length() / bytesPerFrame
	return pcm, frames
}
