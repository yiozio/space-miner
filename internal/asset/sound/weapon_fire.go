package sound

import (
	"encoding/binary"
	"math"
	"math/rand"
	"time"
)

// weapon_fire.go はゲームの効果音（ワンショット）を手続き的に合成して提供する。
//
//   - 破裂音（Burst）: 指数減衰するホワイトノイズの破裂。Ball スタイルの通常弾用。
//   - ザップ音（Zap）  : 高→低へ掃引する低音ノコギリ波にビブラートを掛けた電子音。Plasma 用。
//   - レーザー音（Laser）: 低音のノコギリ波＋ホワイトノイズに軽いアタックとサステイン。Laser 用。
//   - 着弾音（Hit）    : 明るいノイズ＋軽い音程を高速減衰させた「パン」という当たり音。
//   - 被ダメージ音（Damage）: 低い正弦波が下降する「ダンッ」という被弾・衝突音（シールド有）。
//   - 被ダメージ破裂音（DamageBurst）: 着弾音より低く少し長い破裂音（シールド無＝直接被弾）。
//   - 小惑星破壊音（AsteroidBreak）: こもったノイズの「ガッ」という岩の砕け音。
//   - 爆発音（Explosion）: 低い轟音＋ノイズが長めに尾を引く海賊撃破時の爆発。
//
// 再生は ebiten 公式 audio 例の playSE 同様、毎回プレイヤーを作って Play する
// fire-and-forget 方式。再生中のプレイヤーは Context が保持し、終端で解放される
// ため参照は持たない（同種音の重ね鳴りも可能）。

// 効果音の合成済み PCM（初期化時に一度だけ構築）。
var (
	fireBurstPCM     []byte
	fireZapPCM       []byte
	fireLaserPCM     []byte
	hitPCM           []byte
	damagePCM        []byte
	damageBurstPCM   []byte
	asteroidBreakPCM []byte
	explosionPCM     []byte
)

// 各効果音の再生音量（0.0〜1.0）。
const (
	fireBurstVol     = 0.3
	fireZapVol       = 0.3
	fireLaserVol     = 0.25
	hitVol           = 0.35
	damageVol        = 0.4
	damageBurstVol   = 0.4
	asteroidBreakVol = 0.35
	explosionVol     = 0.5
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

// 被ダメージ音の合成パラメータ。低い正弦波を下降させた「ダンッ」。
const (
	damageDur      = 200 * time.Millisecond // 全体長
	damageFreqHi   = 150.0                  // 開始周波数 [Hz]（ここから下降）
	damageFreqLo   = 70.0                   // 終了周波数 [Hz]
	damageDecay    = 60 * time.Millisecond  // 指数減衰の時定数
	damageNoiseAmp = 0.25                   // パンチを与える低域ノイズ量
	damageNoiseCut = 400.0                  // 低域ノイズのローパス [Hz]
)

// 被ダメージ破裂音の合成パラメータ（シールド無）。着弾音(Hit)より低く・少し長い破裂。
const (
	damageBurstDur      = 140 * time.Millisecond // 着弾音より少し長い
	damageBurstDecay    = 30 * time.Millisecond  // 破裂感のある減衰
	damageBurstCut      = 1200.0                 // 着弾音(6000)より低くこもった破裂
	damageBurstToneFreq = 110.0                  // 低い芯の音程 [Hz]
	damageBurstToneAmp  = 0.35                   // 音程成分の混合量
)

// 小惑星破壊音の合成パラメータ。こもったノイズの岩の砕け音。
const (
	asteroidBreakDur   = 130 * time.Millisecond // 全体長
	asteroidBreakDecay = 28 * time.Millisecond  // 指数減衰の時定数
	asteroidBreakCut   = 2500.0                 // ローパスのカットオフ [Hz]
)

// 爆発音の合成パラメータ。低い轟音＋ノイズが長めに尾を引く。
const (
	explosionDur     = 500 * time.Millisecond // 全体長（長めに尾を引く）
	explosionDecay   = 150 * time.Millisecond // 指数減衰の時定数
	explosionCut     = 1500.0                 // ノイズのローパス [Hz]
	explosionBoomHi  = 90.0                   // 轟音の開始周波数 [Hz]（下降）
	explosionBoomLo  = 40.0                   // 轟音の終了周波数 [Hz]
	explosionBoomAmp = 0.6                    // 轟音（低音正弦）の混合量
)

// buildSfx は全効果音 PCM を生成する。initAudio から呼ばれる。
func buildSfx() {
	fireBurstPCM = genBurstPCM()
	fireZapPCM = genZapPCM()
	fireLaserPCM = genLaserPCM()
	hitPCM = genHitPCM()
	damagePCM = genDamagePCM()
	damageBurstPCM = genDamageBurstPCM()
	asteroidBreakPCM = genAsteroidBreakPCM()
	explosionPCM = genExplosionPCM()
	buildUISounds()
	buildWarpSounds()
	buildBeeps()
}

// PlayFireBurst は破裂音を、PlayFireZap はザップ音を、PlayFireLaser はレーザー音を、
// PlayHit は着弾音を、それぞれワンショット再生する。
func PlayFireBurst()     { playOneShot(fireBurstPCM, fireBurstVol) }
func PlayFireZap()       { playOneShot(fireZapPCM, fireZapVol) }
func PlayFireLaser()     { playOneShot(fireLaserPCM, fireLaserVol) }
func PlayHit()           { playOneShot(hitPCM, hitVol) }
func PlayDamage()        { playOneShot(damagePCM, damageVol) }
func PlayDamageBurst()   { playOneShot(damageBurstPCM, damageBurstVol) }
func PlayAsteroidBreak() { playOneShot(asteroidBreakPCM, asteroidBreakVol) }
func PlayExplosion()     { playOneShot(explosionPCM, explosionVol) }

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

// genDamagePCM は高→低へ下降する低音正弦波に低域ノイズを足し、指数減衰させた
// 「ダンッ」という被ダメージ音を作る。
func genDamagePCM() []byte {
	n := frameCount(damageDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(19))
	alpha := lowpassAlpha(damageNoiseCut)
	tau := float64(damageDecay) / float64(time.Second)
	var phase float64 // 0〜1 に正規化した正弦位相
	var lp float32
	for i := range n {
		frac := float64(i) / float64(n-1) // 0→1
		freq := damageFreqHi + (damageFreqLo-damageFreqHi)*frac
		phase += freq / sampleRate
		phase -= math.Floor(phase)
		tone := float32(math.Sin(2 * math.Pi * phase))
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		mono[i] = (tone + damageNoiseAmp*lp) * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genDamageBurstPCM はシールド無しで直接被弾したときの破裂音を作る。着弾音(Hit)を
// 低く・少し長くした構成で、こもったノイズに低い音程を足して高速減衰させる。
func genDamageBurstPCM() []byte {
	n := frameCount(damageBurstDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(23))
	alpha := lowpassAlpha(damageBurstCut)
	tau := float64(damageBurstDecay) / float64(time.Second)
	step := 2 * math.Pi * damageBurstToneFreq / sampleRate
	var lp float32
	for i := range n {
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		tone := float32(math.Sin(step * float64(i)))
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		mono[i] = (lp + damageBurstToneAmp*tone) * env
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genAsteroidBreakPCM はこもったノイズを高速減衰させた岩の砕け音を作る。
func genAsteroidBreakPCM() []byte {
	n := frameCount(asteroidBreakDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(29))
	alpha := lowpassAlpha(asteroidBreakCut)
	tau := float64(asteroidBreakDecay) / float64(time.Second)
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

// genExplosionPCM は下降する低音正弦（轟音）に低域ノイズを混ぜ、長めに尾を引く
// 指数減衰を掛けて爆発音を作る。
func genExplosionPCM() []byte {
	n := frameCount(explosionDur)
	mono := make([]float32, n)
	rng := rand.New(rand.NewSource(31))
	alpha := lowpassAlpha(explosionCut)
	tau := float64(explosionDecay) / float64(time.Second)
	var lp float32
	var phase float64 // 0〜1 に正規化した正弦位相
	for i := range n {
		frac := float64(i) / float64(n-1) // 0→1
		white := rng.Float32()*2 - 1
		lp += alpha * (white - lp)
		freq := explosionBoomHi + (explosionBoomLo-explosionBoomHi)*frac
		phase += freq / sampleRate
		phase -= math.Floor(phase)
		boom := float32(math.Sin(2 * math.Pi * phase))
		env := float32(math.Exp(-float64(i) / sampleRate / tau))
		mono[i] = (lp + explosionBoomAmp*boom) * env
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
