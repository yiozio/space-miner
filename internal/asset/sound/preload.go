package sound

import (
	"bytes"
	"io"
	"log"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// preload.go は起動時の音声プリロードを担う。audio.Context の初期化（効果音 PCM 合成）と
// BGM のフルデコードを先行して行い、完了を AudioReady で公開する。スプラッシュ画面が
// これと惑星アセットの両方の完了を待つことで、タイトル表示時に BGM を即再生できる
// （特にブラウザでの「タイトル表示→音楽再生」のタイムラグを解消する）。

var (
	audioReady  atomic.Bool
	preloadOnce sync.Once

	titleBGMPCM []byte // 事前フルデコード済みのタイトル BGM（F32 PCM）。nil ならその場でデコード。
	gameBGMPCM  []byte // 事前フルデコード済みのゲーム BGM（F32 PCM）。
)

// Preload は音声の初期化（PCM 合成）と BGM のフルデコードを行い、完了で AudioReady を立てる。
// 重い処理なので別ゴルーチンから呼ぶ前提（複数回呼んでも一度だけ実行）。
func Preload() {
	preloadOnce.Do(func() {
		Context() // audio.Context 初期化＋効果音 PCM 合成（重い）
		if s, err := mp3.DecodeF32(bytes.NewReader(cosmicNoiseMP3)); err == nil {
			titleBGMPCM = readAll("title bgm", s)
		} else {
			log.Printf("sound: preload decode title bgm: %v", err)
		}
		if s, err := wav.DecodeF32(bytes.NewReader(spaceNoiseWAV)); err == nil {
			gameBGMPCM = readAll("game bgm", s)
		} else {
			log.Printf("sound: preload decode game bgm: %v", err)
		}
		audioReady.Store(true)
	})
}

// AudioReady は Preload（効果音 PCM 合成・BGM フルデコード）の完了を返す。
func AudioReady() bool { return audioReady.Load() }

// readAll はデコーダ（ストリーム）を最後まで読み切り、フル PCM バッファを返す。失敗時は nil。
func readAll(name string, r io.Reader) []byte {
	buf, err := io.ReadAll(r)
	if err != nil {
		log.Printf("sound: preload read %s: %v", name, err)
		return nil
	}
	return buf
}
