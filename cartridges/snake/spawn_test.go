// SPDX-License-Identifier: GPL-2.0-or-later

package main

// freeScan takes its pick as an index rather than drawing RandomU64, so
// these tests walk every candidate on every level.

import "testing"

func resetSpawnTestBoard() {
	field = board{}
}

func TestFreeFieldScanNeverOverlapsWalls(t *testing.T) {
	for lvl := 1; lvl <= numLevels(); lvl++ {
		resetSpawnTestBoard()
		free, _, _ := field.freeScan(lvl, keepClear{}, -1)
		if free <= 0 {
			t.Fatalf("level %d: freeScan reports 0 free cells on an empty board", lvl)
		}
		for pick := 0; pick < free; pick++ {
			_, c, ok := field.freeScan(lvl, keepClear{}, pick)
			if !ok {
				t.Fatalf("level %d: pick %d of %d free cells returned ok=false", lvl, pick, free)
			}
			x, y := c.x, c.y
			if int(x) < fieldMin || int(x) > fieldMax || int(y) < fieldMin || int(y) > fieldMax {
				t.Fatalf("level %d: candidate (%d,%d) is outside the interior bounds [%d,%d]", lvl, x, y, fieldMin, fieldMax)
			}
			if wallAt(lvl, int(x), int(y)) {
				t.Fatalf("level %d: candidate (%d,%d) at pick %d IS a wall cell — this is exactly the "+
					"spawn-embedding bug: a spot freeScan offers as a legal spawn must never be a wall "+
					"in the level it was scanned against", lvl, x, y, pick)
			}
		}
	}
}

func TestFreeFieldScanKeepsClearOfASpriteBox(t *testing.T) {
	for lvl := 1; lvl <= numLevels(); lvl++ {
		resetSpawnTestBoard()
		all, _, _ := field.freeScan(lvl, keepClear{}, -1)
		_, centre, ok := field.freeScan(lvl, keepClear{}, all/2)
		if !ok {
			t.Fatalf("level %d: no free cell to centre the box on", lvl)
		}
		avoid := clearOf(centre)

		inBox := 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				x, y := int(centre.x)+dx, int(centre.y)+dy
				if x >= fieldMin && x <= fieldMax && y >= fieldMin && y <= fieldMax && !wallAt(lvl, x, y) {
					inBox++
				}
			}
		}
		free, _, _ := field.freeScan(lvl, avoid, -1)
		if free != all-inBox {
			t.Fatalf("level %d: %d free cells clear of %+v, want %d of %d", lvl, free, centre, all-inBox, all)
		}
		for pick := 0; pick < free; pick++ {
			_, c, ok := field.freeScan(lvl, avoid, pick)
			if !ok {
				t.Fatalf("level %d: pick %d of %d free cells returned ok=false", lvl, pick, free)
			}
			if avoid.covers(c) || wallAt(lvl, int(c.x), int(c.y)) {
				t.Fatalf("level %d: pick %d = %+v, inside the box around %+v or a wall", lvl, pick, c, centre)
			}
		}
	}
}

// TestFreeFieldScanEveryLevelPairHasCandidatesAfterAdvance: whatever the old
// level allowed, commitLevelSwap's re-scan against the new one must find a
// cell.
func TestFreeFieldScanEveryLevelPairHasCandidatesAfterAdvance(t *testing.T) {
	for from := 1; from <= numLevels(); from++ {
		to := upcomingLevel(from)
		resetSpawnTestBoard()
		free, _, _ := field.freeScan(to, keepClear{}, -1)
		if free <= 0 {
			t.Fatalf("level %d -> %d: no free cells on the new layout for a post-advance respawn", from, to)
		}
	}
}
