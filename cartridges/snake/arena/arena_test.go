// SPDX-License-Identifier: GPL-2.0-or-later

package arena

import (
	"strings"
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/levels"
)

// layout is an open arena with walls wherever wall says, the border always.
func layout(wall func(x, y int) bool) *levels.Level {
	var b strings.Builder
	for y := 0; y < Rows; y++ {
		for x := 0; x < Cols; x++ {
			if x == 0 || y == 0 || x == Cols-1 || y == Rows-1 || wall(x, y) {
				b.WriteByte(Wall)
			} else {
				b.WriteByte(Floor)
			}
		}
	}
	return &levels.Level{Name: "test", W: Cols, H: Rows, Cells: b.String()}
}

func open(x, y int) bool { return false }

func TestWallAtCountsOffGridAsWall(t *testing.T) {
	lv := layout(open)
	for _, c := range [][2]int{{-1, 5}, {5, -1}, {Cols, 5}, {5, Rows}} {
		if !WallAt(lv, c[0], c[1]) {
			t.Errorf("off-grid (%d,%d) is not a wall", c[0], c[1])
		}
	}
	if WallAt(lv, 1, 1) {
		t.Error("(1,1) of an open arena is a wall")
	}
}

func TestSafeStartNeedsClearanceOnBothAxes(t *testing.T) {
	lv := layout(func(x, y int) bool { return x == 32 && y == 32 })
	cases := []struct {
		x, y int
		safe bool
	}{
		{32, 32, false}, // the wall itself
		{32, 30, false}, {32, 29, true},
		{30, 32, false}, {29, 32, true},
		{34, 32, false}, {35, 32, true},
		{31, 31, true}, // diagonals do not count
		{3, 3, true}, {2, 3, false}, {3, 2, false},
		{60, 60, true}, {61, 60, false}, {60, 61, false},
	}
	for _, c := range cases {
		if got := SafeStart(lv, c.x, c.y); got != c.safe {
			t.Errorf("SafeStart(%d,%d) = %v, want %v", c.x, c.y, got, c.safe)
		}
	}
}

func TestValidateCountsTheOpenArenasSafeStarts(t *testing.T) {
	n, err := Validate(layout(open))
	if err != nil {
		t.Fatal(err)
	}
	if n != MaxSafeStarts {
		t.Errorf("open arena has %d safe starts, want all %d", n, MaxSafeStarts)
	}
}

func TestValidateRejectsUnplayableLayouts(t *testing.T) {
	small := layout(open)
	small.W, small.H, small.Cells = 2, 2, "####"

	leaky := layout(open)
	leaky.Cells = "#." + leaky.Cells[2:]

	pocket := layout(func(x, y int) bool {
		return (x == 10 || x == 12) && y >= 10 && y <= 12 || (y == 10 || y == 12) && x >= 10 && x <= 12
	})

	cramped := layout(func(x, y int) bool { return x%3 == 0 && y >= 2 })

	for name, lv := range map[string]*levels.Level{
		"wrong size": small, "open border": leaky, "walled-off floor": pocket, "no safe start": cramped,
	} {
		if _, err := Validate(lv); err == nil {
			t.Errorf("%s: Validate accepted it", name)
		}
	}
}
