// Package font はゲーム全体のテキスト描画に使う TTF フォント
// （Noto Sans JP Regular）を埋め込みで提供する。
// フォントは SIL Open Font License 1.1 で配布されている。
// ライセンス全文は同梱の OFL.txt を参照（フォントと共に再配布すること）。
package font

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed NotoSansJP-Regular.ttf
var notoSansJPRegular []byte

var (
	once   sync.Once
	source *text.GoTextFaceSource
)

// Source は埋め込みフォントの GoTextFaceSource を返す（初回呼び出しでパース）。
func Source() *text.GoTextFaceSource {
	once.Do(func() {
		s, err := text.NewGoTextFaceSource(bytes.NewReader(notoSansJPRegular))
		if err != nil {
			// 埋め込みフォントが壊れている場合のみ起こり得る（ビルド時に確定する）
			panic("font: embedded Noto Sans JP のパースに失敗: " + err.Error())
		}
		source = s
	})
	return source
}
