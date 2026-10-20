// SPDX-License-Identifier: GPL-2.0-or-later

//go:build !tinygo

package arena

import (
	"fmt"

	"github.com/stealthmonkey99/arkade/sdk/levels"
)

// Validate rejects a layout the arena cannot hold or a snake cannot play:
// it must be Cols by Rows, closed by a complete border, one connected
// floor, and offer at least one safe start. It returns the safe-start
// count of a layout that passes.
func Validate(lv *levels.Level) (safeStarts int, err error) {
	if lv.W != Cols || lv.H != Rows {
		return 0, fmt.Errorf("is %dx%d, but the arena is the framebuffer: a fixed %dx%d", lv.W, lv.H, Cols, Rows)
	}
	for i := 0; i < Cols; i++ {
		for _, c := range [][2]int{{i, 0}, {i, Rows - 1}, {0, i}, {Cols - 1, i}} {
			if !WallAt(lv, c[0], c[1]) {
				return 0, fmt.Errorf("border cell (%d, %d) is open — the arena leaks", c[0], c[1])
			}
		}
	}
	if unreached, total := unreachedFloor(lv); unreached > 0 {
		return 0, fmt.Errorf("%d of %d floor cells are walled off from the rest — food can spawn where the snake cannot go", unreached, total)
	}
	for y := 0; y < Rows; y++ {
		for x := 0; x < Cols; x++ {
			if SafeStart(lv, x, y) {
				safeStarts++
			}
		}
	}
	if safeStarts == 0 {
		return 0, fmt.Errorf("has no safe respawn cell (no open cell with %d cells of clearance on both axes)", SafeRadius)
	}
	return safeStarts, nil
}

// unreachedFloor flood-fills the floor from its first cell.
func unreachedFloor(lv *levels.Level) (unreached, total int) {
	seen := make([]bool, lv.W*lv.H)
	start := -1
	for i := 0; i < len(lv.Cells); i++ {
		if lv.Cells[i] != Wall {
			total++
			if start < 0 {
				start = i
			}
		}
	}
	if start < 0 {
		return 0, 0
	}
	stack := []int{start}
	seen[start] = true
	reached := 0
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		reached++
		x, y := i%lv.W, i/lv.W
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if WallAt(lv, nx, ny) {
				continue
			}
			if j := ny*lv.W + nx; !seen[j] {
				seen[j] = true
				stack = append(stack, j)
			}
		}
	}
	return total - reached, total
}
