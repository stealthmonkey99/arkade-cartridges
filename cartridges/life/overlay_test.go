// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// bootOverlay boots Life and opens the overlay with A, as a player would.
func bootOverlay(t *testing.T) (*hosttest.Host, *lifeGame) {
	t.Helper()
	h := bootLife(t)
	press(h, cartlib.KeyJoyA)
	if game.machine.State() != stOverlay {
		t.Fatalf("A on the running field left state %d, want the overlay", game.machine.State())
	}
	return h, game
}

func TestRenderOverlayDrawsBorderAndFocusedRow(t *testing.T) {
	_, g := bootOverlay(t)

	var fb cartlib.Framebuffer
	fb.Clear(0, 0, 0)
	renderOverlay(&fb, g.speed)

	// The border highlight at the panel's corner must be visibly non-black.
	if r, gr, b := pixelAt(&fb, overlayPanelX, overlayPanelY); r == 0 && gr == 0 && b == 0 {
		t.Errorf("panel border pixel at (%d,%d) is pure black, want a visible border/dim backdrop", overlayPanelX, overlayPanelY)
	}

	// Row 0 starts focused. Sample just inside each row's left edge to land
	// on the fill rather than a glyph.
	row0Y := overlayMenu.Y + 1
	row1Y := overlayMenu.Y + overlayItemH + overlaySpacing + 1
	fr, fg, fb2 := pixelAt(&fb, overlayMenu.X+1, row0Y)
	ur, ug, ub := pixelAt(&fb, overlayMenu.X+1, row1Y)
	if fr == ur && fg == ug && fb2 == ub {
		t.Errorf("focused row 0 background (%d,%d,%d) == unfocused row 1 background (%d,%d,%d), want visibly different fills",
			fr, fg, fb2, ur, ug, ub)
	}
}

// TestOverlayRowLabelIsOpticallyCentred asserts the rendered label has one
// blank row above and below, so it also catches a shift in sdk/ui's font
// metrics. Inside a row box a pure black pixel can only be glyph ink.
func TestOverlayRowLabelIsOpticallyCentred(t *testing.T) {
	_, g := bootOverlay(t)

	var fb cartlib.Framebuffer
	fb.Clear(0, 0, 0)
	renderOverlay(&fb, g.speed)

	// ui.Border takes the box's first and last rows.
	top := overlayMenu.Y
	bandTop, bandBottom := top+1, top+overlayItemH-2

	inked := func(y int) bool {
		for x := overlayMenu.X; x < overlayMenu.X+overlayMenu.W; x++ {
			if r, gr, b := pixelAt(&fb, x, y); r == 0 && gr == 0 && b == 0 {
				return true
			}
		}
		return false
	}

	firstInk, lastInk := -1, -1
	for y := bandTop; y <= bandBottom; y++ {
		if !inked(y) {
			continue
		}
		if firstInk < 0 {
			firstInk = y
		}
		lastInk = y
	}
	if firstInk < 0 {
		t.Fatalf("no glyph ink found in row 0's label band (y %d..%d) — the row drew no text at all", bandTop, bandBottom)
	}

	above, below := firstInk-bandTop, bandBottom-lastInk
	if above != below {
		t.Errorf("row 0 label sits %d blank row(s) below the band's top and %d above its bottom (ink on y %d..%d of band %d..%d), want them equal — the label reads top-aligned, not centred",
			above, below, firstInk, lastInk, bandTop, bandBottom)
	}
	if above != 1 {
		t.Errorf("row 0 label has %d blank row(s) above its ink, want exactly 1 (band %d..%d, ink %d..%d)", above, bandTop, bandBottom, firstInk, lastInk)
	}
}

func TestRenderOverlayDrawsSliderOnlyWhenSpeedFocused(t *testing.T) {
	_, g := bootOverlay(t)

	// A pixel into the handle's width lands inside it, not on a tick.
	x := speedDetentX(overlayMenu.X, overlayMenu.W, int(g.speed)) + 1
	y := speedSliderY + 1

	overlayMenu.Move(ui.DirDown)
	overlayMenu.Move(ui.DirDown) // DRAW, not SPEED
	if overlayMenu.Selected() != overlayRowDraw {
		t.Fatalf("setup: focused row = %d, want overlayRowDraw (%d)", overlayMenu.Selected(), overlayRowDraw)
	}
	var fbUnfocused cartlib.Framebuffer
	fbUnfocused.Clear(0, 0, 0)
	renderOverlay(&fbUnfocused, g.speed)
	ur, ug, ub := pixelAt(&fbUnfocused, x, y)

	overlayMenu.Move(ui.DirUp)
	if overlayMenu.Selected() != overlayRowSpeed {
		t.Fatalf("setup: focused row = %d, want overlayRowSpeed (%d)", overlayMenu.Selected(), overlayRowSpeed)
	}
	var fbFocused cartlib.Framebuffer
	fbFocused.Clear(0, 0, 0)
	renderOverlay(&fbFocused, g.speed)
	fr, fg, fb2 := pixelAt(&fbFocused, x, y)

	if ur == fr && ug == fg && ub == fb2 {
		t.Errorf("handle pixel (%d,%d) unfocused (%d,%d,%d) == focused (%d,%d,%d), want the slider to only render while SPEED is focused",
			x, y, ur, ug, ub, fr, fg, fb2)
	}
	if fr < 100 {
		t.Errorf("focused handle pixel (%d,%d) = (%d,%d,%d), want a light color (sliderHandleColor), got something dark", x, y, fr, fg, fb2)
	}
}

func TestRefreshOverlayLabelsReflectsCurrentValue(t *testing.T) {
	_, g := bootOverlay(t)

	if overlayMenu.Items[overlayRowStyle] != "STYLE: CLASSIC" {
		t.Errorf("initial STYLE row = %q, want %q", overlayMenu.Items[overlayRowStyle], "STYLE: CLASSIC")
	}
	g.style = styleHeat
	g.refreshOverlayLabels()
	if overlayMenu.Items[overlayRowStyle] != "STYLE: HEAT" {
		t.Errorf("STYLE row after switching to HEAT = %q, want %q", overlayMenu.Items[overlayRowStyle], "STYLE: HEAT")
	}

	g.speed = 8 // fastest detent, 20_000
	g.refreshOverlayLabels()
	if overlayMenu.Items[overlayRowSpeed] != "SPEED: 20K" {
		t.Errorf("SPEED row after moving to the fastest detent = %q, want %q", overlayMenu.Items[overlayRowSpeed], "SPEED: 20K")
	}
}

func TestNextStyleLoopsBothWays(t *testing.T) {
	cases := []struct {
		from uint8
		dir  ui.Direction
		want uint8
	}{
		{styleClassic, ui.DirRight, styleGlow},
		{styleGlow, ui.DirRight, styleHeat},
		{styleHeat, ui.DirRight, styleClassic}, // wraps forward past the end
		{styleClassic, ui.DirLeft, styleHeat},  // wraps backward past the start
		{styleHeat, ui.DirLeft, styleGlow},
		{styleGlow, ui.DirUp, styleGlow}, // non-horizontal is a no-op
	}
	for _, c := range cases {
		if got := nextStyle(c.from, c.dir); got != c.want {
			t.Errorf("nextStyle(%d, %v) = %d, want %d", c.from, c.dir, got, c.want)
		}
	}
}

// TestApplyStyleResetsFadeBuffersAndLabel checks that applyStyle moves the
// style and label, clears the entered style's envelope, and reports false when
// nothing changed.
func TestApplyStyleResetsFadeBuffersAndLabel(t *testing.T) {
	_, g := bootOverlay(t)

	gameplayHeatFall[10][10] = 200
	if !g.applyStyle(styleHeat) {
		t.Fatal("applyStyle(styleHeat) from CLASSIC reported no change, want true")
	}
	if g.style != styleHeat {
		t.Fatalf("style after applyStyle(styleHeat) = %d, want %d", g.style, styleHeat)
	}
	if gameplayHeatFall[10][10] != 0 {
		t.Errorf("gameplayHeatFall[10][10] = %d after entering HEAT, want 0 (a clean envelope, not a leftover flash)", gameplayHeatFall[10][10])
	}
	if overlayMenu.Items[overlayRowStyle] != "STYLE: HEAT" {
		t.Errorf("STYLE row after applyStyle(styleHeat) = %q, want %q", overlayMenu.Items[overlayRowStyle], "STYLE: HEAT")
	}

	if g.applyStyle(styleHeat) {
		t.Error("applyStyle(styleHeat) while ALREADY on HEAT reported a change, want false (setStyle relies on this to skip a redundant save)")
	}
}

func TestNextSpeedDetentStepsAndClamps(t *testing.T) {
	if got := nextSpeedDetent(speedDetentDefault, ui.DirLeft); got != speedDetentDefault-1 {
		t.Fatalf("nextSpeedDetent(default, Left) = %d, want %d", got, speedDetentDefault-1)
	}
	if got := nextSpeedDetent(speedDetentDefault-1, ui.DirRight); got != speedDetentDefault {
		t.Fatalf("nextSpeedDetent(default-1, Right) = %d, want back to %d", got, speedDetentDefault)
	}
	if got := nextSpeedDetent(0, ui.DirLeft); got != 0 {
		t.Fatalf("nextSpeedDetent(0, Left) = %d, want still 0 (clamped, not wrapped)", got)
	}
	if got := nextSpeedDetent(speedDetentCount-1, ui.DirRight); got != speedDetentCount-1 {
		t.Fatalf("nextSpeedDetent(max, Right) = %d, want still %d (clamped, not wrapped)", got, speedDetentCount-1)
	}
	if got := nextSpeedDetent(speedDetentDefault, ui.DirUp); got != speedDetentDefault {
		t.Fatalf("nextSpeedDetent(default, Up) = %d, want unchanged %d (non-horizontal is a no-op)", got, speedDetentDefault)
	}
}

// TestHandleOverlayInputRoutesHorizontalByRow checks that LEFT/RIGHT on a
// value row adjusts and saves the value without moving the selection, and
// does nothing on an action row.
func TestHandleOverlayInputRoutesHorizontalByRow(t *testing.T) {
	h, g := bootOverlay(t)
	left := cartlib.InputRecord{Kind: cartlib.KindJoystick, Joy: cartlib.JoyW}
	right := cartlib.InputRecord{Kind: cartlib.KindJoystick, Joy: cartlib.JoyE}

	g.handleOverlayInput(right)
	if g.style != styleGlow || overlayMenu.Selected() != overlayRowStyle {
		t.Fatalf("RIGHT on STYLE: style %d on row %d, want GLOW (%d) on the STYLE row", g.style, overlayMenu.Selected(), styleGlow)
	}

	g.handleOverlayInput(cartlib.InputRecord{Kind: cartlib.KindJoystick, Joy: cartlib.JoyS})
	if overlayMenu.Selected() != overlayRowSpeed {
		t.Fatalf("setup: focused row = %d, want overlayRowSpeed (%d)", overlayMenu.Selected(), overlayRowSpeed)
	}
	g.handleOverlayInput(left)
	if g.speed != speedDetentDefault-1 || overlayMenu.Selected() != overlayRowSpeed {
		t.Fatalf("LEFT on SPEED: detent %d on row %d, want %d on the SPEED row", g.speed, overlayMenu.Selected(), speedDetentDefault-1)
	}
	if saved := h.Storage[settingsStorageKey]; len(saved) != settingsEncodedLen || saved[0] != styleGlow || saved[1] != speedDetentDefault-1 {
		t.Errorf("stored settings = %v, want GLOW and detent %d saved", saved, speedDetentDefault-1)
	}

	g.handleOverlayInput(cartlib.InputRecord{Kind: cartlib.KindJoystick, Joy: cartlib.JoyS})
	if overlayMenu.Selected() != overlayRowDraw {
		t.Fatalf("setup: focused row = %d, want overlayRowDraw (%d)", overlayMenu.Selected(), overlayRowDraw)
	}
	g.handleOverlayInput(left)
	if g.speed != speedDetentDefault-1 || g.style != styleGlow {
		t.Fatalf("LEFT on the DRAW row changed a value (style %d, speed %d), want both untouched", g.style, g.speed)
	}
	if overlayMenu.Selected() != overlayRowDraw {
		t.Fatalf("focused row after LEFT on DRAW = %d, want still overlayRowDraw (%d) (Left is a no-op on a FocusVertical menu)", overlayMenu.Selected(), overlayRowDraw)
	}
}

// TestHandleOverlayInputPairedRealStickRecordStepsMenuOnce feeds the real
// cabinet shape for one stick push, a KindJoystick record followed by its
// non-virtual KindKeypress twin (docs/CARTRIDGE.md, "Input records"), and
// asserts the selection moves exactly one row.
func TestHandleOverlayInputPairedRealStickRecordStepsMenuOnce(t *testing.T) {
	_, g := bootOverlay(t)
	if overlayMenu.Selected() != overlayRowStyle {
		t.Fatalf("setup: focused row = %d, want overlayRowStyle (%d)", overlayMenu.Selected(), overlayRowStyle)
	}

	g.HandleInput([]cartlib.InputRecord{
		{Kind: cartlib.KindJoystick, Joy: cartlib.JoyS, Player: 1},
		{Kind: cartlib.KindKeypress, Key: cartlib.KeyDown, Player: 1, Flags: 0},
	})

	if got := overlayMenu.Selected(); got != overlayRowSpeed {
		t.Fatalf("focused row after one paired real-stick push = %d, want %d (overlayRowSpeed) — got %d rows of movement instead of exactly 1 (the double-fire bug if this is 2+)",
			got, overlayRowSpeed, got-overlayRowStyle)
	}

	g.HandleInput([]cartlib.InputRecord{
		{Kind: cartlib.KindJoystick, Joy: cartlib.JoyS, Player: 1},
		{Kind: cartlib.KindKeypress, Key: cartlib.KeyDown, Player: 1, Flags: 0},
	})
	if got := overlayMenu.Selected(); got != overlayRowDraw {
		t.Fatalf("focused row after a second paired push = %d, want %d (overlayRowDraw)", got, overlayRowDraw)
	}
}
