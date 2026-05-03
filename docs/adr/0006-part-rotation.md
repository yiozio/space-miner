# 0006. パーツ回転と前後スラスタ

## Status
Accepted

## Context
- スラスタの向きを変えて後方推進・横推進を構成できるようにしたい。
- 一方で、前方の最高速度は順方向スラスタ、後方の最高速度は逆方向スラスタで
  別々に決まることをはっきりさせたい（横向きスラスタは無効）。
- ガンも回転に応じた方向に弾が出るようにしたい。

## Decision

### Part に回転を持たせる
- `Part.Rotation int` を 0..3（90° 単位の時計回り）として追加。
- エディタで `R` キーが回転を制御：
  - カーソル上に既設パーツがあればそれを 90° 回転（Cockpit を除く）。
  - カーソル上が空なら「ブラシ回転」を循環し、次回配置時に適用。
- 描画は `DrawPart(..., rotation int)` が一時画像経由で 90° 単位の回転 blit を行う。
  Ship/Pirate の船体画像生成 (`ensureImage`) もこの API を使う。

### スラスタ方向分類
- `Part.ThrustDir()` が `Forward (R=0) / Sideways (R=1,3) / Backward (R=2)` を返す。
- `Player.thrusterStatsByDir()` が前向き集計と後ろ向き集計を別々に返す。
  Sideways は推進・燃料消費・炎いずれにも寄与しない（無意味）。
- `W / ArrowUp` で前向き集計を使用、`S / ArrowDown` で後ろ向き集計を使用。
- 速度上限はベクトル全体ではなく**機軸成分**でクランプ：
  - 前向き成分 (`V·forward`) を `fwdCap` で、後ろ向き成分の絶対値を `bckCap` で制限。
  - 横（接線）成分はクランプしない（衝突や慣性の横成分は保持）。
- 動的キャップ `fwdCap` / `bckCap` はブースト中に瞬時上昇、解除後は線形減衰
  （`updateCap` ヘルパで方向別に処理）。
- Thruster が 0 のときに限り `Cockpit` の最低限スラスタ性能を**前向きにのみ**フォールバック。

### 炎描画の追従
- `Ship.ThrustActiveDir` フィールドを Player が毎フレーム設定。
- `Ship.thrustEmitters()` が現在燃焼している方向のスラスタのみ返す。
- `drawAfterburners` は各パーツの Rotation から「後端中心」「後方ベクトル」「炎幅方向」を
  90° 単位回転で導出し、回転に追従した炎を描く。

### ガンの射撃方向
- `Player.Shoot` / `Pirate.shoot` は各 Gun の Rotation から前方単位ベクトルを 90° CW 回転で求める：
  `R=0:(0,-1)` `R=1:(-1,0)` `R=2:(0,1)` `R=3:(1,0)`（パーツローカル系）。
- これを船体ローカル → ワールド変換 (`R(angle + π/2)`) して発射位置・速度・レーザー方向に使う。

## Consequences

- 後ろ撃ちガン・側面ガン・後方ブースタなど、機体構成の自由度が大きく向上する。
- 横向きスラスタは「無意味」が明示的なルール（無駄な選択を防ぐ）。
- ブースト中に方向を切り替える挙動は、`fwdCap` / `bckCap` の独立減衰でなめらかに表現される。
- 旧セーブには `Rotation` が無いが、JSON `omitempty` 経由で `0` がデフォルト復元され
  互換性が保たれる。

## 関連
- `internal/entity/part.go`, `player.go`, `ship.go`, `pirate.go`
- `internal/scene/station_editor.go`
