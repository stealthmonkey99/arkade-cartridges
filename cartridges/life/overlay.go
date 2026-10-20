// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// Overlay row indices, in overlayItemsBuf slot order.
const (
	overlayRowStyle = iota
	overlayRowSpeed
	overlayRowDraw
	overlayRowReseed
)

// overlayItemsBuf is overlayMenu's backing array; rewriting a slot needs no
// SetItems call.
var overlayItemsBuf [4]string

const overlayRowDrawLabel = "DRAW"
const overlayRowReseedLabel = "RESEED"

// The LCD legend while the overlay is open. It has no key hints because the
// cabinet's buttons carry colours, not letters.
const overlayLCD1 = "MENU"
const overlayLCD2 = ""

var overlayMenu *ui.Menu
var drawPickMenu *ui.Menu

// Panel geometry shared by both menus.
const (
	overlayPanelX = 3
	overlayPanelY = 3
	overlayPanelW = cartlib.FBWidth - 2*overlayPanelX
	overlayPanelH = cartlib.FBHeight - 2*overlayPanelY
)

const (
	overlayItemH   = 9
	overlaySpacing = 1
)

// overlayTextDY nudges each label down a pixel. A row's 7 interior rows
// are centred on the LED font's 7-row bitmap, but capitals only ink rows
// 1-5 of it, so +1 leaves one blank row above and below.
const overlayTextDY = 1

// initOverlayWidgets constructs both menus; Init only.
func (g *lifeGame) initOverlayWidgets() {
	g.refreshOverlayLabels()
	overlayMenu = ui.NewMenu(overlayItemsBuf[:])
	overlayMenu.X, overlayMenu.Y, overlayMenu.W = overlayPanelX+3, overlayPanelY+11, overlayPanelW-6
	overlayMenu.ItemH, overlayMenu.Spacing = overlayItemH, overlaySpacing
	overlayMenu.TextDY = overlayTextDY
	// H <= 0 disables scrolling; all four rows fit.

	drawPickMenu = ui.NewMenu(drawPickLabels[:1])
	drawPickMenu.X, drawPickMenu.Y, drawPickMenu.W = overlayPanelX+3, overlayPanelY+11, overlayPanelW-6
	drawPickMenu.ItemH, drawPickMenu.Spacing = overlayItemH, overlaySpacing
	drawPickMenu.TextDY = overlayTextDY
	// The picker can hold more items than fit, so it needs an H to scroll.
	drawPickMenu.H = overlayPanelH - 13
}

// refreshOverlayLabels rewrites the four slots from g's current values.
func (g *lifeGame) refreshOverlayLabels() {
	overlayItemsBuf[overlayRowStyle] = styleRowLabels[g.style]
	overlayItemsBuf[overlayRowSpeed] = speedDetentLabels[g.speed]
	overlayItemsBuf[overlayRowDraw] = overlayRowDrawLabel
	overlayItemsBuf[overlayRowReseed] = overlayRowReseedLabel
}

// openOverlay layers the overlay over play and freezes the sim.
func (g *lifeGame) openOverlay() { g.machine.Push(stOverlay) }

// closeOverlay pops back to the screen underneath, whose entry hook restores
// its readout and lamp.
func (g *lifeGame) closeOverlay() { g.machine.Pop() }

// activateOverlay runs the focused row's action. On a value row, A steps the
// value as RIGHT does. DRAW opens the seed picker; RESEED installs a fresh
// world and returns to play.
func (g *lifeGame) activateOverlay() {
	switch overlayMenu.Activate() {
	case overlayRowStyle, overlayRowSpeed:
		g.adjustRow(overlayMenu.Selected(), ui.DirRight)
	case overlayRowDraw:
		g.enterDrawPicker()
	case overlayRowReseed:
		g.field.seedWorld()
		g.cancelTerminalFade()
		g.setText()
		g.closeOverlay()
	}
}

// adjustRow nudges row's value one step toward d; action rows are a no-op.
func (g *lifeGame) adjustRow(row int, d ui.Direction) {
	switch row {
	case overlayRowStyle:
		g.setStyle(nextStyle(g.style, d))
	case overlayRowSpeed:
		g.adjustSpeed(d)
	}
}

// nextStyle steps a style ordinal one place toward d, wrapping at both ends.
// Speed clamps instead, since wrapping would jump slowest to fastest.
func nextStyle(style uint8, d ui.Direction) uint8 {
	switch d {
	case ui.DirLeft:
		return (style + styleCount - 1) % styleCount
	case ui.DirRight:
		return (style + 1) % styleCount
	}
	return style
}

// applyStyle switches the display style and refreshes the row label without
// persisting, and reports whether anything moved. Entering GLOW or HEAT
// clears that style's fade buffers.
func (g *lifeGame) applyStyle(next uint8) bool {
	if next == g.style || next >= styleCount {
		return false
	}
	g.style = next
	switch g.style {
	case styleGlow:
		resetGameplayGlow()
	case styleHeat:
		resetGameplayHeat()
	}
	g.refreshOverlayLabels()
	return true
}

// setStyle is applyStyle plus a save when the value moved.
func (g *lifeGame) setStyle(next uint8) {
	if g.applyStyle(next) {
		g.saveSettings()
	}
}

// nextSpeedDetent steps the slider toward slow on LEFT and fast on RIGHT,
// clamped at both ends.
func nextSpeedDetent(speed uint8, d ui.Direction) uint8 {
	switch d {
	case ui.DirLeft:
		if speed > 0 {
			return speed - 1
		}
	case ui.DirRight:
		if speed < speedDetentCount-1 {
			return speed + 1
		}
	}
	return speed
}

// adjustSpeed moves the slider one detent and saves if it changed.
func (g *lifeGame) adjustSpeed(d ui.Direction) {
	next := nextSpeedDetent(g.speed, d)
	if next == g.speed {
		return
	}
	g.speed = next
	g.refreshOverlayLabels()
	g.saveSettings()
}

// handleOverlayInput drives one record while the overlay is open. LEFT/RIGHT
// on a value row shifts the value; B closes the overlay rather than exiting.
func (g *lifeGame) handleOverlayInput(r cartlib.InputRecord) {
	if d := r.PressDir(); d != ui.DirNone {
		row := overlayMenu.Selected()
		if (d == ui.DirLeft || d == ui.DirRight) && (row == overlayRowStyle || row == overlayRowSpeed) {
			g.adjustRow(row, d)
		} else {
			overlayMenu.Move(d)
		}
		return
	}
	switch {
	case !r.IsPress():
	case r.Key == cartlib.KeyJoyA:
		g.activateOverlay()
	case r.IsQuit():
		g.closeOverlay()
	}
}

// handleDrawPickInput drives one record while the seed picker is open.
func (g *lifeGame) handleDrawPickInput(r cartlib.InputRecord) {
	if d := r.PressDir(); d != ui.DirNone {
		drawPickMenu.Move(d)
		return
	}
	switch {
	case !r.IsPress():
	case r.Key == cartlib.KeyJoyA:
		g.confirmDrawPick()
	case r.IsQuit():
		g.machine.Pop()
	}
}

// drawOverlayBackdrop dims the frozen game and draws the panel both menus
// render inside.
func drawOverlayBackdrop(fb *cartlib.Framebuffer) {
	ui.FillRect(fb, 0, 0, cartlib.FBWidth, cartlib.FBHeight, ui.Color{R: 0, G: 0, B: 0, A: 150})
	ui.DarkEdgeBorder(overlayPanelX, overlayPanelY, overlayPanelW, overlayPanelH, ui.Color{R: 20, G: 20, B: 28, A: 255}).Draw(fb)
}

// renderOverlay draws the backdrop, title and rows, plus the slider while the
// SPEED row is focused.
func renderOverlay(fb *cartlib.Framebuffer, speed uint8) {
	drawOverlayBackdrop(fb)
	ui.Font3x5{}.DrawText(fb, overlayPanelX+3, overlayPanelY+3, "MENU", 1, ui.White)
	overlayMenu.Draw(fb)
	if overlayMenu.Selected() == overlayRowSpeed {
		drawSpeedSlider(fb, speed)
	}
}

func renderDrawPicker(fb *cartlib.Framebuffer) {
	drawOverlayBackdrop(fb)
	ui.Font3x5{}.DrawText(fb, overlayPanelX+3, overlayPanelY+3, "SEED", 1, ui.White)
	drawPickMenu.Draw(fb)
}

// The slider's strip sits directly below the menu's four rows.
const (
	speedSliderY = overlayPanelY + 11 + 4*overlayItemH + 3*overlaySpacing
	speedSliderH = 7
)

const (
	speedSliderHandleW = 3
	speedSliderHandleH = 5
	speedSliderTickH   = 3
)

// Slider palette, distinct from ui.Button's fills.
var (
	sliderTrackColor      = ui.Color{R: 70, G: 70, B: 80, A: 255}
	sliderTickColor       = ui.Color{R: 130, G: 130, B: 145, A: 255}
	sliderCenterTickColor = ui.Color{R: 210, G: 190, B: 80, A: 255} // amber "home" marker at NORMAL
	sliderHandleColor     = ui.Color{R: 235, G: 235, B: 245, A: 255}
	sliderHandleStroke    = ui.Color{R: 40, G: 40, B: 50, A: 255}
)

// speedDetentX maps a detent index onto its handle's left edge, with the
// extremes flush with the track ends.
func speedDetentX(trackX, trackW, i int) int {
	travel := trackW - speedSliderHandleW
	return trackX + i*travel/(speedDetentCount-1)
}

// drawSpeedSlider paints the track, a tick per detent (the centre one
// highlighted) and the handle.
func drawSpeedSlider(fb *cartlib.Framebuffer, speed uint8) {
	trackX, trackW := overlayMenu.X, overlayMenu.W
	trackY := speedSliderY + speedSliderH/2

	ui.Rect{X: trackX, Y: trackY, W: trackW, H: 1, Fill: sliderTrackColor}.Draw(fb)

	for i := 0; i < speedDetentCount; i++ {
		tickX := speedDetentX(trackX, trackW, i) + speedSliderHandleW/2
		tickColor := sliderTickColor
		if i == speedDetentDefault {
			tickColor = sliderCenterTickColor
		}
		ui.Rect{X: tickX, Y: trackY - speedSliderTickH/2, W: 1, H: speedSliderTickH, Fill: tickColor}.Draw(fb)
	}

	idx := int(speed)
	if idx >= speedDetentCount {
		idx = speedDetentDefault // defensive; settings.go already clamps
	}
	hx := speedDetentX(trackX, trackW, idx)
	ui.Rect{
		X: hx, Y: speedSliderY, W: speedSliderHandleW, H: speedSliderHandleH,
		Fill: sliderHandleColor, Stroke: sliderHandleStroke, StrokeW: 1,
	}.Draw(fb)
}
