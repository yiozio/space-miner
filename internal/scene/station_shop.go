package scene

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/asset/sound"
	"github.com/yiozio/space-miner/internal/entity"
	"github.com/yiozio/space-miner/internal/i18n"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	shopGridCols   = 4
	shopGridRows   = 5
	shopSlotCount  = shopGridCols * shopGridRows
	shopCellSize   = 64
	shopCellPad    = 8
	shopSideWidth  = shopGridCols*shopCellSize + (shopGridCols-1)*shopCellPad
	shopSideHeight = shopGridRows*shopCellSize + (shopGridRows-1)*shopCellPad
	shopInfoW      = 280.0 // 中央の商品情報カラムの幅
	shopColGap     = 24.0  // カラム間の隙間
)

// shopItem はスロット内アイテムの内容。
// パーツの場合は PartID（バリアント識別）を保持し、価格・名前・説明は PartDef から引く。
type shopItem struct {
	Name        string
	Description string
	Price       int
	IsResource  bool
	PartID      entity.PartID
	PartKind    entity.PartKind // 描画アイコン用（カテゴリ）
	ResType     entity.ResourceType
}

// shopSlot は1セル。Quantity が 0 なら空。
type shopSlot struct {
	Item     shopItem
	Quantity int
}

// StationShop はパーツ店舗シーン。
// 左4列に店在庫、右4列に自分の在庫（resources + spare parts）、
// 中央にセッション集計、右端にカーソル中アイテムの情報を表示する。
type StationShop struct {
	player      *entity.Player
	shopSlots   [shopSlotCount]shopSlot
	playerSlots [shopSlotCount]shopSlot
	side        int // 0 = shop, 1 = player
	index       int // 0..shopSlotCount-1

	// セッション集計（金額ベース）
	sessionBuyTotal  int // 購入額。購入済みを売り戻すと差し引かれる
	sessionSellTotal int // 購入していない所持品を売った額
	sessionNet       int // 収支（クレジット増減の累計 = sell - buy）

	// このセッションで購入し、まだ売り戻していない数量。
	// 売却時に「購入分の売り戻し」か「元々の所持品の売却」かを判定する。
	boughtRes   map[entity.ResourceType]int
	boughtParts map[entity.PartID]int
}

// NewStationShop は店舗シーンを生成する。
// 店在庫の構成（並び順・初期入荷数）は data_shop.go を参照。
// 自分側は毎フレーム refreshPlayerSlots でプレイヤー状態から再構築する。
func NewStationShop(p *entity.Player) *StationShop {
	s := &StationShop{
		player:      p,
		boughtRes:   map[entity.ResourceType]int{},
		boughtParts: map[entity.PartID]int{},
	}
	for i, id := range shopStockIDs {
		if i >= shopSlotCount {
			break
		}
		def := entity.PartDefByID(id)
		if def == nil {
			continue
		}
		s.shopSlots[i] = shopSlot{Item: itemFromDef(def), Quantity: shopInitialQuantity(def)}
	}
	s.refreshPlayerSlots()
	return s
}

// itemFromDef は PartDef から店舗用 shopItem を組み立てる。
// 表示文字列は i18n から取得する（数値 (Price 等) は data_*.go を参照）。
func itemFromDef(d *entity.PartDef) shopItem {
	return shopItem{
		Name:        i18n.PartName(d.ID),
		Description: i18n.PartDesc(d.ID),
		Price:       d.Price,
		PartID:      d.ID,
		PartKind:    d.Kind,
	}
}

// refreshPlayerSlots はプレイヤーの現在状態から自分グリッドを再構築する。
// 資源を上から、スペアパーツをその後ろに並べる。
func (ss *StationShop) refreshPlayerSlots() {
	for i := range ss.playerSlots {
		ss.playerSlots[i] = shopSlot{}
	}
	idx := 0
	sh := i18n.S().Shop
	for _, rt := range entity.AllResourceTypes() {
		qty := ss.player.Inventory[rt]
		if qty <= 0 || idx >= shopSlotCount {
			continue
		}
		name := i18n.ResourceName(rt)
		ss.playerSlots[idx] = shopSlot{
			Item: shopItem{
				Name:        name,
				Description: fmt.Sprintf(sh.OreDescFmt, name),
				Price:       rt.Price(),
				IsResource:  true,
				ResType:     rt,
			},
			Quantity: qty,
		}
		idx++
	}
	for _, def := range entity.AllPlaceablePartDefs() {
		qty := ss.player.PartsInventory[def.ID]
		if qty <= 0 || idx >= shopSlotCount {
			continue
		}
		item := itemFromDef(def)
		item.Description = fmt.Sprintf(sh.SpareFmt, i18n.PartName(def.ID))
		ss.playerSlots[idx] = shopSlot{Item: item, Quantity: qty}
		idx++
	}
}

func (ss *StationShop) currentSlot() *shopSlot {
	if ss.side == 0 {
		return &ss.shopSlots[ss.index]
	}
	return &ss.playerSlots[ss.index]
}

func (ss *StationShop) Update(d Director) error {
	ss.refreshPlayerSlots()

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		sound.PlayMenuCancel()
		d.Pop()
		return nil
	}

	row := ss.index / shopGridCols
	col := ss.index % shopGridCols
	oldIndex, oldSide := ss.index, ss.side

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if row > 0 {
			row--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if row < shopGridRows-1 {
			row++
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if col > 0 {
			col--
		} else if ss.side == 1 {
			ss.side = 0
			col = shopGridCols - 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if col < shopGridCols-1 {
			col++
		} else if ss.side == 0 {
			ss.side = 1
			col = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		ss.side = 1 - ss.side
	}
	ss.index = row*shopGridCols + col
	if ss.index != oldIndex || ss.side != oldSide {
		sound.PlayMenuMove()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		sound.PlayMenuSelect()
		ss.transferOne()
	}
	return nil
}

// itemUnitWeight は shopItem 1 単位あたりの所持重量を返す。
func itemUnitWeight(item shopItem) float64 {
	if item.IsResource {
		return item.ResType.Info().Weight
	}
	if d := entity.PartDefByID(item.PartID); d != nil {
		return d.Weight
	}
	return 0
}

// transferOne はカーソル位置のアイテムを1つだけ反対側に移す（=購入 or 売却）。
// プレイヤーの所持状態（Inventory / PartsInventory / Credits）に直接反映される。
// 購入時はクレジットと積載重量の両方を満たす場合のみ成立。
func (ss *StationShop) transferOne() {
	slot := ss.currentSlot()
	if slot.Quantity <= 0 {
		return
	}
	price := slot.Item.Price
	if ss.side == 0 {
		// 購入
		if ss.player.Credits < price {
			return
		}
		if !ss.player.CanAddWeight(itemUnitWeight(slot.Item)) {
			return
		}
		ss.player.Credits -= price
		ss.sessionBuyTotal += price
		ss.sessionNet -= price
		slot.Quantity--
		if slot.Item.IsResource {
			ss.player.Inventory[slot.Item.ResType]++
			ss.boughtRes[slot.Item.ResType]++
		} else {
			ss.player.PartsInventory[slot.Item.PartID]++
			ss.boughtParts[slot.Item.PartID]++
		}
		return
	}

	// 売却
	ss.player.Credits += price
	ss.sessionNet += price
	slot.Quantity--
	// このセッションで購入した分を売り戻したなら購入額から差し引き、
	// それ以外（元々の所持品）は売却額に計上する。
	bought := false
	if slot.Item.IsResource {
		if ss.boughtRes[slot.Item.ResType] > 0 {
			ss.boughtRes[slot.Item.ResType]--
			bought = true
		}
		ss.player.Inventory[slot.Item.ResType]--
	} else {
		if ss.boughtParts[slot.Item.PartID] > 0 {
			ss.boughtParts[slot.Item.PartID]--
			bought = true
		}
		ss.player.PartsInventory[slot.Item.PartID]--
	}
	if bought {
		ss.sessionBuyTotal -= price
	} else {
		ss.sessionSellTotal += price
	}
	// 売った品はショップ在庫に戻す（在庫が無ければ一時的に追加する）
	ss.addToShopStock(slot.Item)
}

// addToShopStock は売却された品をショップ在庫へ加える。
// 同一の品が既にあれば数量を増やし、無ければ空きスロットに一時追加する。
// 空きが無い場合は何もしない（在庫枠は shopSlotCount 個まで）。
func (ss *StationShop) addToShopStock(item shopItem) {
	for i := range ss.shopSlots {
		s := &ss.shopSlots[i]
		if s.Quantity > 0 && shopItemSame(s.Item, item) {
			s.Quantity++
			return
		}
	}
	for i := range ss.shopSlots {
		s := &ss.shopSlots[i]
		if s.Quantity == 0 {
			s.Item = shopDisplayItem(item)
			s.Quantity = 1
			return
		}
	}
}

// shopItemSame は同一の品（資源種別 or パーツ ID が一致）かを判定する。
func shopItemSame(a, b shopItem) bool {
	if a.IsResource != b.IsResource {
		return false
	}
	if a.IsResource {
		return a.ResType == b.ResType
	}
	return a.PartID == b.PartID
}

// shopDisplayItem は売却品をショップ表示向けの内容に組み直す。
// パーツは PartDef からショップ用説明で作り直し、資源はそのまま使う。
func shopDisplayItem(item shopItem) shopItem {
	if !item.IsResource {
		if d := entity.PartDefByID(item.PartID); d != nil {
			return itemFromDef(d)
		}
	}
	return item
}

func (ss *StationShop) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	header := i18n.S().Station.Shop
	hw, hh := ui.MeasureText(header, headerScale)
	ui.DrawText(dst, header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	// 上段は「店在庫 | 商品情報 | 所持品」の3カラム構成（中央寄せ）
	totalW := float64(shopSideWidth)*2 + shopInfoW + shopColGap*2
	startX := (float64(sw) - totalW) / 2

	shopX := startX
	infoX := shopX + float64(shopSideWidth) + shopColGap
	playerX := infoX + shopInfoW + shopColGap

	// ヘッダー直下に店主画像（プレースホルダ枠）を横幅いっぱいに大きく表示する
	portraitY := 24 + hh + 8
	drawStationPortrait(dst, theme, "SHOPKEEPER", sw, portraitY)

	// 画像の下にグリッドと商品情報を配置する
	sideY := portraitY + stationPortraitH + 36

	sh2 := i18n.S().Shop
	ui.DrawText(dst, sh2.Header, shopX, sideY-22, 1.6, theme.LineDim)
	ui.DrawText(dst, sh2.Inventory, playerX, sideY-22, 1.6, theme.LineDim)

	ss.drawGrid(dst, theme, shopX, sideY, ss.shopSlots[:], ss.side == 0)
	ss.drawGrid(dst, theme, playerX, sideY, ss.playerSlots[:], ss.side == 1)
	ss.drawInfo(dst, theme, infoX, sideY, shopInfoW)

	// 収支は中央カラム（グリッドの間）にグリッド下端合わせで配置する
	summaryH := 100.0
	summaryY := sideY + float64(shopSideHeight) - summaryH
	ss.drawSummary(dst, theme, infoX+shopInfoW/2, summaryY)

	ui.DrawText(dst, sh2.Hint, 20, float64(sh)-30, 1.4, theme.LineDim)
}

func (ss *StationShop) drawGrid(dst *ebiten.Image, theme *ui.Theme, x, y float64, slots []shopSlot, focused bool) {
	cs := float64(shopCellSize)
	for i := range slots {
		col := i % shopGridCols
		row := i / shopGridCols
		cx := x + float64(col)*(cs+shopCellPad)
		cy := y + float64(row)*(cs+shopCellPad)
		vector.StrokeRect(dst, float32(cx), float32(cy), float32(cs), float32(cs), 1, theme.LineDim, false)
		if slots[i].Quantity > 0 {
			ss.drawSlotIcon(dst, theme, cx, cy, cs, slots[i])
			qty := fmt.Sprintf("x%d", slots[i].Quantity)
			qw, _ := ui.MeasureText(qty, 1.0)
			ui.DrawText(dst, qty, cx+cs-qw-4, cy+cs-12, 1.0, theme.LineDim)
		}
		if focused && i == ss.index {
			vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
		}
	}
}

// drawSlotIcon は資源なら名前先頭、パーツならミニアイコンをセル中央に描く。
func (ss *StationShop) drawSlotIcon(dst *ebiten.Image, theme *ui.Theme, cx, cy, cs float64, s shopSlot) {
	if s.Item.IsResource {
		label := s.Item.Name
		if len(label) > 4 {
			label = label[:4]
		}
		lw, lh := ui.MeasureText(label, 2.0)
		ui.DrawText(dst, label, cx+(cs-lw)/2, cy+(cs-lh)/2-4, 2.0, theme.Line)
		return
	}
	// パーツはカテゴリで決まるミニアイコン
	iconCell := float32(cs * 0.8)
	ix := float32(cx) + (float32(cs)-iconCell)/2
	iy := float32(cy) + (float32(cs)-iconCell)/2 - 4
	entity.DrawPart(dst, s.Item.PartKind, ix, iy, iconCell, theme, 0)
}

// drawSummary は cx を中心に、収支情報を中央寄せで描画する。
func (ss *StationShop) drawSummary(dst *ebiten.Image, theme *ui.Theme, cx, y float64) {
	sh := i18n.S().Shop
	drawCentered := func(s string, cy, scale float64, c color.Color) {
		w, _ := ui.MeasureText(s, scale)
		ui.DrawText(dst, s, cx-w/2, cy, scale, c)
	}
	// 購入額・売却額を横並びにする
	buy := fmt.Sprintf(sh.BuyAmountFmt, ss.sessionBuyTotal)
	sell := fmt.Sprintf(sh.SellAmountFmt, ss.sessionSellTotal)
	bw, _ := ui.MeasureText(buy, 1.4)
	sw, _ := ui.MeasureText(sell, 1.4)
	colGap := 32.0
	bx := cx - (bw+colGap+sw)/2
	ui.DrawText(dst, buy, bx, y, 1.4, theme.LineDim)
	ui.DrawText(dst, sell, bx+bw+colGap, y, 1.4, theme.LineDim)
	lineY := y + 38
	sign := "+"
	if ss.sessionNet < 0 {
		sign = ""
	}
	drawCentered(fmt.Sprintf(sh.NetFmt, sign, ss.sessionNet), lineY, 1.7, theme.Line)
	lineY += 44
	drawCentered(fmt.Sprintf(sh.CreditsFmt, ss.player.Credits), lineY, 1.4, theme.Line)
}

func (ss *StationShop) drawInfo(dst *ebiten.Image, theme *ui.Theme, x, y, width float64) {
	sh := i18n.S().Shop
	slot := ss.currentSlot()
	if slot.Quantity == 0 {
		ui.DrawText(dst, i18n.S().Common.Empty, x, y, 1.3, theme.LineDim)
		return
	}
	lineY := y
	ui.DrawText(dst, slot.Item.Name, x, lineY, 1.8, theme.Line)
	lineY += 32
	var action string
	if ss.side == 0 {
		action = fmt.Sprintf(sh.BuyPriceFmt, slot.Item.Price)
	} else {
		action = fmt.Sprintf(sh.SellPriceFmt, slot.Item.Price)
	}
	ui.DrawText(dst, action, x, lineY, 1.4, theme.Line)
	lineY += 32
	// 説明文は領域幅を超えたら折り返す
	for _, line := range wrapByWidth(slot.Item.Description, width, 1.1) {
		ui.DrawText(dst, line, x, lineY, 1.1, theme.LineDim)
		lineY += 18
	}
	// 単位重量（カーゴ計算用）
	lineY += 4
	ui.DrawText(dst, fmt.Sprintf(sh.WeightFmt, itemUnitWeight(slot.Item)), x, lineY, 1.1, theme.LineDim)
	// パーツの場合は性能ステータスを補足表示
	if !slot.Item.IsResource {
		if d := entity.PartDefByID(slot.Item.PartID); d != nil {
			lineY += 22
			for _, stat := range partStatLines(d) {
				for _, line := range wrapByWidth(stat, width, 1.1) {
					ui.DrawText(dst, line, x, lineY, 1.1, theme.LineDim)
					lineY += 18
				}
			}
		}
	}
}

// wrapByWidth は maxWidth(px) を超えないように文字(rune)単位で折り返す。
// 日本語のように空白を含まない文字列にも対応し、文字列中の既存の改行は保持する。
func wrapByWidth(s string, maxWidth, scale float64) []string {
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, r := range para {
			candidate := line + string(r)
			if w, _ := ui.MeasureText(candidate, scale); w > maxWidth && line != "" {
				lines = append(lines, line)
				line = string(r)
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// partStatLines は def の Kind に応じたステータス文字列を返す。
func partStatLines(d *entity.PartDef) []string {
	sh := i18n.S().Shop
	switch d.Kind {
	case entity.PartGun:
		style := sh.BulletStyleTrail
		switch d.GunBulletStyle {
		case entity.BulletStyleBall:
			style = sh.BulletStyleBall
		case entity.BulletStyleLaser:
			style = sh.BulletStyleLaser
		}
		impact := ""
		if d.GunBulletImpact {
			impact = sh.ImpactFXSuffix
		}
		return []string{
			fmt.Sprintf(sh.GunDmgCdFmt, d.GunDamage, d.GunCooldown),
			fmt.Sprintf(sh.GunBulletSpdFmt, d.GunBulletSpeed),
			fmt.Sprintf(sh.GunStyleFmt, style, impact),
		}
	case entity.PartThruster:
		return []string{
			fmt.Sprintf(sh.ThrusterAccelFmt, d.ThrustAccel, d.ThrustMaxSpeed),
			fmt.Sprintf(sh.ThrusterBoostFmt, d.ThrustBoostAccelMul, d.ThrustBoostMaxSpeed),
			fmt.Sprintf(sh.ThrusterFuelFmt, d.ThrustBoostFuelCost),
		}
	case entity.PartFuel:
		return []string{
			fmt.Sprintf(sh.FuelCapFmt, d.FuelCapacity),
		}
	case entity.PartArmor:
		return []string{
			fmt.Sprintf(sh.ArmorHPFmt, d.ArmorHP),
		}
	case entity.PartShield:
		return []string{
			fmt.Sprintf(sh.ShieldHPFmt, d.ShieldHP),
			sh.ShieldRegenNote,
		}
	case entity.PartCargo:
		return []string{
			fmt.Sprintf(sh.CargoCapFmt, d.CargoCapacity),
		}
	case entity.PartAutoAim:
		return []string{
			fmt.Sprintf(sh.AutoAimRangeFmt, d.AutoAimRange, d.AutoAimDPS),
			sh.AutoAimNote,
		}
	}
	return nil
}
