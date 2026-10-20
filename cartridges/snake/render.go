// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-snake-cartridge/arena"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

var (
	colSnake = cartlib.Color{R: 0, G: 255, B: 0, A: 255}
	colFood  = cartlib.Color{R: 255, G: 0, B: 0, A: 255}
	colWall  = cartlib.Color{R: 0, G: 183, B: 239, A: 255}
	colText  = cartlib.Color{R: 255, G: 255, B: 255, A: 255}

	// 2P matches the cabinet joysticks; the apple goes green to read against both.
	colSnakeP1 = cartlib.Color{R: 235, G: 45, B: 45, A: 255}
	colSnakeP2 = cartlib.Color{R: 80, G: 175, B: 255, A: 255}
	colFood2P  = cartlib.Color{R: 0, G: 220, B: 0, A: 255}
)

func snakeColor(i int) cartlib.Color {
	if numPlayers == 2 {
		if i == 1 {
			return colSnakeP2
		}
		return colSnakeP1
	}
	return colSnake
}

func foodColor() cartlib.Color {
	if numPlayers == 2 {
		return colFood2P
	}
	return colFood
}

func centerX(widthPx int) int { return (arena.Cols - widthPx) / 2 }

func renderFrame(fb *cartlib.Framebuffer) {
	fb.Clear(0, 0, 0)
	switch machine.State() {
	case stateTitle:
		renderTitle(fb)
	case stateGetReady:
		renderGetReady(fb)
	case statePlaying, stateLostLife:
		renderPlaying(fb)
	case stateConfirmExit:
		if machine.Under() == stateNameEntry {
			renderNameEntry(fb)
		} else {
			renderPlaying(fb)
		}
		confirmExit.Draw(fb)
	case stateGameOver:
		renderGameOver(fb)
	case stateNameEntry:
		renderNameEntry(fb)
	}
}

// Font3x5 measures in character cells; each label counts its trailing space.
const (
	hiLabelCells    = len("HI ")
	scoreLabelCells = len("SCORE ")
)

func drawHiScore(fb *cartlib.Framebuffer, y int) {
	f := ui.Font3x5{}
	hiW := f.TextWidth(hiLabelCells, 1) + 1 + f.TextWidth(f.NumDigits(highScore), 1)
	x := centerX(hiW)
	f.DrawText(fb, x, y, "HI", 1, colWall)
	f.DrawUint(fb, x+f.TextWidth(hiLabelCells, 1)+1, y, highScore, 1, colWall)
}

func drawSnakeBanner(fb *cartlib.Framebuffer, y int) {
	ui.Label{X: arena.Cols / 2, Y: y, Text: "SNAKE", Scale: 2, Align: ui.AlignCenter, Font: ui.Font3x5{}, Color: colSnake}.Draw(fb)
}

func renderTitle(fb *cartlib.Framebuffer) {
	drawSnakeBanner(fb, 6)
	titleMenu.Draw(fb)
	drawHiScore(fb, 56)
}

func renderGetReady(fb *cartlib.Framebuffer) {
	drawSnakeBanner(fb, 10)
	f := ui.Font3x5{}
	w := f.TextWidth(len("GET READY"), 1)
	f.DrawText(fb, centerX(w), 32, "GET READY", 1, colText)
	drawHiScore(fb, 46)
}

// renderPlaying draws no score: it lives on the LCD.
func renderPlaying(fb *cartlib.Framebuffer) {
	fx := soloEffects()
	lv := layout(field.level)
	wallCol := wallColorForEffect(fx)
	for y := 0; y < arena.Rows; y++ {
		row := lv.Row(y)
		for x := 0; x < arena.Cols; x++ {
			if row[x] == arena.Wall {
				fb.SetPixel(x, y, wallCol.R, wallCol.G, wallCol.B)
			}
		}
	}

	fc := foodColor()
	fb.SetPixel(int(field.food.x), int(field.food.y), fc.R, fc.G, fc.B)

	if fx != nil {
		fx.drawItem(fb)
	}

	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if !s.onBoard() {
			continue
		}
		c := snakeColor(i)
		if ghosting() {
			c = ghostDim(c)
		}
		for n, idx := 0, s.tailIdx; n < s.snakeLen; n, idx = n+1, (idx+1)%ringSize {
			cc := s.body[idx]
			fb.SetPixel(int(cc.x), int(cc.y), c.R, c.G, c.B)
		}
	}

	if fx != nil {
		fx.drawAnnounce(fb) // last, over the snake
	}
}

// drawSeatScore's label carries a colon so the seat number does not read as
// a score digit.
func drawSeatScore(fb *cartlib.Framebuffer, f ui.Font3x5, y int, label string, score uint32, col cartlib.Color) {
	w := f.TextWidth(len(label), 1) + 1 + f.TextWidth(f.NumDigits(score), 1)
	x := centerX(w)
	f.DrawText(fb, x, y, label, 1, col)
	f.DrawUint(fb, x+f.TextWidth(len(label), 1)+1, y, score, 1, col)
}

// gameOverHint shows a green disc, not a letter: the cabinet's buttons are
// unlabelled. Built once, since Render must not allocate.
var gameOverHint = ui.HintRow{
	Y:          48,
	LabelColor: colText,
	Hints:      []ui.Hint{{Text: "TITLE", Color: ui.ButtonGreen}},
}

func renderGameOverCard(fb *cartlib.Framebuffer) {
	f := ui.Font3x5{}
	if numPlayers == 1 {
		w := f.TextWidth(len("GAME OVER"), 1)
		f.DrawText(fb, centerX(w), 6, "GAME OVER", 1, colFood)

		s := &snakes[0]
		scoreW := f.TextWidth(scoreLabelCells, 1) + 1 + f.TextWidth(f.NumDigits(s.score), 1)
		x := centerX(scoreW)
		f.DrawText(fb, x, 18, "SCORE", 1, colText)
		f.DrawUint(fb, x+f.TextWidth(scoreLabelCells, 1)+1, 18, s.score, 1, colText)
	} else {
		label := "P1 WINS"
		col := snakeColor(0)
		if winnerIndex == 1 {
			label = "P2 WINS"
			col = snakeColor(1)
		}
		w := f.TextWidth(len(label), 1)
		f.DrawText(fb, centerX(w), 6, label, 1, col)

		drawSeatScore(fb, f, 18, "P1:", snakes[0].score, snakeColor(0))
		drawSeatScore(fb, f, 26, "P2:", snakes[1].score, snakeColor(1))
	}

	drawHiScore(fb, 36)
}

func renderGameOver(fb *cartlib.Framebuffer) {
	renderGameOverCard(fb)
	gameOverHint.X = centerX(gameOverHint.Width())
	gameOverHint.Draw(fb)
}

func renderNameEntry(fb *cartlib.Framebuffer) {
	renderGameOverCard(fb)
	nameEntryWidget.Draw(fb)
}
