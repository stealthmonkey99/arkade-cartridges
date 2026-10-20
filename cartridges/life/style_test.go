// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
)

// pixelAt reads the RGB at (x, y) of a framebuffer a test rendered into
// directly.
func pixelAt(fb *cartlib.Framebuffer, x, y int) (r, g, b uint8) {
	o := (y*cartlib.FBWidth + x) * 4
	return fb.Pix[o], fb.Pix[o+1], fb.Pix[o+2]
}

// TestHeatFadeColorMapping checks heatFadeColor's mapping. Each case sets only
// the envelope meaningful for its state and leaves the other at 0.
func TestHeatFadeColorMapping(t *testing.T) {
	cases := []struct {
		name                string
		rise, fall          uint8
		alive               bool
		wantR, wantG, wantB uint8
		wantOK              bool
	}{
		{"alive, just born is pure green at full brightness", 0, 0, true, 0, 255, 0, true},
		{"alive, low rise is still green-dominant", 10, 0, true, 10, 255, 10, true},
		{"alive, fully risen is white", 255, 0, true, 255, 255, 255, true},
		{"alive, mid-rise is a green/white blend", 128, 0, true, 128, 255, 128, true},
		{"dead, just died is bright red at full brightness", 0, 255, false, 255, 0, 0, true},
		{"dead, mid-fade is dim red", 0, 128, false, 128, 0, 0, true},
		{"dead, fully faded draws nothing", 0, 0, false, 0, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, g, b, ok := heatFadeColor(c.rise, c.fall, c.alive)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if r != c.wantR || g != c.wantG || b != c.wantB {
				t.Fatalf("heatFadeColor(%d, %d, %v) = (%d,%d,%d), want (%d,%d,%d)",
					c.rise, c.fall, c.alive, r, g, b, c.wantR, c.wantG, c.wantB)
			}
		})
	}
}

// TestStepHeatRisesWhileAliveAndFallsWhenDead checks rise climbs and clamps
// at 255, then fall snaps to 255 on death and counts down, clamping at 0.
func TestStepHeatRisesWhileAliveAndFallsWhenDead(t *testing.T) {
	var cur grid
	var rise, fall glowGrid
	const riseStep, fallStep uint8 = 6, 6

	cur[5][5] = true
	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if rise[5][5] != riseStep {
		t.Fatalf("after 1 tick alive, rise = %d, want %d (first rise step)", rise[5][5], riseStep)
	}
	if fall[5][5] != 0 {
		t.Fatalf("after 1 tick alive, fall = %d, want 0 (no death fade while alive)", fall[5][5])
	}
	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if rise[5][5] != 2*riseStep {
		t.Fatalf("after 2 ticks alive, rise = %d, want %d", rise[5][5], 2*riseStep)
	}

	for i := 0; i < 100; i++ {
		stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	}
	if rise[5][5] != 255 {
		t.Fatalf("rise after many alive ticks = %d, want 255 (clamped, no wraparound)", rise[5][5])
	}

	cur[5][5] = false
	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if fall[5][5] != 255 {
		t.Fatalf("fall on the tick a settled cell dies = %d, want 255 (immediate bright-red flash)", fall[5][5])
	}
	if rise[5][5] != 0 {
		t.Fatalf("rise on the tick a cell dies = %d, want 0 (reset)", rise[5][5])
	}

	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if fall[5][5] != 255-fallStep {
		t.Fatalf("fall after 2nd tick dead = %d, want %d", fall[5][5], 255-fallStep)
	}

	for i := 0; i < 100; i++ {
		stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	}
	if fall[5][5] != 0 {
		t.Fatalf("fall after many dead ticks = %d, want 0 (clamped, no underflow)", fall[5][5])
	}
}

// A single envelope shared by rise and fall would flash this death dim.
func TestStepHeatDeathFlashIsBrightRegardlessOfHowLongTheCellLived(t *testing.T) {
	var cur grid
	var rise, fall glowGrid
	const riseStep, fallStep uint8 = 6, 6

	cur[2][2] = true
	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if rise[2][2] == 0 || rise[2][2] == 255 {
		t.Fatalf("setup: rise after 1 tick alive = %d, want a low, unsettled value", rise[2][2])
	}

	cur[2][2] = false
	stepHeat(&cur, &rise, &fall, riseStep, fallStep)
	if fall[2][2] != 255 {
		t.Fatalf("fall on the tick a barely-risen cell dies = %d, want 255 (bright flash), not a dim carryover of rise", fall[2][2])
	}
}

func TestRenderHeatBirthFlashesGreenThenEasesToWhite(t *testing.T) {
	resetGameplayHeat()
	t.Cleanup(resetGameplayHeat)

	var f lifeField
	f.cur, f.nxt = &f.a, &f.b
	f.cur[0][0] = true

	// renderHeat only composites, so each render needs a cleared fb, as
	// Render provides.
	var fb cartlib.Framebuffer
	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	r0, g0, b0 := pixelAt(&fb, 0, 0)
	if g0 != 255 || r0 == 0 || r0 >= 255 || r0 != b0 {
		t.Fatalf("birth frame (0,0) = (%d,%d,%d), want green-dominant (G=255, 0<R=B<255)", r0, g0, b0)
	}

	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	r1, g1, b1 := pixelAt(&fb, 0, 0)
	if g1 != 255 || r1 <= r0 || r1 != b1 {
		t.Fatalf("second render (0,0) = (%d,%d,%d), want easing whiter than the birth frame (%d,%d,%d)", r1, g1, b1, r0, g0, b0)
	}

	// 255/fadeStep is about 43 renders to settle.
	for i := 0; i < 50; i++ {
		fb.Clear(0, 0, 0)
		renderHeat(&fb, &f)
	}
	if r, g, b := pixelAt(&fb, 0, 0); r != 255 || g != 255 || b != 255 {
		t.Fatalf("long-settled survivor (0,0) = (%d,%d,%d), want white", r, g, b)
	}
}

func TestRenderHeatDeathFadesThroughRedRatherThanVanishing(t *testing.T) {
	resetGameplayHeat()
	t.Cleanup(resetGameplayHeat)

	var f lifeField
	f.cur, f.nxt = &f.a, &f.b
	f.cur[0][0] = true

	var fb cartlib.Framebuffer
	for i := 0; i < 50; i++ {
		fb.Clear(0, 0, 0)
		renderHeat(&fb, &f)
	}
	if r, g, b := pixelAt(&fb, 0, 0); r != 255 || g != 255 || b != 255 {
		t.Fatalf("setup: settled cell = (%d,%d,%d), want white before killing it", r, g, b)
	}

	f.cur[0][0] = false
	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	r0, g0, b0 := pixelAt(&fb, 0, 0)
	if g0 != 0 || b0 != 0 || r0 != 255 {
		t.Fatalf("first frame after death (0,0) = (%d,%d,%d), want a bright-red flash (G=B=0, R=255), not vanished or dim", r0, g0, b0)
	}

	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	r1, _, _ := pixelAt(&fb, 0, 0)
	if r1 >= r0 {
		t.Fatalf("second frame after death red = %d, want dimmer than the first frame's %d", r1, r0)
	}

	for i := 0; i < 100; i++ {
		fb.Clear(0, 0, 0)
		renderHeat(&fb, &f)
	}
	if r, g, b := pixelAt(&fb, 0, 0); r != 0 || g != 0 || b != 0 {
		t.Fatalf("fully-faded dead cell = (%d,%d,%d), want black (nothing drawn)", r, g, b)
	}
}

// TestRenderHeatShortLivedBirthStillFlashesBrightRedOnDeath is the render
// counterpart of TestStepHeatDeathFlashIsBrightRegardlessOfHowLongTheCellLived.
func TestRenderHeatShortLivedBirthStillFlashesBrightRedOnDeath(t *testing.T) {
	resetGameplayHeat()
	t.Cleanup(resetGameplayHeat)

	var f lifeField
	f.cur, f.nxt = &f.a, &f.b
	f.cur[0][0] = true

	var fb cartlib.Framebuffer
	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	rBirth, gBirth, _ := pixelAt(&fb, 0, 0)
	if gBirth != 255 || rBirth == 0 || rBirth >= 255 {
		t.Fatalf("setup: single-tick birth frame = R%d G%d, want a low, unsettled green", rBirth, gBirth)
	}

	f.cur[0][0] = false
	fb.Clear(0, 0, 0)
	renderHeat(&fb, &f)
	r, g, b := pixelAt(&fb, 0, 0)
	if g != 0 || b != 0 || r != 255 {
		t.Fatalf("death frame after a single-tick life = (%d,%d,%d), want a full-brightness red flash (255,0,0), not dim", r, g, b)
	}
}

func TestRenderHeatUntouchedCellStaysBlack(t *testing.T) {
	resetGameplayHeat()
	t.Cleanup(resetGameplayHeat)

	var f lifeField
	f.cur, f.nxt = &f.a, &f.b

	var fb cartlib.Framebuffer
	for i := 0; i < 5; i++ {
		renderHeat(&fb, &f)
	}
	if r, g, b := pixelAt(&fb, 3, 3); r != 0 || g != 0 || b != 0 {
		t.Errorf("never-alive cell (3,3) = (%d,%d,%d), want black", r, g, b)
	}
}

// TestGenerationStepFor pins the detent table, and that NORMAL equals
// generationStepMicros, which keeps the goldens valid at the default speed.
func TestGenerationStepFor(t *testing.T) {
	cases := []struct {
		detent uint8
		want   uint64
	}{
		{0, 800_000},
		{1, 500_000},
		{2, 300_000},
		{3, 200_000},
		{4, 133_000},
		{5, 100_000},
		{6, 66_000},
		{7, 40_000},
		{8, 20_000},
		{99, 100_000}, // out of range falls back to NORMAL
	}
	for _, c := range cases {
		if got := generationStepFor(c.detent); got != c.want {
			t.Errorf("generationStepFor(%d) = %d, want %d", c.detent, got, c.want)
		}
	}
	if generationStepFor(speedDetentDefault) != generationStepMicros {
		t.Errorf("the CENTER/NORMAL detent must equal the original generationStepMicros constant")
	}
}

// TestStyleAndSpeedLabelTablesAreComplete catches a blank overlay row when a
// count changes without its label table.
func TestStyleAndSpeedLabelTablesAreComplete(t *testing.T) {
	for i := uint8(0); i < styleCount; i++ {
		if styleRowLabels[i] == "" {
			t.Errorf("styleRowLabels[%d] is empty", i)
		}
	}
	for i := 0; i < speedDetentCount; i++ {
		if speedDetentLabels[i] == "" {
			t.Errorf("speedDetentLabels[%d] is empty", i)
		}
	}
}

// TestOldSpeedToDetentTableShape checks oldSpeedToDetent has four valid
// entries and that old SLOW, NORMAL and TURBO keep their exact cadence.
func TestOldSpeedToDetentTableShape(t *testing.T) {
	if len(oldSpeedToDetent) != 4 {
		t.Fatalf("len(oldSpeedToDetent) = %d, want 4 (old SLOW/NORMAL/FAST/TURBO)", len(oldSpeedToDetent))
	}
	for i, detent := range oldSpeedToDetent {
		if detent >= speedDetentCount {
			t.Errorf("oldSpeedToDetent[%d] = %d, out of range [0,%d)", i, detent, speedDetentCount)
		}
	}
	exact := map[int]uint64{0: 200_000, 1: 100_000, 3: 20_000} // old SLOW, NORMAL, TURBO
	for oldIdx, wantMicros := range exact {
		got := generationStepFor(oldSpeedToDetent[oldIdx])
		if got != wantMicros {
			t.Errorf("old speed ordinal %d migrated to a detent worth %d µs/gen, want an EXACT match at %d", oldIdx, got, wantMicros)
		}
	}
}
