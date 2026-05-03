# 0005. パーツ定義レジストリとデータ分離

## Status
Accepted

## Context
- 同カテゴリ（ガン、スラスタ等）に複数のバリアント（Mk-I / Mk-II / Heavy ...）を
  個別の性能で定義したい。
- バランス調整は頻繁に行われるため、データ（性能・名前・価格）と
  実装（型定義・レジストリ）を分けて編集しやすくしたい。

## Decision

- 2 段階のキー：
  - `PartKind`（カテゴリ）— Cockpit / Gun / Thruster / Fuel / Cargo / Armor / Shield / AutoAim / Warp。
    描画・振る舞い分岐の単位。
  - `PartID`（バリアント）— `PartIDGunMkI` 等。インベントリ・装備・ショップのキー。
- `PartDef` がバリアント単位の性能を保持：価格、武器系（Damage / Cooldown / BulletSpeed / Style / Width / Impact）、
  推進系（Accel / MaxSpeed / BoostAccelMul / BoostMaxSpeed / BoostFuelCost）、
  装甲（ArmorHP）/ シールド（ShieldHP）/ 燃料（FuelCapacity）/ 積載（CargoCapacity）/
  AutoAim（Range / DPS）/ 重量（Weight）。
- レジストリ実装は `partdef.go`、データ登録は `init()` で `data_parts.go`。
- `Part` 構造体は `DefID PartID` を持ち、`Kind() PartKind` / `Def() *PartDef` で参照。

### data_*.go 命名規約
データ専用ファイルは `data_` プレフィックスを付け、新しい入力（敵パターン、ゾーン定義、
クエストパラメータ、店在庫など）は `data_*.go` に集約する。実装側は型・API のみ持つ。

| 領域 | データ | 実装 |
| --- | --- | --- |
| パーツ | `internal/entity/data_parts.go` | `internal/entity/partdef.go` |
| 資源 | `internal/entity/data_resources.go` | `internal/entity/resource.go` |
| ワールド | `internal/entity/data_world.go` | `internal/entity/world.go`, `celestial.go` |
| 海賊 | `internal/entity/data_pirates.go` | `internal/entity/pirate.go` |
| クエスト生成 | `internal/entity/data_quests.go` | `internal/entity/quest.go` |
| 店在庫 | `internal/scene/data_shop.go` | `internal/scene/station_shop.go` |
| キャラクター | `internal/dialog/data_characters.go` | `internal/dialog/dialog.go` |
| ダイアログ | `internal/dialog/data_scripts.go` | `internal/dialog/dialog.go` |

## Consequences

- 新バリアント追加は `data_*.go` への 1 つの登録で完結する。
- 同じ Kind であれば共通の見た目（アイコン）と振る舞い分岐を共有できる。
- 派生ステータス（MaxHP / MaxFuel / MaxShieldHP / MaxCargo）はパーツ集計から
  `Player.recomputeStats()` で動的に算出する（保存しない）。
- ショップ・編集 UI も `entity.AllPlaceablePartDefs()` を介して自動的に新バリアントを拾う。

## 関連
- ADR [0006 (回転と前後スラスタ)](./0006-part-rotation.md), [0007 (弾丸)](./0007-bullets-and-lasers.md)
