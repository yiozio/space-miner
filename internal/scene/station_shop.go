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

// shopItem は店または自分の在庫の1スロットが保持するアイテム。
type shopItem struct {
	Name        string
	Description string
	Price       int
	IsResource  bool
	PartKind    entity.PartKind     // 非資源
	ResType     entity.ResourceType // 資源
}

// shopSlot は1セルの内容。Quantity が 0 なら空セル扱い。
type shopSlot struct {
	Item     shopItem
	Quantity int
}

// StationShop はパーツ店舗シーン。
// 画面左に店在庫の 4 列グリッド、画面右に自分の在庫の 4 列グリッド、
// その間に取引サマリ、自分グリッドの右側にカーソル中アイテムの情報を表示する。
type StationShop struct {
	player      *entity.Player
	shopSlots   [shopSlotCount]shopSlot
	playerSlots [shopSlotCount]shopSlot
	side        int // 0 = shop, 1 = player
	index       int // 0..shopSlotCount-1

	// セッションサマリ（このシーンが開いている間の累計）
	sessionBuyCount  int
	sessionSellCount int
	sessionNet       int // sells - buys（プレイヤー視点での増減）
}

// NewStationShop は店舗シーンを生成する。
// 店在庫はサンプルパーツで初期化、自分在庫はプレイヤー所持資源から生成する。
func NewStationShop(p *entity.Player) *StationShop {
	s := &StationShop{player: p}
	s.shopSlots[0] = shopSlot{Item: shopItem{Name: "Gun", Description: "Forward-firing gun.", Price: 80, PartKind: entity.PartGun}, Quantity: 5}
	s.shopSlots[1] = shopSlot{Item: shopItem{Name: "Thruster", Description: "Engine module.", Price: 120, PartKind: entity.PartThruster}, Quantity: 3}
	s.shopSlots[2] = shopSlot{Item: shopItem{Name: "Cargo", Description: "Resource storage.", Price: 60, PartKind: entity.PartCargo}, Quantity: 6}
	s.shopSlots[3] = shopSlot{Item: shopItem{Name: "Armor", Description: "Hardened plating.", Price: 100, PartKind: entity.PartArmor}, Quantity: 4}
	s.shopSlots[4] = shopSlot{Item: shopItem{Name: "Auto-Aim", Description: "Auto-targets nearby asteroids.", Price: 250, PartKind: entity.PartAutoAim}, Quantity: 2}

	for i, rt := range entity.AllResourceTypes() {
		info := rt.Info()
		s.playerSlots[i] = shopSlot{
			Item: shopItem{
				Name:        info.Name,
				Description: info.Name + " ore. Mining material.",
				Price:       priceForResource(rt),
				IsResource:  true,
				ResType:     rt,
			},
			Quantity: p.Inventory[rt],
		}
	}
	return s
}

// priceForResource は資源1単位の売却価格。
func priceForResource(r entity.ResourceType) int {
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

func (ss *StationShop) currentSlot() *shopSlot {
	if ss.side == 0 {
		return &ss.shopSlots[ss.index]
	}
	return &ss.playerSlots[ss.index]
}

func (ss *StationShop) Update(d Director) error {
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

// transferOne はカーソル位置のアイテムを1つだけ反対側へ移す（=購入 or 売却）。
func (ss *StationShop) transferOne() {
	slot := ss.currentSlot()
	if slot.Quantity <= 0 {
		return
	}
	if ss.side == 0 {
		// Buy: shop → player
		if ss.player.Credits < slot.Item.Price {
			return
		}
		ss.player.Credits -= slot.Item.Price
		ss.sessionBuyCount++
		ss.sessionNet -= slot.Item.Price
		slot.Quantity--
		ss.addPartToPlayerSlots(slot.Item)
	} else {
		// Sell: player → shop
		ss.player.Credits += slot.Item.Price
		ss.sessionSellCount++
		ss.sessionNet += slot.Item.Price
		slot.Quantity--
		if slot.Item.IsResource {
			ss.player.Inventory[slot.Item.ResType]--
		}
	}
}

// addPartToPlayerSlots は購入したパーツを自分インベントリの空きスロットに追加する。
// 既に同種パーツのスロットがあれば数量加算、なければ新規スロット。
func (ss *StationShop) addPartToPlayerSlots(it shopItem) {
	for i := range ss.playerSlots {
		ps := &ss.playerSlots[i]
		if ps.Quantity > 0 && !ps.Item.IsResource && ps.Item.PartKind == it.PartKind {
			ps.Quantity++
			return
		}
	}
	for i := range ss.playerSlots {
		ps := &ss.playerSlots[i]
		if ps.Quantity == 0 {
			ps.Item = it
			ps.Quantity = 1
			return
		}
	}
}

func (ss *StationShop) Draw(dst *ebiten.Image, d Director) {
	theme := d.Theme()
	sw, sh := dst.Bounds().Dx(), dst.Bounds().Dy()

	vector.DrawFilledRect(dst, 0, 0, float32(sw), float32(sh),
		color.NRGBA{0, 0, 0, 220}, false)

	// ヘッダ
	headerScale := 3.0
	header := "PARTS SHOP"
	hw, hh := ui.MeasureText(header, headerScale)
	ui.DrawText(dst, header, (float64(sw)-hw)/2, 24, headerScale, theme.Line)

	// レイアウト
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

	// セクションラベル
	ui.DrawText(dst, "SHOP", shopX, sideY-22, 1.6, theme.LineDim)
	ui.DrawText(dst, "INVENTORY", playerX, sideY-22, 1.6, theme.LineDim)

	// 左右グリッド
	ss.drawGrid(dst, theme, shopX, sideY, ss.shopSlots[:], ss.side == 0)
	ss.drawGrid(dst, theme, playerX, sideY, ss.playerSlots[:], ss.side == 1)

	// 中央サマリ
	ss.drawSummary(dst, theme, summaryX, sideY)
	// 右側インフォ
	ss.drawInfo(dst, theme, infoX, sideY, infoW)

	ui.DrawText(dst, "[ WASD/Arrows: Move    Tab: Switch Side    Enter/Space: Buy/Sell    Esc: Leave ]",
		20, float64(sh)-30, 1.4, theme.LineDim)
}

// drawGrid は与えられたスロット群を縦横グリッド状に描画する。
// focused が true なら現在カーソル位置に強調枠を描く。
func (ss *StationShop) drawGrid(dst *ebiten.Image, theme *ui.Theme, x, y float64, slots []shopSlot, focused bool) {
	cs := float64(shopCellSize)
	for i := range slots {
		col := i % shopGridCols
		row := i / shopGridCols
		cx := x + float64(col)*(cs+shopCellPad)
		cy := y + float64(row)*(cs+shopCellPad)
		// 枠
		vector.StrokeRect(dst, float32(cx), float32(cy), float32(cs), float32(cs), 1, theme.LineDim, false)
		// 中身（簡易: 名前の最初の数文字をアイコン代わりに表示）
		if slots[i].Quantity > 0 {
			label := slots[i].Item.Name
			if len(label) > 4 {
				label = label[:4]
			}
			lw, lh := ui.MeasureText(label, 2.0)
			ui.DrawText(dst, label, cx+(cs-lw)/2, cy+(cs-lh)/2-4, 2.0, theme.Line)
			qty := fmt.Sprintf("x%d", slots[i].Quantity)
			qw, _ := ui.MeasureText(qty, 1.0)
			ui.DrawText(dst, qty, cx+cs-qw-4, cy+cs-12, 1.0, theme.LineDim)
		}
		// カーソル
		if focused && i == ss.index {
			vector.StrokeRect(dst, float32(cx-2), float32(cy-2), float32(cs+4), float32(cs+4), 2, theme.Line, false)
		}
	}
}

// drawSummary は中央のセッション集計を描画する。
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

// drawInfo は右側のカーソル中アイテム詳細を描画する。
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
