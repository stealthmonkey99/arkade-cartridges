// SPDX-License-Identifier: GPL-2.0-or-later

package main

// StorageWrite refuses an illegal key with false, and the cartridge does not
// check the result, so the data would vanish. FakeHost does no key
// validation, so goldens cannot catch it; this asserts against
// carttest.StorageKeyOK.

import (
	"testing"

	"github.com/stealthmonkey99/arkade/carttest"
)

// TestLifeStorageKeysAreHostLegal covers both storage keys, the only ones the
// cartridge uses.
func TestLifeStorageKeysAreHostLegal(t *testing.T) {
	keys := map[string]string{
		"settingsStorageKey": settingsStorageKey,
		"drawingsStorageKey": drawingsStorageKey,
	}
	for name, key := range keys {
		if !carttest.StorageKeyOK(key) {
			t.Errorf("%s = %q would be REJECTED by the real cabinet host — that data would never persist, silently", name, key)
		}
	}
	// A shared key would let the two blobs overwrite each other.
	if settingsStorageKey == drawingsStorageKey {
		t.Fatalf("settingsStorageKey and drawingsStorageKey are both %q — they would clobber one another", settingsStorageKey)
	}
}
