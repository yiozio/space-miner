# 0009. 海賊 NPC

## Status
Accepted

## Context
- 戦闘要素として敵 NPC を導入したい。
- 出没エリア（FullMap 内のゾーン）で湧き、複数のパターン（軽量・中型・重型）を
  バランスよく出現させたい。
- パーツ構成と AI パラメータを 1 つの定義データから引けるようにしたい。

## Decision

### 型と AI
- `Pirate` は `Ship` を継承し、HP / Pattern / fireTimer を持つ。
  機体構成・描画・当たり判定はプレイヤー機と同じ「ベース船体（土台）+ グリッドパーツ」方式を共有する
  （`internal/entity/shipbase.go`。コックピットはベースが内包するため `Parts` にコックピットは置かない）。
  `Pattern.BaseID` でベースを選べる（ゼロ値は `ShipBasePebble`=3x3）。
- AI（`Pirate.Update`）は単純な追跡 + 射撃：
  - プレイヤー方向への最短旋回（`Pattern.TurnSpeed`）。
  - 距離が `PreferredDist + 80` を超えれば追跡推進、近ければ慣性で滑る。
  - `MaxSpeed` でクランプ、軽いドラッグ (0.99) で暴走を抑制。
  - 距離 < `FireRange` かつ機首ずれ < 0.35 rad で発射。
- `Pirate.Update` は `([]Bullet, []LaserShot, []Drone)` を返す。`shoot()`（弾・レーザー）は
  ADR 0007 と同じ規約。ドローンランチャー（`PartDroneLauncher`）搭載時は、ガン発射とは
  独立した `droneTimer` に従い、`FireRange` 内で**自機を狙う** Hostile なドローンを設置する
  （プレイヤー設置ドローンが小惑星/海賊を狙うのと対になる挙動）。
- 描画は共通の `Ship.DrawAt`（ベース船体 + グリッドパーツ）に委譲し、`Ship.LineColor` で
  ライン色を赤 (`pirateLineColor=#ff6040`) に上書きする。画像キャッシュは元テーマのポインタで
  判定するため、色上書きが固定でもキャッシュは有効。敵識別は赤いライン色で行い、輪郭リングは描かない。
- 被弾判定はベース船体の外形を近似する衝突円群（`Ship.HullColliders`）で取り、
  プレイヤー機の被弾判定と同一方式に揃える（旧来の半径 30 円判定は撤去）。
  弾命中・レーザー命中の双方がハル外形で当たる。

### パターン定義
- `PiratePattern`（`internal/entity/data_pirates.go`）が機体構成 + AI パラメータ + ドロップを保持：
  - `Parts`、`BaseID`（ゼロ値 = Pebble）、`MaxHP`、`TurnSpeed` / `ThrustAccel` / `MaxSpeed` / `PreferredDist` / `FireRange`
  - `DropCreditsMin..Max`、`PartDropRate`、`PartDrops []PartID`
- 初期 3 パターン: `Scout`（軽量速攻）/ `Brawler`（中型 2 連装）/ `Cruiser`（重型ガン + 装甲）。

### 出没エリア
- `PirateZone`（`FullMap.PirateZones`）— 円形の出没エリア、`MaxPirates`、出現候補 `Patterns`。
- 同時出現上限はゾーンフォールオフ × MaxPirates の合算 (`World.PirateSpawnCap`)。
- スポーン: 毎フレーム `Exploration.trySpawnPirate()` がリング上で位置を選び、
  `World.PickPiratePattern` で出現パターンを抽選 → `entity.NewPirate` で生成。
- カル距離超過 (`asteroidCullDist=4000`) で破棄。

### ドロップ
- 撃破時に credits を `[Min..Max]` でランダム付与（即時 `player.Credits` に加算）。
- `PartDropRate` の確率で `PartDrops` から 1 つを `NewPartPickup` で空間に生成。
- 撃破時に `world.Containing(pr.X, pr.Y)` の FullMap 名で `PiratesKilledByMap[name]++`
  （Bounty クエストの進捗根拠、[ADR 0010](./0010-tavern-quests.md)）。

## Consequences

- パーツ構成を変えるだけで新パターンを追加できる。
- 海賊船の見た目はプレイヤー機と同じ「Ship 描画」を共有するため、新パーツが追加されても
  描画コードを増やさずに新パターンを作れる。
- 海賊 vs 小惑星 / 海賊 vs 海賊の物理は未実装（パススルー）。
  これは [KNOWN_GAPS](../KNOWN_GAPS.md) で将来課題として扱う。
- 海賊撃破は Bounty クエストの主要進捗源になる。

## 関連
- `internal/entity/pirate.go`, `data_pirates.go`
- `internal/scene/exploration.go` (`trySpawnPirate`, `cullPiratesAndDrop`)
