// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

const (
	attractBodyLen    = 26
	attractStepMicros = 55_000
	attractTurnMicros = 900_000
)

// The attract instance is separate from a live game, so this state is its own.
var (
	atReady   bool
	atBody    [attractBodyLen]cell
	atLen     int
	atDir     cartlib.Direction
	atStepAcc uint64
	atTurnAcc uint64
)

// Attract implements cartlib.Attractor.
func (snakeGame) Attract(dtMicros uint64, fb *cartlib.Framebuffer) {
	fb.ClearTransparent() // overlay on the starfield

	advanceAttractSnake(dtMicros)
	drawAttractSnake(fb)

	top, board := readAttractScores()
	// Ranks stays off: a 3-digit score already fills the 44px row.
	ui.AttractPanel{
		X: 13, Y: 4, W: 44, H: 34,
		Title:       "SNAKE",
		Top:         top,
		Board:       board,
		Trophy:      true,
		Color:       ui.Color{R: 210, G: 210, B: 210, A: 255},
		AccentColor: ui.Color{R: 0, G: 255, B: 0, A: 255},
	}.Draw(fb)
}

const attractRanks = 5

var (
	attractScoreBuf [8 + cartlib.MaxNameLen]byte
	attractBoard    [attractRanks - 1]ui.AttractScore
	attractNames    [attractRanks]heldName
)

// heldName converts a leaderboard name to a string only when it changes: a
// conversion every frame would leak under -gc=leaking.
type heldName struct {
	b [cartlib.MaxNameLen]byte
	n int
	s string
}

func (h *heldName) of(name []byte) string {
	if len(name) == h.n && sameBytes(h.b[:h.n], name) {
		return h.s
	}
	h.n = copy(h.b[:], name)
	h.s = string(h.b[:h.n])
	return h.s
}

func sameBytes(a, b []byte) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// readAttractScores puts rank 1 on the trophy line and ranks 2-5 on the board.
func readAttractScores() (top ui.AttractScore, board []ui.AttractScore) {
	n := 0
	for rank := 1; rank <= attractRanks; rank++ {
		score, name, ok := cartlib.ScoreRead(rank, attractScoreBuf[:])
		if !ok {
			break
		}
		entry := ui.AttractScore{Name: attractNames[rank-1].of(name), Score: score}
		if rank == 1 {
			top = entry
			continue
		}
		attractBoard[n] = entry
		n++
	}
	return top, attractBoard[:n]
}

func advanceAttractSnake(dtMicros uint64) {
	if !atReady {
		atReady = true
		atDir = cartlib.DirRight
		atLen = 1
		atBody[0] = cell{x: 32, y: 40} // below the panel
	}

	atTurnAcc += dtMicros
	for atTurnAcc >= attractTurnMicros {
		atTurnAcc -= attractTurnMicros
		if d, ok := attractPickTurn(); ok {
			atDir = d
		}
	}

	atStepAcc += dtMicros
	for atStepAcc >= attractStepMicros {
		atStepAcc -= attractStepMicros
		stepAttractSnake()
	}
}

func attractSafe(x, y int) bool {
	if x < fieldMin || x > fieldMax || y < fieldMin || y > fieldMax {
		return false
	}
	for i := 0; i < atLen; i++ {
		if int(atBody[i].x) == x && int(atBody[i].y) == y {
			return false
		}
	}
	return true
}

func attractNext(dir cartlib.Direction, x, y int) (int, int) {
	switch dir {
	case cartlib.DirUp:
		y--
	case cartlib.DirDown:
		y++
	case cartlib.DirLeft:
		x--
	case cartlib.DirRight:
		x++
	}
	return x, y
}

// attractPickTurn never reverses: that would run the head back through the body.
func attractPickTurn() (cartlib.Direction, bool) {
	head := atBody[0]
	x, y := int(head.x), int(head.y)
	a, b := cartlib.DirLeft, cartlib.DirRight
	if atDir.Horizontal() {
		a, b = cartlib.DirUp, cartlib.DirDown
	}
	ax, ay := attractNext(a, x, y)
	bx, by := attractNext(b, x, y)
	aSafe, bSafe := attractSafe(ax, ay), attractSafe(bx, by)
	switch {
	case aSafe && bSafe:
		if cartlib.RandomU64()&1 == 0 {
			return a, true
		}
		return b, true
	case aSafe:
		return a, true
	case bSafe:
		return b, true
	}
	return atDir, false
}

func stepAttractSnake() {
	head := atBody[0]
	x, y := int(head.x), int(head.y)
	nx, ny := attractNext(atDir, x, y)
	if !attractSafe(nx, ny) {
		d, ok := attractPickTurn()
		if !ok {
			// Boxed in: retract the tail until a turn opens.
			if atLen > 1 {
				atLen--
			}
			return
		}
		atDir = d
		nx, ny = attractNext(atDir, x, y)
	}
	for i := atLen; i > 0; i-- {
		if i < attractBodyLen {
			atBody[i] = atBody[i-1]
		}
	}
	atBody[0] = cell{x: uint8(nx), y: uint8(ny)}
	if atLen < attractBodyLen {
		atLen++
	}
}

func drawAttractSnake(fb *cartlib.Framebuffer) {
	for i := atLen - 1; i >= 1; i-- {
		c := atBody[i]
		fb.SetPixel(int(c.x), int(c.y), 0, 130, 0)
	}
	if atLen > 0 {
		h := atBody[0]
		fb.SetPixel(int(h.x), int(h.y), 40, 230, 40)
	}
}
