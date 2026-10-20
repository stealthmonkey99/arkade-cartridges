// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// resetDrawingsState clears the package-level saved-drawings state.
func resetDrawingsState() {
	savedDrawings = [maxDrawings][drawGridSize]uint64{}
	savedCount = 0
}

func TestPackUnpackRowRoundTrip(t *testing.T) {
	var row [drawGridSize]bool
	for x := 0; x < drawGridSize; x += 3 {
		row[x] = true
	}
	bits := packRow(&row)

	var back [drawGridSize]bool
	unpackRow(bits, &back)
	if back != row {
		t.Fatalf("unpackRow(packRow(row)) != row")
	}
}

func TestPackRowAllDeadIsZero(t *testing.T) {
	var row [drawGridSize]bool
	if got := packRow(&row); got != 0 {
		t.Errorf("packRow(all dead) = %d, want 0", got)
	}
}

func TestLEUint64RoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 0xFF, 0x1234567890ABCDEF, ^uint64(0)}
	for _, v := range cases {
		var buf [8]byte
		putLEUint64(buf[:], v)
		if got := leUint64(buf[:]); got != v {
			t.Errorf("leUint64(putLEUint64(%d)) = %d", v, got)
		}
	}
}

func TestDrawingsSurviveARestart(t *testing.T) {
	resetDrawingsState()
	t.Cleanup(resetDrawingsState)
	hosttest.New(t)

	var g grid
	g[3][4], g[60][61] = true, true
	prependDrawing(&g)

	resetDrawingsState()
	loadDrawings()
	if savedCount != 1 {
		t.Fatalf("after a restart: %d saved drawings, want 1", savedCount)
	}
	loadDrawingIntoBuf(0)
	if drawBuf != g {
		t.Error("the saved drawing came back different")
	}
}

func TestEncodeDecodeDrawingsBlobRoundTrip(t *testing.T) {
	resetDrawingsState()
	t.Cleanup(resetDrawingsState)

	var g1, g2 grid
	g1[0][0] = true
	g1[63][63] = true
	g2[10][20] = true

	prependDrawingSlots(&g1) // becomes slot 0
	prependDrawingSlots(&g2) // becomes slot 0; g1 shifts to slot 1

	if savedCount != 2 {
		t.Fatalf("savedCount = %d, want 2", savedCount)
	}

	var buf [drawingsEncodedLen]byte
	n := encodeDrawingsBlob(buf[:])

	resetDrawingsState()
	got := decodeDrawingsBlob(buf[:], n)
	if got != 2 {
		t.Fatalf("decodeDrawingsBlob returned count %d, want 2", got)
	}

	var back grid
	loadDrawingIntoBuf(0)
	back = drawBuf
	if back != g2 {
		t.Errorf("slot 0 after round trip != g2")
	}
	loadDrawingIntoBuf(1)
	back = drawBuf
	if back != g1 {
		t.Errorf("slot 1 after round trip != g1")
	}
}

func TestDecodeDrawingsBlobRejectsMismatchedLength(t *testing.T) {
	resetDrawingsState()
	t.Cleanup(resetDrawingsState)

	var buf [drawingsEncodedLen]byte
	buf[0] = 3
	if got := decodeDrawingsBlob(buf[:], 1+2*drawingEncodedLen); got != 0 {
		t.Errorf("decodeDrawingsBlob(mismatched length) = %d, want 0", got)
	}
	buf[0] = byte(maxDrawings + 1)
	if got := decodeDrawingsBlob(buf[:], 1+(maxDrawings+1)*drawingEncodedLen); got != 0 {
		t.Errorf("decodeDrawingsBlob(count > max) = %d, want 0", got)
	}
	if got := decodeDrawingsBlob(buf[:], 0); got != 0 {
		t.Errorf("decodeDrawingsBlob(n=0) = %d, want 0", got)
	}
}

func TestPrependDrawingSlotsCapsAtMaxAndDropsOldest(t *testing.T) {
	resetDrawingsState()
	t.Cleanup(resetDrawingsState)

	// Drawing i lights cell (i,0).
	mk := func(i int) grid {
		var g grid
		g[0][i] = true
		return g
	}

	for i := 0; i < maxDrawings; i++ {
		g := mk(i)
		prependDrawingSlots(&g)
	}
	if savedCount != maxDrawings {
		t.Fatalf("savedCount = %d, want %d after filling every slot", savedCount, maxDrawings)
	}
	loadDrawingIntoBuf(maxDrawings - 1)
	if drawBuf != mk(0) {
		t.Fatalf("oldest slot != drawing 0 before the 11th save")
	}

	g := mk(maxDrawings) // the 11th drawing
	prependDrawingSlots(&g)
	if savedCount != maxDrawings {
		t.Fatalf("savedCount = %d after the 11th save, want it still capped at %d", savedCount, maxDrawings)
	}
	loadDrawingIntoBuf(0)
	if drawBuf != mk(maxDrawings) {
		t.Fatalf("slot 0 after the 11th save != the newest drawing")
	}
	loadDrawingIntoBuf(maxDrawings - 1)
	if drawBuf == mk(0) {
		t.Fatalf("drawing 0 (the oldest) should have fallen off after the 11th save")
	}
	if drawBuf != mk(1) {
		t.Fatalf("oldest remaining slot != drawing 1 (drawing 0 should have been dropped)")
	}
}

func TestStepDrawCursorWraps(t *testing.T) {
	var g lifeGame

	g.drawCX, g.drawCY = 0, 0
	g.stepDrawCursor(ui.DirUp)
	if g.drawCY != drawGridSize-1 {
		t.Errorf("DirUp from row 0: drawCY = %d, want %d", g.drawCY, drawGridSize-1)
	}

	g.drawCX, g.drawCY = 0, 0
	g.stepDrawCursor(ui.DirLeft)
	if g.drawCX != drawGridSize-1 {
		t.Errorf("DirLeft from col 0: drawCX = %d, want %d", g.drawCX, drawGridSize-1)
	}

	g.drawCX, g.drawCY = drawGridSize-1, drawGridSize-1
	g.stepDrawCursor(ui.DirDown)
	if g.drawCY != 0 {
		t.Errorf("DirDown from last row: drawCY = %d, want 0", g.drawCY)
	}

	g.drawCX, g.drawCY = drawGridSize-1, drawGridSize-1
	g.stepDrawCursor(ui.DirRight)
	if g.drawCX != 0 {
		t.Errorf("DirRight from last col: drawCX = %d, want 0", g.drawCX)
	}
}

func TestUpdateDrawFirstStepLandsImmediately(t *testing.T) {
	var g lifeGame
	g.drawCX, g.drawCY = 32, 32
	g.drawHeld.SetJoy(ui.DirRight)
	g.updateDraw(1000)
	if g.drawCX != 33 {
		t.Fatalf("drawCX after first tick with DirRight held = %d, want 33 (immediate first step)", g.drawCX)
	}
}

func TestUpdateDrawRepeatsAtFixedCadence(t *testing.T) {
	var g lifeGame
	g.drawCX, g.drawCY = 0, 32
	g.drawHeld.SetJoy(ui.DirRight)
	g.updateDraw(0) // the first step fires on dt=0 too
	if g.drawCX != 1 {
		t.Fatalf("drawCX after priming = %d, want 1", g.drawCX)
	}

	g.updateDraw(drawMoveStepMicros - 1)
	if g.drawCX != 1 {
		t.Fatalf("drawCX just before the repeat threshold = %d, want still 1", g.drawCX)
	}
	g.updateDraw(1)
	if g.drawCX != 2 {
		t.Fatalf("drawCX right at the repeat threshold = %d, want 2", g.drawCX)
	}
	g.updateDraw(2 * drawMoveStepMicros)
	if g.drawCX != 4 {
		t.Fatalf("drawCX after two more full repeat intervals = %d, want 4", g.drawCX)
	}
}

func TestUpdateDrawBlinkTogglesPhase(t *testing.T) {
	var g lifeGame
	if !g.drawCursorOn() {
		t.Fatalf("cursor should start in the ON phase")
	}
	g.updateDraw(drawBlinkHalfPeriodMicros - 1)
	if !g.drawCursorOn() {
		t.Fatalf("cursor should still be ON just before the half period")
	}
	g.updateDraw(2) // crosses into the OFF half
	if g.drawCursorOn() {
		t.Fatalf("cursor should be OFF just after the half period")
	}
	g.updateDraw(drawBlinkHalfPeriodMicros) // crosses back into ON, wrapping
	if !g.drawCursorOn() {
		t.Fatalf("cursor should be back ON after a full period")
	}
}
