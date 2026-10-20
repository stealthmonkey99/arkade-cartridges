// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"strings"
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
	"github.com/stealthmonkey99/arkade/sdk/fsm"
)

// repaired is a panel press from seat by the pad paired as slot: what the
// cabinet sends once a pad re-pairs to the other port and keeps its slot.
func repaired(seat cartlib.Seat, slot cartlib.Slot, key uint8) []cartlib.InputRecord {
	rs := hosttest.Press(seat, key)
	for i := range rs {
		rs[i].Player = slot
	}
	return rs
}

func hearts(line string) int { return strings.Count(line, "♥") }

func stepUntil(t *testing.T, h *hosttest.Host, want fsm.State, maxSteps int) {
	t.Helper()
	for i := 0; i < maxSteps && machine.State() != want; i++ {
		h.Tick(stepMicros)
	}
	if machine.State() != want {
		t.Fatalf("state %d after %d steps, want %d", machine.State(), maxSteps, want)
	}
}

// TestThePanelFollowsTheSeatNotTheSlot: the pick's Physical, not its
// pairing slot, decides which LCD and seat lock the run gets.
func TestThePanelFollowsTheSeatNotTheSlot(t *testing.T) {
	h := newHost(t)
	h.Init(game)
	h.AdvancePastSplash()
	h.Input(repaired(cartlib.SeatP1, cartlib.SlotP2, cartlib.KeyJoyA)...)

	if line1, _, _ := h.LCD(cartlib.SeatP1); line1 != getReadyText1 {
		t.Errorf("seat one's LCD = %q, want %q where the player stands", line1, getReadyText1)
	}
	if line1, _, _ := h.LCD(cartlib.SeatP2); line1 == getReadyText1 {
		t.Errorf("GET READY reached seat two, the pairing slot's default seat")
	}
	if last := h.Strategy[len(h.Strategy)-1]; last.Op != "lockseat" || last.Seat != cartlib.SeatP1 {
		t.Errorf("last strategy call = %+v, want lockseat to seat one", last)
	}
}

func TestASoloPadThatRepairsTakesItsPanelAlong(t *testing.T) {
	h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(32, 31), foodIdx(10, 10))
	h.Tick(stepMicros) // the apple, so the run places
	if line1, _, _ := h.LCD(cartlib.SeatP2); line1 != "" {
		t.Fatalf("setup: seat two's LCD shows %q in a run picked from seat one", line1)
	}

	h.Input(repaired(cartlib.SeatP2, cartlib.SlotP1, cartlib.KeyLeft)...)

	if line1, _, _ := h.LCD(cartlib.SeatP2); hearts(line1) != startLives {
		t.Errorf("seat two's LCD after the re-pair = %q, want the run's %d hearts", line1, startLives)
	}
	if line1, line2, _ := h.LCD(cartlib.SeatP1); line1 != "" || line2 != "" {
		t.Errorf("seat one's LCD after the re-pair = %q / %q, want it blank", line1, line2)
	}
	if last := h.Strategy[len(h.Strategy)-1]; last.Op != "lockseat" || last.Seat != cartlib.SeatP2 {
		t.Errorf("last strategy call = %+v, want the display locked to seat two", last)
	}
	if snakes[0].queuedDir != cartlib.DirLeft {
		t.Error("the re-paired pad's press did not steer")
	}

	stepUntil(t, h, stateLostLife, 40)
	if line1, _, _ := h.LCD(cartlib.SeatP2); hearts(line1) != startLives-1 {
		t.Errorf("the lost heart went to seat two as %q, want %d hearts", line1, startLives-1)
	}
	if line1, _, _ := h.LCD(cartlib.SeatP1); line1 != "" {
		t.Errorf("the lost heart reached the vacated seat one: %q", line1)
	}

	for deaths := 1; deaths < startLives; deaths++ {
		h.Tick(freezeDelayMicros)
		stepUntil(t, h, stateLostLife, 80)
	}
	h.Tick(freezeDelayMicros)
	if machine.State() != stateNameEntry {
		t.Fatalf("state after the last life = %d, want NAME ENTRY (%d)", machine.State(), stateNameEntry)
	}
	if h.Lamps[1][0] == 0 || h.Lamps[0] != (cartlib.LampLevels{}) {
		t.Errorf("name entry lamps = seat one %v, seat two %v; want only seat two's A lit", h.Lamps[0], h.Lamps[1])
	}
	if line1, _, _ := h.LCD(cartlib.SeatP2); !strings.HasSuffix(line1, "!") {
		t.Errorf("seat two's LCD in name entry = %q, want the placement banner", line1)
	}
}

// TestTwoPadsThatSwapPortsSwapPanels: P2 eats first so the two LCDs differ,
// then each pad re-pairs to the other port.
func TestTwoPadsThatSwapPortsSwapPanels(t *testing.T) {
	h := bootTwoPlayerRun(t, spawnIdx(3, 3), spawnIdx(32, 32), foodIdx(32, 31)-1, foodIdx(50, 50)-5)
	h.Tick(stepMicros)
	if snakes[1].score != applePoints() {
		t.Fatalf("setup: P2 scored %d, want the first apple", snakes[1].score)
	}
	if _, line2, _ := h.LCD(cartlib.SeatP2); !strings.HasPrefix(line2, "3 ") {
		t.Fatalf("setup: seat two shows %q, want P2's 3", line2)
	}

	h.Input(repaired(cartlib.SeatP2, cartlib.SlotP1, cartlib.KeyLeft)...)
	h.Input(repaired(cartlib.SeatP1, cartlib.SlotP2, cartlib.KeyRight)...)

	if _, line2, _ := h.LCD(cartlib.SeatP1); !strings.HasPrefix(line2, "3 ") {
		t.Errorf("seat one after the swap shows %q, want P2's 3", line2)
	}
	if _, line2, _ := h.LCD(cartlib.SeatP2); !strings.HasPrefix(line2, "0 ") {
		t.Errorf("seat two after the swap shows %q, want P1's 0", line2)
	}

	stepUntil(t, h, stateLostLife, 10)
	if !snakes[0].diedThisTick || snakes[1].diedThisTick {
		t.Fatal("setup: P1, steered into the left wall, should be the one to die")
	}
	if line1, _, _ := h.LCD(cartlib.SeatP2); hearts(line1) != startLives-1 {
		t.Errorf("P1's lost heart showed on seat two as %q, want %d hearts", line1, startLives-1)
	}
	if line1, _, _ := h.LCD(cartlib.SeatP1); hearts(line1) != startLives {
		t.Errorf("seat one, P2's panel now, shows %q, want all %d hearts", line1, startLives)
	}
}
