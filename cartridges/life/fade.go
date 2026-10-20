// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// terminalFadeMicros matches the span GLOW's and HEAT's embers fade over
// (fadeStep), so the veil and the embers finish together.
const terminalFadeMicros uint64 = 1_400_000

// terminalFadeAlpha ramps the veil linearly from clear to opaque.
func terminalFadeAlpha(elapsed uint64) uint8 {
	if elapsed >= terminalFadeMicros {
		return 255
	}
	return uint8(elapsed * 255 / terminalFadeMicros)
}

// drawTerminalFade composites a black veil over the rendered field, so all
// three styles fade alike.
func drawTerminalFade(fb *cartlib.Framebuffer, elapsed uint64) {
	ui.FillRect(fb, 0, 0, cartlib.FBWidth, cartlib.FBHeight,
		ui.Color{R: 0, G: 0, B: 0, A: terminalFadeAlpha(elapsed)})
}

// beginTerminalFade starts the dissolve on a world nextGen reported terminal
// and reports whether it did. An empty board reseeds at once, since there is
// nothing to dissolve. The generation counter is kept so the LCD describes
// the fading world, and accMicros is cleared so the next world starts on a
// full step.
func (g *lifeGame) beginTerminalFade() bool {
	if g.field.population() == 0 {
		g.field.seedWorld()
		g.setText()
		return false
	}
	g.fading = true
	g.fadeMicros = 0
	g.accMicros = 0
	g.setText()
	return true
}

// updateTerminalFade advances the fade and reseeds when it completes. No
// generation advances meanwhile.
func (g *lifeGame) updateTerminalFade(dtMicros uint64) {
	g.fadeMicros += dtMicros
	if g.fadeMicros < terminalFadeMicros {
		return
	}
	g.fading = false
	g.fadeMicros = 0
	g.field.seedWorld()
	g.setText()
}

// cancelTerminalFade abandons a fade without reseeding, for paths that
// install their own world.
func (g *lifeGame) cancelTerminalFade() {
	g.fading = false
	g.fadeMicros = 0
}
