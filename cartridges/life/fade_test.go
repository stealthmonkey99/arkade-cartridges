// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
)

// TestTerminalFadeAlphaRamp checks the veil is clear at 0, opaque by
// terminalFadeMicros, monotonic between, and clamped rather than wrapping.
func TestTerminalFadeAlphaRamp(t *testing.T) {
	if got := terminalFadeAlpha(0); got != 0 {
		t.Errorf("terminalFadeAlpha(0) = %d, want 0 (the fade starts fully clear)", got)
	}
	if got := terminalFadeAlpha(terminalFadeMicros); got != 255 {
		t.Errorf("terminalFadeAlpha(terminalFadeMicros) = %d, want 255", got)
	}
	if got := terminalFadeAlpha(terminalFadeMicros * 10); got != 255 {
		t.Errorf("terminalFadeAlpha(10x the duration) = %d, want 255 (clamped, never wrapped)", got)
	}
	if got := terminalFadeAlpha(terminalFadeMicros / 2); got < 120 || got > 135 {
		t.Errorf("terminalFadeAlpha(halfway) = %d, want roughly half opaque (~127)", got)
	}

	prev := terminalFadeAlpha(0)
	for step := uint64(1); step <= 50; step++ {
		got := terminalFadeAlpha(step * terminalFadeMicros / 50)
		if got < prev {
			t.Fatalf("alpha fell from %d to %d at step %d/50 — the ramp must never brighten mid-fade", prev, got, step)
		}
		prev = got
	}
}

func TestDrawTerminalFadeDarkensTheWholeFrame(t *testing.T) {
	var fb cartlib.Framebuffer
	fb.Clear(200, 200, 200)
	drawTerminalFade(&fb, terminalFadeMicros/2)

	for _, p := range [][2]int{{0, 0}, {63, 0}, {0, 63}, {63, 63}, {32, 32}} {
		r, g, b := pixelAt(&fb, p[0], p[1])
		if r >= 200 || g >= 200 || b >= 200 {
			t.Errorf("pixel (%d,%d) = (%d,%d,%d) halfway through the fade, want visibly darker than the 200-grey it started at", p[0], p[1], r, g, b)
		}
	}

	fb.Clear(200, 200, 200)
	drawTerminalFade(&fb, terminalFadeMicros)
	if r, g, b := pixelAt(&fb, 32, 32); r != 0 || g != 0 || b != 0 {
		t.Errorf("pixel (32,32) = (%d,%d,%d) at the end of the fade, want fully black", r, g, b)
	}
}
