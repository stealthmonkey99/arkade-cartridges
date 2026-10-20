// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

const (
	// historyDepth is how many recent generations the cycle detector
	// remembers, enough for the still lifes and oscillators a 64x64 field
	// produces.
	historyDepth = 40
	// maxGenerations rotates out worlds that never settle, such as a glider
	// gun.
	maxGenerations = 1200
)

// grid is one generation's cell state, sized to the framebuffer.
type grid [cartlib.FBHeight][cartlib.FBWidth]bool

// lifeField owns the double-buffered grid and cycle-detection state. The
// zero value is not usable until seedWorld has run.
type lifeField struct {
	a, b grid
	cur  *grid
	nxt  *grid // scratch nextGen writes into, then swaps in

	gen int

	// emptied is true when the generation nextGen just produced has no
	// live cells; it says nothing about grids stamped by seedWorld or
	// seedFrom.
	emptied bool

	// A fixed ring of the last historyDepth grid hashes. A map would also
	// work, but its growth after Init allocates, and nothing may.
	histHash [historyDepth]uint64
	histLen  int // valid entries so far, <= historyDepth
	histPos  int // ring write cursor
}

// seedWorld stamps a freshly drawn seed recipe onto a cleared grid and
// resets the generation counter and cycle history.
func (f *lifeField) seedWorld() {
	f.resetHistory()
	r := pickRecipe()
	r.place(f.cur)
	f.recordSeen(f.hash())
}

// seedFrom resets the field like seedWorld but stamps a caller-owned grid.
// It draws no RandomU64.
func (f *lifeField) seedFrom(g *grid) {
	f.resetHistory()
	copy(f.cur[:], g[:]) // not *f.cur = *g, which TinyGo routes through a 4 KB heap temporary
	f.recordSeen(f.hash())
}

// resetHistory binds the buffer pointers on first use, clears both grids
// and resets the generation counter and the ring.
func (f *lifeField) resetHistory() {
	if f.cur == nil {
		f.cur, f.nxt = &f.a, &f.b
	}
	*f.cur = grid{}
	*f.nxt = grid{}
	f.gen = 1
	f.histLen = 0
	f.histPos = 0
}

// recordSeen appends h to the ring, overwriting the oldest entry once full.
func (f *lifeField) recordSeen(h uint64) {
	f.histHash[f.histPos] = h
	f.histPos = (f.histPos + 1) % historyDepth
	if f.histLen < historyDepth {
		f.histLen++
	}
}

// hasSeen reports whether h is still inside the ring's window.
func (f *lifeField) hasSeen(h uint64) bool {
	for i := 0; i < f.histLen; i++ {
		if f.histHash[i] == h {
			return true
		}
	}
	return false
}

// hash is an FNV-1a over the cells.
func (f *lifeField) hash() uint64 {
	h := uint64(14695981039346656037)
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			b := byte(0)
			if f.cur[y][x] {
				b = 1
			}
			h = (h ^ uint64(b)) * 1099511628211
		}
	}
	return h
}

// neighbors counts live neighbours; edges do not wrap.
func (f *lifeField) neighbors(x, y int) int {
	n := 0
	y0, y1 := max(0, y-1), min(cartlib.FBHeight-1, y+1)
	x0, x1 := max(0, x-1), min(cartlib.FBWidth-1, x+1)
	for yy := y0; yy <= y1; yy++ {
		for xx := x0; xx <= x1; xx++ {
			if xx == x && yy == y {
				continue
			}
			if f.cur[yy][xx] {
				n++
			}
		}
	}
	return n
}

func (f *lifeField) population() int {
	n := 0
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			if f.cur[y][x] {
				n++
			}
		}
	}
	return n
}

// nextGen advances one generation and reports whether the world is terminal:
// a repeat of a grid still in the ring, or past maxGenerations. It leaves the
// reseed to the caller, since Update fades first and attract reseeds at once.
// A terminal grid is not recorded, as every way onward resets the ring.
func (f *lifeField) nextGen() bool {
	live := 0
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			n := f.neighbors(x, y)
			alive := f.cur[y][x]
			next := alive
			if alive {
				if n < 2 || n > 3 {
					next = false
				}
			} else if n == 3 {
				next = true
			}
			if next {
				live++
			}
			f.nxt[y][x] = next
		}
	}
	f.cur, f.nxt = f.nxt, f.cur
	f.gen++
	f.emptied = live == 0

	h := f.hash()
	if f.hasSeen(h) || f.gen >= maxGenerations {
		return true
	}
	f.recordSeen(h)
	return false
}
