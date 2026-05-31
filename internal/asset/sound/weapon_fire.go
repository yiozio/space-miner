package sound

import (
	"encoding/binary"
	"math"
	"math/rand"
	"time"
)

// weapon_fire.go は武器の発射・着弾効果音（ワンショット）を手続き的に合成して提供する。
//
//   - 破裂音（Burst）: 指数減衰するホワイトノイズの破裂。Ball スタイルの通常弾用。
//   - ザップ音（Zap）  : 高→低へ掃引する低音ノコギリ波にビブラートを掛けた電子音。Plasma 用。
//   - レーザー音（Laser）: 低音のノコギリ波＋ホワイトノイズに軽いアタックとサステイン。Laser 用。
//   - 着弾音（Hit）    : 明るいノイズ＋軽い音程を高速減衰させた「パン」という当たり音。
//
// 再生は ebiten 公式 audio 例の playSE 同様、毎回プレイヤーを作って Play する
// fire-and-forget 方式。再生中のプレイヤーは Context が保持し、終端で解放される
// ため参照は持たない（同種音の重ね鳴りも可能）。

// 効果音の合成済み PCM（初期化時に一度だけ構築）。
var (
	fireBurstPCM []byte
	fireZapPCM   []byte
	fireLaserPCM []byte
	hitPCM       []byte
)

// 各効果音の再生音量（0.0〜1.0）。
const (
	fireBurstVol = 0.3
	fireZapVol   = 0.3
	fireLaserVol = 0.25
	hitVol       = 0.35
)

// 破裂音の合成パラメータ。
const (
	fireBurstDur   = 180 * time.Millisecond // 全体長
	fireBurstDecay = 35 * time.Millisecond  // 指数減衰の時定数
	fireBurstCut   = 3500.0                 // ローパスのカットオフ [Hz]（角を取る）
)

// ザップ音の合成パラメータ。
const (
	fireZapDur     = 160 * time.Millisecond // 全体長
	fireZapFreqHi  = 700.0                  // 掃引開始周波数 [Hz]（低音）
	fireZapFreqLo  = 120.0                  // 掃引終了周波数 [Hz]（低音）
	fireZapDecay   = 50 * time.Millisecond  // 指数減衰の時定数
	fireZapVibRate = 30.0                   // ビブラートの周波数 [Hz]
	fireZapVibDept = 0.3                    // ビブラートの深さ（周波数の ±割合）
)

// レーザー音の合成パラメータ。
const (
	fireLaserDur      = 180 * time.Millisecond // 全体長
	fireLaserFreq     = 300.0                  // ノコギリ波の周波数 [Hz]（低音）
	fireLaserSawAmp   = 0.7                    // ノコギリ波の混合量
	fireLaserNoiseAmp = 0.4                    // ホワイトノイズの混合量
	fireLaserAttack   = 12 * time.Millisecond  // 立ち上がり
	fireLaserRelease  = 40 * time.Millisecond  // 立ち下がり（残りはサステイン）
)

// 着弾音の合成パラメータ。
const (
	hitDur      = 90 * time.Millisecond // 全体長（短く弾ける）
	hitDecay    = 12 * time.Millisecond // 指数減衰の時定数（速い＝スナップ感）
	hitCut      = 6000.0                // ローパスのカットオフ [Hz]（明るめに残す）
	hitToneFreq = 900.0                 // 芯になる音程成分 [Hz]
	hitToneAmp  = 0.4                   // 音程成分の混合量
)

// buildWeaponSounds は全効果音 PCM を生成する。initAudio から呼ばれる。
func buildWeaponSounds() {
	fireBurstPCM = genBurstPCM()
	fireZapPCM = genZapPCM()
	fireLaserPCM = genLaserPCM()
	hitPCM = genHitPCM()
}

// PlayFireBurst は破裂音を、PlayFireZap はザップ音を、PlayFireLaser はレーザー音を、
// PlayHit は着弾音を、それぞれワンショット再生する。
func PlayFireBurst() { playOneShot(fireBurstPCM, fireBurstVol) }
func PlayFireZap()   { playOneShot(fireZapPCM, fireZapVol) }
func PlayFireLaser() { playOneShot(fireLaserPCM, fireLaserVol) }
func PlayHit()       { playOneShot(hitPCM, hitVol) }

// playOneShot は pcm を新しいプレイヤーで一度だけ鳴らす（fire-and-forget）。
func playOneShot(pcm []byte, vol float64) {
	Context() // 初期化保証（PCM 生成も含む）
	if len(pcm) == 0 {
		return
	}
	p := ctx.NewPlayerF32FromBytes(pcm)
	p.SetVolume(vol)
	p.Play()
}

// genBurstPCM は指数減衰するホワイトノイズ（軽くローパス）で破裂音を作る。
func genBurstPCM() []byte {
	n := frameCount(fireBurstDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(11))
	alpha := lowpassAlpha(fireBurstCut)
	tau := float64(fireBurstDecay) / float64(time.Second)
	var lp float32
	for i := range n {
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		mono[i] = lp * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genZapPCM は高→低へ掃引する低音のノコギリ波にビブラート（周波数の周期揺らぎ）と
// 指数減衰を掛けてザップ音を作る。
func genZapPCM() []byte {
	n := frameCount(fireZapDur)
	mono := make([]float32, n)
	tau := float64(fireZapDecay) / float64(time.Second)
	var phase float64 // 0〜1 に正規化したノコギリ位相
	for i := range n {
		t := float64(i) / sampleRate
		frac := float64(i) / float64(n-1) // 0→1
		sweep := fireZapFreqHi + (fireZapFreqLo-fireZapFreqHi)*frac
		// ビブラート: LFO で瞬時周波数を ±fireZapVibDept だけ周期変調する。
		freq := sweep * (1 + fireZapVibDept*math.Sin(2*math.Pi*fireZapVibRate*t))
		phase += freq / sampleRate
		phase -= math.Floor(phase)
		saw := float32(2*phase - 1)
		env := float32(math.Exp(-t / tau))
		mono[i] = saw * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genLaserPCM は低音のノコギリ波にホワイトノイズを混ぜ、軽いアタック／リリースの
// エンベロープを掛けてレーザー音を作る（中間はサステイン＝一定音量）。
func genLaserPCM() []byte {
	n := frameCount(fireLaserDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(13))
	atk := frameCount(fireLaserAttack)
	rel := frameCount(fireLaserRelease)
	period := sampleRate / fireLaserFreq
	for i := range n {
		saw := float32(2*(math.Mod(float64(i), period)/period) - 1)
		white := rng.Float32()*2 - 1
		env := float32(1)
		switch {
		case i < atk:
			env = float32(i) / float32(atk)
		case i >= n-rel:
			env = float32(n-1-i) / float32(rel)
		}
		mono[i] = (fireLaserSawAmp*saw + fireLaserNoiseAmp*white) * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genHitPCM は明るいノイズに軽い音程成分を足し、高速減衰させた「パン」という
// 着弾音を作る。減衰が速いほど鋭いスナップ音になる。
func genHitPCM() []byte {
	n := frameCount(hitDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(17))
	alpha := lowpassAlpha(hitCut)
	tau := float64(hitDecay) / float64(time.Second)
	step := 2 * math.Pi * hitToneFreq / sampleRate
	var lp float32
	for i := range n {
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		tone := float32(math.Sin(step * float64(i)))
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		mono[i] = (lp + hitToneAmp*tone) * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// frameCount は時間長 d に対応するフレーム数を返す。
func frameCount(d time.Duration) int {
	return int(int64(d) * sampleRate / int64(time.Second))
}

// lowpassAlpha はカットオフ周波数 fc [Hz] の一極ローパス係数を返す。
func lowpassAlpha(fc float64) float32 {
	wc := 2 * math.Pi * fc / sampleRate
	return float32(wc / (wc + 1))
}

// normalizeMono は mono のピーク振幅が peak になるよう全体をスケールする。
func normalizeMono(mono []float32, peak float32) {
	var mx float32
	for _, v := range mono {
		if a := float32(math.Abs(float64(v))); a > mx {
			mx = a
		}
	}
	if mx == 0 {
		return
	}
	g := peak / mx
	for i := range mono {
		mono[i] *= g
	}
}

// monoToStereoPCM は mono float サンプル列を 32bit float ステレオ PCM へ複製する。
func monoToStereoPCM(mono []float32) []byte {
	pcm := make([]byte, len(mono)*bytesPerFrame)
	for i, v := range mono {
		b := math.Float32bits(v)
		binary.LittleEndian.PutUint32(pcm[i*bytesPerFrame:], b)
		binary.LittleEndian.PutUint32(pcm[i*bytesPerFrame+4:], b)
	}
	return pcm
}
