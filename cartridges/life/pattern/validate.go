// SPDX-License-Identifier: GPL-2.0-or-later

//go:build !tinygo

package pattern

import (
	"errors"
	"fmt"

	"github.com/stealthmonkey99/arkade/sdk/cartlib"
	"github.com/stealthmonkey99/arkade/sdk/levels"
)

// Validate rejects a pattern the field cannot hold (the centred and fleet
// stamps would compute negative ranges), one with no live cells, or one whose
// weight is missing or below 1, which would make its recipe undrawable. It
// returns the live-cell count and weight of a pattern that passes.
func Validate(lv *levels.Level) (live, weight int, err error) {
	if lv.W > cartlib.FBWidth || lv.H > cartlib.FBHeight {
		return 0, 0, fmt.Errorf("is %dx%d, bigger than the %dx%d field it is stamped on", lv.W, lv.H, cartlib.FBWidth, cartlib.FBHeight)
	}
	live = lv.Count(Live)
	if live == 0 {
		return 0, 0, errors.New("has no live cells — a reseed that draws a blank field")
	}
	weight, ok := lv.Int("weight")
	if !ok {
		return 0, 0, errors.New("has no integer `weight` — its recipe could never be drawn")
	}
	if weight < 1 {
		return 0, 0, fmt.Errorf("has weight %d; a recipe needs at least 1 to be reachable", weight)
	}
	return live, weight, nil
}
