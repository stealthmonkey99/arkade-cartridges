// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

const (
	pathEat           = "assets/omnomnom.arkadesound.wav"
	pathLost          = "assets/lost.arkadesound.wav"
	pathGameOver      = "assets/gameover.arkadesound.wav"
	pathLevelUp       = "assets/levelup.arkadesound.wav"
	pathPickupSpawn   = "assets/pickup_spawn.arkadesound.wav"
	pathPickupCollect = "assets/pickup_collect.arkadesound.wav"
)

var (
	soundEat           uint32
	soundLost          uint32
	soundGameOver      uint32
	soundLevelUp       uint32
	soundPickupSpawn   uint32
	soundPickupCollect uint32
)

// loadSounds runs from Init. Without bundled assets each handle is 0, which
// PlaySound ignores.
func loadSounds(s *cartlib.Setup) {
	soundEat = s.LoadSound(pathEat)
	soundLost = s.LoadSound(pathLost)
	soundGameOver = s.LoadSound(pathGameOver)
	soundLevelUp = s.LoadSound(pathLevelUp)
	soundPickupSpawn = s.LoadSound(pathPickupSpawn)
	soundPickupCollect = s.LoadSound(pathPickupCollect)
}
