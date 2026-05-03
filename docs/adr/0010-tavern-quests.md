# 0010. 酒場クエスト（Tavern Quests）

## Status
Accepted

## Context
- ステーションごとに自動生成されるクエスト掲示板（3 枠）が欲しい。
- クエストは「資源納品（Delivery）」と「海賊討伐（Bounty）」の 2 種を持ちたい。
- 同じステーションを再訪したら掲示板の内容は引き継がれ、
  クリア / 破棄したスロットだけが入れ替わる。
- 海賊討伐はそのステーションのある FullMap の海賊だけを対象にしたい。

## Decision

### データ構造
- `entity.Quest`：
  - `Kind`: `QuestKindDelivery` | `QuestKindBounty`
  - Delivery: `Resource`, `Amount`
  - Bounty: `MapName`, `PirateTarget`, `PirateBaseline`
  - 報酬: `RewardCredits`, `RewardPart` + `HasPartReward`
  - 破棄コスト: `DiscardCost`
- `TavernBoard`: `[3]Quest`。`Quest.IsEmpty()` でスロット空判定（ID="" のとき）。
- `Player.Tavern map[stationName]*TavernBoard`：ステーション専用の掲示板を永続化。
- `Player.PiratesKilledByMap map[string]int`：FullMap 名 → 累計撃破数。
  Bounty 進捗 = `PiratesKilledByMap[Q.MapName] - Q.PirateBaseline`。

### 生成
- `GenerateQuest(rng, world, kills, currentMap)` がランダム生成。
  - 自マップに `PirateZone` がある場合のみ Bounty を 40% で抽選、
    それ以外は Delivery（鉄/青銅/氷から抽選、5..49 個）。
  - Bounty は **`MapName = currentMap`** に固定し、
    `PirateBaseline = kills[currentMap]` を出題時にスナップショット。
- 同じステーションを再訪してもボードは保存されており、空スロットは生成済みのまま残る。
- `EnsureTavernBoard` が初回訪問時の 3 スロット生成と取得を兼ねる。

### UI（StationTavern シーン）
- 3 枚のカード（種別タグ / 説明 / 進捗 / 報酬 / 破棄コスト）を縦並び表示。
- `↑/↓` でカーソル、`Enter`：要件達成 + パーツ報酬を積載できれば CLEAR 実行 → スロット差し替え。
- `D`: 破棄コスト支払い → スロット差し替え。
- 積載超過で報酬パーツを受領できない場合は赤字注意書きで CLEAR を阻止。

### 進捗計算
- Delivery: `Player.Inventory[Resource]` と `Amount` を比較。CLEAR で資源を消費。
- Bounty: 海賊撃破 (`Exploration.cullPiratesAndDrop`) で `PiratesKilledByMap` がインクリメント
  → 各 Bounty の進捗が同時に伸びる。

## Consequences

- 同じマップで複数の Bounty があれば全部に進捗が走る（重複達成可）。
- ステーション切り替え（ワープ）すると別の掲示板が見える。
  各ステーションの経済・依頼が独立する設計。
- ベースライン方式により、過去の撃破は新規 Bounty に流用されない。
- セーブには `Tavern` と `PiratesKilledByMap` が含まれる
  ([ADR 0013](./0013-save.md))。

## 関連
- `internal/entity/quest.go`, `data_quests.go`
- `internal/scene/station_tavern.go`, `station_menu.go`
