package entity

import (
	"fmt"
	"math/rand"
)

// data_quests.go は酒場クエストの生成パラメータ。
// バランス調整はここで行う。

// 納品依頼で要求対象になる資源プール。
var deliveryResources = []ResourceType{ResourceIron, ResourceBronze, ResourceIce}

// 報酬として付与されうるパーツ（コックピット等のレア・必須は除外）。
var rewardablePartIDs = []PartID{
	PartIDGunMkI,
	PartIDGunMkII,
	PartIDGunRapid,
	PartIDThrusterStd,
	PartIDThrusterLight,
	PartIDArmorStd,
	PartIDShieldStd,
	PartIDFuelStd,
	PartIDCargoStd,
	PartIDAutoAimStd,
}

// GenerateQuest はランダム Quest を生成する。
// 各ステーションの掲示板はそのマップ専用なので、討伐対象も自マップに限定する。
//   - rng:           乱数ソース（呼び出し側で seed 制御可能）
//   - world:         参照のみ（マップ存在確認）
//   - kills:         PiratesKilledByMap 相当。Bounty の PirateBaseline 設定に使う（nil でも可）
//   - currentMap:    出題するステーション（FullMap）名。Bounty はここに固定される
func GenerateQuest(rng *rand.Rand, world *World, kills map[string]int, currentMap string) Quest {
	kind := QuestKindDelivery
	// 自マップに海賊ゾーンがあれば、40% 程度で Bounty を選ぶ
	if mapHasPirateZone(world, currentMap) && rng.Float64() < 0.40 {
		kind = QuestKindBounty
	}

	q := Quest{
		ID:   fmt.Sprintf("q%d", rng.Int63()),
		Kind: kind,
	}

	switch kind {
	case QuestKindDelivery:
		r := deliveryResources[rng.Intn(len(deliveryResources))]
		amount := 5 + rng.Intn(45) // 5..49
		q.Resource = r
		q.Amount = amount
		// 報酬: 市場価格 × 1.4..2.0 倍
		baseValue := r.Price() * amount
		q.RewardCredits = int(float64(baseValue) * (1.4 + rng.Float64()*0.6))
		// 10% でレアパーツ追加
		if rng.Float64() < 0.10 {
			q.RewardPart = rewardablePartIDs[rng.Intn(len(rewardablePartIDs))]
			q.HasPartReward = true
		}
		q.DiscardCost = q.RewardCredits / 10

	case QuestKindBounty:
		target := 3 + rng.Intn(6) // 3..8
		q.MapName = currentMap
		q.PirateTarget = target
		if kills != nil {
			q.PirateBaseline = kills[currentMap]
		}
		// 報酬: 1 体あたり 80..120 cr 程度
		q.RewardCredits = target*80 + rng.Intn(target*40+1)
		// 15% でレアパーツ追加
		if rng.Float64() < 0.15 {
			q.RewardPart = rewardablePartIDs[rng.Intn(len(rewardablePartIDs))]
			q.HasPartReward = true
		}
		q.DiscardCost = q.RewardCredits / 8
	}
	return q
}

// mapHasPirateZone は指定 FullMap に PirateZone が 1 つ以上あるか返す。
func mapHasPirateZone(w *World, name string) bool {
	if w == nil {
		return false
	}
	for i := range w.Maps {
		m := &w.Maps[i]
		if m.Name == name && len(m.PirateZones) > 0 {
			return true
		}
	}
	return false
}
