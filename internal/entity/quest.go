package entity

import (
	"fmt"
	"math/rand"
)

// QuestKind は依頼の種別。
type QuestKind int

const (
	QuestKindDelivery QuestKind = iota // 資源納品: Resource を Amount 個渡す
	QuestKindBounty                    // 海賊討伐: MapName で PirateTarget 体倒す
)

// Quest は宇宙酒場に掲示される 1 件の依頼。
// Delivery は Inventory の資源を消費して即時完了。
// Bounty は出題時の累計撃破数を PirateBaseline に保存し、
// (現在の撃破数 - PirateBaseline) >= PirateTarget で完了。
type Quest struct {
	ID    string
	Kind  QuestKind
	Title string

	// Delivery 用
	Resource ResourceType
	Amount   int

	// Bounty 用
	MapName        string
	PirateTarget   int
	PirateBaseline int

	// 報酬: クレジットは必ず付与、HasPartReward が true なら RewardPart を 1 つ追加付与
	RewardCredits int
	RewardPart    PartID
	HasPartReward bool

	// 破棄コスト（クレジット）
	DiscardCost int
}

// IsEmpty は未生成スロット用の判定（ID が空文字）。
func (q *Quest) IsEmpty() bool { return q.ID == "" }

// Description は要求内容の表示文字列。
func (q *Quest) Description() string {
	switch q.Kind {
	case QuestKindDelivery:
		return fmt.Sprintf("Deliver %d %s", q.Amount, q.Resource.Info().Name)
	case QuestKindBounty:
		return fmt.Sprintf("Defeat %d pirates in %s", q.PirateTarget, q.MapName)
	}
	return "?"
}

// TavernBoard は 1 ステーションの 3 スロット掲示板。
// 各スロットは Quest（IsEmpty()=true で空扱い）。
type TavernBoard struct {
	Slots [3]Quest
}

// QuestProgress は (現在進捗, 目標) を返す。Delivery では持ち資源量、Bounty では撃破差分。
func (p *Player) QuestProgress(q *Quest) (current, target int) {
	switch q.Kind {
	case QuestKindDelivery:
		return p.Inventory[q.Resource], q.Amount
	case QuestKindBounty:
		killed := p.PiratesKilledByMap[q.MapName]
		return killed - q.PirateBaseline, q.PirateTarget
	}
	return 0, 0
}

// CanClearQuest は要件を満たしているか返す。
func (p *Player) CanClearQuest(q *Quest) bool {
	cur, target := p.QuestProgress(q)
	return cur >= target
}

// CanReceiveReward は報酬パーツを積載できるか返す（パーツ報酬無しなら常に true）。
func (p *Player) CanReceiveReward(q *Quest) bool {
	if !q.HasPartReward {
		return true
	}
	d := PartDefByID(q.RewardPart)
	if d == nil {
		return true
	}
	return p.CanAddWeight(d.Weight)
}

// ClearQuest は完了処理: 資源消費 + 報酬付与。完了済み判定は呼び出し側で行うこと。
// パーツ報酬で積載超過なら付与をスキップ（クレジットだけ渡す）。
func (p *Player) ClearQuest(q *Quest) {
	if q.Kind == QuestKindDelivery {
		if p.Inventory == nil {
			p.Inventory = make(map[ResourceType]int)
		}
		p.Inventory[q.Resource] -= q.Amount
		if p.Inventory[q.Resource] < 0 {
			p.Inventory[q.Resource] = 0
		}
	}
	p.Credits += q.RewardCredits
	if q.HasPartReward {
		p.AddSparePart(q.RewardPart, 1)
	}
}

// EnsureTavernBoard はステーションの掲示板を返す。未生成ならランダム 3 枠を生成する。
// クエストはステーション（=同名 FullMap）専用に生成され、Bounty は自マップ固定。
func (p *Player) EnsureTavernBoard(stationName string, world *World, rng *rand.Rand) *TavernBoard {
	if p.Tavern == nil {
		p.Tavern = make(map[string]*TavernBoard)
	}
	b, ok := p.Tavern[stationName]
	if !ok {
		b = &TavernBoard{}
		for i := 0; i < 3; i++ {
			b.Slots[i] = GenerateQuest(rng, world, p.PiratesKilledByMap, stationName)
		}
		p.Tavern[stationName] = b
	}
	return b
}

// RegenerateSlot は掲示板の指定スロットを差し替える。
func (p *Player) RegenerateSlot(stationName string, idx int, world *World, rng *rand.Rand) {
	b := p.EnsureTavernBoard(stationName, world, rng)
	if idx < 0 || idx >= len(b.Slots) {
		return
	}
	b.Slots[idx] = GenerateQuest(rng, world, p.PiratesKilledByMap, stationName)
}
