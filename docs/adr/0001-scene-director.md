# 0001. シーン管理と Director

## Status
Accepted

## Context
タイトル・探索・ステーションメニュー・ショップ・編集画面・ダイアログなど、
画面（シーン）が増える前提で、scene パッケージから game パッケージへ
逆参照せずにシーン遷移を表現したい。
オーバーレイ表示（メニュー、ダイアログ、確認モーダル）も必要。

## Decision

- 画面単位は `scene.Scene` インタフェース (`Update(d Director) error`, `Draw(dst, d)`) で抽象化する。
- 遷移操作は `scene.Director` インタフェースに集約：
  - `Push(Scene)` / `Pop()` / `Replace(Scene)` / `Quit()`
  - `Theme()` / `SetTheme()`
- ゲームコアはシーンスタックを持ち、最上位だけ `Update` する一方、
  `Draw` は下位から順に呼ぶ（オーバーレイの背後に下のシーンが残る）。
- 遷移の指示は `Director.Push` 等を通じて行い、`Scene` は内部状態のみ持つ。
  状態 ID（`game.StateID`: `StateTitle` 等）は概念列挙であり、
  実体はシーンスタックである。

## Consequences

- メニュー / 確認モーダル / ダイアログ / 全体マップ / 恒星マップ等を
  すべてオーバーレイ Push で実装でき、下層を再描画するだけで透けて見える UI ができる。
- シーン同士は直接参照せず、`Director` 経由で疎結合。
- `Director.Replace` を使えば現在のシーンを差し替えられる（New Game / Continue / Quit To Title 等で使用）。
- 一方、グローバルなプレイヤー状態などは Push 元から渡す必要がある（DI）。
  `NewMenu(save.Context)` や `NewStationMenu(player, world, stationName)` のように
  シグネチャが膨らむことがある。

## 関連
- 各シーンは `internal/scene/` 配下
- 入力は `inpututil.IsKeyJustPressed` を基本とする（ホールドが意味を持つ箇所のみ `IsKeyPressed`）
