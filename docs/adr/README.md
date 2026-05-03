# Architecture Decision Records

各 ADR は「決定の文脈・採用案・帰結」を簡潔に残す。
実装の細部はコードを正とし、ADR は why と前提条件の記録に徹する。

将来課題や未実装の既知ギャップは [`../KNOWN_GAPS.md`](../KNOWN_GAPS.md) を参照。

| ID | タイトル | 主領域 |
| --- | --- | --- |
| 0001 | [シーン管理と Director](./0001-scene-director.md) | `internal/scene` |
| 0002 | [テーマシステム](./0002-theme.md) | `internal/ui` |
| 0003 | [ワールド階層](./0003-world-hierarchy.md) | `internal/entity` |
| 0004 | [小惑星スポーンモデル](./0004-asteroid-spawn.md) | `internal/scene/exploration` |
| 0005 | [パーツ定義レジストリとデータ分離](./0005-part-defs.md) | `internal/entity` |
| 0006 | [パーツ回転と前後スラスタ](./0006-part-rotation.md) | `internal/entity` |
| 0007 | [弾丸スタイルと瞬間命中レーザー](./0007-bullets-and-lasers.md) | `internal/entity`, `internal/scene` |
| 0008 | [戦闘ループとピックアップ](./0008-combat-loop.md) | `internal/scene/exploration` |
| 0009 | [海賊 NPC](./0009-pirates.md) | `internal/entity`, `internal/scene` |
| 0010 | [酒場クエスト](./0010-tavern-quests.md) | `internal/entity`, `internal/scene` |
| 0011 | [ダイアログ基盤](./0011-dialog.md) | `internal/dialog`, `internal/scene` |
| 0012 | [恒星マップとワープ](./0012-warp.md) | `internal/scene` |
| 0013 | [セーブシステム](./0013-save.md) | `internal/save` |
| 0014 | [HUD とパララックス](./0014-hud-parallax.md) | `internal/scene/exploration` |

## テンプレ

```
# NNNN. タイトル

## Status
Accepted | Superseded by NNNN

## Context
意思決定が必要になった背景・前提・制約。

## Decision
採用した方式と、その実装上のキーとなる仕組み。

## Consequences
得られたもの／犠牲にしたもの／後続に影響する条件。
```
