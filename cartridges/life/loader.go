// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-life-cartridge/pattern"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/levels"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// lifeLoadReadyMicros is how long the load screen runs, inside the required
// 2.0-3.0s window.
const lifeLoadReadyMicros uint64 = 2_500_000

const loaderGenStepMicros uint64 = 300_000

// The mini grid sits below drawTitle's wordmark, which ends at y=30.
const (
	loaderCellPx  = 2
	loaderCols    = 26
	loaderRows    = 14
	loaderOriginX = (cartlib.FBWidth - loaderCols*loaderCellPx) / 2
	loaderOriginY = 33
)

// loaderColor differs from gameplay's white cells so the loader reads as a
// preview.
var loaderColor = cartlib.Color{R: 80, G: 255, B: 140, A: 255}

// loaderGrid is the preview's own grid, separate from the real field. The
// stamp and step below repeat field.go's at this size, since Go generics
// cannot abstract over an array's length.
type loaderGrid [loaderRows][loaderCols]bool

var loaderA, loaderB loaderGrid

// loaderStamp is seeds.go's stamp typed to the mini grid's bounds.
func loaderStamp(g *loaderGrid, p *levels.Level, x, y int) {
	for dy := 0; dy < p.H; dy++ {
		row := p.Row(dy)
		for dx := 0; dx < len(row); dx++ {
			if row[dx] != pattern.Live {
				continue
			}
			cx, cy := x+dx, y+dy
			if cx < 0 || cy < 0 || cx >= loaderCols || cy >= loaderRows {
				continue
			}
			g[cy][cx] = true
		}
	}
}

// loaderSeed stamps a centred r-pentomino, which keeps evolving so the
// preview never looks frozen, plus two unflipped gliders.
func loaderSeed(g *loaderGrid) {
	*g = loaderGrid{}
	loaderStamp(g, art(patRPentomino), (loaderCols-3)/2, (loaderRows-3)/2)
	loaderStamp(g, art(patGlider), 1, 1)
	loaderStamp(g, art(patGlider), loaderCols-9, loaderRows-7)
}

// loaderNeighbors counts live neighbours with clamped edges, as field.go does.
func loaderNeighbors(g *loaderGrid, x, y int) int {
	n := 0
	y0, y1 := max(0, y-1), min(loaderRows-1, y+1)
	x0, x1 := max(0, x-1), min(loaderCols-1, x+1)
	for yy := y0; yy <= y1; yy++ {
		for xx := x0; xx <= x1; xx++ {
			if xx == x && yy == y {
				continue
			}
			if g[yy][xx] {
				n++
			}
		}
	}
	return n
}

func loaderNextGen(cur, nxt *loaderGrid) {
	for y := 0; y < loaderRows; y++ {
		for x := 0; x < loaderCols; x++ {
			n := loaderNeighbors(cur, x, y)
			alive := cur[y][x]
			var next bool
			if alive {
				next = n == 2 || n == 3
			} else {
				next = n == 3
			}
			nxt[y][x] = next
		}
	}
}

// loaderFrame replays gens generations from loaderSeed on every call. The
// frame is then a pure function of elapsedMicros, however often or rarely the
// host calls LoadFrame, and no RandomU64 is drawn during the load phase.
func loaderFrame(gens int) *loaderGrid {
	cur, nxt := &loaderA, &loaderB
	loaderSeed(cur)
	for i := 0; i < gens; i++ {
		loaderNextGen(cur, nxt)
		cur, nxt = nxt, cur
	}
	return cur
}

func drawLoaderSim(fb *cartlib.Framebuffer, elapsedMicros uint64) {
	gens := int(elapsedMicros / loaderGenStepMicros)
	g := loaderFrame(gens)
	c := loaderColor
	for y := 0; y < loaderRows; y++ {
		for x := 0; x < loaderCols; x++ {
			if !g[y][x] {
				continue
			}
			fb.FillRect(loaderOriginX+x*loaderCellPx, loaderOriginY+y*loaderCellPx, loaderCellPx, loaderCellPx, c.R, c.G, c.B)
		}
	}
}

// drawTitle draws the "LIFE" wordmark as LED-font text; the logo's glider
// would sit under the loader's grid, so the loader does not blit it.
func drawTitle(fb *cartlib.Framebuffer) {
	font := ui.FontLED{}
	const scale = 3
	const title = "LIFE"
	tw := font.TextWidth(title, scale)
	tx := (cartlib.FBWidth - tw) / 2
	font.DrawText(fb, tx, 6, title, scale, ui.White)
}

// LoadFrame implements cartlib.Loader and reports ready once the window has
// elapsed. The bloom chime plays on the first call only.
func (g *lifeGame) LoadFrame(elapsedMicros uint64, fb *cartlib.Framebuffer) bool {
	if pendingLoaderBloom {
		pendingLoaderBloom = false
		cartlib.PlaySound(soundBloom, 0)
	}

	fb.Clear(0, 0, 0)
	drawTitle(fb)
	drawLoaderSim(fb, elapsedMicros)
	return elapsedMicros >= lifeLoadReadyMicros
}

var _ cartlib.Loader = (*lifeGame)(nil)
