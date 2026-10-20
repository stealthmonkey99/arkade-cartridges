// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-snake-cartridge/arena"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
)

// board is the state both snakes share. Both mark the one occupancy grid,
// so cross-snake collision needs no special case.
type board struct {
	occupied [arena.Rows][arena.Cols]bool
	level    int // 1-based
	swap     levelSwap
	food     cell

	safeStarts     [arena.MaxSafeStarts]cell // for the pending level while a swap waits
	safeStartCount int

	stepAcc uint64
}

// levelSwap waits to land until no snake overlaps the next layout's walls.
type levelSwap struct {
	pending bool
	to      int
}

var field board

// start leaves the apple unplaced until the snakes have spawned.
func (b *board) start() {
	b.level = 1
	b.swap = levelSwap{}
	for y := range b.occupied {
		for x := range b.occupied[y] {
			b.occupied[y][x] = false
		}
	}
	b.safeStartCount = computeSafeStarts(b.level, &b.safeStarts)
}

func (b *board) restartStepClock() { b.stepAcc = 0 }

func (b *board) scheduleLevel(to int) {
	b.swap = levelSwap{pending: true, to: to}
	b.safeStartCount = computeSafeStarts(to, &b.safeStarts)
}

func (b *board) tryLandLevel() bool {
	if !b.swap.pending {
		return false
	}
	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if s.onBoard() && ringOverlapsLevel(b.swap.to, s.body[:], s.tailIdx, s.snakeLen) {
			return false
		}
	}
	b.commitLevelSwap()
	return true
}

// commitLevelSwap moves an apple or pickup that the new walls cover: they
// were placed against the old layout, possibly many ticks ago.
func (b *board) commitLevelSwap() {
	b.level = b.swap.to
	b.swap = levelSwap{}
	if wallAt(b.level, int(b.food.x), int(b.food.y)) {
		b.placeFood(b.level)
	}
	if fx := soloEffects(); fx != nil {
		fx.revalidate(b)
	}
}

// keepClear is the 3×3 a pickup's sprite covers around a cell. The apple and
// a pickup never share one, so neither is drawn over the other.
type keepClear struct {
	around cell
	set    bool
}

func clearOf(c cell) keepClear { return keepClear{around: c, set: true} }

func (k keepClear) covers(c cell) bool {
	return k.set && within1(c.x, k.around.x) && within1(c.y, k.around.y)
}

func within1(a, b uint8) bool { return a+1 >= b && b+1 >= a }

// freeScan counts free interior cells in row-major order, skipping those
// avoid covers, and returns the pick-th; pick -1 only counts.
func (b *board) freeScan(lvl int, avoid keepClear, pick int) (free int, at cell, ok bool) {
	for py := fieldMin; py <= fieldMax; py++ {
		for px := fieldMin; px <= fieldMax; px++ {
			if b.occupied[py][px] || wallAt(lvl, px, py) {
				continue
			}
			c := cell{x: uint8(px), y: uint8(py)}
			if avoid.covers(c) {
				continue
			}
			if pick == free {
				at, ok = c, true
			}
			free++
		}
	}
	return
}

// placeFood keeps the apple clear of a running pickup's sprite; on a board
// too full for both, the pickup gives way. On a full board the apple stays
// where it is: which snake to end would be ambiguous in 2P.
func (b *board) placeFood(lvl int) {
	avoid := pickups.footprint()
	free, _, _ := b.freeScan(lvl, avoid, -1)
	if free <= 0 && avoid.set {
		pickups.window.stop()
		avoid = keepClear{}
		free, _, _ = b.freeScan(lvl, avoid, -1)
	}
	if free <= 0 {
		return
	}
	if _, c, ok := b.freeScan(lvl, avoid, int(cartlib.RandomU64()%uint64(free))); ok {
		b.food = c
	}
}

func (b *board) pickSafeStart() cell {
	if b.safeStartCount <= 0 {
		// Unreachable: arena.Validate refuses a layout without one.
		return cell{x: 32, y: 32}
	}
	return b.safeStarts[int(cartlib.RandomU64()%uint64(b.safeStartCount))]
}
