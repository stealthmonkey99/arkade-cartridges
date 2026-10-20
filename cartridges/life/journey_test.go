// SPDX-License-Identifier: GPL-2.0-or-later

package main

// These boot the real Game through hosttest and read what the fake cabinet
// saw. The wasm goldens keep pinning the frames; this pins the rules.

import (
	"strings"
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
)

// recipeRoll is the RandomU64 value that makes pickRecipe choose name: it
// walks the table in order, subtracting weights, and a roll under the total
// is its own modulus.
func recipeRoll(name string) uint64 {
	roll := 0
	for _, r := range recipes() {
		if r.name == name {
			return uint64(roll)
		}
		roll += r.weight
	}
	panic("no seed recipe " + name)
}

// diehardScript seeds diehard at zero jitter: centeredStamp's two rolls of
// 10 make randIntn(21)-10 zero.
func diehardScript() []uint64 { return []uint64{recipeRoll("diehard"), 10, 10} }

// blinkerSoupScript seeds the soup with exactly three live cells, a
// horizontal blinker at y=32, x=31..33: 99 is dead and 0 alive against
// soupDensityPercent, in placeSoup's row-major scan of its centred patch.
func blinkerSoupScript() []uint64 {
	const dead, alive uint64 = 99, 0
	x0, y0 := (cartlib.FBWidth-soupSize)/2, (cartlib.FBHeight-soupSize)/2
	script := make([]uint64, 1+soupSize*soupSize)
	script[0] = recipeRoll("soup")
	for i := 1; i < len(script); i++ {
		script[i] = dead
	}
	for _, x := range []int{31, 32, 33} {
		script[1+(32-y0)*soupSize+(x-x0)] = alive
	}
	return script
}

// bootLife starts a fresh instance past the splash, seeded with diehard at
// zero jitter. The process outlives one cabinet instance, so the game is
// zeroed first.
func bootLife(t *testing.T) *hosttest.Host {
	t.Helper()
	*game = lifeGame{}
	h := hosttest.New(t)
	h.Rand = diehardScript()
	h.Init(game)
	h.AdvancePastSplash()
	return h
}

func press(h *hosttest.Host, key uint8) { h.Input(hosttest.Press(cartlib.SeatP1, key)...) }

func stick(h *hosttest.Host, joy uint8) {
	h.Input(hosttest.Stick(cartlib.SeatP1, joy), hosttest.Stick(cartlib.SeatP1, cartlib.JoyCenter))
}

func lcd(h *hosttest.Host) (line1, line2 string) {
	line1, line2, _ = h.LCD(cartlib.SeatP1)
	return line1, line2
}

func TestEveryQuitKeyLeavesTheFieldAtOnce(t *testing.T) {
	for _, key := range []uint8{cartlib.KeyQuit, cartlib.KeyEscape, cartlib.KeyJoyB} {
		h := bootLife(t)
		h.Input(hosttest.KeyboardPress(key))
		if h.Exits != 1 {
			t.Errorf("key %d: %d exits, want 1 with no hold-confirm", key, h.Exits)
		}
	}
}

func TestBClosesTheOverlayBeforeItQuits(t *testing.T) {
	h := bootLife(t)
	press(h, cartlib.KeyJoyA)
	if game.machine.State() != stOverlay {
		t.Fatalf("A on the field: state %d, want the overlay", game.machine.State())
	}
	press(h, cartlib.KeyJoyB)
	if h.Exits != 0 || game.machine.State() != stPlaying {
		t.Fatalf("B on the overlay: %d exits, state %d; want it closed, not quit", h.Exits, game.machine.State())
	}
	press(h, cartlib.KeyJoyB)
	if h.Exits != 1 {
		t.Errorf("B on the field: %d exits, want 1", h.Exits)
	}
}

func TestTheMenuLCDsNameTheMenuAndNoKeys(t *testing.T) {
	h := bootLife(t)
	press(h, cartlib.KeyJoyA)
	if l1, l2 := lcd(h); l1 != "MENU" || l2 != "" {
		t.Errorf("overlay LCD = %q / %q, want the title alone", l1, l2)
	}
	stick(h, cartlib.JoyS) // STYLE -> SPEED
	stick(h, cartlib.JoyS) // SPEED -> DRAW
	press(h, cartlib.KeyJoyA)
	if l1, l2 := lcd(h); l1 != "PICK A SEED" || l2 != "" {
		t.Errorf("seed picker LCD = %q / %q, want the title alone", l1, l2)
	}
}

// TestStyleAndSpeedApplyAndPersistAtOnce: the settings blob is
// [style, speed detent, 0xA5].
func TestStyleAndSpeedApplyAndPersistAtOnce(t *testing.T) {
	h := bootLife(t)
	if game.style != styleClassic {
		t.Fatalf("setup: style %d, want the default CLASSIC", game.style)
	}

	press(h, cartlib.KeyJoyA)
	stick(h, cartlib.JoyW) // STYLE wraps CLASSIC -> HEAT
	press(h, cartlib.KeyJoyB)
	if game.style != styleHeat {
		t.Errorf("style after LEFT on STYLE = %d, want HEAT (%d)", game.style, styleHeat)
	}
	h.Render()
	if c := firstLiveCell(h); c.G != 255 || c.R == 255 {
		t.Errorf("a live cell renders %+v, want a fresh HEAT cell's green", c)
	}
	if blob := h.Storage[settingsStorageKey]; len(blob) != 3 || blob[0] != styleHeat {
		t.Fatalf("settings blob after the style change = %v, want style %d persisted at once", blob, styleHeat)
	}

	press(h, cartlib.KeyJoyA)
	stick(h, cartlib.JoyS)
	press(h, cartlib.KeyJoyA) // one detent up from the centre
	if blob := h.Storage[settingsStorageKey]; len(blob) != 3 || blob[1] != speedDetentDefault+1 {
		t.Fatalf("settings blob after A on SPEED = %v, want detent %d", blob, speedDetentDefault+1)
	}
}

func firstLiveCell(h *hosttest.Host) cartlib.Color {
	for y := 0; y < cartlib.FBHeight; y++ {
		for x := 0; x < cartlib.FBWidth; x++ {
			if c := h.Pixel(x, y); c.R != 0 || c.G != 0 || c.B != 0 {
				return c
			}
		}
	}
	return cartlib.Color{}
}

// TestAnEmptyWorldReseedsWithoutAFade: diehard dies out at generation 131,
// whose empty hash 132 repeats, so the next step reseeds at once.
func TestAnEmptyWorldReseedsWithoutAFade(t *testing.T) {
	h := bootLife(t)
	for i := 0; i < 130; i++ {
		h.Tick(generationStepMicros)
	}
	if l1, _ := lcd(h); l1 != "Gen: 131" || game.field.population() != 0 {
		t.Fatalf("setup: LCD %q, population %d; want diehard extinct at Gen: 131", l1, game.field.population())
	}
	h.Tick(generationStepMicros)
	if l1, _ := lcd(h); l1 != "Gen: 1" || game.fading {
		t.Errorf("one step past extinction: LCD %q, fading %v; want a fresh world at once", l1, game.fading)
	}
}

// TestTheReadoutRefreshesAtFourHertz runs a second at 30 FPS, faster than
// the throttle; each refresh shows a new generation, so changes count pushes.
func TestTheReadoutRefreshesAtFourHertz(t *testing.T) {
	h := bootLife(t)
	refreshes, last := 0, ""
	for i := 0; i < 30; i++ {
		h.Tick(33_333)
		if l1, _ := lcd(h); l1 != last {
			refreshes, last = refreshes+1, l1
		}
	}
	// 4 Hz, plus slack for an extinction push and frame alignment.
	if refreshes > 6 || refreshes == 0 {
		t.Errorf("%d readout refreshes in one second, want 1..6", refreshes)
	}
	if !strings.HasPrefix(last, "Gen: ") {
		t.Errorf("the readout shows %q, want the generation", last)
	}
}
