package entity

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/ui"
)

// PartKind はパーツのカテゴリ（振る舞い分類）。
// 同じ Kind でも複数のバリアント（PartID）が存在しうる。
type PartKind int

const (
	PartCockpit PartKind = iota
	PartGun
	PartThruster
	PartFuel
	PartCargo
	PartArmor
	PartShield
	PartAutoAim
	PartWarp
)

// Part はグリッド配置されたパーツ。
// (GX, GY) はコックピットを原点 (0, 0) としたローカル座標。
// +y はビジュアル上の後方（ローカル -y が機体の進行方向）。
// DefID は性能・名前・価格などのバリアント情報の参照。
// Rotation はパーツの向き（0..3、90° 単位の時計回り回転）。
//   - 0: デフォルト（後方に推進など）
//   - 1: 90° 時計回り
//   - 2: 180°（後ろ向きスラスタ等）
//   - 3: 270°
type Part struct {
	DefID    PartID
	GX, GY   int
	Rotation int
}

// Def は DefID に対応する PartDef を返す。
func (p Part) Def() *PartDef { return PartDefByID(p.DefID) }

// Kind は Def 経由でカテゴリを返す。
func (p Part) Kind() PartKind { return p.Def().Kind }

// ThrustDir はパーツ（主にスラスタ）の推進方向カテゴリを返す。
// Rotation のみで決まり、Kind に依存しない（呼び出し側で Kind を確認すること）。
type ThrustDir int

const (
	ThrustDirForward  ThrustDir = iota // R=0
	ThrustDirSideways                  // R=1, R=3
	ThrustDirBackward                  // R=2
)

// ThrustDir はスラスタの推進方向を返す。
func (p Part) ThrustDir() ThrustDir {
	switch ((p.Rotation % 4) + 4) % 4 {
	case 0:
		return ThrustDirForward
	case 2:
		return ThrustDirBackward
	}
	return ThrustDirSideways
}

// DrawPart は指定 Kind のアイコンを image 上の (x, y) を該当グリッド左上として描画する。
// cellSize はグリッド一辺の論理ピクセル数で、エディタのような拡大表示にも対応する。
// rotation は 90° 単位の時計回り回転（0..3）。0 ならそのまま、それ以外は一時画像経由で回転 blit。
// バリアント間で見た目は共通（Kind で分岐）。
func DrawPart(dst *ebiten.Image, kind PartKind, x, y, cellSize float32, theme *ui.Theme, rotation int) {
	r := ((rotation % 4) + 4) % 4
	if r == 0 {
		drawPartRaw(dst, kind, x, y, cellSize, theme)
		return
	}
	tmp := ebiten.NewImage(int(cellSize), int(cellSize))
	drawPartRaw(tmp, kind, 0, 0, cellSize, theme)
	op := &ebiten.DrawImageOptions{}
	half := float64(cellSize) / 2
	op.GeoM.Translate(-half, -half)
	op.GeoM.Rotate(float64(r) * (3.141592653589793 / 2))
	op.GeoM.Translate(float64(x)+half, float64(y)+half)
	dst.DrawImage(tmp, op)
}

// drawPartRaw は回転なしの素体描画。
func drawPartRaw(dst *ebiten.Image, kind PartKind, x, y, cellSize float32, theme *ui.Theme) {
	g := cellSize
	inset := g * 0.12
	cx := x + g/2
	cy := y + g/2
	line := theme.Line

	switch kind {
	case PartCockpit:
		vector.StrokeLine(dst, cx, y+inset, x+inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+g-inset, x+g-inset, y+g-inset, 1, line, false)
	case PartGun:
		barrel := g * 0.18
		vector.StrokeRect(dst, x+inset, cy-barrel/2, g-inset*2, barrel, 1, line, false)
		vector.StrokeLine(dst, cx, cy-barrel/2, cx, y, 2, line, false)
	case PartThruster:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g*0.55, 1, line, false)
		vector.StrokeLine(dst, x+g*0.32, y+g*0.7, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*0.68, y+g*0.7, cx, y+g-inset, 1, line, false)
	case PartFuel:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+g/3, y+inset, x+g/3, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g*2/3, y+inset, x+g*2/3, y+g-inset, 1, line, false)
	case PartCargo:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, x+inset, y+inset, x+g-inset, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+g-inset, y+inset, x+inset, y+g-inset, 1, line, false)
	case PartArmor:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 2, line, false)
	case PartShield:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		crossInset := g * 0.20
		vector.StrokeLine(dst, cx, y+crossInset, cx, y+g-crossInset, 1, line, false)
		vector.StrokeLine(dst, x+crossInset, cy, x+g-crossInset, cy, 1, line, false)
	case PartAutoAim:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeLine(dst, cx, y+inset, cx, y+g-inset, 1, line, false)
		vector.StrokeLine(dst, x+inset, cy, x+g-inset, cy, 1, line, false)
	case PartWarp:
		vector.StrokeRect(dst, x+inset, y+inset, g-inset*2, g-inset*2, 1, line, false)
		vector.StrokeRect(dst, x+g*0.3, y+g*0.3, g*0.4, g*0.4, 1, line, false)
	}
}
