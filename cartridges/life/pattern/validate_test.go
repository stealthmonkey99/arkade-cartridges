// SPDX-License-Identifier: GPL-2.0-or-later

package pattern

import (
	"strings"
	"testing"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/levels"
)

const noWeight = -1

func level(w, h int, cells string, weight int) *levels.Level {
	lv := &levels.Level{Name: "test", W: w, H: h, Cells: cells}
	if weight != noWeight {
		lv.Meta = []levels.Entry{{Key: "weight", Kind: levels.KindInt, Int: weight}}
	}
	return lv
}

func TestValidate(t *testing.T) {
	big := cartlib.FBWidth + 1
	cases := []struct {
		name    string
		lv      *levels.Level
		wantErr string
	}{
		{"a glider", level(3, 3, ".#...####", 3), ""},
		{"wider than the field", level(big, 1, strings.Repeat("#", big), 1), "bigger than"},
		{"all dead", level(2, 1, "..", 1), "no live cells"},
		{"no weight", level(1, 1, "#", noWeight), "no integer `weight`"},
		{"zero weight", level(1, 1, "#", 0), "weight 0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			live, weight, err := Validate(c.lv)
			if c.wantErr == "" {
				if err != nil || live != 5 || weight != 3 {
					t.Fatalf("Validate = %d, %d, %v; want 5 live, weight 3", live, weight, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("Validate error = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}
