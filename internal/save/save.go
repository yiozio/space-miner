package save

// セーブ・ロード機能。3 スロットの単純なファイルベース実装。
// 動的状態のうち Player のみを保存する。小惑星・ピックアップなどはゾーン定義から
// 再生成される一時状態のため保存しない。
// メタ情報（保存時刻・累計プレイ時間・所持金・宙域名）はスロット選択 UI で表示する。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yiozio/space-miner/internal/entity"
)

const (
	appDirName  = "space-miner"
	saveVersion = 1

	// SlotCount はセーブスロット数（1..SlotCount）。
	SlotCount = 3
)

// State は保存ファイルのトップレベル構造。
type State struct {
	Version int         `json:"version"`
	Meta    Meta        `json:"meta"`
	Player  PlayerState `json:"player"`
}

// Meta はスロット選択 UI で表示する補助情報。
// SavedAt は ISO 8601、Playtime は秒。MapName は当該セーブ時点の宙域（FullMap）名。
type Meta struct {
	SavedAt  time.Time `json:"saved_at"`
	Playtime float64   `json:"playtime"`
	Credits  int       `json:"credits"`
	MapName  string    `json:"map_name"`
}

// PlayerState は Player の永続化対象フィールド。
// 派生ステータス（MaxHP / MaxFuel / MaxCargo 等）はパーツから再計算するため保存しない。
type PlayerState struct {
	Parts           []PartState     `json:"parts"`
	X               float64         `json:"x"`
	Y               float64         `json:"y"`
	Angle           float64         `json:"angle"`
	HP              int             `json:"hp"`
	ShieldHP        int             `json:"shield_hp"`
	Fuel            float64         `json:"fuel"`
	Credits         int             `json:"credits"`
	Inventory       map[int]int     `json:"inventory"`        // ResourceType -> qty
	PartsInventory  map[int]int     `json:"parts_inventory"`  // PartID -> qty
	VisitedStations map[string]bool `json:"visited_stations"` // 初回ダイアログ済みステーション名
}

// PartState は配置済みパーツ 1 つの保存形式。
type PartState struct {
	DefID int `json:"def_id"`
	GX    int `json:"gx"`
	GY    int `json:"gy"`
}

// Context は Save が呼び出し側から受け取る追加情報。
type Context struct {
	Player   *entity.Player
	Playtime float64
	MapName  string
}

// LoadResult は Load が返す復元結果。
type LoadResult struct {
	Player   *entity.Player
	Playtime float64
}

// SlotPath はスロット番号 (1..SlotCount) のセーブファイル絶対パスを返す。
func SlotPath(slot int) (string, error) {
	if !validSlot(slot) {
		return "", fmt.Errorf("invalid save slot: %d", slot)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName, fmt.Sprintf("save_%d.json", slot)), nil
}

func validSlot(slot int) bool { return slot >= 1 && slot <= SlotCount }

// SlotExists は指定スロットにセーブが存在するか返す。
func SlotExists(slot int) bool {
	p, err := SlotPath(slot)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// AnyExists はいずれかのスロットにセーブが存在するか返す。
func AnyExists() bool {
	for i := 1; i <= SlotCount; i++ {
		if SlotExists(i) {
			return true
		}
	}
	return false
}

// LoadMeta はスロットのメタ情報のみを返す。スロットが空の場合 nil。
func LoadMeta(slot int) (*Meta, error) {
	if !SlotExists(slot) {
		return nil, nil
	}
	st, err := readState(slot)
	if err != nil {
		return nil, err
	}
	m := st.Meta
	return &m, nil
}

// LatestSlot は SavedAt が最も新しいスロット番号を返す。空のときは 0。
func LatestSlot() int {
	best := 0
	var bestT time.Time
	for i := 1; i <= SlotCount; i++ {
		m, err := LoadMeta(i)
		if err != nil || m == nil {
			continue
		}
		if best == 0 || m.SavedAt.After(bestT) {
			best = i
			bestT = m.SavedAt
		}
	}
	return best
}

// Save は ctx の状態を指定スロットに書き出す。
func Save(slot int, ctx Context) error {
	if !validSlot(slot) {
		return fmt.Errorf("invalid save slot: %d", slot)
	}
	if ctx.Player == nil {
		return fmt.Errorf("save: player is nil")
	}
	p, err := SlotPath(slot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	state := State{
		Version: saveVersion,
		Meta: Meta{
			SavedAt:  time.Now(),
			Playtime: ctx.Playtime,
			Credits:  ctx.Player.Credits,
			MapName:  ctx.MapName,
		},
		Player: makePlayerState(ctx.Player),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	// 一時ファイル経由で原子的に書き換え（途中でクラッシュしても旧データが残る）
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Load は指定スロットから Player を復元する。
// 派生ステータスはパーツ構成から再計算し、現在値（HP/Shield/Fuel）を上限内にクランプする。
// 速度は 0 にして読み込む（ロード直後は静止状態）。
func Load(slot int) (*LoadResult, error) {
	if !validSlot(slot) {
		return nil, fmt.Errorf("invalid save slot: %d", slot)
	}
	st, err := readState(slot)
	if err != nil {
		return nil, err
	}
	return &LoadResult{
		Player:   restorePlayer(st.Player),
		Playtime: st.Meta.Playtime,
	}, nil
}

func readState(slot int) (*State, error) {
	p, err := SlotPath(slot)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

func makePlayerState(player *entity.Player) PlayerState {
	parts := make([]PartState, len(player.Parts))
	for i, pt := range player.Parts {
		parts[i] = PartState{DefID: int(pt.DefID), GX: pt.GX, GY: pt.GY}
	}
	inv := make(map[int]int, len(player.Inventory))
	for k, v := range player.Inventory {
		if v > 0 {
			inv[int(k)] = v
		}
	}
	pinv := make(map[int]int, len(player.PartsInventory))
	for k, v := range player.PartsInventory {
		if v > 0 {
			pinv[int(k)] = v
		}
	}
	visited := make(map[string]bool, len(player.VisitedStations))
	for k, v := range player.VisitedStations {
		if v {
			visited[k] = true
		}
	}
	return PlayerState{
		Parts:           parts,
		X:               player.X,
		Y:               player.Y,
		Angle:           player.Angle,
		HP:              player.HP,
		ShieldHP:        player.ShieldHP,
		Fuel:            player.Fuel,
		Credits:         player.Credits,
		Inventory:       inv,
		PartsInventory:  pinv,
		VisitedStations: visited,
	}
}

func restorePlayer(ps PlayerState) *entity.Player {
	parts := make([]entity.Part, len(ps.Parts))
	for i, pt := range ps.Parts {
		parts[i] = entity.Part{
			DefID: entity.PartID(pt.DefID),
			GX:    pt.GX,
			GY:    pt.GY,
		}
	}
	inv := make(map[entity.ResourceType]int, len(ps.Inventory))
	for k, v := range ps.Inventory {
		inv[entity.ResourceType(k)] = v
	}
	pinv := make(map[entity.PartID]int, len(ps.PartsInventory))
	for k, v := range ps.PartsInventory {
		pinv[entity.PartID(k)] = v
	}
	visited := make(map[string]bool, len(ps.VisitedStations))
	for k, v := range ps.VisitedStations {
		if v {
			visited[k] = true
		}
	}
	p := &entity.Player{
		Ship: entity.Ship{
			Parts: parts,
			X:     ps.X,
			Y:     ps.Y,
			Angle: ps.Angle,
		},
		Credits:         ps.Credits,
		Inventory:       inv,
		PartsInventory:  pinv,
		VisitedStations: visited,
	}
	p.OnPartsChanged()
	p.HP = clampInt(ps.HP, 0, p.MaxHP)
	p.ShieldHP = clampInt(ps.ShieldHP, 0, p.MaxShieldHP)
	p.Fuel = clampFloat(ps.Fuel, 0, p.MaxFuel)
	return p
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
