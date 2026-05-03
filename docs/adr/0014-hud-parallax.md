# 0014. HUD とパララックス

## Status
Accepted

## Context
- HP / シールド / 燃料は数値テキストでなく、自機下のバーで直感的に把握したい。
- ステーション・小惑星・惑星・星空が混在する画面で、奥行き感を維持したい。
- ミニマップは小惑星の色で素材を判別したい。

## Decision

### 自機下バー
- HP / Shield / Fuel を縦に並べ、自機のパーツバウンディング半径
  (`max_part hypot(GX*g, GY*g) + g/2`) + 余白 18px の位置に表示。
  機体の向きに依らず、円形バウンディングなのでパーツに被らない。
- バー寸法: 80×6 px、ギャップ 4px。
- 色:
  - HP: 赤 `#ff6060`
  - Shield: シアン `#60c0ff`（`MaxShieldHP > 0` のみ表示）
  - Fuel: 黄 `#ffe060`（`MaxFuel > 0` のみ表示）
- 背景は `theme.LineDim` の半透明、フィルは比率分、枠は `theme.LineDim`。
- テキスト HUD は `CARGO m/M   CR n` のみ（HP / Shield / FUEL は撤去）。

### パララックス階層
- 共有定数 (`internal/scene/starfield.go`)：
  - `farStarParallax = 0.0` ← 遠景星は完全静止。
  - `nearStarParallax = 0.10` ← 近景星と惑星バックドロップ共通の係数。
- 惑星バックドロップ (`drawCelestialBackdrop`) は FullMap 中心を anchor として
  `screenPos = body.BackdropOffset + (mapCenter - camera) × P + screenCenter`
  の式で配置。プレイヤーがステーション付近にいるときは設計位置どおりに見え、
  自機が動いた分だけ少しだけ流れる。

### 全体マップ・ミニマップ
- ミニマップ：自機を中心とした小型俯瞰。
  - 小惑星はその構成素材色（資源の `Info().Color`）で点描。
  - 海賊は赤色の点。
  - ステーションは枠付き四角（`theme.Line`）。
- 全体マップ画面 (`WorldMapView`)：FullMap 全体を俯瞰。素材ゾーン円・PirateZone（`HOSTILE` 表示）・
  ステーション（ひし形）・自機（向きを示す二等辺三角形）を描画。
- 恒星マップ画面 (`StarMap`)：恒星系俯瞰、ワープ先選択 UI 兼用 ([ADR 0012](./0012-warp.md))。

## Consequences

- HP / Shield / Fuel が瞬時に判別できる。
  数値が必要な場面（過剰積載のチェックなど）はテキスト HUD に残している。
- 視差が「遠景星=静止 / 近景星=ゆっくり / 惑星=同じ近景レイヤ」と階層化されることで、
  ステーション付近では惑星が常に空に居続ける。
- 機体が大きくなる（パーツが増える）とバーも自動的に下に移動する。
- ミニマップに資源色を載せたことで、プレイヤーは色を頼りに鉱床を判別する設計が成り立つ。

## 関連
- `internal/scene/exploration.go` (`drawHUD`, `drawPlayerVitalBars`, `drawCelestialBackdrop`)
- `internal/scene/starfield.go`
- `internal/scene/worldmap.go`, `star_map.go`
