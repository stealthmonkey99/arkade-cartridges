// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

// Display style ordinals; CLASSIC is the default.
const (
	styleClassic uint8 = iota
	styleGlow
	styleHeat
	styleCount // also the wrap modulus
)

// g.speed indexes speedDetentMicros. speedDetentDefault is the slider's
// centre and the fresh-storage default.
const (
	speedDetentCount   = 9
	speedDetentDefault = 5
)

// speedDetentMicros is each detent's generation cadence, from 1.25 gens/sec
// (800,000) to 50 (20,000), the fastest that still reads as individual
// generations. The slow side has finer gaps than the fast side.
var speedDetentMicros = [speedDetentCount]uint64{
	800_000, 500_000, 300_000, 200_000, 133_000,
	generationStepMicros,
	66_000, 40_000, 20_000,
}

// oldSpeedToDetent maps the 2-byte settings format's SLOW/NORMAL/FAST/TURBO
// ordinal onto a detent. The old FAST (50,000µs) fell between detents 6 and
// 7 and rounds to 7.
var oldSpeedToDetent = [4]uint8{3, 5, 7, 8}

// generationStepFor returns detent's cadence, falling back to the centre
// detent for an out-of-range value.
func generationStepFor(detent uint8) uint64 {
	if detent >= speedDetentCount {
		return speedDetentMicros[speedDetentDefault]
	}
	return speedDetentMicros[detent]
}

// The overlay rows embed their value in the label text, so a change is an
// array assignment rather than a string build.
var styleRowLabels = [styleCount]string{
	styleClassic: "STYLE: CLASSIC",
	styleGlow:    "STYLE: GLOW",
	styleHeat:    "STYLE: HEAT",
}

var speedDetentLabels = [speedDetentCount]string{
	"SPEED: 800K", "SPEED: 500K", "SPEED: 300K", "SPEED: 200K", "SPEED: 133K",
	"SPEED: NORMAL",
	"SPEED: 66K", "SPEED: 40K", "SPEED: 20K",
}

// gameplayGlow is GLOW's fade-trail buffer; attract.go's atGlow is its own
// because attract runs as a separate wasm instance.
var gameplayGlow glowGrid

// resetGameplayGlow clears the trail when the player switches into GLOW or
// commits a drawing, but not on reseeds, where the trail should fade out.
func resetGameplayGlow() { gameplayGlow = glowGrid{} }

// HEAT's per-cell envelopes: rise blends a live cell from its green birth
// flash toward white, fall dims a dead cell from its red death flash toward
// black. They are separate so a death always flashes at full brightness.
var gameplayHeatRise glowGrid
var gameplayHeatFall glowGrid

// resetGameplayHeat clears both envelopes when the player switches into HEAT.
func resetGameplayHeat() {
	gameplayHeatRise = glowGrid{}
	gameplayHeatFall = glowGrid{}
}

// stepHeat advances HEAT's envelopes one render tick. A live cell climbs
// rise and has fall cut to 0, so a respawn is a clean green flash. A dead
// cell with a nonzero rise died this tick, so fall snaps to 255 and rise
// is zeroed.
func stepHeat(cur *grid, rise, fall *glowGrid, riseStep, fallStep uint8) {
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			if cur[y][x] {
				r := rise[y][x]
				if r > 255-riseStep {
					r = 255
				} else {
					r += riseStep
				}
				rise[y][x] = r
				fall[y][x] = 0
				continue
			}
			switch {
			case rise[y][x] != 0:
				fall[y][x] = 255
				rise[y][x] = 0
			case fall[y][x] > fallStep:
				fall[y][x] -= fallStep
			default:
				fall[y][x] = 0
			}
		}
	}
}

// heatFadeColor returns HEAT's colour for one cell, and false when a dead
// cell has fully faded. A live cell blends from green toward white with rise;
// a dead cell's red channel is fall.
func heatFadeColor(rise, fall uint8, alive bool) (r, g, b uint8, ok bool) {
	if alive {
		return rise, 255, rise, true
	}
	if fall == 0 {
		return 0, 0, 0, false
	}
	return fall, 0, 0, true
}

// renderHeat advances both envelopes at the one fade rate, so a birth and a
// death take the same span, then paints every cell with anything left to
// show.
func renderHeat(fb *cartlib.Framebuffer, f *lifeField) {
	stepHeat(f.cur, &gameplayHeatRise, &gameplayHeatFall, fadeStep, fadeStep)
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			if r, g, b, ok := heatFadeColor(gameplayHeatRise[y][x], gameplayHeatFall[y][x], f.cur[y][x]); ok {
				fb.SetPixel(x, y, r, g, b)
			}
		}
	}
}

func renderClassic(fb *cartlib.Framebuffer, f *lifeField) {
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			if f.cur[y][x] {
				fb.SetPixel(x, y, 255, 255, 255)
			}
		}
	}
}

// renderField draws the current generation in the given style onto a
// framebuffer already cleared to black.
func renderField(fb *cartlib.Framebuffer, f *lifeField, style uint8) {
	switch style {
	case styleGlow:
		stepGlow(f.cur, &gameplayGlow, fadeStep)
		drawGlow(fb, &gameplayGlow)
	case styleHeat:
		renderHeat(fb, f)
	default: // styleClassic, and any out-of-range value
		renderClassic(fb, f)
	}
}
