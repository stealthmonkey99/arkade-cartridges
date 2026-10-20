// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

// Panels and lamps go to the seat (a record's Physical); steering goes by
// pairing slot (Player). They part once someone picks up the other pad.

const (
	titleText1    = "SNAKE"
	titleText2    = "SELECT MODE" // the buttons are unlabelled, so name the verb
	getReadyText1 = "GET"
	getReadyText2 = "READY!"
	gameOverText1 = "GAME OVER"
	winText1      = "YOU WIN!"
)

var lcdLine1Buf [24]byte
var lcdLine2Buf [24]byte

const heartBytes = "♥"

func showTitlePeripherals() {
	cartlib.SetText(cartlib.SeatBoth, titleText1, titleText2)
	cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA, true)
}

func enterGetReadyPeripherals() {
	cartlib.SetLights(cartlib.SeatBoth, false)
	cartlib.SetText(cartlib.SeatBoth, "", "")
	for i := 0; i < numPlayers; i++ {
		cartlib.SetText(seats[i], getReadyText1, getReadyText2)
	}
}

// buildLcdLine1 writes "Lives: ♥ ♥ ♥" and the level on one 16-column line.
// At most 3 lives and a one-digit level, so the two never collide.
func buildLcdLine1(buf []byte, lives, lvl int) int {
	n := copy(buf, "Lives: ")
	leftRunes := 7
	for i := 0; i < lives; i++ {
		if i > 0 {
			buf[n] = ' '
			n++
			leftRunes++
		}
		n += copy(buf[n:], heartBytes)
		leftRunes++
	}
	for leftRunes < 14 {
		buf[n] = ' '
		n++
		leftRunes++
	}
	buf[n] = 'L'
	n++
	buf[n] = byte('0' + lvl)
	n++
	return n
}

// buildLcdLine2 writes the score and "high: n" on one 16-column line.
func buildLcdLine2(buf []byte, score, highScore uint32) int {
	high := score
	if highScore > high {
		high = highScore
	}

	var highBuf [24]byte
	hn := copy(highBuf[:], "high: ")
	hn = cartlib.AppendUint(highBuf[:], hn, high)

	var scoreBuf [16]byte
	sn := cartlib.AppendUint(scoreBuf[:], 0, score)

	var padded [16]byte
	pn := copy(padded[:], scoreBuf[:sn])
	for i := pn; i < 16; i++ {
		padded[i] = ' '
	}

	keep := 16 - hn
	if keep < 0 {
		keep = 0
	}
	if keep > 16 {
		keep = 16
	}
	n := copy(buf, padded[:keep])
	n += copy(buf[n:], highBuf[:hn])
	return n
}

// updatePlayLcd shows a running effect's countdown in place of the score:
// there is no room for both, and the field itself never shows the countdown.
func updatePlayLcd(i int) {
	s := &snakes[i]
	n1 := buildLcdLine1(lcdLine1Buf[:], s.lives, field.level)
	var n2 int
	if fx := soloEffects(); fx != nil && fx.effect != pickupNone {
		n2 = buildEffectLcdLine(lcdLine2Buf[:], fx.effect, fx.shownSeconds)
	} else {
		n2 = buildLcdLine2(lcdLine2Buf[:], s.score, highScore)
	}
	cartlib.SetTextBytes(seats[i], lcdLine1Buf[:n1], lcdLine2Buf[:n2])
}

func updateNameEntryLcd() {
	banner := "TOP 10!"
	if cartlib.ScoreRank(int(eligibleScore)) == 1 {
		banner = "NEW HIGH!"
	}
	n1 := copy(lcdLine1Buf[:], banner)
	n2 := buildLcdLine2(lcdLine2Buf[:], eligibleScore, highScore)
	cartlib.SetTextBytes(eligibleSeat, lcdLine1Buf[:n1], lcdLine2Buf[:n2])
}

func updateGameOverLcd() {
	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		label := gameOverText1
		if numPlayers == 2 && i == winnerIndex {
			label = winText1
		}
		n1 := copy(lcdLine1Buf[:], label)
		n2 := buildLcdLine2(lcdLine2Buf[:], s.score, highScore)
		cartlib.SetTextBytes(seats[i], lcdLine1Buf[:n1], lcdLine2Buf[:n2])
	}
}

const blinkPeriodMicros = 500_000 // game-over A lamp, half period

var gameOverLightsOn bool

func tickGameOver() {
	on := (machine.Elapsed()/blinkPeriodMicros)%2 == 0
	if on == gameOverLightsOn {
		return
	}
	gameOverLightsOn = on
	if on {
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA, true)
	} else {
		cartlib.SetButtonLights(cartlib.SeatBoth, 0, true)
	}
}
