// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"arkade-snake-cartridge/arena"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
)

// TestPackFitsTheArena is where the pack's layout rules live; genlevelpack
// only parses, so this holds the compiled pack to arena.Validate.
func TestPackFitsTheArena(t *testing.T) {
	if numLevels() != 6 {
		t.Errorf("the campaign has %d layouts, want 6", numLevels())
	}
	if got := wallsPack.Legend.Runes; got != string([]byte{arena.Floor, arena.Wall}) {
		t.Errorf("legend is %q, want floor then wall", got)
	}
	var dst [arena.MaxSafeStarts]cell
	for lvl := 1; lvl <= numLevels(); lvl++ {
		want, err := arena.Validate(layout(lvl))
		if err != nil {
			t.Errorf("level %d %s: %v", lvl, layout(lvl).Name, err)
			continue
		}
		if got := computeSafeStarts(lvl, &dst); got != want {
			t.Errorf("level %d: the cartridge keeps %d safe starts, the arena rules allow %d", lvl, got, want)
		}
		if want < 100 {
			t.Errorf("level %d: only %d safe starts — suspiciously few for a level layout", lvl, want)
		}
	}
}

func TestUpcomingLevelWraps(t *testing.T) {
	want := map[int]int{1: 2, 2: 3, 3: 4, 4: 5, 5: 6, 6: 1}
	for cur, next := range want {
		if got := upcomingLevel(cur); got != next {
			t.Errorf("upcomingLevel(%d) = %d, want %d", cur, got, next)
		}
	}
}

// TestTheFirstTwoLayouts pins the walls the other tests place things
// against: level 1 is border-only, level 2 adds levelTwoBar.
func TestTheFirstTwoLayouts(t *testing.T) {
	for y := 0; y < arena.Rows; y++ {
		for x := 0; x < arena.Cols; x++ {
			border := x == 0 || y == 0 || x == arena.Cols-1 || y == arena.Rows-1
			if wallAt(1, x, y) != border {
				t.Fatalf("level 1: wallAt(%d,%d) = %v, want %v", x, y, !border, border)
			}
			if wallAt(2, x, y) != (border || levelTwoBar(x, y)) {
				t.Fatalf("level 2: wallAt(%d,%d) = %v, want %v", x, y, !(border || levelTwoBar(x, y)), border || levelTwoBar(x, y))
			}
		}
	}
}

func levelTwoBar(x, y int) bool { return x >= 19 && x <= 44 && (y == 31 || y == 32) }

// spawnIdx is the RandomU64 value that spawns a snake on (x, y) of level 1,
// whose safe starts are the [3,60] square in row-major order.
func spawnIdx(x, y int) uint64 { return uint64((y-3)*58 + (x - 3)) }

// foodIdx is the RandomU64 value that places an apple or pickup on (x, y)
// of level 1 when nothing earlier in the row-major scan is taken. Each
// taken cell before (x, y) lowers it by one.
func foodIdx(x, y int) uint64 { return uint64((y-1)*62 + (x - 1)) }

// TestComputeSafeStartsLevel1 pins the row-major order spawnIdx indexes.
func TestComputeSafeStartsLevel1(t *testing.T) {
	var dst [arena.MaxSafeStarts]cell
	if n := computeSafeStarts(1, &dst); n != arena.MaxSafeStarts {
		t.Fatalf("level 1 safe starts = %d, want %d", n, arena.MaxSafeStarts)
	}
	for _, c := range []cell{{3, 3}, {60, 3}, {32, 32}, {3, 60}, {60, 60}} {
		if got := dst[spawnIdx(int(c.x), int(c.y))]; got != c {
			t.Errorf("safe start at spawnIdx%v = %+v", c, got)
		}
	}
}

// TestFoodIdxIndexesAnEmptyBoard pins foodIdx against the scan placeFood
// draws from.
func TestFoodIdxIndexesAnEmptyBoard(t *testing.T) {
	var b board
	for _, c := range []cell{{1, 1}, {62, 1}, {32, 31}, {1, 62}, {62, 62}} {
		if _, got, ok := b.freeScan(1, keepClear{}, int(foodIdx(int(c.x), int(c.y)))); !ok || got != c {
			t.Errorf("free cell at foodIdx%v = %+v (ok %v)", c, got, ok)
		}
	}
}

func TestSpawnDirectionFacesTheCentre(t *testing.T) {
	cases := []struct {
		x, y int
		want cartlib.Direction
	}{
		{10, 32, cartlib.DirRight}, // far left -> head right
		{54, 32, cartlib.DirLeft},  // far right -> head left
		{32, 10, cartlib.DirDown},  // far up -> head down
		{32, 54, cartlib.DirUp},    // far down -> head up
		{32, 32, cartlib.DirUp},    // dead center: |dx| > |dy| false, y < 32 false -> up
		{30, 30, cartlib.DirDown},  // tie |dx| == |dy|: vertical wins (strict >), y < 32 -> down
		{3, 4, cartlib.DirRight},   // |29| > |28| -> horizontal, x < 32 -> right
		{4, 3, cartlib.DirDown},    // |28| < |29| -> vertical, y < 32 -> down
	}
	for _, c := range cases {
		if got := spawnDirection(c.x, c.y); got != c.want {
			t.Errorf("spawnDirection(%d,%d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestRingOverlapsLevel(t *testing.T) {
	ring := make([]cell, 8)

	ring[0] = cell{x: 10, y: 10}
	ring[1] = cell{x: 11, y: 10}
	if ringOverlapsLevel(2, ring, 0, 2) {
		t.Error("snake at (10..11,10) does not overlap level 2's walls")
	}

	ring[1] = cell{x: 32, y: 31}
	if !ringOverlapsLevel(2, ring, 0, 2) {
		t.Error("snake with a cell at (32,31) overlaps level 2's bar")
	}

	// Tail near the end of the ring, head wrapped to index 0.
	ring[6] = cell{x: 20, y: 20}
	ring[7] = cell{x: 21, y: 20}
	ring[0] = cell{x: 32, y: 32} // level 2 bar cell, reached via wrap
	if !ringOverlapsLevel(2, ring, 6, 3) {
		t.Error("wrapped ring with a bar cell must report overlap")
	}
	ring[0] = cell{x: 22, y: 20}
	if ringOverlapsLevel(2, ring, 6, 3) {
		t.Error("wrapped ring fully clear of walls must not report overlap")
	}

	// Level 1 is border-only, so interior cells never block the wrap to it.
	ring[0] = cell{x: 32, y: 31}
	if ringOverlapsLevel(1, ring, 0, 1) {
		t.Error("(32,31) is open on level 1")
	}
}
