// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/carttest"
	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
)

// newSnakeHost is the wasm golden host with the bundle's assets mounted, as
// the cabinet mounts the .rcart.
func newSnakeHost(t *testing.T) *carttest.FakeHost {
	t.Helper()
	h := carttest.NewFakeHost()
	if err := h.MountAssets("."); err != nil {
		t.Fatal(err)
	}
	return h
}

// newHost is hosttest.New with the bundle's assets mounted.
func newHost(t *testing.T) *hosttest.Host {
	t.Helper()
	h := hosttest.New(t)
	if err := h.MountAssets("."); err != nil {
		t.Fatal(err)
	}
	return h
}

// TestEverySpriteTheGameDrawsIsOnItsSheet: a misnamed cel would load as
// blank and draw nothing, silently.
func TestEverySpriteTheGameDrawsIsOnItsSheet(t *testing.T) {
	newHost(t).Init(game)
	if titleArt == &blank {
		t.Errorf("%s has no cel %q", pathTitleSheet, "title")
	}
	if slashGlyph == &blank {
		t.Errorf("%s has no cel %q", pathPickupSheet, "slash")
	}
	for i, name := range pickupCelNames {
		if pickupCels[i] == &blank {
			t.Errorf("%s has no cel %q", pathPickupSheet, name)
		}
	}
}

// TestPickupsDifferInSilhouette: a colour-blind player tells the kinds apart
// by shape, so no two pickup cels may share an opaque footprint.
func TestPickupsDifferInSilhouette(t *testing.T) {
	newHost(t).Init(game)
	seen := map[string]string{}
	for i, name := range pickupCelNames {
		cel := pickupCels[i]
		if cel == &blank {
			t.Fatalf("no %q cel", name)
		}
		var shape []byte
		for j := 3; j < len(cel.Pix); j += 4 {
			shape = append(shape, cel.Pix[j]/255)
		}
		if other, dup := seen[string(shape)]; dup {
			t.Errorf("%s and %s have the same silhouette", other, name)
		}
		seen[string(shape)] = name
	}
}
