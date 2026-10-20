// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

const pathSoundBloom = "assets/bloom.arkadesound.wav"

var soundBloom uint32

// pendingLoaderBloom is set in Init and consumed by the first LoadFrame,
// since the load phase can collapse into a single call under test.
var pendingLoaderBloom bool

// loadSounds registers the bloom chime from Init. Without bundled assets it
// gets handle 0, which PlaySound ignores.
func loadSounds(s *cartlib.Setup) {
	soundBloom = s.LoadSound(pathSoundBloom)
}
