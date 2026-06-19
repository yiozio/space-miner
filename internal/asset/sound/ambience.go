package sound

import (
	"bytes"
	"io"
	"log"

	_ "embed"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// ambience.go は BGM（タイトル／ゲーム中）を管理する。BGM チャンネルは1つで、
// タイトル BGM とゲーム BGM のどちらか一方だけが鳴る。
//
//   - タイトル BGM: cosmic_noise_sound.mp3 を再生終了後 2 秒待って繰り返す
//     （ストリーミング再生＋毎フレームの TickTitleBGM）。
//   - ゲーム BGM  : space_noise.wav を継ぎ目なくループ（ストリーミング再生）。
//     メニュー／ステーション等のサブ画面表示中は StopBGM で止める。

//go:embed cosmic_noise_sound.mp3
var cosmicNoiseMP3 []byte

//go:embed space_noise.wav
var spaceNoiseWAV []byte

const (
	titleBGMVol       = 0.5
	titleBGMGapFrames = 120 // タイトル BGM の再生終了後の待ち（2 秒 @60fps）
	gameBGMVol        = 0.4
)

type bgmKind int

const (
	bgmNone bgmKind = iota
	bgmTitle
	bgmGame
)

var (
	bgmPlayer *audio.Player
	bgmCur    bgmKind
	bgmActive bool // タイトル BGM の繰り返し制御（tick 用）
	bgmGap    int  // タイトル BGM の次の再生までの残フレーム
)

// PlayTitleBGM はタイトル BGM の繰り返し再生を開始する。
// TickTitleBGM を毎フレーム呼ぶことで「終了 → 2 秒待機 → 再生」を繰り返す。
func PlayTitleBGM() {
	StopBGM()
	bgmCur = bgmTitle
	bgmActive = true
	bgmGap = 0
	startTitleClip()
}

// TickTitleBGM は毎フレーム呼び、タイトル BGM の終了を検知して 2 秒後に再生し直す。
func TickTitleBGM() {
	if !bgmActive {
		return
	}
	if bgmPlayer != nil {
		if bgmPlayer.IsPlaying() {
			return
		}
		_ = bgmPlayer.Close()
		bgmPlayer = nil
		bgmGap = titleBGMGapFrames
		return
	}
	if bgmGap > 0 {
		bgmGap--
		return
	}
	startTitleClip()
}

// PlayGameBGM はゲーム中の BGM（space_noise.wav）を継ぎ目なくループ再生する。
// 既にゲーム BGM が鳴っていれば何もしない（毎フレーム呼んでよい）。
func PlayGameBGM() {
	if bgmCur == bgmGame && bgmPlayer != nil && bgmPlayer.IsPlaying() {
		return
	}
	StopBGM()
	ctx := Context()
	// 事前デコード済みバッファがあれば即再生（デコード待ちなし）。無ければその場でデコード。
	var src io.ReadSeeker
	var length int64
	if gameBGMPCM != nil {
		src = bytes.NewReader(gameBGMPCM)
		length = int64(len(gameBGMPCM))
	} else {
		s, err := wav.DecodeF32(bytes.NewReader(spaceNoiseWAV))
		if err != nil {
			log.Printf("sound: decode game bgm: %v", err)
			return
		}
		src = s
		length = s.Length()
	}
	loop := audio.NewInfiniteLoopF32(src, length)
	p, err := ctx.NewPlayerF32(loop)
	if err != nil {
		log.Printf("sound: game bgm player: %v", err)
		return
	}
	p.SetVolume(gameBGMVol)
	p.Play()
	bgmPlayer = p
	bgmCur = bgmGame
}

// StopBGM は再生中の BGM を停止・解放する。
func StopBGM() {
	bgmActive = false
	bgmGap = 0
	bgmCur = bgmNone
	if bgmPlayer != nil {
		bgmPlayer.Pause()
		_ = bgmPlayer.Close()
		bgmPlayer = nil
	}
}

// startTitleClip はタイトル BGM の mp3 を 1 回ストリーミング再生する。
func startTitleClip() {
	ctx := Context()
	// 事前デコード済みバッファがあれば即再生（デコード待ちなし）。無ければその場でデコード。
	var src io.Reader
	if titleBGMPCM != nil {
		src = bytes.NewReader(titleBGMPCM)
	} else {
		s, err := mp3.DecodeF32(bytes.NewReader(cosmicNoiseMP3))
		if err != nil {
			log.Printf("sound: decode title bgm: %v", err)
			return
		}
		src = s
	}
	p, err := ctx.NewPlayerF32(src)
	if err != nil {
		log.Printf("sound: title bgm player: %v", err)
		return
	}
	p.SetVolume(titleBGMVol)
	p.Play()
	bgmPlayer = p
}
