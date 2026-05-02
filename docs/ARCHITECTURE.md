# Space Miner — アーキテクチャ

## 設計方針
- クリーンアーキテクチャを意識し、ゲームコア / シーン / エンティティ / UI / アセットを疎結合に保つ
- ゲーム全体の状態管理はステートマシンパターン
- シーン（画面）も差し替え可能なインターフェースとして扱う

## ディレクトリ構成
```
space-miner/
├── cmd/
│   └── game/
│       └── main.go          # エントリーポイント
├── internal/
│   ├── game/                # ゲームコア
│   │   ├── game.go         # メインゲームループ
│   │   └── state.go        # ゲーム状態管理
│   ├── scene/               # シーン管理
│   │   ├── scene.go        # シーンインターフェース
│   │   ├── exploration.go  # 探索シーン
│   │   └── warp.go         # ワープ先選択シーン
│   ├── entity/              # エンティティ
│   │   ├── enemy.go        # 敵（ほぼプレイヤーと同じ）
│   │   ├── player.go       # プレイヤー
│   │   └── object.go       # インタラクト可能オブジェクト
│   ├── dialog/              # 会話通信システム
│   │   └── dialog.go
│   ├── asset/               # アセット管理
│   │   └── loader.go
│   └── ui/                  # 共通UI
│       └── component.go
├── assets/                   # ゲームアセット
│   ├── images/
│   ├── fonts/
│   └── data/                # ゲームデータ（JSON/YAML）
├── docs/                     # ドキュメント
├── go.mod
├── go.sum
└── CLAUDE.md
```

## パッケージ責務
| パッケージ             | 責務                               |
|-------------------|----------------------------------|
| `cmd/game`        | ebitengine の起動・依存性の組み立て          |
| `internal/game`   | ゲーム全体のループとステートマシン                |
| `internal/scene`  | 画面単位の Update/Draw を提供する Scene 実装 |
| `internal/entity` | プレイヤー・敵・採掘対象などの振る舞いを持つオブジェクト     |
| `internal/dialog` | 通信ログ・会話イベントの管理                   |
| `internal/asset`  | 画像・フォント・データファイルのロードとキャッシュ        |
| `internal/ui`     | HUD・ボタン・ミニマップなど共通描画部品            |

## ステートマシン
ゲームコアは `GameState` を切り替えるステートマシンとして実装する。各状態は対応する Scene を保持し、入力に応じて遷移する。

### 主要な状態
- `StateBoot`: 初期化・アセットロード
- `StateTitle`: スタート画面（タイトル）
- `StateExploration`: 探索（メインゲームプレイ）
- `StateStation`: 宇宙ステーション内（売買・改造）
- `StateWarpSelect`: ワープ先選択
- `StateDialog`: 会話・通信イベント
- `StateMenu`: ゲーム中ポーズメニュー（直前画面の上にオーバーレイ表示）
- `StateSettings`: 設定画面（タイトル / メニュー双方から到達）
- `StateWorldMap`: 全体マップ画面（探索の上にオーバーレイ表示。最後に入った FullMap を俯瞰）

### 遷移概要
```
Boot ──▶ Title ⇄ Settings
           │
           ▼
        Exploration ⇄ Station
           │  ▲         │
           ▼  │         ▼
         Menu ┘      Dialog
           ⇅
        Settings

Exploration ──▶ WarpSelect ──▶ Exploration（別宙域）
```
- `Menu` は呼び出し元の状態（Exploration / Station）を保持し、閉じると元に戻る
- `Settings` は呼び出し元（Title / Menu）を保持し、`Back` で元の状態へ復帰する
- `Menu` の `Quit To Title` は確認モーダル経由で `Title` へ遷移し、未保存の進行は破棄

## シーンインターフェース
```go
type Scene interface {
    Update() error
    Draw(screen *ebiten.Image)
}
```
シーン切り替えはゲームコア側で行い、Scene 自体は遷移を直接呼び出さず「次の状態を要求する」形に留める。

## テーマシステム
UI は「暗い背景 + 単色ライン」のレトロベクター風デザインで統一する（詳細は SCREENS.md「ビジュアル方針」参照）。配色は単一の `Theme` オブジェクトに集約し、すべての描画コードがこれを参照することで切替を一括化する。

### 構造
```go
// internal/ui/theme.go（責務上 internal/ui に配置。必要に応じて専用パッケージへ分離）
type Theme struct {
    Name       string       // "Black" / "Navy" / "DarkGreen"
    Background color.Color  // 画面塗り色
    Line       color.Color  // ライン・文字・アイコンの基本色
    LineDim    color.Color  // 補助色（グレーアウト・サブ情報用）
}
```
- 実体はプリセット定数として用意し、設定値（テーマ名）から解決する
- 描画側はハードコードされた色を持たず、必ず `Theme` 経由で色を取得する

### プリセット例
| テーマ名 | 背景 | ライン |
| --- | --- | --- |
| Black | 黒 | 緑系 |
| Navy | 紺 | シアン系 |
| DarkGreen | 濃い緑 | 琥珀系 |

具体的な RGB 値は実装時に詰める。プリセットは追加・差し替えが容易な形で持つ。

### 提供と参照
- ゲームコアが現在のテーマを保持し、シーン / UI / エンティティ描画に共有する（描画コンテキストや DI で受け渡す）
- 設定画面でテーマを変更すると即座に反映され、永続化される
- 背景色だけ・ライン色だけといった部分変更は許可しない。常にプリセット単位で切り替える

## 世界モデル
世界の空間構造は `internal/entity` の以下の型で表現する。仕様詳細は GAME_DESIGN.md「宙域構造 / 小惑星の出現」を参照。

| 型 | 役割 |
| --- | --- |
| `World` | 宇宙全体。境界を持たず、複数の `FullMap` を保持する。座標から含む区画を解決する |
| `FullMap` | 矩形の区画。中心座標・半幅/半高と、内部の `ResourceZone` 群を持つ |
| `ResourceZone` | 円形の素材ゾーン。中心・半径・最大生成数・素材重み（`ResourceWeight`）を持つ |

主要 API:
- `World.Containing(x, y)`: 座標を含む `FullMap` を返す（区画外は nil）
- `World.InBounds(x, y)`: いずれかの `FullMap` に含まれるか
- `World.SpawnCap(px, py)`: 現フレームの小惑星同時生成上限。区画内のゾーン群のフォールオフ合算
- `World.PickResource(x, y, rng)`: 重なるゾーンの重みをフォールオフ込みで合算し素材を抽選

探索シーンは毎フレーム `SpawnCap` を確認し、ミニマップ外のリングに新規小惑星を生成する。`FullMap` 外では何も生成されないため、区画間の空虚は構造的に保証される。

## データフロー
- 静的データ（パーツ性能・資源マスタ等）は `assets/data/` の JSON/YAML から読み込む
- 動的状態（プレイヤー所持資源・船構成）はゲームコアの State が保持
- アセットは `internal/asset` を通じて取得し、エンティティや UI から参照する
- テーマ（描画色）は `internal/ui` の `Theme` を全描画コードで共有し、設定画面から切り替える
