package scene

import (
	"log"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// planet_sphere.go は惑星を「立体的な球体」に見せる描画を提供する。
// 実際の 3D ではなく、正距円筒テクスチャを Kage シェーダで円盤へ球面マッピングし、
// 経度オフセット（Rotation）を時間で進めることで自転しているように見せる。

// planetShaderSrc は球面マッピング + 簡易ライティングのフラグメントシェーダ。
// 円盤ローカル座標 (dstPos - Center)/Radius から球面法線を起こし、経度・緯度を
// テクスチャ UV に変換してサンプルする。光源方向 LightDir との内積で陰影を付ける。
const planetShaderSrc = `
//kage:unit pixels
package main

var Center vec2   // 円盤中心（スクリーンpx）
var Radius float  // 円盤半径（px）
var TexSize vec2  // テクスチャの元サイズ（px）
var LightDir vec3 // 光源方向（正規化前で可）
var Rotation float // 自転の経度オフセット（0..1 で1周）

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	local := (dstPos.xy - Center) / Radius
	d2 := dot(local, local)
	if d2 > 1.0 {
		return vec4(0.0)
	}
	nz := sqrt(1.0 - d2)
	normal := vec3(local.x, local.y, nz)
	// 経度（自転で回す）と緯度を UV へ。
	lon := atan2(local.x, nz)/(2.0*3.14159265) + 0.5 + Rotation
	u := lon - floor(lon)
	lat := 0.5 - asin(clamp(local.y, -1.0, 1.0))/3.14159265
	uv := clamp(vec2(u, lat)*TexSize, vec2(0.5), TexSize-vec2(0.5))
	c := imageSrc0At(imageSrc0Origin() + uv)
	// ランバート + 環境光。球の陰影で立体感を出す。
	diff := clamp(dot(normal, normalize(LightDir)), 0.0, 1.0)
	shade := 0.22 + 0.78*diff
	// 縁のアンチエイリアス（アルファをフェード）。
	alpha := 1.0 - smoothstep(0.97, 1.0, d2)
	return vec4(c.rgb*shade*alpha, alpha)
}
`

var (
	planetShaderOnce sync.Once
	planetShaderInst *ebiten.Shader
)

// planetShader はシェーダを初回のみコンパイルして返す。失敗時は nil（呼び出し側で平面描画にフォールバック）。
func planetShader() *ebiten.Shader {
	planetShaderOnce.Do(func() {
		sh, err := ebiten.NewShader([]byte(planetShaderSrc))
		if err != nil {
			log.Printf("planet shader compile failed: %v", err)
			return
		}
		planetShaderInst = sh
	})
	return planetShaderInst
}

// drawPlanetSphere は (sx, sy) を中心・半径 r の球体として惑星テクスチャ tex を描く。
// spin は自転の経度オフセット（0..1 で1周）。tex はアニメフレーム（毎フレーム差し替え可）。
// シェーダ・テクスチャが使えない場合は false を返し、呼び出し側が平面描画にフォールバックする。
func drawPlanetSphere(dst, tex *ebiten.Image, sx, sy, r, spin float64) bool {
	sh := planetShader()
	if sh == nil || tex == nil {
		return false
	}
	tb := tex.Bounds()
	tw, th := float32(tb.Dx()), float32(tb.Dy())

	x0, y0 := float32(sx-r), float32(sy-r)
	x1, y1 := float32(sx+r), float32(sy+r)
	col := struct{ r, g, b, a float32 }{1, 1, 1, 1}
	vs := []ebiten.Vertex{
		{DstX: x0, DstY: y0, SrcX: 0, SrcY: 0, ColorR: col.r, ColorG: col.g, ColorB: col.b, ColorA: col.a},
		{DstX: x1, DstY: y0, SrcX: tw, SrcY: 0, ColorR: col.r, ColorG: col.g, ColorB: col.b, ColorA: col.a},
		{DstX: x0, DstY: y1, SrcX: 0, SrcY: th, ColorR: col.r, ColorG: col.g, ColorB: col.b, ColorA: col.a},
		{DstX: x1, DstY: y1, SrcX: tw, SrcY: th, ColorR: col.r, ColorG: col.g, ColorB: col.b, ColorA: col.a},
	}
	is := []uint16{0, 1, 2, 1, 3, 2}

	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Images[0] = tex
	op.Uniforms = map[string]any{
		"Center":   []float32{float32(sx), float32(sy)},
		"Radius":   float32(r),
		"TexSize":  []float32{tw, th},
		"LightDir": []float32{-0.5, -0.6, 0.62}, // 左上・手前から
		"Rotation": float32(spin),
	}
	dst.DrawTrianglesShader(vs, is, sh, op)
	return true
}
