package entity

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	assetimage "github.com/yiozio/space-miner/internal/asset/image"
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
	PartMineLayer
	PartDroneLauncher
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
// 画像の噴射口は CW 90°×R 回転で配置されるため、推進はその反対方向になる。
type ThrustDir int

const (
	ThrustDirForward  ThrustDir = iota // R=0 噴射が機体後方 → 推進は機体前方
	ThrustDirRight                     // R=1 噴射が機体左側 → 推進は機体右方向
	ThrustDirBackward                  // R=2 噴射が機体前方 → 推進は機体後方
	ThrustDirLeft                      // R=3 噴射が機体右側 → 推進は機体左方向
)

// ThrustDir はスラスタの推進方向を返す。
func (p Part) ThrustDir() ThrustDir {
	switch ((p.Rotation % 4) + 4) % 4 {
	case 0:
		return ThrustDirForward
	case 1:
		return ThrustDirRight
	case 2:
		return ThrustDirBackward
	case 3:
		return ThrustDirLeft
	}
	return ThrustDirForward
}

// partSpriteCell は Kind に対応するパーツシートのセル (col, row) を返す。
// シート未収録の Kind（Cockpit）は ok=false で、ベクター描画にフォールバックする。
// スラスタはアイドル時のセルを返す（点火時セル・炎は ship.go の推進描画が別途重ねる）。
func partSpriteCell(kind PartKind) (col, row int, ok bool) {
	switch kind {
	case PartGun:
		return 0, 1, true
	case PartThruster:
		return 1, 1, true // アイドル
	case PartShield:
		return 3, 1, true
	case PartArmor:
		return 0, 2, true
	case PartMineLayer:
		return 1, 2, true
	case PartDroneLauncher:
		return 2, 2, true
	case PartWarp:
		return 3, 2, true
	case PartAutoAim:
		return 0, 3, true // 左下
	case PartCargo:
		return 2, 3, true
	case PartFuel:
		return 3, 3, true
	}
	return 0, 0, false
}

// DrawPart は指定 Kind のアイコンを image 上の (x, y) を該当グリッド左上として描画する。
// cellSize はグリッド一辺の論理ピクセル数で、エディタのような拡大表示にも対応する。
// rotation は 90° 単位の時計回り回転（0..3）。
// スプライトがある Kind はピクセル画像を、無い Kind は従来のベクター描画を使う。
func DrawPart(dst *ebiten.Image, kind PartKind, x, y, cellSize float32, theme *ui.Theme, rotation int) {
	if col, row, ok := partSpriteCell(kind); ok {
		drawCellSprite(dst, assetimage.Cell(col, row), float64(x), float64(y), float64(cellSize), rotation)
		return
	}
	// ベクターフォールバック（AutoAim 等）。
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

// drawCellSprite は 16x16 のセル画像を cellSize に拡大し、セル中心まわりに
// rotation（90° 単位 CW）回転して (x, y) を左上とするセルへ描く。ニアレスト補間でドット感を保つ。
func drawCellSprite(dst, sub *ebiten.Image, x, y, cellSize float64, rotation int) {
	r := ((rotation % 4) + 4) % 4
	scale := cellSize / float64(assetimage.CellSize)
	half := cellSize / 2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(assetimage.CellSize)/2, -float64(assetimage.CellSize)/2)
	op.GeoM.Scale(scale, scale)
	if r != 0 {
		op.GeoM.Rotate(float64(r) * (math.Pi / 2))
	}
	op.GeoM.Translate(x+half, y+half)
	dst.DrawImage(sub, op)
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
	case PartMineLayer:
		// 機雷本体（円）と放射状のスパイクで「設置兵器」を表現する。
		vector.StrokeCircle(dst, cx, cy, g*0.26, 1, line, false)
		for i := range 6 {
			ang := float64(i) / 6 * (2 * 3.141592653589793)
			dx := float32(math.Cos(ang))
			dy := float32(math.Sin(ang))
			vector.StrokeLine(dst, cx+dx*g*0.26, cy+dy*g*0.26, cx+dx*g*0.40, cy+dy*g*0.40, 1, line, false)
		}
	case PartDroneLauncher:
		// 菱形のドローン本体と、左右に張り出した射出レール。
		dr := g * 0.22
		vector.StrokeLine(dst, cx, cy-dr, cx+dr, cy, 1, line, false)
		vector.StrokeLine(dst, cx+dr, cy, cx, cy+dr, 1, line, false)
		vector.StrokeLine(dst, cx, cy+dr, cx-dr, cy, 1, line, false)
		vector.StrokeLine(dst, cx-dr, cy, cx, cy-dr, 1, line, false)
		vector.StrokeLine(dst, x+inset, cy, cx-dr, cy, 1, line, false)
		vector.StrokeLine(dst, cx+dr, cy, x+g-inset, cy, 1, line, false)
	}
}
