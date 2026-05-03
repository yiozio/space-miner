# 0003. ワールド階層

## Status
Accepted

## Context
- ゲームは「だだっ広い宇宙」であり、その中に区画として複数の宙域（FullMap）が点在する。
- 各 FullMap には宇宙ステーションと素材ゾーンがあり、海賊出没エリアもある。
- 恒星マップ（ワープ先選択 UI）は同じ世界座標系をそのままレイアウトに使い、
  ワープ先選択時の方向感とゲーム内の方向が一致するようにしたい。

## Decision

- 3 階層モデル：
  - `World`（境界なし）— `Star Celestial` と `[]FullMap`
  - `FullMap`（中心 CX,CY と半幅 30000）— `[]ResourceZone`、`[]PirateZone`、`Body Celestial`
  - `ResourceZone`（円形、素材重み）/ `PirateZone`（円形、出現パターン候補）

- 区画は重ならないように配置する（`World.Containing(x,y)` は最初に含む区画を返す）。
- 区画外（FullMap 外）では小惑星も海賊も生成されない（明示的な空虚）。
- 恒星 (`Star Celestial`) は世界座標 (0, 0) を中心に置く規約とし、
  各 FullMap の Body は `m.CX, m.CY` をそのまま恒星マップの位置に使う。
- これにより「恒星マップ上で見た方向」と「ワープ後に向く方向」が一致する。
- データは `internal/entity/data_world.go` に集約。FullMap 名 = ステーション名 = `Body.Name`。
- 新マップを足す手順は ADR 末尾に記載。

## Consequences

- 区画間の物理距離は意図的に大きくする（数十万 px）。間にプレイヤーが入っても何も湧かない。
- 区画外でもプレイヤーは航行可能（ワープなしで他マップへたどり着くこともできる）。
- ステーション座標 = FullMap 中心とすることで、恒星マップ・ワープ・初回ダイアログ・
  Bounty クエスト判定が同一の名前空間で結ばれる。
- 既存セーブのプレイヤー座標が空虚にあるケースが起きうる（マップ再配置時）。
  ロード時の区画判定はベストエフォートで、空虚なら `lastMap=nil`。

## 新マップ追加手順
1. `data_world.go` に `FullMap` を追加（Name / CX, CY / Body / Zones / PirateZones）。
2. ステーションは `NewExploration` の初期化ループで `m.CX, m.CY` に自動配置される。
3. 必要なら初回入船ダイアログを `dialog.ScriptForStation` に登録（[ADR 0011](./0011-dialog.md)）。
