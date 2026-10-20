// SPDX-License-Identifier: GPL-2.0-or-later

// Command life is Conway's Game of Life for the Årkade cabinet, built on
// cartlib.Game. The simulation runs as soon as Init returns; A opens a
// settings overlay (display style, speed) and a DRAW mode for hand-drawn
// seeds. Time comes only from Update's dtMicros and randomness only from
// cartlib.RandomU64, so a scripted run is reproducible frame for frame.
package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/fsm"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

//go:generate go run github.com/stealthmonkey99/arkade/tools/genlevelpack .
//go:generate go run github.com/stealthmonkey99/arkade/sdk/tools/manifestgen .

// generationStepMicros is the NORMAL speed setting's cadence, and the
// slider's centre detent in style.go.
const generationStepMicros uint64 = 100_000

// Screens HandleInput, Update and Render dispatch on. The overlay and the
// picker are modals pushed over the frozen world, so backing out lands on
// whatever they covered.
const (
	stPlaying  fsm.State = iota // the running simulation; A opens the overlay
	stOverlay                   // the top-level STYLE/SPEED/DRAW/RESEED list
	stDrawPick                  // DRAW's "pick a starting canvas" sub-list
	stDraw                      // the cursor-driven pattern editor
)

// lifeEdges lists the transitions Push and Go may take; Pop is not an edge.
var lifeEdges = []uint16{
	stPlaying:  1 << stOverlay,
	stOverlay:  1 << stDrawPick,
	stDrawPick: 1 << stDraw,
	stDraw:     0,
}

// lifeGame implements cartlib.Game.
type lifeGame struct {
	field     lifeField
	accMicros uint64 // dtMicros accumulated toward the next generation step
	lcdMicros uint64 // dtMicros accumulated toward the next LCD refresh
	lcdGen    uint32 // the generation the LCD is currently showing

	machine fsm.Machine

	style uint8 // style.go's styleClassic/styleGlow/styleHeat
	speed uint8 // index into style.go's speedDetentMicros

	// Terminal-state fade (fade.go); a modifier of stPlaying, not a screen.
	fading     bool
	fadeMicros uint64

	// DRAW session state, zeroed on entry to stDraw.
	drawCX, drawCY int
	drawHeld       ui.HeldDirs
	drawPrevDir    ui.Direction
	drawMoveAccum  uint64
	drawBlinkAccum uint64
}

func (g *lifeGame) Manifest() []byte { return manifestBytes }

// Init seeds the first world and turns the cabinet lights off. loadSettings
// and loadDrawings must not draw RandomU64, or they would shift the draw
// seedWorld consumes.
func (g *lifeGame) Init(s *cartlib.Setup) {
	loadSounds(s)
	pendingLoaderBloom = true
	g.loadSettings()
	loadDrawings()
	g.field.seedWorld()
	cartlib.SetLights(cartlib.SeatBoth, false)
	g.initOverlayWidgets()
	g.machine.OnEnter(g.enterScreen)
	g.machine.Edges(lifeEdges)
	g.machine.Reset(stPlaying)
}

// enterScreen sets each screen's LCD legend, button lamps and fresh session
// state. It runs on every arrival, including a Pop back to a covered screen.
func (g *lifeGame) enterScreen(prev, next fsm.State) {
	switch next {
	case stPlaying:
		g.setText()
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA, true)
	case stOverlay:
		cartlib.SetText(cartlib.SeatBoth, overlayLCD1, overlayLCD2)
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA|cartlib.MaskJoyB, true)
	case stDrawPick:
		// Title only, no key hints (see overlayLCD2).
		cartlib.SetText(cartlib.SeatBoth, "PICK A SEED", "")
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA|cartlib.MaskJoyB, true)
	case stDraw:
		g.drawCX, g.drawCY = drawGridSize/2, drawGridSize/2
		g.drawHeld.Reset()
		g.drawPrevDir = ui.DirNone
		g.drawMoveAccum = 0
		g.drawBlinkAccum = 0
		// Two lit lamps cannot identify a button, so the LCD names colours.
		cartlib.SetText(cartlib.SeatBoth, "DRAW: GREEN", "YELLOW = RUN IT")
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA|cartlib.MaskJoyC, true)
	}
}

// OwnsQuitKeys keeps cartlib from answering B with an immediate exit: B
// backs out of Life's overlays, and only the running field's B quits.
func (g *lifeGame) OwnsQuitKeys() bool { return true }

func (g *lifeGame) HandleInput(records []cartlib.InputRecord) {
	for _, r := range records {
		switch g.machine.State() {
		case stOverlay:
			g.handleOverlayInput(r)
		case stDrawPick:
			g.handleDrawPickInput(r)
		case stDraw:
			g.handleDrawInput(r)
		default: // stPlaying
			g.handlePlayingInput(r)
		}
	}
}

// handlePlayingInput opens the overlay on A and exits on B. Exit is
// immediate, not ui.HoldConfirm: Life has no score or progress to lose.
func (g *lifeGame) handlePlayingInput(r cartlib.InputRecord) {
	switch {
	case !r.IsPress():
	case r.Key == cartlib.KeyJoyA:
		g.openOverlay()
	case r.IsQuit():
		cartlib.Exit()
	}
}

// Update advances one generation per whole step of accumulated dtMicros,
// so a slow host tick can advance several. The sim is frozen on every
// screen but stPlaying, and while a terminal world fades. No machine.Tick:
// every clock here is an accumulator, so nothing reads fsm.Elapsed.
func (g *lifeGame) Update(dtMicros uint64) {
	switch g.machine.State() {
	case stOverlay:
		overlayMenu.Update(dtMicros)
	case stDrawPick:
		drawPickMenu.Update(dtMicros)
	case stDraw:
		g.updateDraw(dtMicros)
	default: // stPlaying
		if g.fading {
			g.updateTerminalFade(dtMicros)
			return
		}
		g.accMicros += dtMicros
		step := generationStepFor(g.speed)
		for g.accMicros >= step {
			g.accMicros -= step
			if g.field.nextGen() && g.beginTerminalFade() {
				return
			}
			if g.field.emptied {
				// Extinction is shown at once, ahead of the LCD throttle.
				g.setText()
			}
		}
		g.tickLcd(dtMicros)
	}
}

// Render draws the current screen. DRAW replaces the field view, since
// drawBuf is unrelated to the field. The fade veil goes under any overlay
// panel so a menu opened mid-fade stays at full contrast.
func (g *lifeGame) Render(fb *cartlib.Framebuffer) {
	if g.machine.State() == stDraw {
		renderDrawEditor(fb, g.drawCX, g.drawCY, g.drawCursorOn())
		return
	}

	fb.Clear(0, 0, 0)
	renderField(fb, &g.field, g.style)
	if g.fading {
		drawTerminalFade(fb, g.fadeMicros)
	}

	switch g.machine.State() {
	case stOverlay:
		renderOverlay(fb, g.speed)
	case stDrawPick:
		renderDrawPicker(fb)
	}
}

// Static LCD line buffers: no allocation after Init.
var (
	genPrefix = []byte("Gen: ")
	popPrefix = []byte("Pop: ")
	genBuf    [16]byte
	popBuf    [16]byte
)

// lcdIntervalMicros throttles the Gen/Pop readout to 4 Hz, the authoring
// guide's ceiling for LCD updates. Every transition out of the running sim
// calls setText, so the readout is never stale.
const lcdIntervalMicros uint64 = 250_000

// tickLcd is the throttled LCD push; setText is the unthrottled one.
func (g *lifeGame) tickLcd(dtMicros uint64) {
	g.lcdMicros += dtMicros
	if g.lcdMicros < lcdIntervalMicros {
		return
	}
	if uint32(g.field.gen) == g.lcdGen {
		// Below 4 gens/sec the throttle fires between generations.
		return
	}
	g.setText()
}

// setText pushes "Gen: n" / "Pop: n" to both LCDs immediately, without
// allocating.
func (g *lifeGame) setText() {
	g.lcdMicros = 0
	g.lcdGen = uint32(g.field.gen)

	gn := copy(genBuf[:], genPrefix)
	gn = cartlib.AppendUint(genBuf[:], gn, uint32(g.field.gen))

	pn := copy(popBuf[:], popPrefix)
	pn = cartlib.AppendUint(popBuf[:], pn, uint32(g.field.population()))

	cartlib.SetTextBytes(cartlib.SeatBoth, genBuf[:gn], popBuf[:pn])
}

// Registered at package level because main is never called (cartlib's
// "The reactor model").
var game = &lifeGame{}
var _ = cartlib.Run(game)

// cart_attract opts Life into drawing its own screensaver.
//
//export cart_attract
func cartAttract(dtMicros uint64) { cartlib.DispatchAttract(dtMicros) }

func main() {}
