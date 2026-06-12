package logo

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TestParse はSVGパスが解釈でき、パスの範囲が viewBox 内に収まることを確認する。
// パスの平坦化は CPU 処理のため GPU 不要で検証できる。
func TestParse(t *testing.T) {
	d := extractPathD(svgData)
	if d == "" {
		t.Fatal("path d を抽出できなかった")
	}
	var p vector.Path
	parsePath(d, &p)
	b := p.Bounds()
	if b.Empty() {
		t.Fatal("パスの範囲が空")
	}
	// translate 補正後の viewBox 範囲（多少のマージンを許容）。
	const margin = 1.0
	if float64(b.Min.X) < -margin || float64(b.Min.Y) < -margin ||
		float64(b.Max.X) > NativeW+margin || float64(b.Max.Y) > NativeH+margin {
		t.Fatalf("パスが viewBox を逸脱: %v", b)
	}
	// ロゴ幅がほぼ viewBox 幅まで広がっていること（パース漏れ検知）。
	if float64(b.Dx()) < NativeW*0.8 {
		t.Fatalf("ロゴ幅が狭すぎる: %d (期待 >= %g)", b.Dx(), NativeW*0.8)
	}
	t.Logf("bounds=%v", b)
}
