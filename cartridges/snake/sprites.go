// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

const (
	pathTitleSheet  = "assets/title.arkadesprite.png"
	pathPickupSheet = "assets/pickups.arkadesprite.png"
)

// pickupCelNames is indexed by kind - pickupChili.
var pickupCelNames = [numPickupKinds]string{"chili", "ice", "shrink", "ghost", "gold"}

// blank stands in for a cel the bundle could not supply, as in a bare .wasm
// run: it draws nothing.
var blank ui.Sprite

var (
	titleArt    = &blank
	slashGlyph  = &blank
	pickupCels  = [numPickupKinds]*ui.Sprite{&blank, &blank, &blank, &blank, &blank}
	pickupTints [numPickupKinds]cartlib.Color
)

// loadSprites runs from Init, the only phase Sheet.Cel may allocate in.
func loadSprites(s *cartlib.Setup) {
	titleArt = celOr(s.LoadSheet(pathTitleSheet), "title")
	sheet := s.LoadSheet(pathPickupSheet)
	slashGlyph = celOr(sheet, "slash")
	for i, name := range pickupCelNames {
		pickupCels[i] = celOr(sheet, name)
		pickupTints[i] = firstOpaque(pickupCels[i])
	}
}

func celOr(sheet cartlib.Sheet, name string) *ui.Sprite {
	if cel := sheet.Cel(name); cel != nil {
		return cel
	}
	return &blank
}

func firstOpaque(s *ui.Sprite) cartlib.Color {
	for i := 0; i+3 < len(s.Pix); i += 4 {
		if s.Pix[i+3] == 255 {
			return cartlib.Color{R: s.Pix[i], G: s.Pix[i+1], B: s.Pix[i+2], A: 255}
		}
	}
	return cartlib.Color{}
}

// drawTinted multiplies each drawn pixel by tint, so a white cel comes out
// exactly tint and ui.White leaves a cel as drawn.
func drawTinted(fb *cartlib.Framebuffer, s *ui.Sprite, x, y int, tint cartlib.Color) {
	for i := 0; i+3 < len(s.Pix); i += 4 {
		if s.Pix[i+3] == 0 {
			continue
		}
		at := i / 4
		fb.SetPixel(x+at%s.W, y+at/s.W, mulByte(s.Pix[i], tint.R), mulByte(s.Pix[i+1], tint.G), mulByte(s.Pix[i+2], tint.B))
	}
}

func mulByte(a, b byte) byte { return byte(uint32(a) * uint32(b) / 255) }
