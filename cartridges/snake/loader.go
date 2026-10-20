// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

const snakeLoadReadyMicros uint64 = 2_500_000 // inside the shell's 2-3 s load window

const loaderInset = 3 // clear of the title art's border

const (
	loaderLeft      = loaderInset
	loaderTop       = loaderInset
	loaderRight     = cartlib.FBWidth - 1 - loaderInset
	loaderBottom    = cartlib.FBHeight - 1 - loaderInset
	loaderW         = loaderRight - loaderLeft
	loaderH         = loaderBottom - loaderTop
	loaderPerimeter = 2*loaderW + 2*loaderH
)

// loaderPerimeterPoint walks clockwise from the top-left corner; d wraps.
func loaderPerimeterPoint(d int) (x, y int) {
	d %= loaderPerimeter
	if d < 0 {
		d += loaderPerimeter
	}
	switch {
	case d <= loaderW:
		return loaderLeft + d, loaderTop
	case d <= loaderW+loaderH:
		return loaderRight, loaderTop + (d - loaderW)
	case d <= 2*loaderW+loaderH:
		return loaderRight - (d - loaderW - loaderH), loaderBottom
	default:
		return loaderLeft, loaderBottom - (d - 2*loaderW - loaderH)
	}
}

const loaderSnakeStepMicros = snakeLoadReadyMicros / loaderPerimeter // one lap per load screen

const loaderSnakeSegments = 10

const loaderFoodStep = loaderPerimeter / 2

// loaderFoodEaten never resets: the loader runs once per process, and the
// attract instance never calls LoadFrame.
var loaderFoodEaten bool

// drawLoaderSnake is a pure function of elapsedMicros and draws no
// RandomU64, so however many load frames a host renders, the gameplay RNG
// script is untouched.
func drawLoaderSnake(fb *cartlib.Framebuffer, elapsedMicros uint64) {
	head := int(elapsedMicros / loaderSnakeStepMicros)
	if head < loaderFoodStep {
		fx, fy := loaderPerimeterPoint(loaderFoodStep)
		fb.SetPixel(fx, fy, colFood.R, colFood.G, colFood.B)
	} else if !loaderFoodEaten {
		loaderFoodEaten = true
		cartlib.PlaySound(soundEat, 0)
	}
	for i := 0; i < loaderSnakeSegments; i++ {
		x, y := loaderPerimeterPoint(head - i)
		if i == 0 {
			fb.SetPixel(x, y, 255, 255, 255)
			continue
		}
		fb.SetPixel(x, y, colSnake.R, colSnake.G, colSnake.B)
	}
}

// LoadFrame implements cartlib.Loader.
func (snakeGame) LoadFrame(elapsedMicros uint64, fb *cartlib.Framebuffer) bool {
	fb.Clear(0, 0, 0)
	titleArt.Draw(fb, 0, 0)
	drawLoaderSnake(fb, elapsedMicros)
	return elapsedMicros >= snakeLoadReadyMicros
}

var _ cartlib.Loader = snakeGame{}
