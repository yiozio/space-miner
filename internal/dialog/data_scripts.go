package dialog

// data_scripts.go は各シーンで再生するスクリプトの定義集。
// 新しいスクリプトを追加するときはこのファイルにエントリを追加する。

// Opening はゲーム開始時に再生する導入スクリプト。
// 話者なし（ナレーション）で全画面背景の上に表示される想定。
var Opening = Script{
	Start: "n1",
	Nodes: map[string]Node{
		"n1": {
			Text: "The Sol-α system. A quiet corner of the frontier, dotted with mining stations and dust.",
			Next: "n2",
		},
		"n2": {
			Text: "You inherited a battered cockpit and a popgun from a relative who never came back from a haul.",
			Next: "n3",
		},
		"n3": {
			Text: "Aurora Station has cleared you for departure. Mine, trade, upgrade. The system is yours to explore.",
		},
	},
}

// AuroraIntro は Aurora Station 初回入船時のスクリプト。選択肢の例も含む。
var AuroraIntro = Script{
	Start: "greet",
	Nodes: map[string]Node{
		"greet": {
			Speaker: "dock_master_aurora",
			Text:    "Welcome to Aurora Station, miner. First time docking, eh?",
			Next:    "ask",
		},
		"ask": {
			Speaker: "dock_master_aurora",
			Text:    "Anything you want to know before heading out?",
			Choices: []Choice{
				{Label: "What's nearby?", Next: "nearby"},
				{Label: "How does the shop work?", Next: "shop"},
				{Label: "Just point me to the airlock.", Next: "bye"},
			},
		},
		"nearby": {
			Speaker: "dock_master_aurora",
			Text:    "Iron ore in the inner belts, ice on Tinker, bronze deposits out at Helix. Watch your cargo limit.",
			Next:    "ask",
		},
		"shop": {
			Speaker: "dock_master_aurora",
			Text:    "Sell ore, buy parts, drag them into your hull from the editor. Standard freelancer kit.",
			Next:    "ask",
		},
		"bye": {
			Speaker: "dock_master_aurora",
			Text:    "Fly safe out there. The void doesn't forgive sloppy pilots.",
		},
	},
}

// TinkerIntro は Tinker (Aurora の衛星) 初回入船時のスクリプト。
var TinkerIntro = Script{
	Start: "n1",
	Nodes: map[string]Node{
		"n1": {
			Speaker: "engineer_tinker",
			Text:    "Tinker outpost. Mostly ice, mostly cold, mostly quiet. We like it that way.",
			Next:    "n2",
		},
		"n2": {
			Speaker: "engineer_tinker",
			Text:    "If you came for water, you came to the right rock. Don't bother haggling, the price is the price.",
		},
	},
}

// HelixIntro は Helix (遠方の惑星) 初回入船時のスクリプト。
var HelixIntro = Script{
	Start: "n1",
	Nodes: map[string]Node{
		"n1": {
			Speaker: "foreman_helix",
			Text:    "Helix Industrial Dock. You're a long way from Aurora, friend.",
			Next:    "n2",
		},
		"n2": {
			Speaker: "foreman_helix",
			Text:    "Bronze fields here can pay better than iron, but the rocks bite back. Keep your shields up.",
		},
	},
}

// ScriptForStation はステーション名（FullMap 名）に対応する初回スクリプトを返す。
// 未定義のステーションでは nil を返す（=会話なし）。
func ScriptForStation(name string) *Script {
	switch name {
	case "Aurora":
		return &AuroraIntro
	case "Tinker":
		return &TinkerIntro
	case "Helix":
		return &HelixIntro
	}
	return nil
}
