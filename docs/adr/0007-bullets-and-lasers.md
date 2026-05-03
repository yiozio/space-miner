# 0007. 弾丸スタイルと瞬間命中レーザー

## Status
Accepted

## Context
- ガンに見た目のバリエーション（線・球・ビーム）を持たせたい。
- レーザーは「撃った瞬間に着弾する」即時ヒットの挙動が期待される。
- 着弾エフェクトの有無もガンごとに切り替えたい。

## Decision

### 通常弾（Bullet）
- 3 つの `BulletStyle`：
  - `BulletStyleTrail`: 既存の見かけスクリーン速度（ワールド速度 − カメラ速度）に沿った短いライン。
    パーツ毎に太さ可変。
  - `BulletStyleBall`: `Width` を半径とする塗り円。低速・大ダメージ向け。
  - `BulletStyleLaser`: Bullet では使わない（後述、LaserShot 経由で別系統）。
- 各 Bullet は `Damage` / `Hostile` / `Style` / `Width` / `ImpactFX` を持つ。
- `PartDef` に `GunBulletStyle` / `GunBulletWidth` / `GunBulletImpact` を追加し、
  発射時に値をコピーする。

### 瞬間命中レーザー（LaserShot）
- `Player.Shoot` / `Pirate.shoot` は `([]Bullet, []LaserShot)` を返す。
  `GunBulletStyle == BulletStyleLaser` のガンだけ Bullet ではなく `LaserShot` を返す。
- `Exploration.fireLaser(shot)` が以下を行う：
  1. 発射点から方向ベクトル沿いに `raySphereHit` で最近接ヒットを探す。
     対象は小惑星グリッド（円近似 `g/2`）、対する船体（敵レーザーならプレイヤー、
     プレイヤーレーザーなら海賊）。
  2. ヒット対象に即時 `Damage` を適用。
  3. `entity.Beam`（短命の直線エフェクト、8 フレーム）を生成して可視化。
     ヒットしなかった場合は射程の終端まで線を引く。
  4. `ImpactFX` が真ならヒット点に `Impact`（広がる円リング）を生成。

### 着弾エフェクト
- `entity.Impact`：半径が広がりながらアルファがフェードする円リング、寿命 16 フレーム。
- 通常弾命中時は `ImpactFX` が真ならその位置に生成（Bullet 経路）。
- レーザーも同様。
- 色は friendly = 琥珀 (`#ffc040`)、hostile = 赤 (`#ff6040`)。

## Consequences

- ガンの個性付けが画一的でなく、見た目と挙動の両面で表現できる。
- レーザーは飛翔オブジェクトを持たないため、物理的な「漏れ」（早撃ち中の弾の追い越し等）が無く、
  クリーンに即時ヒットする。
- レーザーの当たり判定はレイ vs 円の解析解 (`raySphereHit`) で軽量。
- 着弾エフェクトは寿命管理だけの単純なエンティティで、毎フレーム独立に更新できる。
- ワープ時には `bullets` / `lasers (Beam)` / `impacts` をすべてクリア
  （古い世界の残骸を持ち越さない）。

## 関連
- `internal/entity/bullet.go`, `laser.go`, `impact.go`
- `internal/scene/exploration.go` (`fireLaser`, `raySphereHit`, `spawnImpact`)
