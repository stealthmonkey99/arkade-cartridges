// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

func isAction(r cartlib.InputRecord) bool {
	if !r.IsPress() {
		return false
	}
	switch r.Key {
	case cartlib.KeyJoyA, cartlib.KeyEnter, cartlib.KeySpace, cartlib.KeyStart:
		return true
	}
	return false
}

// steerDir keeps a stick's keypress twin, which PressDir drops: queueing the
// same turn twice is harmless.
func steerDir(r cartlib.InputRecord) cartlib.Direction {
	if r.Kind == cartlib.KindKeydown || r.Kind == cartlib.KindKeyup {
		return cartlib.DirNone
	}
	return r.Dir()
}

func handleInputRecord(r cartlib.InputRecord) {
	heldDirs.Feed(r)
	if inRound() {
		followSeat(r)
	}
	quit := r.IsPress() && r.IsQuit()

	switch machine.State() {
	case stateTitle:
		handleTitleInput(r, quit)
	case stateGetReady:
		if quit {
			returnToTitle()
		}
	case statePlaying:
		handlePlayingInput(r, quit)
	case stateConfirmExit:
		// Any press the hold does not take cancels.
		if !confirmExit.Feed(r) && r.IsPress() {
			resumeFromConfirmExit()
		}
	case stateGameOver:
		if quit {
			cartlib.Exit()
		} else if isAction(r) {
			returnToTitle()
		}
	case stateNameEntry:
		handleNameEntryInput(r, quit)
	}
}

// seatOf falls back to the slot's seat for a seatless source like a keyboard.
func seatOf(r cartlib.InputRecord, slot cartlib.Slot) cartlib.Seat {
	if r.Physical.Single() {
		return r.Physical
	}
	return cartlib.Seat(slot - cartlib.SlotP1)
}

func handleTitleInput(r cartlib.InputRecord, quit bool) {
	if quit {
		cartlib.Exit()
		return
	}
	if d := r.PressDir(); d != cartlib.DirNone {
		titleMenu.Move(d)
	}
	if !isAction(r) {
		return
	}
	slot := r.Player
	if slot != cartlib.SlotP1 && slot != cartlib.SlotP2 {
		slot = cartlib.SlotP1 // defensive; the ABI never sends anything else
	}
	seat := seatOf(r, slot)
	switch titleMenu.Activate() {
	case 0:
		numPlayers = 1
		seats = [2]cartlib.Seat{seat, cartlib.SeatNone}
		soloSlot = slot
		// Lock now, so GET READY already faces the player.
		cartlib.LockPlayerToSeat(seat)
	case 1:
		numPlayers = 2
		soloSlot = cartlib.SlotNone
		seats[slot-cartlib.SlotP1] = seat
		seats[cartlib.SlotP2-slot] = seat.Other()
	default:
		return
	}
	machine.Go(stateGetReady)
}

// playerOf is the player whose snake r steers, or -1. 1P steers from
// soloSlot, not slot 1, so a run started from P2's pad still steers.
func playerOf(r cartlib.InputRecord) int {
	if numPlayers == 1 {
		if r.Player != soloSlot {
			return -1
		}
		return 0
	}
	i := int(r.Player) - int(cartlib.SlotP1)
	if i < 0 || i >= numPlayers {
		return -1
	}
	return i
}

// followSeat moves a player's panel to the seat their record came from, so a
// pad that re-pairs mid-round takes its LCD with it. A vacated seat goes
// dark; the arrival shows what the round shows there.
func followSeat(r cartlib.InputRecord) {
	i := playerOf(r)
	if i < 0 || !r.Physical.Single() || seats[i] == r.Physical {
		return
	}
	from := seats[i]
	seats[i] = r.Physical
	if numPlayers == 1 {
		cartlib.LockPlayerToSeat(r.Physical)
	}
	if seats[1-i] != from {
		cartlib.SetLights(from, false)
		cartlib.SetText(from, "", "")
	}
	switch machine.State() {
	case stateGetReady:
		cartlib.SetText(r.Physical, getReadyText1, getReadyText2)
	case statePlaying, stateLostLife:
		updatePlayLcd(i)
	}
}

// inRound reports whether a round holds the seats: from the pick to the
// final freeze, the quit overlay over play included.
func inRound() bool {
	switch machine.State() {
	case stateGetReady, statePlaying, stateLostLife:
		return true
	case stateConfirmExit:
		return machine.Under() != stateNameEntry
	}
	return false
}

// handlePlayingInput takes quit first, so either pad can leave.
func handlePlayingInput(r cartlib.InputRecord, quit bool) {
	if quit {
		enterConfirmExit(r)
		return
	}
	d := steerDir(r)
	if d == cartlib.DirNone {
		return
	}
	i := playerOf(r)
	if i < 0 {
		return
	}
	s := &snakes[i]
	if s.onBoard() && perpendicular(d, s.dir) {
		s.queuedDir = d
	}
}

// enterConfirmExit counts the opening B press toward the hold, since it is
// already down.
func enterConfirmExit(r cartlib.InputRecord) {
	confirmExit.Open()
	if r.Virtual() {
		confirmExit.Pulse()
	} else {
		confirmExit.KeyDown()
	}
	machine.Push(stateConfirmExit)
}

func resumeFromConfirmExit() {
	confirmExit.Close()
	machine.Pop()
}

// handleNameEntryInput routes B to the hold-to-quit overlay: a stray press
// here would throw away a top-ten placement.
func handleNameEntryInput(r cartlib.InputRecord, quit bool) {
	switch {
	case quit:
		enterConfirmExit(r)
		confirmExit.Avoid(nameEntryWidget.LetterBand())
	case isAction(r):
		nameEntryWidget.Do(ui.ActionActivate)
	case r.IsPress() && r.Key == cartlib.KeyJoyC:
		nameEntryWidget.Do(ui.ActionErase)
	}
}
