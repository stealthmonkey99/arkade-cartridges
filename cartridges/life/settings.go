// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

const settingsStorageKey = "settings"

// The current format is 3 bytes: style ordinal, speed detent index and a
// marker. The earlier 2-byte format's speed byte was a 0-3 ordinal that
// overlaps detent values, so length plus marker tells them apart. The marker
// is nonzero so a zeroed buffer never reads as valid. Any STYLE or SPEED
// change rewrites a 2-byte blob, so its decode path can go once no cabinet
// still holds one.
const (
	settingsEncodedLenV1   = 2
	settingsEncodedLen     = 3
	settingsFormatV2Marker = 0xA5
)

// decodeSettings decodes the n bytes StorageRead put in buf. StorageRead
// reports only a length, so a missing key, a short read and a stale blob look
// alike; anything but a marked 3-byte blob or a 2-byte blob falls back to
// CLASSIC and NORMAL.
func decodeSettings(buf []byte, n int) (style, speed uint8) {
	style, speed = styleClassic, speedDetentDefault

	switch {
	case n == settingsEncodedLen && buf[2] == settingsFormatV2Marker:
		if s := buf[0]; s < styleCount {
			style = s
		}
		if sp := buf[1]; sp < speedDetentCount {
			speed = sp
		}
	case n == settingsEncodedLenV1:
		if s := buf[0]; s < styleCount {
			style = s
		}
		if old := buf[1]; int(old) < len(oldSpeedToDetent) {
			speed = oldSpeedToDetent[old]
		}
	}
	return
}

// loadSettings restores style and speed from storage; Init only.
func (g *lifeGame) loadSettings() {
	n := cartlib.StorageRead(settingsStorageKey, settingsBuf[:])
	g.style, g.speed = decodeSettings(settingsBuf[:], n)
}

// saveSettings persists on every change, since a cartridge has no shutdown
// hook.
func (g *lifeGame) saveSettings() {
	settingsBuf = [settingsEncodedLen]byte{g.style, g.speed, settingsFormatV2Marker}
	cartlib.StorageWrite(settingsStorageKey, settingsBuf[:])
}

// settingsBuf is package-level because a local array handed to the host
// escapes to the heap.
var settingsBuf [settingsEncodedLen]byte
