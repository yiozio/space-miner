package sound

import (
	"bytes"
	"log"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// ambience.go はタイトル画面の BGM（cosmic_noise_sound.mp3）を、
// 再生終了後に少し間を置いて繰り返し流す（ストリーミング再生）。

//go:embed cosmic_noise_sound.mp3
var cosmicNoiseMP3 []byte

const (
	titleBGMVol       = 0.5
	titleBGMGapFrames = 120 // 再生終了後の待ち時間（2 秒 @60fps）
)

var (
	bgmPlayer *audio.Player
	bgmActive bool // タイトル BGM を回し続けるか
	bgmGap    int  // 次の再生までの残フレーム（プレイヤー無しのとき有効）
)

// PlayTitleBGM はタイトル BGM の繰り返し再生を開始する。
// TickTitleBGM を毎フレーム呼ぶことで「終了 → 2 秒待機 → 再生」を繰り返す。
func PlayTitleBGM() {
	StopBGM()
	bgmActive = true
	bgmGap = 0
	startBGMClip()
}

// TickTitleBGM は毎フレーム呼び、再生終了を検知して 2 秒後に再生し直す。
func TickTitleBGM() {
	if !bgmActive {
		return
	}
	if bgmPlayer != nil {
		if bgmPlayer.IsPlaying() {
			return
		}
		// 再生終了 → 解放して待機開始
		_ = bgmPlayer.Close()
		bgmPlayer = nil
		bgmGap = titleBGMGapFrames
		return
	}
	// プレイヤー無し = 待機中
	if bgmGap > 0 {
		bgmGap--
		return
	}
	startBGMClip()
}

// StopBGM はタイトル BGM を停止する。ゲーム開始時などに呼ぶ。
func StopBGM() {
	bgmActive = false
	bgmGap = 0
	if bgmPlayer != nil {
		bgmPlayer.Pause()
		_ = bgmPlayer.Close()
		bgmPlayer = nil
	}
}

// startBGMClip は MP3 を 1 回ストリーミング再生する（全展開しないので省メモリ）。
func startBGMClip() {
	ctx := Context()
	s, err := mp3.DecodeF32(bytes.NewReader(cosmicNoiseMP3))
	if err != nil {
		log.Printf("sound: decode title bgm: %v", err)
		return
	}
	p, err := ctx.NewPlayerF32(s)
	if err != nil {
		log.Printf("sound: title bgm player: %v", err)
		return
	}
	p.SetVolume(titleBGMVol)
	p.Play()
	bgmPlayer = p
}
