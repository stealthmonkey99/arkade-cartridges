// SPDX-License-Identifier: GPL-2.0-or-later

// Command snake is the Årkade Snake cartridge: a single-player run, a
// two-player shared arena, temporary pickups, and high-score name entry.
//
// The build is TinyGo, then the wasmfix post-pass that turns cart_manifest's
// hidden output pointer into a real two-result export (make in the repo root).
//
// Nothing allocates after Init except a few string conversions, which
// TinyGo's -print-allocs report cannot see; each runs once per qualifying
// round or rarer.
package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

//go:generate go run github.com/stealthmonkey99/arkade/tools/genlevelpack .
//go:generate go run github.com/stealthmonkey99/arkade/sdk/tools/manifestgen .

// snakeGame is a stateless shell over the package-level game state.
type snakeGame struct{}

func (snakeGame) Manifest() []byte { return manifestBytes }

// OwnsQuitKeys keeps cartlib from answering B with an immediate exit:
// snake runs its own hold-to-quit confirm.
func (snakeGame) OwnsQuitKeys() bool { return true }

func (snakeGame) Init(s *cartlib.Setup) {
	// Unlocked, the solo strategy takes either controller and never rotates
	// the display: what the title menu and a 2P run both need. A 1P pick locks it.
	cartlib.PushPlayerStrategy(cartlib.StrategyOnePlayerSolo)

	loadHighScore()
	loadSounds(s)
	loadSprites(s)
	initWidgets()
	machine.OnEnter(enterScreen)
	machine.Edges(legalEdges)
	// Reset fires the title's entry hook, which sets the LCD and lamps.
	machine.Reset(stateTitle)
}

func (snakeGame) HandleInput(records []cartlib.InputRecord) {
	for _, r := range records {
		handleInputRecord(r)
	}
}

func (snakeGame) Update(dtMicros uint64) { update(dtMicros) }

func (snakeGame) Render(fb *cartlib.Framebuffer) { renderFrame(fb) }

// Registered at package level: under the reactor model main is never called.
var game snakeGame
var _ = cartlib.Run(game)

// cart_attract's presence is the opt-in to drawing Snake's own screensaver.
//
//export cart_attract
func cartAttract(dtMicros uint64) { cartlib.DispatchAttract(dtMicros) }

func main() {}
