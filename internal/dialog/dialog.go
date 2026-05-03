package dialog

// 会話ダイアログの共通型。
// 個別のキャラクター・スクリプトのデータは data_*.go を参照。

// Choice はノードに付随する選択肢。
// Next は次に進むノード ID。空文字なら会話終了。
type Choice struct {
	Label string
	Next  string
}

// Node はダイアログ 1 単位（1 つのテキスト表示）。
// Speaker は CharacterByID で解決される ID。空文字ならアバター無しのナレーション扱い。
// Choices が空の場合は Next で次ノードへ進む。Next も空なら会話終了。
type Node struct {
	Speaker string
	Text    string
	Next    string
	Choices []Choice
}

// Script は会話全体。Start から開始し、Nodes をたどる。
type Script struct {
	Start string
	Nodes map[string]Node
}

// Node は安全にノードを取得する。未登録の id なら ok=false。
func (s *Script) Node(id string) (Node, bool) {
	if s == nil {
		return Node{}, false
	}
	n, ok := s.Nodes[id]
	return n, ok
}
