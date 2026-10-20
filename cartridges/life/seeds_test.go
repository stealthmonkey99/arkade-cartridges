// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"arkade-life-cartridge/pattern"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
	"github.com/stealthmonkey99/arkade/sdk/levels"
)

// flipReference is the allocating copy-and-reverse oracle stampFlipped must
// agree with. It stays in the test file because every call would leak under
// -gc=leaking.
func flipReference(art []string, horiz, vert bool) []string {
	out := make([]string, len(art))
	copy(out, art)
	if vert {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	if horiz {
		for i, row := range out {
			b := []byte(row)
			for a, z := 0, len(b)-1; a < z; a, z = a+1, z-1 {
				b[a], b[z] = b[z], b[a]
			}
			out[i] = string(b)
		}
	}
	return out
}

// flipCases covers the art fleetStamp mirrors (glider, LWSS) plus the widest
// shape, the pentadecathlon.
var flipCases = []struct {
	name string
	art  *levels.Level
}{
	{"glider", art(patGlider)},
	{"lwss", art(patLWSS)},
	{"pentadecathlon", art(patPentadecathlon)},
}

// asRows renders a pattern as the []string flipReference wants.
func asRows(p *levels.Level) []string {
	out := make([]string, p.H)
	for y := range out {
		out[y] = p.Row(y)
	}
	return out
}

// stampRows is stamp typed to []string, to draw flipReference's output.
func stampRows(g *grid, rows []string, x, y int) {
	for dy, row := range rows {
		for dx := 0; dx < len(row); dx++ {
			if row[dx] != pattern.Live {
				continue
			}
			cx, cy := x+dx, y+dy
			if cx < 0 || cy < 0 || cx >= cartlib.FBWidth || cy >= cartlib.FBHeight {
				continue
			}
			g[cy][cx] = true
		}
	}
}

// TestPatternPackShape is where the pack's rules live; genlevelpack only
// parses, so this holds the compiled pack to pattern.Validate. seeds.go also
// indexes it by position.
func TestPatternPackShape(t *testing.T) {
	want := []string{"acorn", "r-pentomino", "gosper-gun", "glider", "lwss", "diehard", "pentadecathlon"}
	if len(patternsPack.Levels) != len(want) {
		t.Fatalf("pack has %d patterns, want %d", len(patternsPack.Levels), len(want))
	}
	for i, name := range want {
		lv := &patternsPack.Levels[i]
		if lv.Name != name {
			t.Errorf("pattern %d is %q, want %q — seeds.go indexes this pack by position", i, lv.Name, name)
		}
		if _, _, err := pattern.Validate(lv); err != nil {
			t.Errorf("%s %v", lv.Name, err)
		}
	}
	if got, want := patternsPack.Legend.Runes, string([]byte{'.', pattern.Live}); got != want {
		t.Errorf("legend is %q, want %q — dead then live", got, want)
	}
}

// TestStampFlippedMatchesCopyAndReverse checks all four orientations, which
// the wasm goldens only reach as their scripted RNG rolls them.
func TestStampFlippedMatchesCopyAndReverse(t *testing.T) {
	// Inside the field, straddling the top-left, straddling the bottom-right.
	offsets := []struct{ x, y int }{
		{20, 20},
		{-2, -1},
		{cartlib.FBWidth - 3, cartlib.FBHeight - 2},
	}
	for _, tc := range flipCases {
		for _, horiz := range []bool{false, true} {
			for _, vert := range []bool{false, true} {
				for _, off := range offsets {
					var got, want grid
					stampFlipped(&got, tc.art, off.x, off.y, horiz, vert)
					stampRows(&want, flipReference(asRows(tc.art), horiz, vert), off.x, off.y)
					if got != want {
						t.Errorf("%s horiz=%v vert=%v at (%d,%d): mirrored stamp differs from copy-and-reverse",
							tc.name, horiz, vert, off.x, off.y)
					}
				}
			}
		}
	}
}

// TestReseedingAllocatesNothing runs every recipe, since a reseed happens
// long after Init.
func TestReseedingAllocatesNothing(t *testing.T) {
	h := hosttest.New(t)
	var f lifeField
	for _, r := range recipes() {
		h.Rand = []uint64{recipeRoll(r.name), 1, 0, 7, 3}
		if n := h.AllocsPerRun(50, f.seedWorld); n != 0 {
			t.Errorf("seedWorld(%s) allocated %v times per run, want 0", r.name, n)
		}
	}
}

func TestSeedRecipeTableBuildsOnce(t *testing.T) {
	first := recipes()
	if len(first) == 0 {
		t.Fatal("recipes() returned an empty table")
	}
	if n := testing.AllocsPerRun(50, func() { recipes() }); n != 0 {
		t.Errorf("recipes() allocated %v times per run after the first call, want 0", n)
	}
	if &recipes()[0] != &first[0] {
		t.Error("recipes() rebuilt its table instead of returning the cached one")
	}
}

// TestLcdReadoutAllocatesNothing checks setText at several digit counts,
// since it runs every quarter second of play.
func TestLcdReadoutAllocatesNothing(t *testing.T) {
	h := hosttest.New(t)
	var g lifeGame
	g.field.resetHistory()
	for y := 0; y < 7; y++ {
		for x := 0; x < 9; x++ {
			g.field.cur[y][x] = true
		}
	}

	for _, gen := range []int{0, 9, 10, 999, 1000, 65535} {
		g.field.gen = gen
		if n := h.AllocsPerRun(50, g.setText); n != 0 {
			t.Errorf("setText at gen %d allocated %v times per run, want 0", gen, n)
		}
	}
	if l1, l2, _ := h.LCD(cartlib.SeatP1); l1 != "Gen: 65535" || l2 != "Pop: 63" {
		t.Errorf("LCD reads %q / %q, want %q / %q", l1, l2, "Gen: 65535", "Pop: 63")
	}
}
