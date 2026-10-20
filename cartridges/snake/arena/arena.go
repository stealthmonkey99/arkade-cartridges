// SPDX-License-Identifier: GPL-2.0-or-later

// Package arena is Snake's playfield: its size and the wall and respawn
// rules a layout is read by. The cartridge plays by these rules and its
// tests hold every layout to them.
package arena

import "github.com/stealthmonkey99/arkade/sdk/levels"

// Cols and Rows are the framebuffer's, cell for pixel.
const (
	Cols = 64
	Rows = 64
)

// Wall and Floor are the layout runes.
const (
	Wall  = '#'
	Floor = '.'
)

// SafeRadius is a respawn's clearance: a safe start sits at least this far
// from the grid edge, with no wall closer along either axis.
const SafeRadius = 3

// MaxSafeStarts is the edge guard's square, so no layout can exceed it.
const MaxSafeStarts = (Cols - 2*SafeRadius) * (Rows - 2*SafeRadius)

// WallAt counts off-grid as wall.
func WallAt(lv *levels.Level, x, y int) bool {
	if x < 0 || y < 0 || x >= lv.W || y >= lv.H {
		return true
	}
	return lv.At(x, y) == Wall
}

// SafeStart reports whether a snake may respawn on (x, y) of lv.
func SafeStart(lv *levels.Level, x, y int) bool {
	if x < SafeRadius || y < SafeRadius || x > Cols-1-SafeRadius || y > Rows-1-SafeRadius {
		return false
	}
	for i := 0; i < SafeRadius; i++ {
		if WallAt(lv, x, y+i) || WallAt(lv, x, y-i) || WallAt(lv, x+i, y) || WallAt(lv, x-i, y) {
			return false
		}
	}
	return true
}
