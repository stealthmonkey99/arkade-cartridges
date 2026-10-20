// SPDX-License-Identifier: GPL-2.0-or-later

package main

// These tests drive the checked-in life.wasm. Time comes only from
// cart_update's dt_micros and randomness only from host_random_u64, so a
// scripted run must reproduce byte-identical frames on any conformant host.

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stealthmonkey99/arkade/carttest"
)

const lifeWasmPath = "life.wasm"

func loadLifeWasm(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(lifeWasmPath)
	if err != nil {
		t.Fatalf("read checked-in cartridges/life/life.wasm (run `make -C ../.. cart-life` to rebuild it): %v", err)
	}
	return data
}

func newLifeCartridge(t *testing.T, host carttest.Host) *carttest.Cartridge {
	t.Helper()
	cart := newLifeCartridgeAtSplash(t, host)
	carttest.AdvancePastSplash(t, cart)
	return cart
}

func newLifeCartridgeAtSplash(t *testing.T, host carttest.Host) *carttest.Cartridge {
	t.Helper()
	cart, err := carttest.NewCartridge(context.Background(), loadLifeWasm(t), host, carttest.Options{}) // default 64x64
	if err != nil {
		t.Fatalf("NewCartridge(life.wasm): %v", err)
	}
	t.Cleanup(func() { _ = cart.Close(context.Background()) })
	if err := cart.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return cart
}

// livePixels returns the (x, y) coordinates of every non-black pixel in a
// blit frame.
func livePixels(pixels []carttest.Pixel) [][2]int {
	var out [][2]int
	for i, p := range pixels {
		if p.R != 0 || p.G != 0 || p.B != 0 {
			out = append(out, [2]int{i % carttest.DefaultWidth, i / carttest.DefaultWidth})
		}
	}
	return out
}

func lifeTick(t *testing.T, cart *carttest.Cartridge, dtMicros uint64, n int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		if err := cart.Update(ctx, dtMicros); err != nil {
			t.Fatalf("Update: %v", err)
		}
		if err := cart.Render(ctx); err != nil {
			t.Fatalf("Render: %v", err)
		}
	}
}

// TestLifeCartridgeManifestAndABI checks a supported ABI version, a
// single-player "life" manifest, and that Render before any Update works.
func TestLifeCartridgeManifestAndABI(t *testing.T) {
	host := carttest.NewFakeHost()
	host.RandScript = diehardScript()

	cart, err := carttest.NewCartridge(context.Background(), loadLifeWasm(t), host, carttest.Options{})
	if err != nil {
		t.Fatalf("NewCartridge(life.wasm): %v", err)
	}
	t.Cleanup(func() { _ = cart.Close(context.Background()) })

	version, err := cart.ABIVersion(context.Background())
	if err != nil {
		t.Fatalf("ABIVersion: %v", err)
	}
	// A host states the range it accepts, so check the range, not one number.
	if version < carttest.MinSupportedABIVersion || version > carttest.SupportedABIVersion {
		t.Fatalf("ABIVersion = %d, want %d..%d",
			version, carttest.MinSupportedABIVersion, carttest.SupportedABIVersion)
	}

	manifestRaw, err := cart.Manifest(context.Background())
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	var manifest struct {
		ABI     int    `json:"abi"`
		Slug    string `json:"slug"`
		Name    string `json:"name"`
		Players []int  `json:"players"`
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatalf("parsing cart_manifest JSON %q: %v", manifestRaw, err)
	}
	if manifest.ABI != 1 || manifest.Slug != "life" || manifest.Name != "Life" {
		t.Fatalf("cart_manifest = %+v, want abi=1 slug=life name=Life", manifest)
	}
	if len(manifest.Players) != 1 || manifest.Players[0] != 1 {
		t.Fatalf("cart_manifest.players = %v, want [1]", manifest.Players)
	}

	if err := cart.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := cart.Render(context.Background()); err != nil {
		t.Fatalf("Render before first Update: %v", err)
	}
	if len(host.Blits) != 1 {
		t.Fatalf("blits after the init render = %d, want 1", len(host.Blits))
	}

	// -1 is the ABI's both-players sentinel.
	const wantBothPlayers = -1
	if len(host.LightSets) != 1 || host.LightSets[0].Player != wantBothPlayers || host.LightSets[0].On {
		t.Errorf("host_set_lights calls = %+v, want one {Player: %d, On: false}", host.LightSets, wantBothPlayers)
	}
}

// TestLifeCartridgeDiehardExtinction checks that diehard, with zero jitter,
// vanishes after exactly 130 generations without an early cycle-detector
// reseed. The field's gen starts at 1, so extinction shows at gen 131 after
// 130 ticks.
func TestLifeCartridgeDiehardExtinction(t *testing.T) {
	const extinctionGeneration = 131
	const extinctionTicks = extinctionGeneration - 1

	host := carttest.NewFakeHost()
	host.RandScript = diehardScript()
	cart := newLifeCartridge(t, host)

	// Diehard has 7 live cells, and Init has already pushed the LCD.
	if err := cart.Render(context.Background()); err != nil {
		t.Fatalf("Render before first Update: %v", err)
	}
	live := livePixels(host.Blits[len(host.Blits)-1])
	if len(live) != 7 {
		t.Fatalf("initial live cell count = %d, want 7 (diehard)", len(live))
	}
	if got := len(host.LCDTexts); got == 0 {
		t.Fatal("no host_set_text calls recorded after Init")
	}
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 1" || last.Line2 != "Pop: 7" {
		t.Fatalf("initial LCD text = %+v, want {Line1: %q, Line2: %q}", last, "Gen: 1", "Pop: 7")
	}

	carttest.GoldenFrame(t, "life-cart-diehard-initial", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))

	lifeTick(t, cart, generationStepMicros, 1)
	if pop := len(livePixels(host.Blits[len(host.Blits)-1])); pop == 0 {
		t.Fatal("diehard died on generation 2 - want it alive until generation 131")
	}
	carttest.GoldenFrame(t, "life-cart-diehard-gen2", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))

	lifeTick(t, cart, generationStepMicros, extinctionTicks-2)
	if pop := len(livePixels(host.Blits[len(host.Blits)-1])); pop == 0 {
		t.Fatalf("diehard died before generation %d - extinction landed early", extinctionGeneration-1)
	}
	lifeTick(t, cart, generationStepMicros, 1)

	final := host.Blits[len(host.Blits)-1]
	if live := livePixels(final); len(live) != 0 {
		t.Fatalf("live cells at generation %d = %v, want none (diehard's canonical extinction)", extinctionGeneration, live)
	}
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 131" || last.Line2 != "Pop: 0" {
		t.Fatalf("extinction LCD text = %+v, want {Line1: %q, Line2: %q}", last, "Gen: 131", "Pop: 0")
	}
	carttest.GoldenFrame(t, "life-cart-diehard-extinct", carttest.MatrixFrame(t, final))
}

// TestLifeCartridgeBlinkerOscillates checks a blinker's horizontal-to-vertical
// toggle, then that generation 3, which repeats generation 1's grid, is
// terminal and the world reseeds after the fade instead of oscillating on.
func TestLifeCartridgeBlinkerOscillates(t *testing.T) {
	horizontal := [][2]int{{31, 32}, {32, 32}, {33, 32}}
	vertical := [][2]int{{32, 31}, {32, 32}, {32, 33}}

	host := carttest.NewFakeHost()
	host.RandScript = blinkerSoupScript()
	cart := newLifeCartridge(t, host)

	if err := cart.Render(context.Background()); err != nil {
		t.Fatalf("Render before first Update: %v", err)
	}
	assertLiveCells(t, "generation 1 (seeded)", livePixels(host.Blits[len(host.Blits)-1]), horizontal)
	carttest.GoldenFrame(t, "life-cart-blinker-horizontal", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))

	lifeTick(t, cart, generationStepMicros, 1)
	assertLiveCells(t, "generation 2 (vertical phase)", livePixels(host.Blits[len(host.Blits)-1]), vertical)
	// The LCD still reads Gen 1: the 4 Hz throttle outpaces 100 ms steps.
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 1" {
		t.Fatalf("LCD text after one tick = %+v, want Line1 %q — the readout is throttled to "+
			"%dus, so one 100ms generation step must NOT have moved it", last, "Gen: 1", 250_000)
	}
	carttest.GoldenFrame(t, "life-cart-blinker-vertical", carttest.MatrixFrame(t, host.Blits[len(host.Blits)-1]))

	// Generation 3 repeats generation 1, so it is terminal but still shown, at
	// full brightness on the fade's first frame.
	lifeTick(t, cart, generationStepMicros, 1)
	assertLiveCells(t, "generation 3 (terminal, first fade frame)", livePixels(host.Blits[len(host.Blits)-1]), horizontal)
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 3" {
		t.Fatalf("LCD text on the terminal generation = %+v, want Line1 %q — the readout describes the world being faded out, not the one replacing it", last, "Gen: 3")
	}

	// 43 frames at 30 FPS is one past the 1.4s fade. The exhausted RNG script
	// clamps to "dead", so the reseeded soup is empty.
	lifeTick(t, cart, 33_333, 43)
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 1" {
		t.Fatalf("LCD text after the fade completed = %+v, want Line1 %q (reseeded)", last, "Gen: 1")
	}
	live := livePixels(host.Blits[len(host.Blits)-1])
	lcd := host.LCDTexts[len(host.LCDTexts)-1]
	if len(live) != 0 || lcd.Line2 != "Pop: 0" {
		t.Fatalf("reseeded world = %v live cells (LCD %+v), want an empty world (blinkerSoupScript's RNG script is exhausted, so the reseed's soup draw is clamped to a 'dead' value for every cell)", live, lcd)
	}
}

// TestLifeCartridgeTerminalWorldFadesOut walks a terminal blinker's dissolve
// at 30 FPS, asserting the same three cells dim, brightness falls
// monotonically, and the reseed lands only at the end.
func TestLifeCartridgeTerminalWorldFadesOut(t *testing.T) {
	horizontal := [][2]int{{31, 32}, {32, 32}, {33, 32}}

	host := carttest.NewFakeHost()
	host.RandScript = blinkerSoupScript()
	cart := newLifeCartridge(t, host)

	// Init seeds generation 1; two ticks reach generation 3, which repeats
	// generation 1's grid and is therefore terminal.
	lifeTick(t, cart, generationStepMicros, 2)

	const frameMicros = 33_333
	// CLASSIC paints live cells white, so this pixel tracks the veil.
	cellIdx := 32*carttest.DefaultWidth + 32

	prev := host.Blits[len(host.Blits)-1][cellIdx].R
	if prev != 255 {
		t.Fatalf("terminal generation's first frame paints the cell at %d (255,-,-) -> got R=%d, want a full-brightness 255 — the fade should start fully clear", cellIdx, prev)
	}

	faded := 0
	for i := 0; i < 60 && prev > 0; i++ {
		lifeTick(t, cart, frameMicros, 1)
		blit := host.Blits[len(host.Blits)-1]
		live := livePixels(blit)
		if len(live) == 0 {
			break // fully dissolved
		}
		assertLiveCells(t, "mid-fade", live, horizontal)
		if got := blit[cellIdx].R; got >= prev {
			t.Fatalf("frame %d of the fade: cell brightness %d, previous %d — want a strictly falling ramp", i+1, got, prev)
		} else {
			prev = got
		}
		faded++
	}

	// 1.4s at 30 FPS is ~42 frames, with slack for the frame boundary.
	if faded < 30 || faded > 50 {
		t.Errorf("the dissolve took %d frames at 30 FPS, want ~42 (terminalFadeMicros is 1.4s)", faded)
	}

	lifeTick(t, cart, frameMicros, 3)
	if last := host.LCDTexts[len(host.LCDTexts)-1]; last.Line1 != "Gen: 1" {
		t.Errorf("LCD text after the fade = %+v, want Line1 %q (a reseeded world)", last, "Gen: 1")
	}
}

func assertLiveCells(t *testing.T, label string, got, want [][2]int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: live cells = %v, want %v", label, got, want)
	}
	seen := make(map[[2]int]bool, len(want))
	for _, c := range want {
		seen[c] = true
	}
	for _, c := range got {
		if !seen[c] {
			t.Fatalf("%s: live cells = %v, want %v", label, got, want)
		}
	}
}

// TestLifeCartridgeDeterminism runs two fresh instances with the same RNG
// script at 30 FPS, past diehard's extinction and reseed, and asserts their
// blits and LCD texts are identical (docs/CARTRIDGE.md rule 2).
func TestLifeCartridgeDeterminism(t *testing.T) {
	const tickMicros = 33333 // 30 FPS
	const ticks = 500        // ~16.6s, past diehard's extinction and reseed

	run := func() *carttest.FakeHost {
		host := carttest.NewFakeHost()
		host.RandScript = diehardScript()
		cart := newLifeCartridge(t, host)
		lifeTick(t, cart, tickMicros, ticks)
		return host
	}
	a, b := run(), run()

	if len(a.Blits) != len(b.Blits) {
		t.Fatalf("runs produced %d vs %d frames", len(a.Blits), len(b.Blits))
	}
	for i := range a.Blits {
		if len(a.Blits[i]) != len(b.Blits[i]) {
			t.Fatalf("frame %d: %d vs %d pixels", i, len(a.Blits[i]), len(b.Blits[i]))
		}
		for j := range a.Blits[i] {
			if a.Blits[i][j] != b.Blits[i][j] {
				t.Fatalf("frame %d differs at pixel %d: %+v vs %+v", i, j, a.Blits[i][j], b.Blits[i][j])
			}
		}
	}
	if len(a.LCDTexts) != len(b.LCDTexts) {
		t.Fatalf("runs produced %d vs %d host_set_text calls", len(a.LCDTexts), len(b.LCDTexts))
	}
	for i := range a.LCDTexts {
		if a.LCDTexts[i] != b.LCDTexts[i] {
			t.Fatalf("host_set_text[%d] differs: %+v vs %+v", i, a.LCDTexts[i], b.LCDTexts[i])
		}
	}

	finalLive := livePixels(a.Blits[len(a.Blits)-1])
	if len(finalLive) == 0 {
		t.Log("note: field is empty at the final tick (a live-but-transient reseed state); determinism still holds by the byte-identical check above")
	}
}

// TestLifeAttractSilence counts host_play_sound calls across cart_init and a
// long attract run and expects zero. The bloom flag stays unconsumed because
// attract never reaches LoadFrame.
func TestLifeAttractSilence(t *testing.T) {
	host := carttest.NewFakeHost()
	cart := newLifeCartridgeAtSplash(t, host)
	if n := host.PlayCount(); n != 0 {
		t.Fatalf("host_play_sound calls after cart_init = %d, want 0", n)
	}
	if !cart.SupportsAttract() {
		t.Fatal("life.wasm does not export cart_attract")
	}

	ctx := context.Background()
	for i := 0; i < 300; i++ { // 300 * 100ms = 30s: many generations, several reseeds
		if err := cart.Attract(ctx, 100_000); err != nil {
			t.Fatalf("Attract tick %d: %v", i, err)
		}
	}
	if n := host.PlayCount(); n != 0 {
		t.Fatalf("host_play_sound calls after cart_init + a 30s attract run = %d, want 0 (attract must stay silent)", n)
	}
}

// TestLifeWasmMatchesSource catches a stale life.wasm, which the goldens
// cannot see because they were recorded from it.
func TestLifeWasmMatchesSource(t *testing.T) {
	carttest.RequireWasmMatchesSource(t, ".", "life.wasm", "make -C ../.. cart-life")
}

func TestLifeManifestCopiesAgree(t *testing.T) {
	carttest.ManifestCopiesAgree(t, newLifeCartridgeAtSplash(t, carttest.NewFakeHost()), "manifest.json", "make -C ../.. cart-life")
}

func TestLifeAllocatesNothingAfterInit(t *testing.T) {
	carttest.RequireNoAllocsAfterInit(t, ".")
}
