// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "testing"

func TestShrinkCutCountRoundsToNearestThird(t *testing.T) {
	cases := []struct {
		length int
		want   int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{3, 1},
		{4, 1},
		{5, 2},
		{6, 2},
		{9, 3},
		{10, 3},
		{11, 4},
		{100, 33},
		{101, 34},
	}
	for _, c := range cases {
		if got := shrinkCutCount(c.length); got != c.want {
			t.Errorf("shrinkCutCount(%d) = %d, want %d", c.length, got, c.want)
		}
	}
}

func TestShrinkSnakeFloorsAtShrinkMinLen(t *testing.T) {
	resetSpawnTestBoard()
	s := &snakes[0]
	*s = snake{}
	const length = 12
	for i := 0; i < length; i++ {
		c := cell{x: uint8(1 + i), y: 5}
		s.body[i] = c
		field.occupied[c.y][c.x] = true
	}
	s.tailIdx = 0
	s.headIdx = length - 1
	s.snakeLen = length

	cut := shrinkCutCount(length)
	if cut != 4 {
		t.Fatalf("shrinkCutCount(%d) = %d, want 4 (sanity check on the case this test drives)", length, cut)
	}
	shrinkSnake(s, cut)
	if s.snakeLen != length-cut {
		t.Errorf("snakeLen after shrinking a %d-long snake by %d = %d, want %d", length, cut, s.snakeLen, length-cut)
	}

	shrinkSnake(s, 100)
	if s.snakeLen != shrinkMinLen {
		t.Errorf("snakeLen after an oversized shrink request = %d, want the floor %d (shrinkMinLen)", s.snakeLen, shrinkMinLen)
	}
}

// TestAnnounceLabelMatchesTuningSpec: MANUAL.md quotes these labels.
func TestAnnounceLabelMatchesTuningSpec(t *testing.T) {
	cases := map[uint8]string{
		pickupChili:  "2X SPEED",
		pickupIce:    "1/2 SPEED",
		pickupShrink: "CUT SHORT",
		pickupGhost:  "INVINCIBLE",
		pickupGold:   "3X POINTS",
	}
	for kind, want := range cases {
		if got := announceLabel(kind); got != want {
			t.Errorf("announceLabel(%d) = %q, want %q", kind, got, want)
		}
	}
}
