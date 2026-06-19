package scene

import (
	"log"
	"math"
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
var Radius float  // 惑星本体の半径（px）
var TexSize vec2  // テクスチャの元サイズ（px）
var LightDir vec3 // 光源方向（正規化前で可）
// 惑星の向きを表す回転行列の列ベクトル（視点法線 → 惑星法線へ写す）。
// トラックボール式に画面軸まわりの微小回転を積んだもの。ジンバルを避けるため Euler 角ではなく行列で渡す。
var RotC0 vec3
var RotC1 vec3
var RotC2 vec3
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
		// 惑星法線 = 回転行列 · 視点法線。テクスチャの UV はこの向きで引く。
		rn := RotC0*normal.x + RotC1*normal.y + RotC2*normal.z
		lon := atan2(rn.x, rn.z)/(2.0*3.14159265) + 0.5
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

// planetBaseLight は基準の光源方向（左上・手前から、惑星基準の太陽）。
var planetBaseLight = [3]float64{-0.5, -0.6, 0.62}

// mat3 は 3x3 回転行列（列優先 [c0x,c0y,c0z, c1x,c1y,c1z, c2x,c2y,c2z]）。
type mat3 [9]float64

func mat3Identity() mat3 { return mat3{1, 0, 0, 0, 1, 0, 0, 0, 1} }

// mulVec は M·v。
func (m mat3) mulVec(v [3]float64) [3]float64 {
	return [3]float64{
		m[0]*v[0] + m[3]*v[1] + m[6]*v[2],
		m[1]*v[0] + m[4]*v[1] + m[7]*v[2],
		m[2]*v[0] + m[5]*v[1] + m[8]*v[2],
	}
}

// mul は M·N（列優先）。
func (m mat3) mul(n mat3) mat3 {
	c0 := m.mulVec([3]float64{n[0], n[1], n[2]})
	c1 := m.mulVec([3]float64{n[3], n[4], n[5]})
	c2 := m.mulVec([3]float64{n[6], n[7], n[8]})
	return mat3{c0[0], c0[1], c0[2], c1[0], c1[1], c1[2], c2[0], c2[1], c2[2]}
}

// transpose は転置（回転行列なら逆行列）。
func (m mat3) transpose() mat3 {
	return mat3{m[0], m[3], m[6], m[1], m[4], m[7], m[2], m[5], m[8]}
}

// rotX/rotY は X/Y 軸まわりの回転行列（シェーダの法線回転規約に合わせる）。
func rotX(a float64) mat3 {
	c, s := math.Cos(a), math.Sin(a)
	return mat3{1, 0, 0, 0, c, s, 0, -s, c}
}
func rotY(a float64) mat3 {
	c, s := math.Cos(a), math.Sin(a)
	return mat3{c, 0, -s, 0, 1, 0, s, 0, c}
}

// orthonormalize は数値誤差で崩れた回転行列を Gram–Schmidt で正規直交化し直す
// （トラックボール式に微小回転を積み続けても歪まないようにする）。
func (m mat3) orthonormalize() mat3 {
	norm := func(v [3]float64) [3]float64 {
		l := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
		if l == 0 {
			return v
		}
		return [3]float64{v[0] / l, v[1] / l, v[2] / l}
	}
	dot := func(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
	c0 := norm([3]float64{m[0], m[1], m[2]})
	c1 := [3]float64{m[3], m[4], m[5]}
	d := dot(c0, c1)
	c1 = norm([3]float64{c1[0] - d*c0[0], c1[1] - d*c0[1], c1[2] - d*c0[2]})
	c2 := [3]float64{ // c0 × c1
		c0[1]*c1[2] - c0[2]*c1[1],
		c0[2]*c1[0] - c0[0]*c1[2],
		c0[0]*c1[1] - c0[1]*c1[0],
	}
	return mat3{c0[0], c0[1], c0[2], c1[0], c1[1], c1[2], c2[0], c2[1], c2[2]}
}

// drawPlanetSphere は (sx, sy) を中心・半径 r の球体として惑星テクスチャ tex を描く。
// rot は惑星の向き（視点法線→惑星法線へ写す回転行列）。テクスチャはこの向きで貼られる。
// orbitLight=true のときは光源も rot に合わせて回し、影がテクスチャと一緒に回る
// （自機が惑星を周回している見せ方）。false なら光源固定（その場自転の見せ方）。
// atmo に大気を指定すると、端ほど濃い青白い大気圏層（＋外周ハロー）を重ねる。
// シェーダ・テクスチャが使えない場合は false を返し、呼び出し側が平面描画にフォールバックする。
func drawPlanetSphere(dst, tex *ebiten.Image, sx, sy, r float64, rot mat3, atmo planetAtmosphere, orbitLight bool) bool {
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

	// 光源: orbitLight=true なら惑星の向き rot に合わせてビュー空間の太陽を回す
	// （太陽は空間固定で自機が周回する想定 → 影がテクスチャと一緒に回る）。
	// rot は視点→惑星なので、ビュー空間の太陽 = rotᵀ · base。false なら固定光。
	lv := planetBaseLight
	if orbitLight {
		lv = rot.transpose().mulVec(planetBaseLight)
	}
	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Images[0] = tex
	op.Uniforms = map[string]any{
		"Center":       []float32{float32(sx), float32(sy)},
		"Radius":       float32(r),
		"TexSize":      []float32{tw, th},
		"LightDir":     []float32{float32(lv[0]), float32(lv[1]), float32(lv[2])},
		"RotC0":        []float32{float32(rot[0]), float32(rot[1]), float32(rot[2])},
		"RotC1":        []float32{float32(rot[3]), float32(rot[4]), float32(rot[5])},
		"RotC2":        []float32{float32(rot[6]), float32(rot[7]), float32(rot[8])},
		"AtmoColor":    []float32{atmo.color[0], atmo.color[1], atmo.color[2]},
		"AtmoStrength": float32(atmo.strength),
		"AtmoOuter":    float32(outer),
	}
	dst.DrawTrianglesShader(vs, is, sh, op)
	return true
}
