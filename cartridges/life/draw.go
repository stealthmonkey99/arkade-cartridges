// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// DRAW mode: the stick moves a cursor, A toggles a cell and C runs the
// drawing as the field's new seed. A drawing is stored packed, one bit per
// cell and a little-endian uint64 per row (512 bytes), and savedDrawings
// keeps that packed shape in memory.

const maxDrawings = 10
const drawGridSize = cartlib.FBWidth // the canvas is the square framebuffer

// drawingEncodedLen is one drawing's packed size: 64 rows of 8 bytes.
const drawingEncodedLen = drawGridSize * 8

const drawingsStorageKey = "drawings"

// drawingsEncodedLen is the whole blob's worst case: a count byte plus
// maxDrawings full drawings.
const drawingsEncodedLen = 1 + maxDrawings*drawingEncodedLen

// drawBuf is the live edit canvas.
var drawBuf grid

// savedDrawings holds committed drawings, slot 0 the newest; savedCount is
// how many slots are in use.
var savedDrawings [maxDrawings][drawGridSize]uint64
var savedCount int

// drawPickLabels is the picker's full item list: the blank canvas, then the
// saved slots newest-first. The menu shows a prefix of it.
var drawPickLabels = [1 + maxDrawings]string{
	"NEW DRAWING",
	"SAVED 1", "SAVED 2", "SAVED 3", "SAVED 4", "SAVED 5",
	"SAVED 6", "SAVED 7", "SAVED 8", "SAVED 9", "SAVED 10",
}

func packRow(row *[drawGridSize]bool) uint64 {
	var bits uint64
	for x := 0; x < drawGridSize; x++ {
		if row[x] {
			bits |= uint64(1) << uint(x)
		}
	}
	return bits
}

func unpackRow(bits uint64, row *[drawGridSize]bool) {
	for x := 0; x < drawGridSize; x++ {
		row[x] = bits&(uint64(1)<<uint(x)) != 0
	}
}

func putLEUint64(b []byte, v uint64) {
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> uint(8*i))
	}
}

func leUint64(b []byte) uint64 {
	var v uint64
	for i := 0; i < 8; i++ {
		v |= uint64(b[i]) << uint(8*i)
	}
	return v
}

// encodeDrawingsBlob serializes the saved slots into buf, which must hold
// drawingsEncodedLen bytes, and returns the bytes written.
func encodeDrawingsBlob(buf []byte) int {
	n := savedCount
	if n > maxDrawings {
		n = maxDrawings
	}
	buf[0] = byte(n)
	pos := 1
	for i := 0; i < n; i++ {
		for y := 0; y < drawGridSize; y++ {
			putLEUint64(buf[pos:pos+8], savedDrawings[i][y])
			pos += 8
		}
	}
	return pos
}

// decodeDrawingsBlob fills savedDrawings from the n bytes StorageRead put in
// buf and returns the count. An implausible count or a length that does not
// match it exactly decodes to zero drawings.
func decodeDrawingsBlob(buf []byte, n int) int {
	if n < 1 {
		return 0
	}
	count := int(buf[0])
	if count < 0 || count > maxDrawings {
		return 0
	}
	if n != 1+count*drawingEncodedLen {
		return 0
	}
	pos := 1
	for i := 0; i < count; i++ {
		for y := 0; y < drawGridSize; y++ {
			savedDrawings[i][y] = leUint64(buf[pos : pos+8])
			pos += 8
		}
	}
	return count
}

// loadDrawings restores the library from storage; Init only, and it draws no
// RandomU64.
func loadDrawings() {
	n := cartlib.StorageRead(drawingsStorageKey, drawingsBuf[:])
	savedCount = decodeDrawingsBlob(drawingsBuf[:], n)
}

// saveDrawingsBlob writes only the bytes in use, which decodeDrawingsBlob's
// exact-length check relies on.
func saveDrawingsBlob() {
	n := encodeDrawingsBlob(drawingsBuf[:])
	cartlib.StorageWrite(drawingsStorageKey, drawingsBuf[:n])
}

// drawingsBuf is package-level so a save does not put 5 KB on the heap.
var drawingsBuf [drawingsEncodedLen]byte

// prependDrawingSlots packs g into slot 0, shifting older drawings down and
// dropping the oldest once full.
func prependDrawingSlots(g *grid) {
	n := savedCount
	if n > maxDrawings-1 {
		n = maxDrawings - 1
	}
	for i := n; i > 0; i-- {
		savedDrawings[i] = savedDrawings[i-1]
	}
	for y := 0; y < drawGridSize; y++ {
		savedDrawings[0][y] = packRow(&g[y])
	}
	if savedCount < maxDrawings {
		savedCount++
	}
}

func prependDrawing(g *grid) {
	prependDrawingSlots(g)
	saveDrawingsBlob()
}

// loadDrawingIntoBuf unpacks a saved slot into drawBuf.
func loadDrawingIntoBuf(idx int) {
	for y := 0; y < drawGridSize; y++ {
		unpackRow(savedDrawings[idx][y], &drawBuf[y])
	}
}

// drawMoveStepMicros is the cursor's repeat cadence while a direction is
// held, after the first step, which lands as the hold begins.
const drawMoveStepMicros uint64 = 130_000

// drawBlinkHalfPeriodMicros is the cursor's blink half-cycle. The cursor
// blinks the cell's own pixel because a cell is one framebuffer pixel.
const drawBlinkHalfPeriodMicros uint64 = 250_000

// enterDrawPicker sizes the picker to the saved drawings and layers it over
// the overlay.
func (g *lifeGame) enterDrawPicker() {
	drawPickMenu.SetItems(drawPickLabels[:1+savedCount])
	g.machine.Push(stDrawPick)
}

// confirmDrawPick loads the selected item into drawBuf and enters the editor,
// ignoring an out-of-range index. It uses Go, not Push, so cancelDrawEditor's
// single Pop lands on the overlay.
func (g *lifeGame) confirmDrawPick() {
	idx := drawPickMenu.Activate()
	switch {
	case idx == 0:
		drawBuf = grid{}
	case idx >= 1 && idx-1 < savedCount:
		loadDrawingIntoBuf(idx - 1)
	default:
		return
	}
	g.machine.Go(stDraw)
}

// cancelDrawEditor backs out to the overlay without saving drawBuf.
func (g *lifeGame) cancelDrawEditor() { g.machine.Pop() }

// commitDrawing makes drawBuf the field's new generation 1, saves it and
// resumes play. It cancels any terminal fade, which would otherwise reseed
// the new drawing away.
func (g *lifeGame) commitDrawing() {
	g.field.seedFrom(&drawBuf)
	prependDrawing(&drawBuf)
	resetGameplayGlow()
	g.cancelTerminalFade()
	// Reset, not Pop, so the overlay's legend does not flash on the way.
	g.machine.Reset(stPlaying)
}

func (g *lifeGame) toggleDrawCell() {
	drawBuf[g.drawCY][g.drawCX] = !drawBuf[g.drawCY][g.drawCX]
}

// stepDrawCursor moves the cursor one cell, wrapping at the grid edges.
func (g *lifeGame) stepDrawCursor(d ui.Direction) {
	switch d {
	case ui.DirUp:
		g.drawCY = (g.drawCY + drawGridSize - 1) % drawGridSize
	case ui.DirDown:
		g.drawCY = (g.drawCY + 1) % drawGridSize
	case ui.DirLeft:
		g.drawCX = (g.drawCX + drawGridSize - 1) % drawGridSize
	case ui.DirRight:
		g.drawCX = (g.drawCX + 1) % drawGridSize
	}
}

// updateDraw samples drawHeld's level once per tick rather than counting
// records, so a real stick's record pairs cannot double-step the cursor.
func (g *lifeGame) updateDraw(dtMicros uint64) {
	g.drawHeld.Update(dtMicros)
	cur := g.drawHeld.Current()
	if cur != g.drawPrevDir {
		// A fresh hold or a turn steps immediately.
		if cur != ui.DirNone {
			g.stepDrawCursor(cur)
		}
		g.drawMoveAccum = 0
		g.drawPrevDir = cur
	} else if cur != ui.DirNone {
		g.drawMoveAccum += dtMicros
		for g.drawMoveAccum >= drawMoveStepMicros {
			g.drawMoveAccum -= drawMoveStepMicros
			g.stepDrawCursor(cur)
		}
	}

	full := 2 * drawBlinkHalfPeriodMicros
	g.drawBlinkAccum += dtMicros
	if g.drawBlinkAccum >= full {
		g.drawBlinkAccum %= full
	}
}

func (g *lifeGame) drawCursorOn() bool {
	return g.drawBlinkAccum < drawBlinkHalfPeriodMicros
}

// renderDrawEditor draws drawBuf in white on black, independent of the
// display style, then the cursor.
func renderDrawEditor(fb *cartlib.Framebuffer, cx, cy int, cursorOn bool) {
	fb.Clear(0, 0, 0)
	for y := 0; y < drawGridSize; y++ {
		for x := 0; x < drawGridSize; x++ {
			if drawBuf[y][x] {
				fb.SetPixel(x, y, 255, 255, 255)
			}
		}
	}
	if cursorOn {
		fb.SetPixel(cx, cy, 255, 220, 0) // amber, blinking
	}
}

// handleDrawInput feeds every record to drawHeld; A, C and B act once per
// press.
func (g *lifeGame) handleDrawInput(r cartlib.InputRecord) {
	g.drawHeld.Feed(r)
	if !r.IsPress() {
		return
	}
	switch r.Key {
	case cartlib.KeyJoyA:
		g.toggleDrawCell()
	case cartlib.KeyJoyC, cartlib.KeyStart:
		g.commitDrawing()
	default:
		if r.IsQuit() {
			g.cancelDrawEditor()
		}
	}
}
