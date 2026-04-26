package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/yiozio/space-miner/internal/scene"
	"github.com/yiozio/space-miner/internal/ui"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

// Game はゲーム全体のループとシーンスタックを管理する。
// scene.Director を実装し、シーン側からスタック操作・テーマ参照を行う。
type Game struct {
	stack       []scene.Scene
	theme       *ui.Theme
	preferences *ui.Preferences
	quit        bool
}

// New はゲームを初期化する。設定を読み、初期シーンとしてタイトルを積む。
func New() *Game {
	prefs := ui.LoadPreferences()
	g := &Game{
		theme:       ui.ThemeByName(prefs.ThemeName),
		preferences: prefs,
	}
	g.Push(scene.NewTitle())
	return g
}

// scene.Director の実装

func (g *Game) Theme() *ui.Theme { return g.theme }

func (g *Game) SetTheme(t *ui.Theme) {
	g.theme = t
	g.preferences.ThemeName = t.Name
	// 設定の永続化失敗はユーザー操作の妨げにならないよう握りつぶす
	_ = ui.SavePreferences(g.preferences)
}

func (g *Game) Push(s scene.Scene) {
	g.stack = append(g.stack, s)
}

func (g *Game) Pop() {
	if len(g.stack) > 0 {
		g.stack = g.stack[:len(g.stack)-1]
	}
}

func (g *Game) Replace(s scene.Scene) {
	if len(g.stack) > 0 {
		g.stack[len(g.stack)-1] = s
		return
	}
	g.stack = append(g.stack, s)
}

func (g *Game) Quit() { g.quit = true }

// ebiten.Game の実装

func (g *Game) Update() error {
	if g.quit {
		return ebiten.Termination
	}
	if len(g.stack) == 0 {
		return nil
	}
	return g.stack[len(g.stack)-1].Update(g)
}

func (g *Game) Draw(screen *ebiten.Image) {
	for _, s := range g.stack {
		s.Draw(screen, g)
	}
}

func (g *Game) Layout(int, int) (int, int) {
	return screenWidth, screenHeight
}
