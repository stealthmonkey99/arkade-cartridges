// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib/hosttest"
)

func TestSettingsSurviveARestart(t *testing.T) {
	hosttest.New(t)
	saved := lifeGame{style: styleHeat, speed: 2}
	saved.saveSettings()

	var restarted lifeGame
	restarted.loadSettings()
	if restarted.style != styleHeat || restarted.speed != 2 {
		t.Errorf("after a restart: style %d, detent %d, want HEAT (%d) and 2", restarted.style, restarted.speed, styleHeat)
	}
}

func TestDecodeSettingsDefaultsOnAbsentOrWrongSize(t *testing.T) {
	var buf [settingsEncodedLen]byte
	// Poison the buffer so a length-only check cannot pass by accident.
	buf[0], buf[1], buf[2] = 200, 200, settingsFormatV2Marker

	for _, n := range []int{0, 1, 4, 100} {
		style, speed := decodeSettings(buf[:], n)
		if style != styleClassic || speed != speedDetentDefault {
			t.Errorf("decodeSettings(buf, %d) = (%d, %d), want (styleClassic, speedDetentDefault)", n, style, speed)
		}
	}
}

func TestDecodeSettingsRejectsOutOfRangeValues(t *testing.T) {
	var buf [settingsEncodedLen]byte
	buf[0], buf[1], buf[2] = styleCount, speedDetentCount, settingsFormatV2Marker // one past the last valid ordinal each
	style, speed := decodeSettings(buf[:], settingsEncodedLen)
	if style != styleClassic || speed != speedDetentDefault {
		t.Errorf("decodeSettings(out-of-range) = (%d, %d), want the CLASSIC/NORMAL defaults", style, speed)
	}
}

func TestDecodeSettingsAcceptsGenuineV2Value(t *testing.T) {
	var buf [settingsEncodedLen]byte
	buf[0], buf[1], buf[2] = styleHeat, 8, settingsFormatV2Marker // detent 8 == 20_000 (fastest)
	style, speed := decodeSettings(buf[:], settingsEncodedLen)
	if style != styleHeat || speed != 8 {
		t.Errorf("decodeSettings(HEAT, detent 8) = (%d, %d), want (%d, %d)", style, speed, styleHeat, 8)
	}
}

func TestDecodeSettingsRejectsV2LengthWithBadMarker(t *testing.T) {
	var buf [settingsEncodedLen]byte
	buf[0], buf[1], buf[2] = styleHeat, 8, 0x00 // right length, wrong marker
	style, speed := decodeSettings(buf[:], settingsEncodedLen)
	if style != styleClassic || speed != speedDetentDefault {
		t.Errorf("decodeSettings(V2-length, bad marker) = (%d, %d), want the CLASSIC/NORMAL defaults", style, speed)
	}
}

// TestDecodeSettingsMigratesV1SpeedToNearestDetent checks the 2-byte format's
// ordinals decode through oldSpeedToDetent.
func TestDecodeSettingsMigratesV1SpeedToNearestDetent(t *testing.T) {
	cases := []struct {
		name       string
		oldSpeed   uint8
		wantDetent uint8
		wantMicros uint64
	}{
		{"old SLOW (200k) -> exact detent", 0, 3, 200_000},
		{"old NORMAL (100k) -> exact detent", 1, 5, 100_000},
		{"old FAST (50k) -> nearest detent (40k, not 66k)", 2, 7, 40_000},
		{"old TURBO (20k) -> exact detent", 3, 8, 20_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf [settingsEncodedLenV1]byte
			buf[0], buf[1] = styleGlow, c.oldSpeed
			style, speed := decodeSettings(buf[:], settingsEncodedLenV1)
			if style != styleGlow {
				t.Errorf("style = %d, want styleGlow (unaffected by the speed migration)", style)
			}
			if speed != c.wantDetent {
				t.Fatalf("decodeSettings(V1 old speed %d) = detent %d, want %d", c.oldSpeed, speed, c.wantDetent)
			}
			if got := generationStepFor(speed); got != c.wantMicros {
				t.Errorf("generationStepFor(migrated detent %d) = %d, want %d", speed, got, c.wantMicros)
			}
		})
	}
}

func TestDecodeSettingsRejectsOutOfRangeV1Speed(t *testing.T) {
	var buf [settingsEncodedLenV1]byte
	buf[0], buf[1] = styleClassic, 200
	style, speed := decodeSettings(buf[:], settingsEncodedLenV1)
	if style != styleClassic || speed != speedDetentDefault {
		t.Errorf("decodeSettings(V1, out-of-range old speed) = (%d, %d), want the CLASSIC/NORMAL defaults", style, speed)
	}
}
