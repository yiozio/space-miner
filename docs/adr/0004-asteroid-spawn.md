# 0004. 小惑星スポーンモデル

## Status
Accepted

## Context
- 採掘対象の小惑星はゾーン中心に近いほど数を増やしたい（豊鉱の表現）。
- ミニマップに新規小惑星が突然出現するのは違和感があるため、
  ミニマップ外で生成して中央に流入してくる挙動が望ましい。
- 1 小惑星 1 素材として、見た目で資源種別が判別できるようにする。
- 全体マップ（FullMap）外では生成しない。

## Decision

- 各 `ResourceZone` は中心 `(CX, CY)`・半径 `Radius`・`MaxAsteroids`・素材重み (`Mix`)。
- `World.SpawnCap(px, py)` は (px, py) を含む FullMap のゾーン群について
  `Σ MaxAsteroids × max(0, 1 - dist/Radius)` を返す（線形フォールオフ）。
- 毎フレーム `Exploration.trySpawnAsteroid()`：
  - 現在数 < cap なら、自機からの距離 2200..3000 のリング上で点をサンプリング
    （ミニマップ表示半径 ~1500px の対角線 ~2120px の外側で生成 = ミニマップ外）。
  - その点が FullMap 内かつ何らかのゾーンに入っていれば、
    重なるゾーンの重みを合算して 1 素材を抽選 (`World.PickResource`)。
  - 1 小惑星 = 1 素材で生成し、自機方向に軽く流入する初速を与える。
- 自機から `asteroidCullDist` (=4000) を超えた小惑星は破棄（再生成に任せる）。
- `entity.NewAsteroid(seed, x, y, size, resource)` は単一素材で構築する。

## Consequences

- ゾーン中心ほど画面に小惑星が多く流入する「鉱床」感が出る。
- カメラが大きく動いても周囲は常に最新のスポーンに置き換わるため、
  古い小惑星のオーバーラン問題が起きない。
- ゾーン外を航行すると静寂になる。プレイヤーはミニマップで赤や鉱石色の点を見て
  ゾーンを発見する設計（[ADR 0014](./0014-hud-parallax.md) も参照）。
- スポーン上限は決定的でないため、フレームによってラグなしで逐次補充される
  （フレームあたり最大 1 体）。一気に湧くフレームはない。

## 関連
- `internal/entity/world.go`, `internal/entity/data_world.go`
- `internal/entity/asteroid.go`
