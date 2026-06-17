package image

import "testing"

// TestShipBaseDecodes は埋め込みベース船体 PNG がデコードでき、想定サイズであることを確認する。
func TestShipBaseDecodes(t *testing.T) {
	w, h := ShipBaseSize()
	if w != 52 || h != 61 {
		t.Fatalf("ShipBaseSize = (%d, %d); want (52, 61)", w, h)
	}
	if ShipBaseW != w || ShipBaseH != h {
		t.Fatalf("ShipBaseW/H consts = (%d, %d); want (%d, %d)", ShipBaseW, ShipBaseH, w, h)
	}
}

// TestPartCells はパーツシートが 4x4=16 セル（各 16x16）取り出せることを確認する。
func TestPartCells(t *testing.T) {
	for row := 0; row < 4; row++ {
		for col := 0; col < 4; col++ {
			c := Cell(col, row)
			b := c.Bounds()
			if b.Dx() != CellSize || b.Dy() != CellSize {
				t.Fatalf("Cell(%d,%d) size = (%d,%d); want (16,16)", col, row, b.Dx(), b.Dy())
			}
		}
	}
}
