package sound

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// RotationSound は機体回転に同期して rotate 音を再生する。
//
//   - 回転開始: 原音 0s→5s を順再生（イントロ）
//   - 5s 到達後: 5s→4s→5s のサステインを継ぎ目なく無限ループ（ループ）
//   - 回転停止（イントロ中）: その位置から 0s まで全部逆再生して停止（フル・アウトロ）
//   - 回転停止（ループ中）: 短い逆再生をフェードアウトして停止（ショート・アウトロ）
//
// イントロとループは NewInfiniteLoopWithIntroF32 による単一プレイヤーで再生し、
// 区間の継ぎ目はすべて同一サンプル（4s/5s）なのでクリックが出ない。
// 再生位置 pos は ebiten の固定タイムステップ（Update 1回 = 1 tick）で自前に
// 進め、停止種別（フル/ショート）と再開位置の判定だけに用いる。audio.Player の
// IsPlaying()/Position() は EOF 付近で不安定なため判定には使わない。
// Update を1フレーム1回呼んで使う。
type RotationSound struct {
	ctx      *audio.Context
	phase    rotPhase
	pos      time.Duration // 原音先頭からの再生位置
	loopDir  int           // ループ中の pos 進行方向（-1: 5→4, +1: 4→5）
	outroEnd time.Duration // アウトロ完了位置（pos がここまで下がったら idle）
	player   *audio.Player
}

// ループの折り返し点。原音の絶対時刻。
const (
	loopLow  = 2 * time.Second // ループ下端（逆再生の戻り先 / 順再生の開始点）
	loopHigh = 3 * time.Second // ループ上端兼イントロ終端
)

// shortOutroDur はループ中に停止したときの短い逆再生（余韻）の長さ。
const shortOutroDur = 1500 * time.Millisecond

// outroAttack はアウトロ先頭の立ち上がり時間。プレイヤー差し替え時に逆再生
// バッファが非ゼロサンプルから始まることによるオンセット・クリック（プツッ音）を抑える。
const outroAttack = 10 * time.Millisecond

// playerBufferSize は各プレイヤーの内部バッファ上限。
// 既定（〜500ms）のままだと Close 後も残り音が長く流れ続けるため抑える。
// ループはシームレス化でプレイヤー差し替えが無くなったため、アンダーラン
// （再生中の枯渇ノイズ）を避けつつ停止時の残り音も短い妥協値にする。
const playerBufferSize = 50 * time.Millisecond

// masterVolume は回転音全体の音量（0.0〜1.0）。全プレイヤーに適用する。
const masterVolume = 0.10

type rotPhase int

const (
	rotIdle  rotPhase = iota
	rotIntro          // 原音 pos→5s 順再生（イントロ）
	rotLoop           // 5s→4s→5s サステインを無限ループ
	rotOutro          // 逆再生して停止
)

// NewRotationSound は audio.Context を初期化して RotationSound を返す。
func NewRotationSound() *RotationSound {
	return &RotationSound{ctx: Context()}
}

// Stop は再生中のプレイヤーを停止・解放し idle に戻す。
// シーン切り替えなど Update が継続しない遷移で必要。
func (r *RotationSound) Stop() {
	r.enterIdle()
}

// Update は rotating（回転入力中フラグ）を受け取り、状態に応じて再生を進める。
func (r *RotationSound) Update(rotating bool) {
	switch r.phase {
	case rotIdle:
		if rotating {
			r.enterPlaying(0)
		}
		return
	case rotOutro:
		if rotating {
			// アウトロ途中で回転再開 → その位置から再生に復帰。
			r.enterPlaying(r.pos)
			return
		}
	default: // rotIntro, rotLoop
		if !rotating {
			r.enterOutro()
			return
		}
	}
	r.advance(tickDur())
}

// advance は pos を1 tick ぶん進め、区切りに達したらフェーズを遷移する。
// イントロ→ループの遷移ではプレイヤーは差し替えない（音声はシームレスに継続）。
func (r *RotationSound) advance(dt time.Duration) {
	switch r.phase {
	case rotIntro:
		r.pos += dt
		if r.pos >= loopHigh {
			r.pos = loopHigh
			r.phase = rotLoop
			r.loopDir = -1
		}
	case rotLoop:
		r.pos += time.Duration(r.loopDir) * dt
		if r.pos <= loopLow {
			r.pos = loopLow
			r.loopDir = 1
		} else if r.pos >= loopHigh {
			r.pos = loopHigh
			r.loopDir = -1
		}
	case rotOutro:
		r.pos -= dt
		if r.pos <= r.outroEnd {
			r.enterIdle()
		}
	}
}

// enterPlaying は原音 from から再生を開始する（回転開始／アウトロからの復帰）。
// イントロ（from→5s）に続けてサステインを無限ループする単一プレイヤーを張る。
// from が 5s 以上ならイントロ長 0 で即ループに入る。
func (r *RotationSound) enterPlaying(from time.Duration) {
	if from < 0 {
		from = 0
	}
	if from > loopHigh {
		from = loopHigh
	}
	// イントロ部のみ 0→1 のフェードインを掛ける。末尾は full のままなので
	// サステインとの継ぎ目は連続。from>=5s だと intro 長 0 で no-op。
	intro := fadeInF32(rotateSound.forwardClip(from, loopHigh))
	buf := make([]byte, 0, len(intro)+len(rotateSustainPCM))
	buf = append(buf, intro...)
	buf = append(buf, rotateSustainPCM...)
	r.playLoop(buf, int64(len(intro)), int64(len(rotateSustainPCM)))

	r.pos = from
	if from >= loopHigh {
		r.phase = rotLoop
		r.loopDir = -1
	} else {
		r.phase = rotIntro
	}
}

// enterOutro は現在のフェーズに応じた逆再生で停止に向かう。
//   - イントロ中（ループ未到達）: pos→0 を全部逆再生（フル）
//   - ループ中: pos から shortOutroDur ぶんだけ逆再生し、フェードアウト（ショート）
func (r *RotationSound) enterOutro() {
	if r.pos <= 0 {
		r.enterIdle()
		return
	}
	switch r.phase {
	case rotLoop:
		// ループ到達後は短い余韻。ループはフル音量なので g0=1 から始め、
		// 末尾を無音まで落として終える（途中サンプルで切れるクリック防止）。
		start := max(r.pos-shortOutroDur, 0)
		r.outroEnd = start
		r.play(outroEnvelopeF32(reverseClip(start, r.pos), 1, outroAttack))
	default: // rotIntro
		// イントロ中はフェードイン途中の音量（pos/loopHigh）から始めると
		// 直前の再生と段差なく繋がる。末尾は 0=無音で着地。
		r.outroEnd = 0
		g0 := float32(r.pos) / float32(loopHigh)
		r.play(outroEnvelopeF32(reverseClip(0, r.pos), g0, outroAttack))
	}
	r.phase = rotOutro
}

// enterIdle は再生を止めて待機状態に戻す。
func (r *RotationSound) enterIdle() {
	r.stopPlayer()
	r.phase = rotIdle
	r.pos = 0
	r.loopDir = 0
	r.outroEnd = 0
}

// play は pcm を新しいプレイヤーで先頭から1回だけ再生する（アウトロ用）。
func (r *RotationSound) play(pcm []byte) {
	r.stopPlayer()
	p := r.ctx.NewPlayerF32FromBytes(pcm)
	p.SetBufferSize(playerBufferSize)
	p.SetVolume(masterVolume)
	p.Play()
	r.player = p
}

// playLoop は buf を「イントロ＋サステイン無限ループ」として再生する。
// introLen バイトを1回再生したあと、続く loopLen バイトを継ぎ目なく繰り返す。
func (r *RotationSound) playLoop(buf []byte, introLen, loopLen int64) {
	r.stopPlayer()
	loop := audio.NewInfiniteLoopWithIntroF32(bytes.NewReader(buf), introLen, loopLen)
	p, err := r.ctx.NewPlayerF32(loop)
	if err != nil {
		log.Printf("sound: rotation loop player: %v", err)
		return
	}
	p.SetBufferSize(playerBufferSize)
	p.SetVolume(masterVolume)
	p.Play()
	r.player = p
}

func (r *RotationSound) stopPlayer() {
	if r.player == nil {
		return
	}
	r.player.Pause()
	_ = r.player.Close()
	r.player = nil
}

// fadeInF32 は先頭0.0→末尾1.0 の線形フェードを掛ける（イントロ用）。
func fadeInF32(pcm []byte) []byte { return rampF32(pcm, 0, 1) }

// outroEnvelopeF32 はアウトロ用の振幅エンベロープを掛ける。
//   - 全体: g0 → 0.0 の線形フェードアウト（末尾を無音で着地させクリック防止）
//   - 先頭 attack ぶん: 0.0 → 1.0 のアタックを重畳（再生開始時のオンセット・クリック防止）
//
// 入力は変更せず新しいバッファを返す。
func outroEnvelopeF32(pcm []byte, g0 float32, attack time.Duration) []byte {
	out := make([]byte, len(pcm))
	copy(out, pcm)
	frames := len(out) / bytesPerFrame
	if frames == 0 {
		return out
	}
	a := min(int(int64(attack)*sampleRate/int64(time.Second)), frames)
	for f := range frames {
		gain := g0
		if frames > 1 {
			gain = g0 * (1 - float32(f)/float32(frames-1))
		}
		if a > 0 && f < a {
			gain *= float32(f) / float32(a)
		}
		for ch := range 2 {
			off := f*bytesPerFrame + ch*4
			v := math.Float32frombits(binary.LittleEndian.Uint32(out[off:]))
			binary.LittleEndian.PutUint32(out[off:], math.Float32bits(v*gain))
		}
	}
	return out
}

// rampF32 は 32bit float ステレオ PCM の振幅を g0→g1 へ線形に変化させる。
// 入力は変更せず新しいバッファを返す。
func rampF32(pcm []byte, g0, g1 float32) []byte {
	out := make([]byte, len(pcm))
	copy(out, pcm)
	frames := len(out) / bytesPerFrame
	if frames <= 1 {
		return out
	}
	for f := range frames {
		gain := g0 + (g1-g0)*float32(f)/float32(frames-1)
		for ch := range 2 {
			off := f*bytesPerFrame + ch*4
			v := math.Float32frombits(binary.LittleEndian.Uint32(out[off:]))
			binary.LittleEndian.PutUint32(out[off:], math.Float32bits(v*gain))
		}
	}
	return out
}

// tickDur は Update 1回ぶんの経過時間（= 1 tick）。
// ebiten は既定で固定タイムステップ（TPS 回/秒）で Update を呼ぶ。
func tickDur() time.Duration {
	tps := ebiten.TPS()
	if tps <= 0 {
		tps = 60
	}
	return time.Second / time.Duration(tps)
}
