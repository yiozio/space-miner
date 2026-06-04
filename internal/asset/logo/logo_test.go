package logo

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TestParse はSVGパスが解釈でき、頂点が viewBox 内に収まることを確認する。
// 三角形分割は CPU 処理のため GPU 不要で検証できる。
func TestParse(t *testing.T) {
	d := extractPathD(svgData)
	if d == "" {
		t.Fatal("path d を抽出できなかった")
	}
	var p vector.Path
	parsePath(d, &p)
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 || len(is) == 0 {
		t.Fatalf("頂点/インデックスが空: verts=%d indices=%d", len(vs), len(is))
	}
	if len(is)%3 != 0 {
		t.Fatalf("インデックス数が3の倍数でない: %d", len(is))
	}
	// translate 補正後の viewBox 範囲（多少のマージンを許容）。
	const margin = 1.0
	minX, minY := float32(1e9), float32(1e9)
	maxX, maxY := float32(-1e9), float32(-1e9)
	for _, v := range vs {
		minX, maxX = min(minX, v.DstX), max(maxX, v.DstX)
		minY, maxY = min(minY, v.DstY), max(maxY, v.DstY)
	}
	if minX < -margin || minY < -margin || maxX > NativeW+margin || maxY > NativeH+margin {
		t.Fatalf("頂点が viewBox を逸脱: x[%g,%g] y[%g,%g]", minX, maxX, minY, maxY)
	}
	// ロゴ幅がほぼ viewBox 幅まで広がっていること（パース漏れ検知）。
	if maxX-minX < NativeW*0.8 {
		t.Fatalf("ロゴ幅が狭すぎる: %g (期待 >= %g)", maxX-minX, NativeW*0.8)
	}
	t.Logf("verts=%d tris=%d bounds=x[%.1f,%.1f] y[%.1f,%.1f]", len(vs), len(is)/3, minX, maxX, minY, maxY)
}
