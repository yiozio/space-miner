package entity

import (
	"math"
	"testing"
)

// TestHullRect は Pebble ベースの当たり判定矩形が、自機画像（52x60, グリッド原点(2,8)）から
// 幅・高さ 4px 縮めた長方形になっていることを検証する。
func TestHullRect(t *testing.T) {
	s := &Ship{BaseID: ShipBasePebble}
	cx, cy, hw, hh := s.HullRect()
	// 期待値: 画像幅52-4=48, 高さ60-4=56（論理pxは元画像px×2）。
	if hw != 48 || hh != 56 {
		t.Fatalf("half extents = (%v, %v); want (48, 56)", hw, hh)
	}
	if cx != 0 {
		t.Fatalf("center x = %v; want 0", cx)
	}
	// 中心 y はグリッド中心(-partPivotShiftY) からスプライト中心ぶん上にオフセット。
	if math.Abs(cy-(-4)) > 1e-9 {
		t.Fatalf("center y = %v; want -4", cy)
	}
}

// TestPivotShift は回転中心がスプライト中心 + 2px 下になることを確認する。
// 画像高さ60ではスプライト中心(y=30)とグリッド中心(y=32)の差(2px)とドロップ量(2px)が
// 相殺し、ピボットはグリッド中心（partPivotShiftY=0）になる。整数除算の取りこぼしも検出する。
func TestPivotShift(t *testing.T) {
	if math.Abs(partPivotShiftY-0.0) > 1e-9 {
		t.Fatalf("partPivotShiftY = %v; want 0.0", partPivotShiftY)
	}
}

// TestHullContains は矩形の内外判定が向きを反映することを確認する（Angle=0 は +x 向き）。
func TestHullContains(t *testing.T) {
	s := &Ship{BaseID: ShipBasePebble, X: 0, Y: 0, Angle: 0}
	if !s.HullContains(0, 0) {
		t.Fatal("pivot point should be inside hull")
	}
	// Angle=0 は機首が +x。前方（+x）100px は半高(56)を超えるので外。
	if s.HullContains(100, 0) {
		t.Fatal("point 100px ahead should be outside hull")
	}
	// 側方（+y）40px は半幅(48)内なので内。
	if !s.HullContains(0, 40) {
		t.Fatal("point 40px to the side should be inside hull")
	}
}
