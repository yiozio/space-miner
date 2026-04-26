package scene

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/yiozio/space-miner/internal/entity"
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
)

// shopItem はスロット内アイテムの内容。
type shopItem struct {
	Name        string
	Description string
	Price       int
	IsResource  bool
	PartKind    entity.PartKind
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

	sessionBuyCount  int
	sessionSellCount int
	sessionNet       int
}

// NewStationShop は店舗シーンを生成する。
// 店在庫はサンプルパーツを固定初期化、自分側は毎フレーム refreshPlayerSlots で
// プレイヤー状態から再構築する。
func NewStationShop(p *entity.Player) *StationShop {
	s := &StationShop{player: p}
	s.shopSlots[0] = shopSlot{Item: shopParts(entity.PartGun, "Forward-firing gun."), Quantity: 5}
	s.shopSlots[1] = shopSlot{Item: shopParts(entity.PartThruster, "Engine module."), Quantity: 3}
	s.shopSlots[2] = shopSlot{Item: shopParts(entity.PartCargo, "Resource storage."), Quantity: 6}
	s.shopSlots[3] = shopSlot{Item: shopParts(entity.PartArmor, "Hardened plating."), Quantity: 4}
	s.shopSlots[4] = shopSlot{Item: shopParts(entity.PartAutoAim, "Auto-targets nearby asteroids."), Quantity: 2}
	s.shopSlots[5] = shopSlot{Item: shopParts(entity.PartFuel, "Auxiliary fuel tank."), Quantity: 4}
	s.shopSlots[6] = shopSlot{Item: shopParts(entity.PartShield, "Shield generator."), Quantity: 2}
	s.refreshPlayerSlots()
	return s
}

// shopParts はパーツ種別から shopItem を組み立てる（共通価格を引く）。
func shopParts(kind entity.PartKind, desc string) shopItem {
	return shopItem{
		Name:        entity.PartName(kind),
		Description: desc,
		Price:       partPrice(kind),
		PartKind:    kind,
	}
}

// partPrice はパーツ売買の共通価格。
func partPrice(kind entity.PartKind) int {
	switch kind {
	case entity.PartGun:
		return 80
	case entity.PartThruster:
		return 120
	case entity.PartFuel:
		return 70
	case entity.PartCargo:
		return 60
	case entity.PartArmor:
		return 100
	case entity.PartShield:
		return 150
	case entity.PartAutoAim:
		return 250
	case entity.PartWarp:
		return 400
	}
	return 30
}

// resourcePrice は資源1単位の価格。
func resourcePrice(r entity.ResourceType) int {
	switch r {
	case entity.ResourceIron:
		return 5
	case entity.ResourceCrystal:
		return 30
	case entity.ResourceIce:
		return 8
	}
	return 1
}

// refreshPlayerSlots はプレイヤーの現在状態から自分グリッドを再構築する。
// 資源を上から、スペアパーツをその後ろに並べる。
func (ss *StationShop) refreshPlayerSlots() {
	for i := range ss.playerSlots {
		ss.playerSlots[i] = shopSlot{}
	}
	idx := 0
	for _, rt := range entity.AllResourceTypes() {
		qty := ss.player.Inventory[rt]
		if qty <= 0 || idx >= shopSlotCount {
			continue
		}
		info := rt.Info()
		ss.playerSlots[idx] = shopSlot{
			Item: shopItem{
				Name:        info.Name,
				Description: info.Name + " ore. Mining material.",
				Price:       resourcePrice(rt),
				IsResource:  true,
				ResType:     rt,
			},
			Quantity: qty,
		}
		idx++
	}
	for _, kind := range entity.AllPlaceablePartKinds() {
		qty := ss.player.PartsInventory[kind]
		if qty <= 0 || idx >= shopSlotCount {
			continue
		}
		ss.playerSlots[idx] = shopSlot{
			Item: shopItem{
				Name:        entity.PartName(kind),
				Description: entity.PartName(kind) + " part (spare).",
				Price:       partPrice(kind),
				PartKind:    kind,
			},
			Quantity: qty,
		}
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
		d.Pop()
		return nil
	}

	row := ss.index / shopGridCols
	col := ss.index % shopGridCols

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

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		ss.transferOne()
	}
	return nil
}

// transferOne はカーソル位置のアイテムを1つだけ反対側に移す（=購入 or 売却）。
// プレイヤーの所持状態（Inventory / PartsInventory / Credits）に直接反映される。
func (ss *StationShop) transferOne() {
	slot := ss.currentSlot()
	if slot.Quantity <= 0 {
		return
	}
	if ss.side == 0 {
		if ss.player.Credits < slot.Item.Price {
			return
		}
		ss.player.Credits -= slot.Item.Price
		ss.sessionBuyCount++
		ss.sessionNet -= slot.Item.Price
		slot.Quantity--
		if slot.Item.IsResource {
			ss.player.Inventory[slot.Item.ResType]++
		} else {
			ss.player.PartsInventory[slot.Item.PartKind]++
		}
	} else {
		ss.player.Credits += slot.Item.Price
		ss.sessionSellCount++
		ss.sessionNet += slot.Item.Price
		slot.Quantity--
		if slot.Item.IsResource {
			ss.player.Inventory[slot.Item.ResType]--
		} else {
			ss.player.PartsInventory[slot.Item.PartKind]--
		}
	}
}

func (ss *StationShop) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	headerScale := 3.0
	header := "PARTS SHOP"
	hw, hh := ui.MeasureText(header, headerScale)
	ui.DrawText(dst, header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	gap := 24.0
	summaryW := 200.0
	infoW := 280.0
	totalW := float64(shopSideWidth)*2 + summaryW + infoW + gap*3
	startX := (float64(sw) - totalW) / 2
	sideY := 24 + hh + 40

	shopX := startX
	summaryX := shopX + float64(shopSideWidth) + gap
	playerX := summaryX + summaryW + gap
	infoX := playerX + float64(shopSideWidth) + gap

	ui.DrawText(dst, "SHOP", shopX, sideY-22, 1.6, theme.LineDim)
	ui.DrawText(dst, "INVENTORY", playerX, sideY-22, 1.6, theme.LineDim)

	ss.drawGrid(dst, theme, shopX, sideY, ss.shopSlots[:], ss.side == 0)
	ss.drawGrid(dst, theme, playerX, sideY, ss.playerSlots[:], ss.side == 1)
	ss.drawSummary(dst, theme, summaryX, sideY)
	ss.drawInfo(dst, theme, infoX, sideY, infoW)

	ui.DrawText(dst, "[ WASD/Arrows: Move    Tab: Switch Side    Enter/Space: Buy/Sell    Esc: Leave ]",
		20, float64(sh)-30, 1.4, theme.LineDim)
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
	// パーツはミニアイコン
	iconCell := float32(cs * 0.8)
	ix := float32(cx) + (float32(cs)-iconCell)/2
	iy := float32(cy) + (float32(cs)-iconCell)/2 - 4
	entity.DrawPart(dst, entity.Part{Kind: s.Item.PartKind}, ix, iy, iconCell, theme)
}

func (ss *StationShop) drawSummary(dst *ebiten.Image, theme *ui.Theme, x, y float64) {
	ui.DrawText(dst, "SESSION", x, y, 1.6, theme.Line)
	lineY := y + 32
	ui.DrawText(dst, fmt.Sprintf("BUY  %d", ss.sessionBuyCount), x, lineY, 1.4, theme.LineDim)
	lineY += 24
	ui.DrawText(dst, fmt.Sprintf("SELL %d", ss.sessionSellCount), x, lineY, 1.4, theme.LineDim)
	lineY += 36
	sign := "+"
	if ss.sessionNet < 0 {
		sign = ""
	}
	ui.DrawText(dst, fmt.Sprintf("NET %s%d", sign, ss.sessionNet), x, lineY, 1.7, theme.Line)
	lineY += 50
	ui.DrawText(dst, fmt.Sprintf("CR %d", ss.player.Credits), x, lineY, 1.4, theme.Line)
}

func (ss *StationShop) drawInfo(dst *ebiten.Image, theme *ui.Theme, x, y, _ float64) {
	ui.DrawText(dst, "INFO", x, y, 1.6, theme.Line)
	slot := ss.currentSlot()
	if slot.Quantity == 0 {
		ui.DrawText(dst, "(empty)", x, y+34, 1.3, theme.LineDim)
		return
	}
	lineY := y + 34
	ui.DrawText(dst, slot.Item.Name, x, lineY, 1.8, theme.Line)
	lineY += 32
	var action string
	if ss.side == 0 {
		action = fmt.Sprintf("BUY %d cr", slot.Item.Price)
	} else {
		action = fmt.Sprintf("SELL %d cr", slot.Item.Price)
	}
	ui.DrawText(dst, action, x, lineY, 1.4, theme.Line)
	lineY += 32
	ui.DrawText(dst, slot.Item.Description, x, lineY, 1.1, theme.LineDim)
}
