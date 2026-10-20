// SPDX-License-Identifier: GPL-2.0-or-later

// Package pattern is how Life reads a seed pattern out of its pack. The
// cartridge stamps by these rules and the pack test refuses what breaks them.
package pattern

// Live is the rune of a live cell; every other rune is dead.
const Live = '#'
