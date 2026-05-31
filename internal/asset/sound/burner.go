package sound

import (
	"bytes"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// BurnerSound はロケットバーナー（推進）に同期して burner 音を再生する。
//
//   - 開始: 原音 0s→10s を順再生（イントロ）
//   - 以降: 原音 1s→10s を継ぎ目なく無限ループ
//   - 終了: 再生を停止する（逆再生などはしない）
//
// イントロとループは NewInfiniteLoopWithIntroF32 による単一プレイヤーで再生する。
// active フラグの立ち上がりで先頭から再生し、立ち下がりで停止する。
type BurnerSound struct {
	ctx      *audio.Context
	buf      []byte // イントロ(0→10s) ++ ループ(1→10s)
	introLen int64
	loopLen  int64
	playing  bool
	boosting bool // 現在ブースト音量で鳴らしているか
	player   *audio.Player
}

// バーナー音のイントロ／ループ区間。原音の絶対時刻。
const (
	burnerIntroEnd  = 10 * time.Second // イントロ終端兼ループ終端
	burnerLoopStart = 1 * time.Second  // ループ先頭
)

// burnerLoopBlend はループ折り返しを滑らかにするためのブレンド長。
// ループ末尾に「ループ先頭からの短い断片」を継ぎ足しておくと、InfiniteLoop が
// それを使ってループ接合部をクロスフェードし、周期的なクリックを抑える。
const burnerLoopBlend = 100 * time.Millisecond

// burnerVolume はバーナー音の通常音量（0.0〜1.0）。
const burnerVolume = 0.005

// burnerBoostMul はシフト（ブースト）中の音量倍率。
const burnerBoostMul = 1.5

// NewBurnerSound は audio.Context を初期化し、再生用バッファを構築して返す。
func NewBurnerSound() *BurnerSound {
	ctx := Context()
	intro := burnerSound.forwardClip(0, burnerIntroEnd)
	loop := burnerSound.forwardClip(burnerLoopStart, burnerIntroEnd)
	// ループ末尾に付けるブレンド用断片（ループ先頭の続き）。InfiniteLoop の
	// 「ループ終端より後ろのデータ」として扱われ、接合部の平滑化にのみ使われる。
	blend := burnerSound.forwardClip(burnerLoopStart, burnerLoopStart+burnerLoopBlend)
	buf := make([]byte, 0, len(intro)+len(loop)+len(blend))
	buf = append(buf, intro...)
	buf = append(buf, loop...)
	buf = append(buf, blend...)
	return &BurnerSound{
		ctx:      ctx,
		buf:      buf,
		introLen: int64(len(intro)),
		loopLen:  int64(len(loop)),
	}
}

// Update は active（推進中フラグ）と boosting（ブースト中フラグ）に応じて
// 再生を開始／停止し、再生中はブースト状態に合わせて音量を切り替える。
func (b *BurnerSound) Update(active, boosting bool) {
	switch {
	case !active:
		if b.playing {
			b.Stop()
		}
	case !b.playing:
		b.start(boosting)
	case boosting != b.boosting:
		// 再生は止めずに音量だけブースト／通常へ切り替える。
		b.boosting = boosting
		if b.player != nil {
			b.player.SetVolume(b.volume())
		}
	}
}

// volume は現在の音量（ブースト中は倍率を掛ける）を返す。
func (b *BurnerSound) volume() float64 {
	if b.boosting {
		return burnerVolume * burnerBoostMul
	}
	return burnerVolume
}

// Stop は再生を停止・解放する。シーン切り替えなどでも使う。
func (b *BurnerSound) Stop() {
	if b.player != nil {
		b.player.Pause()
		_ = b.player.Close()
		b.player = nil
	}
	b.playing = false
}

// start はイントロ＋ループ無限再生を先頭から始める。
func (b *BurnerSound) start(boosting bool) {
	b.Stop()
	b.boosting = boosting
	loop := audio.NewInfiniteLoopWithIntroF32(bytes.NewReader(b.buf), b.introLen, b.loopLen)
	p, err := b.ctx.NewPlayerF32(loop)
	if err != nil {
		log.Printf("sound: burner player: %v", err)
		return
	}
	// 停止時に残り音が長く流れ続けないよう内部バッファを抑える。
	p.SetBufferSize(playerBufferSize)
	p.SetVolume(b.volume())
	p.Play()
	b.player = p
	b.playing = true
}
