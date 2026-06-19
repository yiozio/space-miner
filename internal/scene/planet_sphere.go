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

var Center vec2    // 円盤中心（スクリーンpx）
var Radius float   // 惑星本体の半径（px）
var TexSize vec2   // テクスチャの元サイズ（px）
var LightDir vec3  // 光源方向（正規化前で可）
var Rotation float // 自転の経度オフセット（0..1 で1周）
var Tilt float     // 緯度方向の回転（ラジアン。周回の上下移動で球を縦に転がす）
// 大気圏層（AtmoStrength=0 で無効）。
var AtmoColor vec3    // 大気の色（青白）
var AtmoStrength float // 大気の濃さ
var AtmoOuter float    // 大気外縁の半径倍率（>1 でハロー、1 で無し）

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	local := (dstPos.xy - Center) / Radius
	d2 := dot(local, local)
	ld := normalize(LightDir)
	outer2 := AtmoOuter * AtmoOuter
	if d2 > outer2 {
		return vec4(0.0)
	}
	if d2 <= 1.0 {
		nz := sqrt(1.0 - d2)
		normal := vec3(local.x, local.y, nz)
		// UV 用には法線を緯度方向（X 軸まわり Tilt）に回した向きを使う。
		// 陰影は視点法線 normal のまま（光源は固定）なので、回しても明暗は動かない。
		ct := cos(Tilt)
		st := sin(Tilt)
		ry := normal.y*ct - normal.z*st
		rz := normal.y*st + normal.z*ct
		rn := vec3(normal.x, ry, rz)
		// 経度（自転で回す）と緯度を UV へ。
		lon := atan2(rn.x, rn.z)/(2.0*3.14159265) + 0.5 + Rotation
		u := lon - floor(lon)
		lat := 0.5 - asin(clamp(rn.y, -1.0, 1.0))/3.14159265
		uv := clamp(vec2(u, lat)*TexSize, vec2(0.5), TexSize-vec2(0.5))
		c := imageSrc0At(imageSrc0Origin() + uv)
		// ランバート + 環境光。球の陰影で立体感を出す。
		diff := clamp(dot(normal, ld), 0.0, 1.0)
		shade := 0.22 + 0.78*diff
		rgb := c.rgb * shade
		// 大気（表面側）: 全体うっすら + 縁ほど濃い。昼側ほど明るい。
		rim := d2 * d2
		atmo := AtmoStrength * (0.12 + 0.88*rim) * (0.25 + 0.75*diff)
		rgb += AtmoColor * atmo
		// 縁のアンチエイリアス（アルファをフェード）。
		alpha := 1.0 - smoothstep(0.985, 1.0, d2)
		return vec4(rgb*alpha, alpha)
	}
	// 大気ハロー（惑星の外側）。リムで濃く外縁で消える。昼側を明るく。
	dist := (sqrt(d2) - 1.0) / max(AtmoOuter-1.0, 0.0001)
	halo := AtmoStrength * (1.0 - dist)
	halo *= halo
	litN := normalize(vec3(local.x, local.y, 0.25))
	lit := clamp(dot(litN, ld), 0.0, 1.0)
	halo *= 0.25 + 0.75*lit
	a := clamp(halo, 0.0, 1.0)
	return vec4(AtmoColor*a, a)
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

// planetAtmosphere は惑星に重ねる大気圏層。strength=0 で無効。
// color は大気色（青白）、outer は大気外縁の半径倍率（>1 でハロー、1 で無し）。
type planetAtmosphere struct {
	strength float64
	color    [3]float32
	outer    float64
}

// drawPlanetSphere は (sx, sy) を中心・半径 r の球体として惑星テクスチャ tex を描く。
// spin は自転の経度オフセット（0..1 で1周）。tex はアニメフレーム（毎フレーム差し替え可）。
// atmo に大気を指定すると、端ほど濃い青白い大気圏層（＋外周ハロー）を重ねる。
// シェーダ・テクスチャが使えない場合は false を返し、呼び出し側が平面描画にフォールバックする。
func drawPlanetSphere(dst, tex *ebiten.Image, sx, sy, r, spin, tilt float64, atmo planetAtmosphere) bool {
	sh := planetShader()
	if sh == nil || tex == nil {
		return false
	}
	tb := tex.Bounds()
	tw, th := float32(tb.Dx()), float32(tb.Dy())

	// 大気ハローを収めるため、外縁倍率ぶん大きい四角形に描く（本体半径 r は uniform で渡す）。
	outer := atmo.outer
	if outer < 1 || atmo.strength <= 0 {
		outer = 1
	}
	qr := r * outer
	x0, y0 := float32(sx-qr), float32(sy-qr)
	x1, y1 := float32(sx+qr), float32(sy+qr)
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
		"Center":       []float32{float32(sx), float32(sy)},
		"Radius":       float32(r),
		"TexSize":      []float32{tw, th},
		"LightDir":     []float32{-0.5, -0.6, 0.62}, // 左上・手前から
		"Rotation":     float32(spin),
		"Tilt":         float32(tilt),
		"AtmoColor":    []float32{atmo.color[0], atmo.color[1], atmo.color[2]},
		"AtmoStrength": float32(atmo.strength),
		"AtmoOuter":    float32(outer),
	}
	dst.DrawTrianglesShader(vs, is, sh, op)
	return true
}
