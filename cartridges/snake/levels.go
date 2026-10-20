// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-snake-cartridge/arena"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/levels"
)

func numLevels() int { return len(wallsPack.Levels) }

// layout is level lvl, 1-based.
func layout(lvl int) *levels.Level { return &wallsPack.Levels[lvl-1] }

func wallAt(lvl, x, y int) bool { return arena.WallAt(layout(lvl), x, y) }

func upcomingLevel(cur int) int {
	if cur < numLevels() {
		return cur + 1
	}
	return 1
}

// computeSafeStarts keeps lvl's safe starts in row-major order, the order
// pickSafeStart's RandomU64 indexes.
func computeSafeStarts(lvl int, dst *[arena.MaxSafeStarts]cell) int {
	lv := layout(lvl)
	n := 0
	for y := 0; y < arena.Rows; y++ {
		for x := 0; x < arena.Cols; x++ {
			if arena.SafeStart(lv, x, y) {
				dst[n] = cell{x: uint8(x), y: uint8(y)}
				n++
			}
		}
	}
	return n
}

// spawnDirection heads toward the middle along the farther-out axis; ties
// go vertical.
func spawnDirection(x, y int) cartlib.Direction {
	if absInt(x-32) > absInt(y-32) {
		if x < 32 {
			return cartlib.DirRight
		}
		return cartlib.DirLeft
	}
	if y < 32 {
		return cartlib.DirDown
	}
	return cartlib.DirUp
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func ringOverlapsLevel(lvl int, ring []cell, tail, n int) bool {
	for i, idx := 0, tail; i < n; i, idx = i+1, (idx+1)%len(ring) {
		c := ring[idx]
		if wallAt(lvl, int(c.x), int(c.y)) {
			return true
		}
	}
	return false
}
