// SPDX-License-Identifier: GPL-2.0-or-later

package main

// The simulation calls the host inline (sounds, LCDs, RandomU64), so these
// tests boot the real Game through hosttest and read what the fake cabinet
// saw. The wasm goldens keep pinning the frames; this pins the rules.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
)

// bootOnePlayerRun takes the cartridge from cart_init to the first playing
// tick the way a player would: through the splash and the title menu.
func bootOnePlayerRun(t *testing.T, rand ...uint64) *hosttest.Host {
	t.Helper()
	return bootRun(t, 1, rand)
}

// bootTwoPlayerRun is bootOnePlayerRun with 2 PLAYERS picked from seat one.
func bootTwoPlayerRun(t *testing.T, rand ...uint64) *hosttest.Host {
	t.Helper()
	return bootRun(t, 2, rand)
}

func bootRun(t *testing.T, players int, rand []uint64) *hosttest.Host {
	t.Helper()
	h := newHost(t)
	h.Rand = rand
	h.Init(game)
	h.AdvancePastSplash()
	if players == 2 {
		h.Input(hosttest.Stick(cartlib.SeatP1, cartlib.JoyS), hosttest.Stick(cartlib.SeatP1, cartlib.JoyCenter))
	}
	h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyJoyA)...)
	if machine.State() != stateGetReady {
		t.Fatalf("after A on the title: state %d, want GET READY (%d)", machine.State(), stateGetReady)
	}
	h.Tick(readyDelayMicros)
	if machine.State() != statePlaying {
		t.Fatalf("after the ready delay: state %d, want PLAYING (%d)", machine.State(), statePlaying)
	}
	if numPlayers != players {
		t.Fatalf("the title started a %dP run, want %dP", numPlayers, players)
	}
	return h
}

func TestEatingTheAppleScoresGrowsAndReplacesIt(t *testing.T) {
	h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(32, 31), foodIdx(10, 10))
	s := &snakes[0]
	if s.head() != (cell{32, 32}) || s.dir != cartlib.DirUp {
		t.Fatalf("spawn: head %+v heading %v, want (32,32) heading up", s.head(), s.dir)
	}
	if field.food != (cell{32, 31}) {
		t.Fatalf("first apple at %+v, want (32,31)", field.food)
	}
	if h.SoundID(pathEat) == 0 {
		t.Fatal("Init did not load the eat sound through its Setup")
	}

	h.Tick(stepMicros)

	if s.head() != (cell{32, 31}) {
		t.Fatalf("after one step the head is at %+v, want (32,31)", s.head())
	}
	if s.score != applePoints() {
		t.Errorf("score after the apple = %d, want %d", s.score, applePoints())
	}
	if !h.Playing(h.SoundID(pathEat)) {
		t.Error("eating did not play the eat sound")
	}
	if field.food != (cell{10, 10}) {
		t.Errorf("the apple was replaced at %+v, want the scripted (10,10)", field.food)
	}
	if field.occupied[field.food.y][field.food.x] {
		t.Error("the new apple sits on the snake")
	}
	if _, _, written := h.LCD(cartlib.SeatP1); !written {
		t.Error("the playing LCD was never written")
	}
}

func TestHittingTheWallCostsALifeAndFreezesTheArena(t *testing.T) {
	h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(10, 10))
	s := &snakes[0]
	for steps := 0; steps < 40 && machine.State() == statePlaying; steps++ {
		h.Tick(stepMicros)
	}
	if machine.State() != stateLostLife {
		t.Fatalf("state after running into the top wall = %d, want LOST LIFE (%d)", machine.State(), stateLostLife)
	}
	if s.lives != startLives-1 {
		t.Errorf("lives = %d, want %d", s.lives, startLives-1)
	}
	if !h.Playing(h.SoundID(pathLost)) {
		t.Error("the collision did not play the lost sound")
	}
	if h.Playing(h.SoundID(pathEat)) {
		t.Error("the snake ate on a column with no apple")
	}
	if s.head().y != 1 {
		t.Errorf("the head stopped at y=%d, want 1, the last cell before the wall", s.head().y)
	}
}

func TestAPlayingTickAllocatesNothingOnHostGo(t *testing.T) {
	h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(10, 10))
	h.Tick(stepMicros)
	if n := h.AllocsPerRun(20, func() { h.Tick(stepMicros) }); n != 0 {
		t.Errorf("a playing tick allocated %v times per run; TinyGo's -print-allocs report has the sites", n)
	}
}

// playOut steps a run until it leaves play for the banner or name entry.
func playOut(t *testing.T, h *hosttest.Host) {
	t.Helper()
	for i := 0; i < 2000 && (machine.State() == statePlaying || machine.State() == stateLostLife); i++ {
		h.Tick(stepMicros)
	}
	if s := machine.State(); s != stateGameOver && s != stateNameEntry {
		t.Fatalf("the run never ended: state %d", s)
	}
}

func TestACancelledCountdownStartsOver(t *testing.T) {
	h := newHost(t)
	h.Init(game)
	h.AdvancePastSplash()
	h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyJoyA)...)
	h.Tick(1_500_000)
	h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyQuit)...)
	if machine.State() != stateTitle || h.Exits != 0 {
		t.Fatalf("quitting the countdown: state %d, %d exits; want the title and no exit", machine.State(), h.Exits)
	}

	h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyJoyA)...)
	h.Tick(readyDelayMicros - 1)
	if machine.State() != stateGetReady {
		t.Fatalf("1 µs short of a fresh countdown: state %d, want GET READY — the old countdown's half second carried over", machine.State())
	}
	h.Tick(1)
	if machine.State() != statePlaying {
		t.Fatalf("at the full countdown: state %d, want PLAYING", machine.State())
	}
}

// TestTheLoaderEatsItsAppleOnce ticks the load screen frame by frame, as a
// cabinet does, where AdvancePastSplash jumps it in one update.
func TestTheLoaderEatsItsAppleOnce(t *testing.T) {
	loaderFoodEaten = false // once per process, and earlier tests ran the loader
	h := newHost(t)
	h.Init(game)
	for elapsed := uint64(0); elapsed < hosttest.SplashMicros; elapsed += 33_333 {
		h.Tick(33_333)
	}
	eat := h.SoundID(pathEat)
	plays := 0
	for _, c := range h.Sounds {
		if c.Op == "play" && c.ID == eat {
			plays++
		}
	}
	if plays != 1 {
		t.Errorf("the eat sound played %d times across the loader, want 1", plays)
	}
}

func stepToHead(t *testing.T, h *hosttest.Host, c cell) {
	t.Helper()
	for i := 0; i < 64 && snakes[0].head() != c; i++ {
		h.Tick(stepMicros)
	}
	if snakes[0].head() != c {
		t.Fatalf("the head is at %+v, never reached %+v", snakes[0].head(), c)
	}
}

func colGold() cartlib.Color { return pickupColor(pickupGold) }

func opaque(c cartlib.Color) cartlib.Color { c.A = 255; return c }

// TestGoldTriplesApplesUntilItExpires: the first apple triggers a pickup
// (applesToGo is set to one), the snake collects it on its way up, and the
// next apple pays triple.
func TestGoldTriplesApplesUntilItExpires(t *testing.T) {
	h := bootOnePlayerRun(t,
		spawnIdx(32, 60),
		foodIdx(32, 59),
		foodIdx(32, 50),   // apple two
		4,                 // pickupChili + 4 is gold
		foodIdx(32, 55)-9, // the pickup; the 3×3 around apple two precedes it
		0,                 // the next cadence
		foodIdx(1, 1),     // apple three, never reached
	)
	pickups.applesToGo = 1

	h.Tick(stepMicros)
	if !pickups.window.running() || pickups.kind != pickupGold || pickups.x != 32 || pickups.y != 55 {
		t.Fatalf("after the first apple: pickup %+v, want gold at (32,55)", pickups)
	}
	if !h.Playing(h.SoundID(pathPickupSpawn)) {
		t.Error("the pickup arrived without its sound")
	}
	if px := opaque(h.Pixel(32, 55)); px != colGold() {
		t.Errorf("pickup centre pixel = %+v, want gold", px)
	}

	stepToHead(t, h, cell{32, 55})
	if left := pickups.effectLeft.left; pickups.effect != pickupGold || left > effectMicros || left < effectMicros-stepMicros {
		t.Fatalf("after collecting: effect %d with %d µs left, want gold with its full %d µs", pickups.effect, left, effectMicros)
	}
	if px := opaque(h.Pixel(0, 32)); px != colGold() {
		t.Errorf("border pixel after collecting = %+v, want gold", px)
	}
	if px := opaque(h.Pixel(14, 2)); px != colGold() {
		t.Errorf("announcement pixel (14,2) = %+v, want the gold \"3X POINTS\"", px)
	}
	if _, line2, _ := h.LCD(cartlib.SeatP1); !strings.HasPrefix(line2, "GOLD X3 ") {
		t.Errorf("LCD line 2 while gold runs = %q, want the effect countdown", line2)
	}

	stepToHead(t, h, cell{32, 50})
	if want := uint32(growPerEat + 3*growPerEat); snakes[0].score != want {
		t.Errorf("score after a gold apple = %d, want %d", snakes[0].score, want)
	}

	for pickups.announceLeft.running() {
		h.Tick(stepMicros)
	}
	h.Tick(stepMicros)
	if px := h.Pixel(14, 2); px.R != 0 || px.G != 0 || px.B != 0 {
		t.Errorf("announcement pixel after its window = %+v, want black", px)
	}

	pickups.effectLeft.left = 1
	h.Tick(stepMicros)
	if pickups.effect != pickupNone {
		t.Fatal("gold outlived its countdown")
	}
	if px := opaque(h.Pixel(0, 32)); px != colWall {
		t.Errorf("border pixel after gold expired = %+v, want the wall colour", px)
	}
	if _, line2, _ := h.LCD(cartlib.SeatP1); !strings.HasSuffix(line2, "high: 12") {
		t.Errorf("LCD line 2 after gold expired = %q, want the score back", line2)
	}
}

// TestAPickupNeverHidesTheApple: the pickup draws a 3×3 around its cell, so
// neither placement may put the apple inside that square.
func TestAPickupNeverHidesTheApple(t *testing.T) {
	t.Run("a new pickup steers clear of the apple", func(t *testing.T) {
		h := bootOnePlayerRun(t,
			spawnIdx(32, 60),
			foodIdx(32, 59),
			foodIdx(32, 50),   // apple two
			0,                 // chili
			foodIdx(33, 50)-1, // beside apple two if only its cell were excluded
			0,
		)
		pickups.applesToGo = 1
		h.Tick(stepMicros)

		if !pickups.window.running() {
			t.Fatal("the first apple did not bring a pickup")
		}
		if pickups.footprint().covers(field.food) {
			t.Errorf("the pickup at (%d,%d) covers the apple at %+v", pickups.x, pickups.y, field.food)
		}
		if px := opaque(h.Pixel(32, 50)); px != foodColor() {
			t.Errorf("apple pixel (32,50) = %+v, want the apple", px)
		}
	})

	t.Run("a new apple steers clear of the pickup", func(t *testing.T) {
		h := bootOnePlayerRun(t,
			spawnIdx(32, 60),
			foodIdx(32, 59),
			foodIdx(32, 50), // under the pickup's corner if nothing were excluded
		)
		pickups.kind, pickups.x, pickups.y = pickupIce, 33, 51
		pickups.window.start(pickupWindowMicros)
		h.Tick(stepMicros)

		if pickups.footprint().covers(field.food) {
			t.Errorf("the apple at %+v sits under the pickup at (33,51)", field.food)
		}
		if px := opaque(h.Pixel(int(field.food.x), int(field.food.y))); px != foodColor() {
			t.Errorf("apple pixel %+v = %+v, want the apple", field.food, px)
		}
	})

	t.Run("on a board too full for both the pickup gives way", func(t *testing.T) {
		h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(32, 31), 4)
		pickups.kind, pickups.x, pickups.y = pickupIce, 10, 10
		pickups.window.start(pickupWindowMicros)
		box := pickups.footprint()
		for y := fieldMin; y <= fieldMax; y++ {
			for x := fieldMin; x <= fieldMax; x++ {
				if !box.covers(cell{x: uint8(x), y: uint8(y)}) {
					field.occupied[y][x] = true
				}
			}
		}
		field.placeFood(field.level)

		if pickups.window.running() {
			t.Error("the pickup is still on the field with no room for the apple beside it")
		}
		if field.food != (cell{10, 10}) {
			t.Errorf("the apple is at %+v, want the scripted centre of the old box (10,10)", field.food)
		}
		h.Render()
		if px := opaque(h.Pixel(10, 10)); px != foodColor() {
			t.Errorf("pixel (10,10) = %+v, want the apple", px)
		}
	})
}

// TestALevelSwapMovesWhatItsWallsCover: the apple, and in 1P a pickup,
// were placed against the old layout; the new one's bar must not hold them.
func TestALevelSwapMovesWhatItsWallsCover(t *testing.T) {
	for _, players := range []int{1, 2} {
		t.Run(fmt.Sprintf("%dP", players), func(t *testing.T) {
			// The snakes start clear of level 2's bar, or the swap would wait.
			rand := []uint64{spawnIdx(10, 50), spawnIdx(50, 10), 0, 0}
			if players == 1 {
				rand = []uint64{spawnIdx(10, 50), 0, 0, 0}
			}
			h := bootRun(t, players, rand)
			field.food = cell{29, 31}
			if players == 1 {
				pickups.kind, pickups.x, pickups.y = pickupGold, 30, 32
				pickups.window.start(pickupWindowMicros)
			}
			field.scheduleLevel(2)
			h.Tick(stepMicros)

			if field.level != 2 {
				t.Fatalf("level %d after the swap's step, want 2", field.level)
			}
			if field.food != (cell{1, 1}) {
				t.Errorf("the apple moved to %+v, want the re-roll's (1,1)", field.food)
			}
			if wallAt(2, int(field.food.x), int(field.food.y)) {
				t.Errorf("the apple at %+v is inside level 2's walls", field.food)
			}
			if players == 1 && (!pickups.window.running() || wallAt(2, int(pickups.x), int(pickups.y))) {
				t.Errorf("the pickup is at (%d,%d), running %v; want it moved clear of the walls", pickups.x, pickups.y, pickups.window.running())
			}
			if players == 1 && pickups.footprint().covers(field.food) {
				t.Errorf("the moved pickup at (%d,%d) covers the apple at %+v", pickups.x, pickups.y, field.food)
			}
			if px := opaque(h.Pixel(1, 1)); px != foodColor() {
				t.Errorf("pixel (1,1) = %+v, want the apple", px)
			}
			for i := 0; i < players; i++ {
				if line1, _, _ := h.LCD(seats[i]); !strings.HasSuffix(line1, "L2") {
					t.Errorf("player %d's LCD line 1 = %q, want level 2", i+1, line1)
				}
			}
		})
	}
}

// TestTheGameOverLampBlinksOnTime: every 500 ms whatever the frame rate. A
// full table keeps the run off it, so the banner shows.
func TestTheGameOverLampBlinksOnTime(t *testing.T) {
	toggles := func(dtMicros uint64, frames int) int {
		h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(32, 31), foodIdx(10, 10))
		for i := 0; i < 10; i++ {
			h.Scores = append(h.Scores, hosttest.ScoreEntry{Score: 100, Name: "AAA"})
		}
		playOut(t, h)
		if machine.State() != stateGameOver {
			t.Fatalf("setup: state %d, want the GAME OVER banner", machine.State())
		}
		n, lit := 0, h.Lamps[0][0]
		for i := 0; i < frames; i++ {
			h.Tick(dtMicros)
			if h.Lamps[0][0] != lit {
				n, lit = n+1, h.Lamps[0][0]
			}
		}
		return n
	}
	// 2.2 s either way: the lamp flips at 0.5, 1.0, 1.5 and 2.0 s.
	if got := toggles(33_333, 66); got != 4 {
		t.Errorf("30 FPS: %d lamp flips in 2.2 s of banner, want 4", got)
	}
	if got := toggles(16_667, 132); got != 4 {
		t.Errorf("60 FPS: %d lamp flips in 2.2 s of banner, want 4", got)
	}
}

// TestRecentNamesHangBelowTheCharset: one scroll down from a blank name
// previews the seat's own history first, then the leaderboard.
func TestRecentNamesHangBelowTheCharset(t *testing.T) {
	cases := []struct {
		name string
		mru  []string
		want string
	}{
		{"the leaderboard when the seat has no history", nil, "ZOE"},
		{"the seat's history before the leaderboard", []string{"AVA"}, "AVA"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := bootOnePlayerRun(t, spawnIdx(32, 32), foodIdx(32, 31), foodIdx(10, 10))
			h.Scores = []hosttest.ScoreEntry{{Score: 500, Name: "ZOE"}}
			h.RecentMRU[0] = c.mru
			playOut(t, h)
			if machine.State() != stateNameEntry {
				t.Fatalf("setup: state %d, want NAME ENTRY", machine.State())
			}

			h.Input(hosttest.KeyboardPress(cartlib.KeyDown))
			h.Tick(33_333)
			h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyJoyA)...) // take it, park on OK
			h.Tick(33_333)
			h.Input(hosttest.Press(cartlib.SeatP1, cartlib.KeyJoyA)...) // submit
			h.Tick(33_333)

			if h.RecentNames[0] != c.want {
				t.Errorf("seat one's recent name = %q, want %q", h.RecentNames[0], c.want)
			}
			if len(h.Scores) != 2 || h.Scores[1].Name != c.want {
				t.Errorf("score table = %+v, want ZOE's 500 and this run under %s", h.Scores, c.want)
			}
		})
	}
}
