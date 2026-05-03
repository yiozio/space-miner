# 0011. ダイアログ基盤

## Status
Accepted

## Context
- ストーリー・NPC 会話・選択肢・オープニングを 1 つの仕組みで扱いたい。
- 外部画像アセットを持たないため、キャラクター画像はベクター描画で表現する。
- ステーション初回入船で自動再生し、再訪では再生しない。

## Decision

### スクリプト型
- `dialog.Node`: `Speaker`（キャラ ID、空ならナレーション）、`Text`、`Next` または `Choices`。
- `dialog.Choice`: `Label` + 次ノード ID。空 ID で会話終了。
- `dialog.Script`: `Start` + `map[string]Node`。
- データ集約: `internal/dialog/data_scripts.go`、`data_characters.go`。

### キャラクター画像
- `dialog.Character` は色 + `AvatarStyle`（Smile / Stern / Goggles / Helmet）を持つ。
- アバター枠の中に固定図形（頭円 + 目 + Style に応じた口やゴーグルや兜）を描画。
- 外部画像は使わず、すべてベクター。シーン全体のレトロ風と統一。

### シーン
- `DialogScene`：暗いオーバーレイ + 下部ダイアログ枠（左にアバター、右にテキスト）。
  タイプライタ表示 (~42 文字/秒)、`Enter`/`Space` で全文表示ショートカット。
  選択肢は本文表示完了後にカーソル選択 → 確定で次ノードへ。`Esc` で全体スキップ。
- `OpeningScene`：全画面背景（恒星シルエット + 軌道線 + `PROLOGUE` 文字）+ アバター無しのナレーション。
  `DialogScene` と同じ実装、`dialogStyle` で分岐。

### 起動契機
- 新規ゲーム: `Title.NewGame` で `Exploration` を Replace した上に `OpeningScene(&Opening)` を Push。
- 初回入船: `Exploration` のドック処理で `!player.VisitedStations[name]` なら
  `dialog.ScriptForStation(name)` を取得して `StationMenu` の上に `DialogScene` を Push。
  訪問済みフラグは `player.VisitedStations` で永続化。

## Consequences

- スクリプトは単純な map なので、簡単な分岐は表現できる。
  グラフ的な複雑な分岐や状態管理は将来課題（[KNOWN_GAPS](../KNOWN_GAPS.md)）。
- アバターは Style 列挙の追加だけで個性を増やせる。
  外部画像を入れる場合は描画関数を差し替え可能なように Character に `Draw` 関数フィールドを
  足す形になりうる（現状は固定）。
- セーブ: `VisitedStations` を含むので、ロード後も再演出が起きない。

## 関連
- `internal/dialog/dialog.go`, `data_characters.go`, `data_scripts.go`
- `internal/scene/dialog_scene.go`
