// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-life-cartridge/pattern"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/levels"
)

// Pattern indices in assets/patterns.arkadelevels order. The weighted roll walks
// the draw table in this order, so reordering the pack changes which recipe a
// given draw selects.
const (
	patAcorn = iota
	patRPentomino
	patGosperGun
	patGlider
	patLWSS
	patDiehard
	patPentadecathlon
)

func art(i int) *levels.Level { return &patternsPack.Levels[i] }

// weightOf reads a pattern's draw weight off the pack; pattern.Validate
// refuses a pattern without one.
func weightOf(i int) int {
	w, _ := art(i).Int("weight")
	return w
}

// seedRecipe is one entry in the weighted draw table.
type seedRecipe struct {
	name   string
	weight int
	place  func(g *grid)
}

// recipeTable is built lazily by recipes rather than as a package-level
// initializer, because package vars must be compile-time foldable under the
// reactor model.
var recipeTable [8]seedRecipe

// recipes returns the weighted draw table, building it on first call.
func recipes() []seedRecipe {
	if recipeTable[0].place == nil {
		// Soup has no pattern, so its weight lives here.
		recipeTable = [len(recipeTable)]seedRecipe{
			{name: "acorn", weight: weightOf(patAcorn), place: placeAcorn},
			{name: "r-pentomino", weight: weightOf(patRPentomino), place: placeRPentomino},
			{name: "gosper-gun", weight: weightOf(patGosperGun), place: placeGosperGun},
			{name: "glider-fleet", weight: weightOf(patGlider), place: placeGliderFleet},
			{name: "spaceship-parade", weight: weightOf(patLWSS), place: placeSpaceshipParade},
			{name: "diehard", weight: weightOf(patDiehard), place: placeDiehard},
			{name: "heartbeat", weight: weightOf(patPentadecathlon), place: placeHeartbeat},
			{name: "soup", weight: 7, place: placeSoup},
		}
	}
	return recipeTable[:]
}

func pickRecipe() seedRecipe {
	rs := recipes()
	total := 0
	for _, r := range rs {
		total += r.weight
	}
	roll := randIntn(total)
	for _, r := range rs {
		roll -= r.weight
		if roll < 0 {
			return r
		}
	}
	return rs[len(rs)-1]
}

// randIntn returns a value in [0, n) from cartlib.RandomU64. Modulo bias is
// negligible at these n.
func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(cartlib.RandomU64() % uint64(n))
}

// stamp draws a pattern with its top-left at (x,y), clipping off-grid cells.
func stamp(g *grid, p *levels.Level, x, y int) {
	stampFlipped(g, p, x, y, false, false)
}

// stampFlipped draws a pattern at (x,y), optionally mirrored. The mirror is
// an index flip on read, because a reversed copy would allocate after Init
// and never be freed under -gc=leaking. Pack grids are rectangular, so a
// row's length is the pattern's width.
func stampFlipped(g *grid, p *levels.Level, x, y int, horiz, vert bool) {
	for dy := 0; dy < p.H; dy++ {
		row := p.Row(dy)
		if vert {
			row = p.Row(p.H - 1 - dy)
		}
		for dx := 0; dx < len(row); dx++ {
			sx := dx
			if horiz {
				sx = len(row) - 1 - dx
			}
			if row[sx] != pattern.Live {
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

// centeredStamp places a pattern mid-grid with random jitter.
func centeredStamp(g *grid, p *levels.Level) {
	jx := randIntn(21) - 10
	jy := randIntn(21) - 10
	stamp(g, p, (cartlib.FBWidth-p.W)/2+jx, (cartlib.FBHeight-p.H)/2+jy)
}

// fleetStamp scatters minN to maxN copies of a pattern in random spots and
// orientations.
func fleetStamp(g *grid, p *levels.Level, minN, maxN int) {
	w, h := p.W, p.H
	n := minN + randIntn(maxN-minN+1)
	for i := 0; i < n; i++ {
		// Draw order is horizontal flip, vertical flip, x, y; the goldens
		// were recorded against this stream.
		horiz := randIntn(2) == 0
		vert := randIntn(2) == 0
		x := 4 + randIntn(cartlib.FBWidth-w-8)
		y := 4 + randIntn(cartlib.FBHeight-h-8)
		stampFlipped(g, p, x, y, horiz, vert)
	}
}

func placeAcorn(g *grid)       { centeredStamp(g, art(patAcorn)) }
func placeRPentomino(g *grid)  { centeredStamp(g, art(patRPentomino)) }
func placeDiehard(g *grid)     { centeredStamp(g, art(patDiehard)) }
func placeGliderFleet(g *grid) { fleetStamp(g, art(patGlider), 4, 7) }

// placeGosperGun puts the gun near the top-left so it fires across the field.
func placeGosperGun(g *grid) {
	stamp(g, art(patGosperGun), 2+randIntn(3), 2+randIntn(3))
}

// placeSpaceshipParade adds a centred r-pentomino to keep the field busy
// after the ships land.
func placeSpaceshipParade(g *grid) {
	fleetStamp(g, art(patLWSS), 2, 4)
	centeredStamp(g, art(patRPentomino))
}

// placeHeartbeat puts a pentadecathlon and an r-pentomino on opposite halves.
func placeHeartbeat(g *grid) {
	pd, rp := art(patPentadecathlon), art(patRPentomino)
	stamp(g, pd, cartlib.FBWidth/4-pd.W/2, (cartlib.FBHeight-pd.H)/2+randIntn(9)-4)
	stamp(g, rp, 3*cartlib.FBWidth/4-rp.W/2, (cartlib.FBHeight-rp.H)/2+randIntn(9)-4)
}

// Soup fills a central patch, since a full-field soup reads as static. The
// density is an integer percent to avoid floats.
const (
	soupSize           = 34
	soupDensityPercent = 35
)

func placeSoup(g *grid) {
	x0 := (cartlib.FBWidth - soupSize) / 2
	y0 := (cartlib.FBHeight - soupSize) / 2
	for y := y0; y < y0+soupSize; y++ {
		for x := x0; x < x0+soupSize; x++ {
			if randIntn(100) < soupDensityPercent {
				g[y][x] = true
			}
		}
	}
}
