// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

const (
	pickupNone uint8 = iota
	pickupChili
	pickupIce
	pickupShrink
	pickupGhost
	pickupGold

	numPickupKinds = 5
)

const (
	// pickupInitialApples is a constant, not a draw, so a run that eats
	// fewer apples consumes no RandomU64 values for pickups at all.
	pickupInitialApples = 8

	pickupMinApples   = 8 // re-rolled at each spawn into [8, 15]
	pickupApplesRange = 8

	pickupWindowMicros      = 8_000_000
	pickupBlinkStartMicros  = 2_500_000
	pickupBlinkPeriodMicros = 200_000 // half-cycle

	effectMicros           = 20_000_000 // chili, ice, ghost and gold alike
	effectBlinkStartMicros = 3_000_000

	// Gold pays no bonus: its payoff is the apple multiplier.
	chiliBonusPoints  = 25
	iceBonusPoints    = 10
	ghostBonusPoints  = 10
	shrinkBonusPoints = 10

	shrinkMinLen = initLength

	boostStepMicros = stepMicros / 2
	slowStepMicros  = stepMicros * 2
)

type countdown struct{ left uint64 }

func (c *countdown) start(micros uint64) { c.left = micros }
func (c *countdown) stop()               { c.left = 0 }
func (c *countdown) running() bool       { return c.left > 0 }

func (c *countdown) tick(dtMicros uint64) (expired bool) {
	if c.left == 0 {
		return false
	}
	if dtMicros >= c.left {
		c.left = 0
		return true
	}
	c.left -= dtMicros
	return false
}

// pickupState holds at most one item on the field and one running effect.
type pickupState struct {
	kind   uint8 // meaningful while window runs
	x, y   uint8
	window countdown

	applesToGo int

	effect       uint8 // never pickupShrink, which is instant
	effectLeft   countdown
	shownSeconds int // last value pushed to the LCD

	pulseMicros uint64

	announced    uint8
	announceLeft countdown
}

var pickups pickupState

// soloEffects is nil in 2P: pickups are a 1P feature.
func soloEffects() *pickupState {
	if numPlayers != 1 {
		return nil
	}
	return &pickups
}

func (p *pickupState) reset() { *p = pickupState{applesToGo: pickupInitialApples} }

func currentStepMicros() uint64 {
	if fx := soloEffects(); fx != nil {
		switch fx.effect {
		case pickupChili:
			return boostStepMicros
		case pickupIce:
			return slowStepMicros
		}
	}
	return stepMicros
}

func applePoints() uint32 {
	if fx := soloEffects(); fx != nil && fx.effect == pickupGold {
		return growPerEat * 3
	}
	return growPerEat
}

// ghosting bypasses body collisions only; walls always kill.
func ghosting() bool {
	fx := soloEffects()
	return fx != nil && fx.effect == pickupGhost
}

// onAppleEaten leaves the countdown at zero when no cell is free, so the
// next apple retries.
func (p *pickupState) onAppleEaten() {
	if p.applesToGo > 0 {
		p.applesToGo--
	}
	if p.applesToGo <= 0 && !p.window.running() {
		p.spawn()
	}
}

// placePickup is placeFood's draw kept clear of the apple, so the pickup's
// sprite never covers it.
func placePickup(b *board) (at cell, ok bool) {
	avoid := clearOf(b.food)
	free, _, _ := b.freeScan(b.level, avoid, -1)
	if free <= 0 {
		return cell{}, false
	}
	_, at, ok = b.freeScan(b.level, avoid, int(cartlib.RandomU64()%uint64(free)))
	return at, ok
}

// spawn draws kind, position, then the next cadence, in that order; the
// gold journey's script follows it. With no free cell it stops after
// the kind.
func (p *pickupState) spawn() {
	kind := pickupChili + uint8(cartlib.RandomU64()%numPickupKinds)
	at, ok := placePickup(&field)
	if !ok {
		return
	}
	p.kind = kind
	p.x, p.y = at.x, at.y
	p.window.start(pickupWindowMicros)
	cartlib.PlaySound(soundPickupSpawn, 0)
	p.applesToGo = pickupMinApples + int(cartlib.RandomU64()%pickupApplesRange)
}

// footprint is what the apple must stay clear of: nothing in 2P or while no
// item is on the field.
func (p *pickupState) footprint() keepClear {
	if soloEffects() == nil || !p.window.running() {
		return keepClear{}
	}
	return clearOf(cell{x: p.x, y: p.y})
}

func (p *pickupState) collidesAt(x, y int) bool {
	return p.window.running() && uint8(x) == p.x && uint8(y) == p.y
}

// revalidate moves an item the level swap just walled in, or drops it if
// there is nowhere else to go.
func (p *pickupState) revalidate(b *board) {
	if !p.window.running() || !wallAt(b.level, int(p.x), int(p.y)) {
		return
	}
	if at, ok := placePickup(b); ok {
		p.x, p.y = at.x, at.y
		return
	}
	p.window.stop()
}

// collect replaces any running effect; shrink cancels it outright.
func (p *pickupState) collect(s *snake) {
	kind := p.kind
	p.window.stop()
	cartlib.PlaySound(soundPickupCollect, 0)

	switch kind {
	case pickupChili:
		p.startEffect(pickupChili, effectMicros)
		s.score += chiliBonusPoints
	case pickupIce:
		p.startEffect(pickupIce, effectMicros)
		s.score += iceBonusPoints
	case pickupGhost:
		p.startEffect(pickupGhost, effectMicros)
		s.score += ghostBonusPoints
	case pickupGold:
		p.startEffect(pickupGold, effectMicros)
	case pickupShrink:
		p.startEffect(pickupNone, 0)
		shrinkSnake(s, shrinkCutCount(s.snakeLen))
		s.score += shrinkBonusPoints
	}
	p.shownSeconds = ceilSeconds(p.effectLeft.left)
	updatePlayLcd(0)

	p.announced = kind
	p.announceLeft.start(announceDurationMicros)
}

func (p *pickupState) startEffect(kind uint8, micros uint64) {
	p.effect = kind
	p.effectLeft.start(micros)
}

// shrinkCutCount is length/3 rounded to nearest; shrinkSnake still floors
// the cut at shrinkMinLen.
func shrinkCutCount(length int) int {
	n, r := length/3, length%3
	if r >= 2 {
		n++
	}
	return n
}

func shrinkSnake(s *snake, n int) {
	for k := 0; k < n && s.snakeLen > shrinkMinLen; k++ {
		s.dropTail()
	}
}

// tick runs only while playing, so timers pause through the freeze and the
// exit dialog.
func (p *pickupState) tick(dtMicros uint64) {
	p.pulseMicros += dtMicros
	p.window.tick(dtMicros)
	p.announceLeft.tick(dtMicros)

	if p.effect == pickupNone {
		return
	}
	if p.effectLeft.tick(dtMicros) {
		p.effect = pickupNone
		p.shownSeconds = 0
		updatePlayLcd(0)
		return
	}
	if secs := ceilSeconds(p.effectLeft.left); secs != p.shownSeconds {
		p.shownSeconds = secs
		updatePlayLcd(0)
	}
}

func ceilSeconds(micros uint64) int {
	return int((micros + 999_999) / 1_000_000)
}

func pickupName(kind uint8) string {
	switch kind {
	case pickupChili:
		return "CHILI BOOST"
	case pickupIce:
		return "ICE SLOW"
	case pickupGhost:
		return "GHOST"
	case pickupGold:
		return "GOLD X3"
	}
	return ""
}

func buildEffectLcdLine(buf []byte, kind uint8, secs int) int {
	n := copy(buf, pickupName(kind))
	buf[n] = ' '
	n++
	n = cartlib.AppendUint(buf, n, uint32(secs))
	buf[n] = 's'
	n++
	return n
}

// pickupColor is the kind's cel colour, so the border tint and the
// announcement follow the art.
func pickupColor(kind uint8) cartlib.Color {
	if kind < pickupChili || kind > pickupGold {
		return cartlib.Color{}
	}
	return pickupTints[kind-pickupChili]
}

// pulseFraction is an integer triangle wave in [0,255]: no float trig per frame.
func pulseFraction(accum, periodMicros uint64) uint8 {
	half := periodMicros / 2
	phase := accum % periodMicros
	if phase < half {
		return uint8(255 * phase / half)
	}
	return uint8(255 * (periodMicros - phase) / half)
}

func scaleColor(c cartlib.Color, num, den uint32) cartlib.Color {
	return cartlib.Color{
		R: byte(uint32(c.R) * num / den),
		G: byte(uint32(c.G) * num / den),
		B: byte(uint32(c.B) * num / den),
		A: 255,
	}
}

// drawItem blinks on and off near expiry, unlike ghost's continuous pulse,
// so "about to expire" reads differently from "ghost".
func (p *pickupState) drawItem(fb *cartlib.Framebuffer) {
	if !p.window.running() {
		return
	}
	if left := p.window.left; left < pickupBlinkStartMicros {
		if (left/pickupBlinkPeriodMicros)%2 != 0 {
			return
		}
	}
	tint := ui.White
	if p.kind == pickupGhost {
		frac := pulseFraction(p.pulseMicros, 1_000_000)
		tint = scaleColor(tint, 128+uint32(frac)/2, 255)
	}
	drawTinted(fb, pickupCels[p.kind-pickupChili], int(p.x)-1, int(p.y)-1, tint)
}

// wallColorForEffect tints the border with the running effect. Ghost's white
// is dimmed and never pulses, so the walls still read as solid: ghost does
// not pass through them.
func wallColorForEffect(fx *pickupState) cartlib.Color {
	if fx == nil || fx.effect == pickupNone {
		return colWall
	}
	col := pickupColor(fx.effect)
	if fx.effect == pickupGhost {
		col = ghostDim(col)
	}
	if left := fx.effectLeft.left; left < effectBlinkStartMicros {
		if (left/pickupBlinkPeriodMicros)%2 != 0 {
			return colWall
		}
	}
	return col
}

// ghostDim stands in for translucency: SetPixel has no alpha.
func ghostDim(c cartlib.Color) cartlib.Color { return scaleColor(c, 3, 5) }

const (
	announceDurationMicros = 1_200_000
	announceFadeMicros     = 400_000
	announceY              = 2 // a fixed row, not the collect position
)

func announceLabel(kind uint8) string {
	switch kind {
	case pickupChili:
		return "2X SPEED"
	case pickupIce:
		return "1/2 SPEED"
	case pickupShrink:
		return "CUT SHORT"
	case pickupGhost:
		return "INVINCIBLE"
	case pickupGold:
		return "3X POINTS"
	}
	return ""
}

func (p *pickupState) drawAnnounce(fb *cartlib.Framebuffer) {
	if !p.announceLeft.running() {
		return
	}
	col := pickupColor(p.announced)
	if left := p.announceLeft.left; left < announceFadeMicros {
		col = scaleColor(col, uint32(left), uint32(announceFadeMicros))
	}
	drawAnnounceText(fb, announceLabel(p.announced), col)
}

// drawAnnounceText draws per character because ui.Font3x5 has no '/'; the
// slash is a cel.
func drawAnnounceText(fb *cartlib.Framebuffer, s string, col cartlib.Color) {
	f := ui.Font3x5{}
	x := centerX(f.TextWidth(len(s), 1))
	for i := 0; i < len(s); i++ {
		cx := x + i*f.Advance(1)
		if s[i] == '/' {
			drawTinted(fb, slashGlyph, cx, announceY, col)
			continue
		}
		f.DrawText(fb, cx, announceY, s[i:i+1], 1, col)
	}
}
