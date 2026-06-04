package logo

import (
	"regexp"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

// svgpath.go は SVG の path データ（d 属性）を ebiten の vector.Path へ変換する
// 最小限のパーサ。本ロゴで使われるコマンド（M/m L/l H/h V/v C/c Z/z）のみ対応する。

// translate はグループの transform="translate(tx,ty)" 補正。
// 全座標にこのオフセットを足して viewBox 座標へ合わせる。
const (
	translateX = -1.1119846
	translateY = -0.96242
)

var pathDRe = regexp.MustCompile(`(?s)\sd="([^"]*)"`)

// extractPathD は SVG 文字列から最初の path の d 属性値を取り出す。
func extractPathD(svg []byte) string {
	m := pathDRe.FindSubmatch(svg)
	if m == nil {
		return ""
	}
	return string(m[1])
}

// parsePath は d データを解釈して p に描き込む。座標には translate 補正を加える。
func parsePath(d string, p *vector.Path) {
	const tx, ty = float32(translateX), float32(translateY)
	t := &scanner{s: d}
	var cx, cy float32 // 現在点（ローカル座標）
	var sx, sy float32 // サブパス開始点（ローカル座標）
	cmd := byte(0)
	for {
		t.skipSep()
		if t.eof() {
			break
		}
		if isCmd(t.peek()) {
			cmd = t.next()
		} else if cmd == 0 {
			break
		}
		switch cmd {
		case 'M', 'm':
			x, y := t.num(), t.num()
			if cmd == 'm' {
				x, y = x+cx, y+cy
			}
			cx, cy, sx, sy = x, y, x, y
			p.MoveTo(cx+tx, cy+ty)
			// 後続のペアは暗黙的に lineto 扱い（SVG 仕様）。
			if cmd == 'm' {
				cmd = 'l'
			} else {
				cmd = 'L'
			}
		case 'L', 'l':
			x, y := t.num(), t.num()
			if cmd == 'l' {
				x, y = x+cx, y+cy
			}
			cx, cy = x, y
			p.LineTo(cx+tx, cy+ty)
		case 'H', 'h':
			x := t.num()
			if cmd == 'h' {
				x += cx
			}
			cx = x
			p.LineTo(cx+tx, cy+ty)
		case 'V', 'v':
			y := t.num()
			if cmd == 'v' {
				y += cy
			}
			cy = y
			p.LineTo(cx+tx, cy+ty)
		case 'C', 'c':
			x1, y1 := t.num(), t.num()
			x2, y2 := t.num(), t.num()
			x, y := t.num(), t.num()
			if cmd == 'c' {
				x1, y1 = x1+cx, y1+cy
				x2, y2 = x2+cx, y2+cy
				x, y = x+cx, y+cy
			}
			p.CubicTo(x1+tx, y1+ty, x2+tx, y2+ty, x+tx, y+ty)
			cx, cy = x, y
		case 'Z', 'z':
			p.Close()
			cx, cy = sx, sy
		default:
			return // 未対応コマンドが来たら打ち切る
		}
	}
}

// scanner は path データの簡易トークナイザ。
type scanner struct {
	s string
	i int
}

func (t *scanner) eof() bool  { return t.i >= len(t.s) }
func (t *scanner) peek() byte { return t.s[t.i] }
func (t *scanner) next() byte { c := t.s[t.i]; t.i++; return c }

// skipSep は区切り（空白・カンマ）を読み飛ばす。
func (t *scanner) skipSep() {
	for t.i < len(t.s) {
		switch t.s[t.i] {
		case ' ', ',', '\n', '\r', '\t':
			t.i++
		default:
			return
		}
	}
}

// num は次の数値を読む。符号・小数・指数表記に対応する。
func (t *scanner) num() float32 {
	t.skipSep()
	start := t.i
	if t.i < len(t.s) && (t.s[t.i] == '+' || t.s[t.i] == '-') {
		t.i++
	}
	for t.i < len(t.s) {
		c := t.s[t.i]
		switch {
		case c >= '0' && c <= '9', c == '.':
			t.i++
		case c == 'e' || c == 'E':
			t.i++
			if t.i < len(t.s) && (t.s[t.i] == '+' || t.s[t.i] == '-') {
				t.i++
			}
		default:
			f, _ := strconv.ParseFloat(t.s[start:t.i], 32)
			return float32(f)
		}
	}
	f, _ := strconv.ParseFloat(t.s[start:t.i], 32)
	return float32(f)
}

// isCmd は SVG path のコマンド文字か判定する。
func isCmd(c byte) bool {
	switch c {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c', 'S', 's', 'Q', 'q', 'T', 't', 'A', 'a', 'Z', 'z':
		return true
	}
	return false
}
