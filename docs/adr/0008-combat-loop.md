# 0008. 戦闘ループとピックアップ

## Status
Accepted

## Context
プレイヤーと海賊が混在するシーンで、弾の所属・小惑星との衝突・ドロップ回収を
一貫したルールで処理したい。

## Decision

### 弾の所属
- `Bullet.Hostile bool` で所属を判別。
  - `false`: プレイヤー弾（小惑星と海賊にダメージ）
  - `true`: 敵弾（プレイヤーにダメージ）
- 小惑星はどちらの弾でも砕ける（敵弾でも採掘される）。
- AutoAim ターゲット更新はプレイヤー弾命中時のみ。

### 命中判定
- 弾 vs 小惑星: `Asteroid.Hit(bx, by, damage)` が点-AABB（自転反映）で判定。
- 弾 vs 海賊: 円-点判定 `pirateHitRadius=30`。`pr.HP <= 0` の海賊はスキップ。
- 弾 vs プレイヤー: 円-点判定 `playerHitRadius=GridSize`。
- レーザーはレイキャスト ([ADR 0007](./0007-bullets-and-lasers.md))。

### 体当たりダメージ
- 自機 ⇄ 小惑星: 自機側のみ移動、相対法線速度から反射＋ダメージ
  （`collisionDamageThreshold=1`、`collisionDamageFactor=3`、`collisionRestitution=0.6`）。
- 自機 ⇄ 海賊: 等質量の弾性衝突インパルス。重なりを双方半分ずつ押し戻し、
  接近相対速度が閾値超えなら**両者同量ダメージ**。
- 海賊 vs 海賊 / 海賊 vs 小惑星: 未実装（パススルー）。

### ダメージ吸収順序
- プレイヤーが受けるダメージは Shield → HP の順に吸収。
- Shield は最後の被弾から `ShieldRegenDelay=120` フレーム経過後に
  `ShieldRegenPerFrame=0.5/frame` で再生（ADR 0014 のバー表示と連動）。
- 被弾後の無敵時間は撤去（ユーザー指示）。連続被弾はそのまま入る。

### ドロップとピックアップ
- 小惑星グリッド破壊 → `Pickup`（資源、`PickupKind=Resource`）が発生。
- 海賊撃破 → credits を即時付与 + `PartDropRate` で稀に `Pickup`（`PickupKind=Part`）を生成。
- ピックアップは半径 `pickupAttractRadius=250` で吸引、`pickupCollectRadius=18` で取得。
- 取得時に積載重量を超えるなら `AddResource` / `AddSparePart` が `false` を返し、
  ピックアップを自機外側へ少し弾く（吸引ループから抜け、寿命まで漂う）。
- ピックアップ寿命: 30 秒（`pickupLifeFrames`）。

## Consequences

- 戦闘・採掘・回収が同じルートで処理されるため、海賊と小惑星が混在する場面でも
  違和感なく動作する。
- 無敵時間が無いことで連続被弾のリスクが上がる一方、シールド再生が緩衝役になる。
- 積載超過時の反発挙動は控えめなため、プレイヤーは積極的に売却・装備見直しの動機付けになる。

## 関連
- `internal/scene/exploration.go` (collision, bullet routing, pickup loop)
- `internal/entity/player.go` (`Damage`, `AddResource`, `AddSparePart`)
- `internal/entity/asteroid.go`, `pickup.go`
