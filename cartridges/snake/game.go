// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"arkade-snake-cartridge/arena"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/fsm"
	"github.com/stealthmonkey99/arkade/sdk/ui"
)

// The quit overlay is pushed, so a cancel pops back to whatever it covered.
const (
	stateTitle fsm.State = iota
	stateGetReady
	statePlaying
	stateLostLife // frozen, timed: at least one snake collided this tick
	stateConfirmExit
	stateGameOver
	stateNameEntry // a qualifying score gets this instead of the banner
)

// legalEdges is one destination bit per state; Pop is not an edge.
var legalEdges = []uint16{
	stateTitle:       1 << stateGetReady,
	stateGetReady:    1<<statePlaying | 1<<stateTitle,
	statePlaying:     1<<stateLostLife | 1<<stateConfirmExit,
	stateLostLife:    1<<statePlaying | 1<<stateGameOver | 1<<stateNameEntry,
	stateConfirmExit: 0,
	stateGameOver:    1 << stateTitle,
	stateNameEntry:   1<<stateTitle | 1<<stateConfirmExit,
}

const (
	stepMicros      = 62500 // 16 px/s
	maxStepsPerTick = 8     // catch-up cap after a stalled frame

	readyDelayMicros  = 2_000_000
	freezeDelayMicros = 1_000_000 // any collision freezes the whole arena

	startLives = 3

	nextLevelLength   = 100
	nextLevelLength2P = 30 // so a 2P round reaches a level change at all

	initLength = 4
	growPerEat = 3 // growth per apple; the score steps by the same amount

	fieldMin = 1 // 0 and arena.Cols-1 are the border wall
	fieldMax = arena.Cols - 2
)

var (
	machine         fsm.Machine
	reportedBlocked uint32

	numPlayers int = 1

	soloSlot = cartlib.SlotNone // the slot that started the 1P run
	// seats is each player's panel, picked on the title and moved by followSeat.
	seats = [2]cartlib.Seat{cartlib.SeatNone, cartlib.SeatNone}

	heldDirs ui.HeldDirs // name entry scrolls on the held level, not the edge

	highScore uint32

	winnerIndex int // 2P only

	eligibleSeat  cartlib.Seat // whose qualifying score the name entry is for
	eligibleScore uint32
)

// enterScreen sets each screen's LCD text and lamps, which the cabinet never
// restores for you. It runs on every arrival, including a Pop.
func enterScreen(prev, next fsm.State) {
	switch next {
	case stateTitle:
		resetTitleMenu()
		showTitlePeripherals()
	case stateGetReady:
		enterGetReadyPeripherals()
	case statePlaying:
		if prev == stateLostLife {
			return // killSnakeThisTick already updated the hearts
		}
		// Play binds no button, so no lamp may stay lit.
		cartlib.SetLights(cartlib.SeatBoth, false)
		for i := 0; i < numPlayers; i++ {
			updatePlayLcd(i)
		}
	case stateLostLife:
	case stateConfirmExit:
		cartlib.SetText(cartlib.SeatBoth, ui.HoldConfirmLCD1, ui.HoldConfirmLCD2)
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA|cartlib.MaskJoyB, true)
	case stateGameOver:
		gameOverLightsOn = true
		cartlib.SetButtonLights(cartlib.SeatBoth, cartlib.MaskJoyA, true)
		updateGameOverLcd()
	case stateNameEntry:
		// heldDirs only decays in this screen's Update, so a pulse from before
		// the quit overlay would otherwise scroll the initials on cancel.
		heldDirs.Reset()
		if prev == stateConfirmExit {
			// The overlay lit both panels; "only" below rewrites one seat.
			cartlib.SetButtonLights(cartlib.SeatBoth, 0, true)
		}
		cartlib.SetButtonLights(eligibleSeat, cartlib.MaskJoyA, true)
		updateNameEntryLcd()
	}
}

// reportRefusedTransitions logs each new refusal, so the golden tests,
// which only see the host, can assert the edge table holds.
func reportRefusedTransitions() {
	if b := machine.Blocked(); b != reportedBlocked {
		reportedBlocked = b
		cartlib.Log(cartlib.LogError, "snake: the screen machine refused a transition")
	}
}

var titleMenuItems = []string{"1 PLAYER", "2 PLAYERS"}

var titleMenu *ui.Menu
var nameEntryWidget *ui.NameEntry
var confirmExit ui.HoldConfirm

func initWidgets() {
	titleMenu = ui.NewMenu(titleMenuItems)
	titleMenu.X, titleMenu.Y, titleMenu.W = 8, 24, 48
	nameEntryWidget = ui.NewNameEntry(8, 44, 0)
	confirmExit.W, confirmExit.H = arena.Cols, arena.Rows
}

func resetTitleMenu() { titleMenu.SetItems(titleMenuItems) }

// loadHighScore reads rank 1 of the cabinet's shared, descending table.
func loadHighScore() {
	score, _, ok := cartlib.ScoreRead(1, scoreScratch[:])
	if !ok {
		highScore = 0
		return
	}
	highScore = uint32(score)
}

var scoreScratch [8 + cartlib.MaxNameLen]byte
var recentNameScratch [cartlib.MaxNameLen]byte

// startGame resets both slots: they persist across runs in the cabinet
// process.
func startGame() {
	field.start()
	pickups.reset()
	for i := range snakes {
		snakes[i].sitOut()
	}
	for i := 0; i < numPlayers; i++ {
		snakes[i].enter()
		respawnSnake(&snakes[i], 0)
	}
	field.restartStepClock()
	field.placeFood(field.level)
	machine.Go(statePlaying)
}

// triggerLevelUp restarts the 1P snake in place, regrowing from its head.
func triggerLevelUp() {
	s := &snakes[0]
	beginSnake(s, s.head(), s.dir, initLength)
	field.restartStepClock()
	field.scheduleLevel(upcomingLevel(field.level))
	cartlib.PlaySound(soundLevelUp, 0)
}

// triggerLevelUp2P respawns both snakes on safe starts: restarting in place
// could stack them or trap one in the new walls.
func triggerLevelUp2P() {
	field.scheduleLevel(upcomingLevel(field.level))
	for i := 0; i < numPlayers; i++ {
		if snakes[i].onBoard() {
			respawnSnake(&snakes[i], 0)
		}
	}
	field.restartStepClock()
	cartlib.PlaySound(soundLevelUp, 0)
}

// tryAdvanceLevel runs before a step's spawns, so they never land against a
// layout about to change.
func tryAdvanceLevel() {
	if !field.tryLandLevel() {
		return
	}
	for i := 0; i < numPlayers; i++ {
		updatePlayLcd(i)
	}
}

// resolveLostLife respawns each dead snake at its length or removes it. The
// round ends as soon as any snake is out: in 2P the survivor wins without
// playing out its lives.
func resolveLostLife() {
	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if !s.diedThisTick {
			continue
		}
		s.diedThisTick = false
		if s.lives > 0 {
			respawnSnake(s, s.snakeLen)
		} else {
			removeSnakeFromBoard(s)
		}
	}
	field.restartStepClock()

	for i := 0; i < numPlayers; i++ {
		if snakes[i].dead {
			enterGameOver()
			return
		}
	}
	machine.Go(statePlaying)
}

// enterGameOver offers only the winner's score to the table, even when the
// loser's is higher. A simultaneous final death goes to the higher score,
// then to P1.
func enterGameOver() {
	winnerIndex = 0
	if numPlayers == 2 {
		switch {
		case !snakes[0].dead:
		case !snakes[1].dead:
			winnerIndex = 1
		case snakes[1].score > snakes[0].score:
			winnerIndex = 1
		}
	}
	winner := &snakes[winnerIndex]

	cartlib.PlaySound(soundGameOver, 0)

	if !cartlib.ScoreQualifies(int(winner.score)) {
		machine.Go(stateGameOver)
		return
	}
	eligibleSeat = seats[winnerIndex]
	eligibleScore = winner.score
	cartlib.LockPlayerToSeat(eligibleSeat)
	prefill, _ := cartlib.RecentName(eligibleSeat, recentNameScratch[:])
	nameEntryWidget.SetName(string(prefill))
	nameEntryWidget.SetRecents(cartlib.RecentNames(eligibleSeat))
	machine.Go(stateNameEntry)
}

func finishNameEntry() {
	name := nameEntryWidget.Name()
	if rank := cartlib.ScoreSubmit(int(eligibleScore), name); rank > 0 {
		loadHighScore()
	}
	if name != "" {
		cartlib.NoteRecentName(eligibleSeat, name)
	}
	returnToTitle()
}

// returnToTitle unlocks the strategy so any controller can pick the game up
// and the display returns upright.
func returnToTitle() {
	cartlib.UnlockPlayerStrategy()
	seats = [2]cartlib.Seat{cartlib.SeatNone, cartlib.SeatNone}
	soloSlot = cartlib.SlotNone
	numPlayers = 1
	heldDirs.Reset()
	machine.Go(stateTitle)
}

func update(dtMicros uint64) {
	machine.Tick(dtMicros)
	switch machine.State() {
	case stateTitle:
		titleMenu.Update(dtMicros)
	case stateGetReady:
		if machine.Elapsed() >= readyDelayMicros {
			startGame()
		}
	case statePlaying:
		field.stepAcc += dtMicros
		for steps := 0; steps < maxStepsPerTick; steps++ {
			sm := currentStepMicros() // an effect can expire mid-catch-up
			if field.stepAcc < sm {
				break
			}
			field.stepAcc -= sm
			stepGameAll()
			if machine.State() != statePlaying {
				return
			}
		}
		if fx := soloEffects(); fx != nil {
			fx.tick(dtMicros)
		}
	case stateLostLife:
		if machine.Elapsed() >= freezeDelayMicros {
			resolveLostLife()
		}
	case stateConfirmExit:
		if confirmExit.Update(dtMicros) {
			cartlib.Exit()
		}
	case stateGameOver:
		tickGameOver()
	case stateNameEntry:
		heldDirs.Update(dtMicros)
		nameEntryWidget.Move(heldDirs.Current())
		nameEntryWidget.Update(dtMicros)
		if nameEntryWidget.Done() {
			finishNameEntry()
		}
	}
}
