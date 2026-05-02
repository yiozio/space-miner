package game

// StateID はステートマシンの状態識別子。
// 実装上はシーンスタック（scene パッケージ）として表現されるが、
// 設計上の状態列挙としてここに定義する。
type StateID int

const (
	StateBoot StateID = iota
	StateTitle
	StateExploration
	StateStation
	StateWarpSelect
	StateDialog
	StateMenu
	StateSettings
	StateWorldMap
)
