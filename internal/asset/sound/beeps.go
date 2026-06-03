package sound

import (
	"math"
	"time"
)

// beeps.go は「ポポポ」という低い3連続ビープ音を提供する。
// 残響として、少し遅らせて音量を落とした同じ3連続ビープを2回重ねる
//（本体 → エコー1（小さめ）→ エコー2（さらに小さめ、もう少し遅れて））。

var (
	beepsPCM   []byte
	warningPCM []byte
)

// 警告音（ノコギリ波のビープ3連続）の合成パラメータ。
const (
	warnVol    = 0.3
	warnFreq   = 700.0                  // ノコギリ波の周波数（警告らしい中高音）
	warnDur    = 100 * time.Millisecond // 1 ビープの長さ
	warnGap    = 70 * time.Millisecond  // ビープ間の空き
	warnCount  = 3                      // 3 連続
	warnEnvAtk = 4 * time.Millisecond
	warnEnvRel = 15 * time.Millisecond
)

// 合成パラメータ。
const (
	beepVol       = 0.1
	beepFreq      = 200.0                  // 低めのビープ
	beepDur       = 65 * time.Millisecond  // 1 ビープの長さ
	beepGap       = 50 * time.Millisecond  // バースト内のビープ間隔（空き）
	beepBurstGap  = 240 * time.Millisecond // エコーまでの基準の空き
	beepEnvAtk    = 5 * time.Millisecond   // ビープの立ち上がり
	beepEnvRel    = 20 * time.Millisecond  // ビープの立ち下がり
	beepsPerBurst = 3                      // 1 バーストのビープ数（ポポポ）
)

// buildBeeps はビープ音 PCM を生成する。buildSfx から呼ばれる。
func buildBeeps() {
	beepsPCM = genBeepsPCM()
	warningPCM = genWarningPCM()
}

// PlayBeeps は「ポポポ」（残響付き）をワンショット再生する。
func PlayBeeps() { playOneShot(beepsPCM, beepVol) }

// PlayWarning は敵出現時の警告音（ノコギリ波3連続）をワンショット再生する。
func PlayWarning() { playOneShot(warningPCM, warnVol) }

// genWarningPCM はノコギリ波のビープを warnCount 回連続させた警告音を作る。
func genWarningPCM() []byte {
	beepN := frameCount(warnDur)
	strideN := frameCount(warnDur + warnGap)
	totalN := (warnCount-1)*strideN + beepN
	mono := make([]float32, totalN)
	period := sampleRate / warnFreq
	atk := frameCount(warnEnvAtk)
	rel := frameCount(warnEnvRel)
	for k := range warnCount {
		off := k * strideN
		for i := range beepN {
			saw := float32(2*(math.Mod(float64(i), period)/period) - 1)
			env := float32(1)
			switch {
			case i < atk:
				env = float32(i) / float32(atk)
			case i >= beepN-rel:
				env = float32(beepN-1-i) / float32(rel)
			}
			mono[off+i] = saw * env
		}
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}

// genBeepsPCM は3連続ビープ＋音量を落とした2回のエコーを1つの PCM にまとめる。
func genBeepsPCM() []byte {
	beepN := frameCount(beepDur)
	strideN := frameCount(beepDur + beepGap)        // ビープ開始間隔
	burstSpanN := (beepsPerBurst-1)*strideN + beepN // 1 バーストの長さ
	gap1 := frameCount(beepBurstGap)
	gap2 := frameCount(beepBurstGap * 3 / 2) // もう少し置く

	// 各バーストの開始フレームと音量（本体→エコー1→エコー2）。
	starts := []int{0, burstSpanN + gap1, 2*burstSpanN + gap1 + gap2}
	amps := []float32{1.0, 0.5, 0.25}
	totalN := starts[2] + burstSpanN

	mono := make([]float32, totalN)
	step := 2 * math.Pi * beepFreq / sampleRate
	atk := frameCount(beepEnvAtk)
	rel := frameCount(beepEnvRel)
	for bi, bs := range starts {
		amp := amps[bi]
		for k := range beepsPerBurst {
			off := bs + k*strideN
			for i := range beepN {
				env := float32(1)
				switch {
				case i < atk:
					env = float32(i) / float32(atk)
				case i >= beepN-rel:
					env = float32(beepN-1-i) / float32(rel)
				}
				mono[off+i] += amp * env * float32(math.Sin(step*float64(i)))
			}
		}
	}
	normalizeMono(mono, 0.9)
	return monoToStereoPCM(mono)
}
