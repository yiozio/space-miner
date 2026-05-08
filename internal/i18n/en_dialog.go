package i18n

// newENDialog は英語のダイアログ行を返す。
func newENDialog() DialogStrings {
	return DialogStrings{
		Opening: []string{
			"The Sol-α system. A quiet corner of the frontier, dotted with mining stations and dust.",
			"You inherited a battered cockpit and a popgun from a relative who never came back from a haul.",
			"Aurora Station has cleared you for departure. Mine, trade, upgrade. The system is yours to explore.",
		},
		StationFirstVisit: map[string][]string{
			"Aurora": {
				"Welcome to Aurora Station, miner. First time docking, eh?",
				"Anything you want to know before heading out?",
				"Iron ore in the inner belts, ice on Tinker, bronze deposits out at Helix. Watch your cargo limit.",
				"Sell ore, buy parts, drag them into your hull from the editor. Standard freelancer kit.",
				"Fly safe out there. The void doesn't forgive sloppy pilots.",
			},
			"Tinker": {
				"Tinker outpost. Mostly ice, mostly cold, mostly quiet. We like it that way.",
				"If you came for water, you came to the right rock. Don't bother haggling, the price is the price.",
			},
			"Helix": {
				"Helix Industrial Dock. You're a long way from Aurora, friend.",
				"Bronze fields here can pay better than iron, but the rocks bite back. Keep your shields up.",
			},
		},
		Characters: map[string]string{
			"dock_master_aurora": "Dockmaster Vex",
			"engineer_tinker":    "Engineer Solis",
			"foreman_helix":      "Foreman Halberg",
			"comms":              "Comms",
		},
		OpeningHeader:  "PROLOGUE",
		HintNext:       "[ Enter / Space: Next   Esc: Skip ]",
		HintChoiceMove: "[ Up/Down: Move   Enter: Select ]",
	}
}
