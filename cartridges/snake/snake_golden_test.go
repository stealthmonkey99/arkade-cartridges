// SPDX-License-Identifier: GPL-2.0-or-later

package main

// Snake's only inputs are dt_micros, host_random_u64 and input records, so
// any host fed these scripts must reproduce these frames from the checked-in
// snake.wasm byte for byte.

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/stealthmonkey99/arkade/carttest"
)

const snakeWasmPath = "snake.wasm"

const snakeTickMicros = 33333 // 30 FPS

// snake1PRandScript draws once per spawn and once per apple placed. A centre
// spawn heads up.
var snake1PRandScript = []uint64{
	spawnIdx(32, 32),
	foodIdx(32, 31),     // directly ahead of the head
	foodIdx(32, 30),     // one ahead again
	foodIdx(50, 50) - 3, // off the snake's column, never eaten; its three cells precede it
	spawnIdx(32, 32),    // respawn after death 1
	spawnIdx(32, 32),    // respawn after death 2
}

func loadSnakeWasm(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(snakeWasmPath)
	if err != nil {
		t.Fatalf("read checked-in snake cartridge (run `make -C ../.. cart-snake` to rebuild it): %v", err)
	}
	return data
}

func newSnakeCartridge(t *testing.T, host carttest.Host) *carttest.Cartridge {
	t.Helper()
	return newSnakeCartridgeWith(t, host, carttest.Options{})
}

func newSnakeCartridgeWith(t *testing.T, host carttest.Host, opts carttest.Options) *carttest.Cartridge {
	t.Helper()
	cart := newSnakeCartridgeAtSplashWith(t, host, opts)
	carttest.AdvancePastSplash(t, cart)
	return cart
}

func newSnakeCartridgeAtSplash(t *testing.T, host carttest.Host) *carttest.Cartridge {
	t.Helper()
	return newSnakeCartridgeAtSplashWith(t, host, carttest.Options{})
}

func newSnakeCartridgeAtSplashWith(t *testing.T, host carttest.Host, opts carttest.Options) *carttest.Cartridge {
	t.Helper()
	cart, err := carttest.NewCartridge(context.Background(), loadSnakeWasm(t), host, opts) // default 64x64
	if err != nil {
		t.Fatalf("NewCartridge(snake.wasm): %v", err)
	}
	t.Cleanup(func() { _ = cart.Close(context.Background()) })
	if err := cart.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return cart
}

func snakeTick(t *testing.T, cart *carttest.Cartridge, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		if err := cart.Update(ctx, snakeTickMicros); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if err := cart.Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
	}
}

func snakePress(t *testing.T, cart *carttest.Cartridge, rec carttest.InputRecord) {
	t.Helper()
	if err := cart.DispatchInput(context.Background(), []carttest.InputRecord{rec}); err != nil {
		t.Fatalf("DispatchInput(%+v): %v", rec, err)
	}
}

// playSnake1PJourney plays a solo run from seat 2: two apples, a quit
// detour, three deaths against the top wall, a blank name entry, and quit.
// checkpoint, if non-nil, sees each named frame.
func playSnake1PJourney(t *testing.T, host *carttest.FakeHost, checkpoint func(name string)) {
	t.Helper()
	playSnake1PJourneyWith(t, host, carttest.Options{}, checkpoint)
}

func playSnake1PJourneyWith(t *testing.T, host *carttest.FakeHost, opts carttest.Options, checkpoint func(name string)) {
	t.Helper()
	ctx := context.Background()
	cart := newSnakeCartridgeWith(t, host, opts)

	mark := func(name string) {
		if checkpoint != nil {
			checkpoint(name)
		}
	}

	if err := cart.Render(ctx); err != nil {
		t.Fatalf("Render: %v", err)
	}
	mark("snake-cart-title")

	// From seat 2, so every peripheral call after this must address seat 2.
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})

	snakeTick(t, cart, 1)
	mark("snake-cart-getready")

	// Play starts at tick 61; the snake eats at ticks 63 and 65.
	snakeTick(t, cart, 74)
	mark("snake-cart-playing")

	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyEscape, Player: 2, Physical: 1})
	snakeTick(t, cart, 3)
	mark("snake-cart-confirm")

	// JoyC cancels the dialog without steering; the snake hits the top wall
	// at tick 125.
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyC, Player: 2, Physical: 1})
	snakeTick(t, cart, 47)
	mark("snake-cart-lostlife")

	// Deaths at ticks 217 and 309; the round ends after the freeze, at 340.
	snakeTick(t, cart, 215)
	mark("snake-cart-nameentry")

	// On a blank slot, the first A jumps to OK and the second submits.
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})

	snakeTick(t, cart, 1) // the submit happens in Update
	mark("snake-cart-title-hiscore")

	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyQuit, Player: 2, Physical: 1})

	if err := cart.Teardown(ctx); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
}

func TestSnakeCartridgeGoldens(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake1PRandScript

	playSnake1PJourney(t, host, func(name string) {
		if len(host.Blits) == 0 {
			t.Fatalf("checkpoint %q reached with no blits recorded", name)
		}
		carttest.GoldenFrame(t, name, carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))
	})

	// A blank name is a legal anonymous submit.
	if score, name, ok := host.ScoreRead(1); !ok || score != 6 || name != "" {
		t.Errorf("score table rank 1 = (%d, %q, %v), want (6, \"\", true)", score, name, ok)
	}
	if n := host.ScoreCount(); n != 1 {
		t.Errorf("score table count = %d, want 1 (the one finished run)", n)
	}
	if name, ok := host.RecentName(1); ok {
		t.Errorf("RecentName(1) = (%q, true), want no recent name noted for a blank submit", name)
	}

	// A fresh instance reads the high score back in Init.
	cart2 := newSnakeCartridge(t, host)
	if err := cart2.Render(context.Background()); err != nil {
		t.Fatalf("Render (fresh instance): %v", err)
	}
	carttest.GoldenFrame(t, "snake-cart-title-hiscore", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))
}

// snakeReachNameEntry replays playSnake1PJourney up to the blank name entry.
func snakeReachNameEntry(t *testing.T, host *carttest.FakeHost) *carttest.Cartridge {
	t.Helper()
	cart := newSnakeCartridge(t, host)
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakeTick(t, cart, 75)
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyEscape, Player: 2, Physical: 1})
	snakeTick(t, cart, 3)
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyC, Player: 2, Physical: 1})
	snakeTick(t, cart, 47+215)
	return cart
}

// TestSnakeCartridgeNameEntryArrowKeys: a terminal sends only virtual
// keypresses, never key release, and those must still move the cursor. A
// control run ticked the same frame cancels out the cursor blink.
func TestSnakeCartridgeNameEntryArrowKeys(t *testing.T) {
	pressedHost := newSnakeHost(t)
	pressedHost.RandScript = snake1PRandScript
	pressedCart := snakeReachNameEntry(t, pressedHost)

	controlHost := newSnakeHost(t)
	controlHost.RandScript = snake1PRandScript
	controlCart := snakeReachNameEntry(t, controlHost)

	if len(pressedHost.Blits) == 0 || len(controlHost.Blits) == 0 {
		t.Fatalf("no blits recorded reaching name entry")
	}
	if !reflect.DeepEqual(pressedHost.Blits[len(pressedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Fatalf("identical scripted journeys produced different name-entry frames before either diverged")
	}

	snakePress(t, pressedCart, carttest.InputRecord{
		Kind: carttest.RecordKindKeypress, Key: carttest.KeyRight, Player: 2, Physical: 1, Flags: carttest.RecordFlagVirtual,
	})
	snakeTick(t, pressedCart, 1)
	snakeTick(t, controlCart, 1) // same elapsed time, no press: isolates the blink-phase confound

	if reflect.DeepEqual(pressedHost.Blits[len(pressedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Fatal("a FlagVirtual keypress right-arrow during name entry produced no visible change — " +
			"the held-direction tracker is dropping keypress-only records (the terminal's exact " +
			"shape), so the arrows are dead")
	}
}

func TestSnakeCartridgeSplash(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake1PRandScript
	cart := newSnakeCartridgeAtSplash(t, host)
	if err := cart.Render(context.Background()); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(host.Blits) == 0 {
		t.Fatal("no splash frame blitted")
	}
	carttest.GoldenFrame(t, "snake-cart-splash", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))
}

func TestSnakeCartridgeAttract(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = []uint64{0, 1, 0, 1, 1, 0} // attract snake's turn coin-flips
	host.Scores.Insert(120, "AAA")
	host.Scores.Insert(90, "BOB")

	cart := newSnakeCartridgeAtSplash(t, host)
	if !cart.SupportsAttract() {
		t.Fatal("snake.wasm does not export cart_attract")
	}
	ctx := context.Background()
	for i := 0; i < 40; i++ {
		if err := cart.Attract(ctx, snakeTickMicros); err != nil {
			t.Fatalf("Attract tick %d: %v", i, err)
		}
	}
	// The attract instance never runs the load phase, so the loader's eat cue
	// must not sound.
	for _, c := range host.Sounds {
		if c.Action == "play" {
			t.Errorf("host_play_sound called during attract: %+v (attract must stay silent)", c)
		}
	}
	if len(host.Blits) == 0 {
		t.Fatal("cart_attract produced no blit")
	}
	frame := host.Blits[len(host.Blits)-1]

	// An overlay: some pixels transparent for the starfield, some opaque.
	var sawTransparent, sawOpaque bool
	for _, p := range frame {
		switch p.A {
		case 0:
			sawTransparent = true
		case 0xFF:
			sawOpaque = true
		}
	}
	if !sawTransparent {
		t.Error("attract frame has no transparent pixels — it isn't overlaying the starfield")
	}
	if !sawOpaque {
		t.Error("attract frame drew nothing opaque")
	}
	carttest.GoldenFrame(t, "snake-cart-attract", carttest.MatrixFrame(t, frame))
}

// snakePeripheralsGoldenPath pins the 1P journey's LCD and lamp calls for any
// host to check itself against.
const snakePeripheralsGoldenPath = "testdata/golden/snake-cart-peripherals.json"

func TestSnakeCartridgePeripherals(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake1PRandScript
	playSnake1PJourney(t, host, nil)

	carttest.GoldenPeripherals(t, host, snakePeripheralsGoldenPath)
	assertNoErrorLogs(t, host)
}

func TestSnakeCartridgeDeterminism(t *testing.T) {
	run := func() *carttest.FakeHost {
		host := newSnakeHost(t)
		host.RandScript = snake1PRandScript
		playSnake1PJourney(t, host, nil)
		return host
	}
	a, b := run(), run()
	carttest.AssertBlitsIdentical(t, a, b)
}

// snake2PRandScript: a tie between axes goes vertical, so (3,3) heads down.
var snake2PRandScript = []uint64{
	spawnIdx(3, 3),      // P1
	spawnIdx(32, 32),    // P2, heading up
	foodIdx(32, 31) - 1, // ahead of P2; P1's spawn cell precedes it
	foodIdx(33, 31) - 3, // beside P2's head, never eaten; three taken cells precede it
	spawnIdx(3, 3),      // P1's respawn after its 1st death
	spawnIdx(3, 3),      // and after its 2nd
}

// playSnake2PJourney: P2 eats one apple while P1, steered left every 5
// ticks, hits the left wall three times. P2 wins and enters a blank name.
func playSnake2PJourney(t *testing.T, host *carttest.FakeHost, checkpoint func(name string)) {
	t.Helper()
	playSnake2PJourneyWith(t, host, carttest.Options{}, checkpoint)
}

func playSnake2PJourneyWith(t *testing.T, host *carttest.FakeHost, opts carttest.Options, checkpoint func(name string)) {
	t.Helper()
	ctx := context.Background()
	cart := newSnakeCartridgeWith(t, host, opts)

	mark := func(name string) {
		if checkpoint != nil {
			checkpoint(name)
		}
	}

	if err := cart.Render(ctx); err != nil {
		t.Fatalf("Render: %v", err)
	}

	// FlagVirtual models a keyboard press; the menu steps on PressDir.
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyDown, Player: 1, Flags: carttest.RecordFlagVirtual})
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 1})

	snakeTick(t, cart, 1)
	mark("snake-cart-2p-getready")

	snakeTick(t, cart, 60) // play starts at tick 61

	steerP1Left := func(elapsed int) {
		if elapsed%5 == 0 {
			snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyLeft, Player: 1})
		}
	}

	// P2 eats at tick 63.
	for i := 62; i <= 65; i++ {
		steerP1Left(i)
		snakeTick(t, cart, 1)
	}
	mark("snake-cart-2p-playing")

	// P1's first death, at tick 71.
	for i := 66; i <= 71; i++ {
		steerP1Left(i)
		snakeTick(t, cart, 1)
	}
	mark("snake-cart-2p-lostlife")

	// Deaths at ticks 110 and 151; P2 wins after the freeze, at 182.
	for i := 72; i <= 182; i++ {
		steerP1Left(i)
		snakeTick(t, cart, 1)
	}
	mark("snake-cart-2p-nameentry")

	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakeTick(t, cart, 1)
	mark("snake-cart-2p-title-hiscore")

	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyQuit, Player: 1})

	if err := cart.Teardown(ctx); err != nil {
		t.Fatalf("Teardown: %v", err)
	}
}

// TestSnakeCompilerMatchesInterpreter guards against a wasm compiler that
// executes a valid build wrongly: wazero v1.9.0's arm64 compiler did, and the
// goldens recorded on it could not tell.
func TestSnakeCompilerMatchesInterpreter(t *testing.T) {
	for _, j := range []struct {
		name   string
		rand   []uint64
		replay func(*testing.T, *carttest.FakeHost, carttest.Options, func(string))
	}{
		{"1P", snake1PRandScript, playSnake1PJourneyWith},
		{"2P", snake2PRandScript, playSnake2PJourneyWith},
	} {
		t.Run(j.name, func(t *testing.T) {
			carttest.RequireRuntimesAgree(t, func(opts carttest.Options) *carttest.FakeHost {
				host := newSnakeHost(t)
				host.RandScript = j.rand
				j.replay(t, host, opts, nil)
				return host
			})
		})
	}
}

func TestSnakeCartridge2PGoldens(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake2PRandScript

	playSnake2PJourney(t, host, func(name string) {
		if len(host.Blits) == 0 {
			t.Fatalf("checkpoint %q reached with no blits recorded", name)
		}
		carttest.GoldenFrame(t, name, carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))
	})

	if score, name, ok := host.ScoreRead(1); !ok || score != 3 || name != "" {
		t.Errorf("score table rank 1 = (%d, %q, %v), want (3, \"\", true) — the WINNER's score only", score, name, ok)
	}
	if n := host.ScoreCount(); n != 1 {
		t.Errorf("score table count = %d, want 1 (only the winner's run)", n)
	}
}

const snake2PPeripheralsGoldenPath = "testdata/golden/snake-cart-2p-peripherals.json"

func TestSnakeCartridge2PPeripherals(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake2PRandScript
	playSnake2PJourney(t, host, nil)

	carttest.GoldenPeripherals(t, host, snake2PPeripheralsGoldenPath)
	assertNoErrorLogs(t, host)
}

func TestSnakeCartridge2PDeterminism(t *testing.T) {
	run := func() *carttest.FakeHost {
		host := newSnakeHost(t)
		host.RandScript = snake2PRandScript
		playSnake2PJourney(t, host, nil)
		return host
	}
	a, b := run(), run()
	carttest.AssertBlitsIdentical(t, a, b)
}

// assertNoErrorLogs catches a refused screen transition, which Snake logs.
func assertNoErrorLogs(t *testing.T, host *carttest.FakeHost) {
	t.Helper()
	host.Mu.Lock()
	defer host.Mu.Unlock()
	for _, l := range host.Logs {
		if l.Level == carttest.LogError {
			t.Errorf("cartridge logged an error: %q", l.Msg)
		}
	}
}

// TestSnakeWasmMatchesSource catches the one drift the goldens cannot: a
// stale checked-in binary.
func TestSnakeWasmMatchesSource(t *testing.T) {
	carttest.RequireWasmMatchesSource(t, ".", "snake.wasm", "make -C ../.. cart-snake")
}

func TestSnakeAllocatesNothingAfterInit(t *testing.T) {
	carttest.RequireNoAllocsAfterInit(t, ".")
}

// TestSnakeCartridgeNameEntryEscapesOnB: C erases, and B reaches the
// cartridge's quit overlay rather than the widget.
func TestSnakeCartridgeNameEntryEscapesOnB(t *testing.T) {
	erasedHost := newSnakeHost(t)
	erasedHost.RandScript = snake1PRandScript
	erasedCart := snakeReachNameEntry(t, erasedHost)

	controlHost := newSnakeHost(t)
	controlHost.RandScript = snake1PRandScript
	controlCart := snakeReachNameEntry(t, controlHost)

	for _, cart := range []*carttest.Cartridge{erasedCart, controlCart} {
		snakePress(t, cart, carttest.InputRecord{
			Kind: carttest.RecordKindKeypress, Key: carttest.KeyDown, Player: 2, Physical: 1, Flags: carttest.RecordFlagVirtual,
		})
		snakeTick(t, cart, 1)
	}
	if !reflect.DeepEqual(erasedHost.Blits[len(erasedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Fatalf("identical scripted journeys diverged before either was given a C press")
	}

	snakePress(t, erasedCart, carttest.InputRecord{
		Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyC, Player: 2, Physical: 1, Flags: carttest.RecordFlagVirtual,
	})
	snakeTick(t, erasedCart, 1)
	snakeTick(t, controlCart, 1)
	if reflect.DeepEqual(erasedHost.Blits[len(erasedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Errorf("C during name entry changed nothing — erase is supposed to live on C now")
	}

	openHost := newSnakeHost(t)
	openHost.RandScript = snake1PRandScript
	openCart := snakeReachNameEntry(t, openHost)

	quietHost := newSnakeHost(t)
	quietHost.RandScript = snake1PRandScript
	quietCart := snakeReachNameEntry(t, quietHost)

	snakePress(t, openCart, carttest.InputRecord{
		Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyB, Player: 2, Physical: 1,
	})
	snakeTick(t, openCart, 1)
	snakeTick(t, quietCart, 1) // same elapsed time, no press: isolates the blink-phase confound
	if reflect.DeepEqual(openHost.Blits[len(openHost.Blits)-1], quietHost.Blits[len(quietHost.Blits)-1]) {
		t.Error("B during name entry changed nothing on screen — the widget is eating the press again, " +
			"which is the wedge that stranded players on the cabinet")
	}
}

// snakeNameEntryHoldB sends the quit gesture as the cabinet reports it: the
// opening press must be released before the hold counts.
func snakeNameEntryHoldB(t *testing.T, cart *carttest.Cartridge) {
	t.Helper()
	b := func(kind uint8) carttest.InputRecord {
		return carttest.InputRecord{Kind: kind, Key: carttest.KeyJoyB, Player: 2, Physical: 1}
	}
	snakePress(t, cart, b(carttest.RecordKindKeydown))
	snakePress(t, cart, b(carttest.RecordKindKeypress)) // opens the overlay
	snakePress(t, cart, b(carttest.RecordKindKeyup))    // arms the hold
	snakeTick(t, cart, 2)

	snakePress(t, cart, b(carttest.RecordKindKeydown)) // the hold proper
	snakeTick(t, cart, 60)                             // ~2.0s: past the 1s fill and the ~0.8s blink
	snakePress(t, cart, b(carttest.RecordKindKeyup))
	snakeTick(t, cart, 2)
}

// TestSnakeCartridgeNameEntryBareBKeepsTheScore: a mash of B neither exits
// nor submits, another button returns to a live entry, and only the full
// hold leaves, abandoning the placement.
func TestSnakeCartridgeNameEntryBareBKeepsTheScore(t *testing.T) {
	host := newSnakeHost(t)
	host.RandScript = snake1PRandScript
	cart := snakeReachNameEntry(t, host)
	if host.Exits != 0 || host.ScoreCount() != 0 {
		t.Fatalf("setup: reached name entry with Exits=%d ScoreCount=%d, want 0 and 0", host.Exits, host.ScoreCount())
	}

	for i := 0; i < 6; i++ {
		snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeydown, Key: carttest.KeyJoyB, Player: 2, Physical: 1})
		snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyB, Player: 2, Physical: 1})
		snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeyup, Key: carttest.KeyJoyB, Player: 2, Physical: 1})
		snakeTick(t, cart, 2)
		if host.Exits != 0 {
			t.Fatalf("bare B press #%d left the cartridge — a fumbled B must never discard a placement", i+1)
		}
		if i == 0 {
			// The overlay must sit over the name entry, not the play field.
			carttest.GoldenFrame(t, "snake-cart-nameentry-confirm", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))
		}
	}
	if host.ScoreCount() != 0 {
		t.Errorf("score table has %d rows after a B mash during name entry, want 0 (nothing was confirmed)", host.ScoreCount())
	}

	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakeTick(t, cart, 1)
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyA, Player: 2, Physical: 1})
	snakeTick(t, cart, 1)
	if host.ScoreCount() != 1 {
		t.Errorf("score table has %d rows after cancelling the overlay and finishing the entry, want 1 — "+
			"the placement did not survive the B detour", host.ScoreCount())
	}
	if host.Exits != 0 {
		t.Errorf("cartridge exited %d times on the resume-and-submit path, want 0", host.Exits)
	}

	holdHost := newSnakeHost(t)
	holdHost.RandScript = snake1PRandScript
	holdCart := snakeReachNameEntry(t, holdHost)
	snakeNameEntryHoldB(t, holdCart)
	if holdHost.Exits == 0 {
		t.Error("a full press-release-hold B during name entry did not leave the cartridge — " +
			"the deliberate quit gesture has to still work, or the modal is a trap again")
	}
	if holdHost.ScoreCount() != 0 {
		t.Errorf("holding B out of name entry submitted %d rows, want 0 — quitting abandons the placement", holdHost.ScoreCount())
	}
}

func TestSnakeManifestCopiesAgree(t *testing.T) {
	carttest.ManifestCopiesAgree(t, newSnakeCartridgeAtSplash(t, newSnakeHost(t)), "manifest.json", "make -C ../.. cart-snake")
}

// TestSnakeCartridgeNameEntryResumeDoesNotScroll: a direction pulse sent
// with B, before any tick, must not scroll the initials once the overlay is
// cancelled. Held credit only decays in name entry's Update, so the overlay
// would otherwise preserve it.
func TestSnakeCartridgeNameEntryResumeDoesNotScroll(t *testing.T) {
	reach := func() (*carttest.Cartridge, *carttest.FakeHost) {
		host := newSnakeHost(t)
		host.RandScript = snake1PRandScript
		return snakeReachNameEntry(t, host), host
	}
	pulsedCart, pulsedHost := reach()
	controlCart, controlHost := reach()

	// 250 ms of held credit, and no tick, so the widget never sees it.
	snakePress(t, pulsedCart, carttest.InputRecord{
		Kind: carttest.RecordKindKeypress, Key: carttest.KeyDown, Player: 2, Physical: 1, Flags: carttest.RecordFlagVirtual,
	})

	for _, cart := range []*carttest.Cartridge{pulsedCart, controlCart} {
		snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyB, Player: 2, Physical: 1})
		snakeTick(t, cart, 1)
	}
	if !reflect.DeepEqual(pulsedHost.Blits[len(pulsedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Fatalf("the runs diverged before the overlay was dismissed — the pulse moved the widget, " +
			"so anything measured after the cancel would not isolate the phantom")
	}

	// Two seconds on the overlay, eight times the pulse's lifetime.
	for _, cart := range []*carttest.Cartridge{pulsedCart, controlCart} {
		snakeTick(t, cart, 60)
		// The overlay consumes JoyC, so it never reaches the entry as an erase.
		snakePress(t, cart, carttest.InputRecord{Kind: carttest.RecordKindKeypress, Key: carttest.KeyJoyC, Player: 2, Physical: 1})
		snakeTick(t, cart, 3)
	}

	if !reflect.DeepEqual(pulsedHost.Blits[len(pulsedHost.Blits)-1], controlHost.Blits[len(controlHost.Blits)-1]) {
		t.Error("cancelling the quit overlay scrolled the initials of the run that had nudged a direction " +
			"before pressing B — the stale held credit survived the modal. Entering stateNameEntry must " +
			"clear heldDirs on EVERY arrival, the resume included (game.go's enterScreen).")
	}
}
