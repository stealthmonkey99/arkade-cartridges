// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"testing"

	"github.com/stealthmonkey99/arkade/carttest"
)

func TestNamesNoButtonByLetter(t *testing.T) { carttest.NoLetteredButtonHints(t) }
