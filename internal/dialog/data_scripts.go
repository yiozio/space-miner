package dialog

import (
	"strconv"

	"github.com/yiozio/space-miner/internal/i18n"
)

// data_scripts.go は各シーンで再生するスクリプトを構築する。
// 表示テキストはすべて i18n パッケージから引いてくる。スクリプトの構造（ノード接続）は
// ここで組み立てるが、文字列リテラルは持たない。

// Opening はゲーム開始時に再生する導入スクリプト。
// 話者なし（ナレーション）で全画面背景の上に表示される。
func Opening() *Script {
	return scriptFromLines("", i18n.S().Dialog.Opening)
}

// ScriptForStation はステーション名（FullMap 名）に対応する初回スクリプトを返す。
// 未定義のステーションでは nil を返す（=会話なし）。話者は固定 (各ステーションごと)。
func ScriptForStation(name string) *Script {
	speaker := stationSpeaker(name)
	lines := i18n.S().Dialog.StationFirstVisit[name]
	if len(lines) == 0 {
		return nil
	}
	return scriptFromLines(speaker, lines)
}

// stationSpeaker はステーションごとの話者キャラクター ID を返す。
func stationSpeaker(stationName string) string {
	switch stationName {
	case "Aurora":
		return "dock_master_aurora"
	case "Tinker":
		return "engineer_tinker"
	case "Helix":
		return "foreman_helix"
	}
	return ""
}

// scriptFromLines は連番ノードとして lines を Script に組み立てる。
// ノードキーは "n0", "n1", ... を使う。最後のノードのみ Next 空 (会話終了)。
func scriptFromLines(speaker string, lines []string) *Script {
	if len(lines) == 0 {
		return nil
	}
	nodes := make(map[string]Node, len(lines))
	for i, line := range lines {
		key := "n" + strconv.Itoa(i)
		next := ""
		if i < len(lines)-1 {
			next = "n" + strconv.Itoa(i+1)
		}
		nodes[key] = Node{
			Speaker: speaker,
			Text:    line,
			Next:    next,
		}
	}
	return &Script{
		Start: "n0",
		Nodes: nodes,
	}
}
