package scene

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yiozio/space-miner/internal/ui"
)

// Director はシーンからゲームコアを操作するためのインターフェース。
// シーンスタックの操作とテーマの取得・更新を提供する。
// scene パッケージから game パッケージへの逆参照を避けるための抽象。
type Director interface {
	Theme() *ui.Theme
	SetTheme(*ui.Theme)
	Push(Scene)
	Pop()
	Replace(Scene)
	Quit()
}

// Scene は1画面分の更新と描画を担う。
// オーバーレイ表示するシーン（メニュー画面など）はスタックに積まれ、
// 下位シーンも続けて Draw されるが、Update は最上位のみ呼ばれる。
type Scene interface {
	Update(d Director) error
	Draw(dst *ebiten.Image, d Director)
}
