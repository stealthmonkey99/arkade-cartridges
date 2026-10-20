// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

// Attract is a slow Conway simulation with a glow trail over the engine's
// starfield. It reseeds a terminal world at once, since the trail already
// dissolves the old one.

const attractGenStepMicros uint64 = 450_000

// fadeStep is how far a glow or heat envelope moves per render call: 6 at
// ~30 FPS fades a cell over ~1.4s. Every fade in this cartridge runs at it.
const fadeStep = 6

var atGlow [cartlib.FBHeight][cartlib.FBWidth]uint8

// Attract implements cartlib.Attractor.
func (g *lifeGame) Attract(dtMicros uint64, fb *cartlib.Framebuffer) {
	g.accMicros += dtMicros
	for g.accMicros >= attractGenStepMicros {
		g.accMicros -= attractGenStepMicros
		if g.field.nextGen() {
			g.field.seedWorld()
		}
		g.setText()
	}

	fb.ClearTransparent()
	stepGlow(g.field.cur, &atGlow, fadeStep)
	drawGlow(fb, &atGlow)
}

// glowGrid is the per-cell brightness buffer for the glow trails.
type glowGrid = [cartlib.FBHeight][cartlib.FBWidth]uint8

// stepGlow advances one fade-trail tick: a live cell snaps to full
// brightness, everything else dims by step.
func stepGlow(cur *grid, glow *glowGrid, step uint8) {
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			b := glow[y][x]
			switch {
			case cur[y][x]:
				b = 255
			case b > step:
				b -= step
			default:
				b = 0
			}
			glow[y][x] = b
		}
	}
}

// drawGlow paints the trail in green with a faint blue lift. Fully faded
// cells are skipped, leaving the starfield or cleared black showing.
func drawGlow(fb *cartlib.Framebuffer, glow *glowGrid) {
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			b := glow[y][x]
			if b == 0 {
				continue
			}
			fb.SetPixel(x, y, 0, b, uint8(uint32(b)/4))
		}
	}
}
