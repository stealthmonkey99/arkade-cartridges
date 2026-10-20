// SPDX-License-Identifier: GPL-2.0-or-later

package main

import "github.com/stealthmonkey99/arkade/sdk/cartlib"

type cell struct{ x, y uint8 }

const ringSize = 4096 // > 62*62 interior cells

type snake struct {
	dir, queuedDir                     cartlib.Direction
	body                               [ringSize]cell
	headIdx, tailIdx, snakeLen, growth int
	score                              uint32
	lives                              int
	inert                              bool // not in this round: P2's slot during 1P
	dead                               bool // out of lives and off the board
	diedThisTick                       bool // collided; resolveLostLife settles it after the freeze
}

func (s *snake) head() cell { return s.body[s.headIdx] }

func (s *snake) onBoard() bool { return !s.inert && !s.dead }

// enter resets s field by field: a struct literal would heap-allocate the
// 8 KB body.
func (s *snake) enter() {
	s.dir, s.queuedDir = cartlib.DirNone, cartlib.DirNone
	s.headIdx, s.tailIdx, s.snakeLen, s.growth = 0, 0, 0, 0
	s.score, s.lives = 0, startLives
	s.inert, s.dead, s.diedThisTick = false, false, false
}

func (s *snake) sitOut() {
	s.enter()
	s.lives = 0
	s.inert = true
}

func (s *snake) unmark(b *board) {
	for n, idx := 0, s.tailIdx; n < s.snakeLen; n, idx = n+1, (idx+1)%ringSize {
		c := s.body[idx]
		b.occupied[c.y][c.x] = false
	}
}

func (s *snake) dropTail() {
	t := s.body[s.tailIdx]
	field.occupied[t.y][t.x] = false
	s.tailIdx = (s.tailIdx + 1) % ringSize
	s.snakeLen--
}

// snakes[1] is P2, inert in 1P.
var snakes [2]snake

// beginSnake makes s a single head cell that regrows to length.
func beginSnake(s *snake, head cell, d cartlib.Direction, length int) {
	s.unmark(&field)
	s.body[0] = head
	s.tailIdx, s.headIdx, s.snakeLen = 0, 0, 1
	field.occupied[head.y][head.x] = true
	s.dir = d
	s.queuedDir = cartlib.DirNone
	s.growth = length
}

func removeSnakeFromBoard(s *snake) {
	s.unmark(&field)
	s.snakeLen = 0
	s.dead = true
}

// respawnSnake drops s on a random safe start; length 0 means initLength.
func respawnSnake(s *snake, length int) {
	c := field.pickSafeStart()
	if length == 0 {
		length = initLength
	}
	beginSnake(s, c, spawnDirection(int(c.x), int(c.y)), length)
}

func perpendicular(a, b cartlib.Direction) bool { return a.Vertical() != b.Vertical() }

// stepGameAll moves every snake one cell on the same tick. Collisions read
// the grid as it stood before the tick, so a tail about to move still kills,
// and a head-to-head kills both. Any death freezes the whole arena.
func stepGameAll() {
	tryAdvanceLevel()

	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if !s.onBoard() {
			continue
		}
		if s.queuedDir != cartlib.DirNone {
			if perpendicular(s.queuedDir, s.dir) {
				s.dir = s.queuedDir
			}
			s.queuedDir = cartlib.DirNone
		}
	}

	var nx, ny [2]int
	var dies [2]bool
	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if !s.onBoard() {
			continue
		}
		h := s.head()
		x, y := int(h.x), int(h.y)
		switch s.dir {
		case cartlib.DirUp:
			y--
		case cartlib.DirDown:
			y++
		case cartlib.DirLeft:
			x--
		case cartlib.DirRight:
			x++
		}
		nx[i], ny[i] = x, y
		hitsBody := field.occupied[y][x] && !ghosting()
		if wallAt(field.level, x, y) || hitsBody {
			dies[i] = true
		}
	}

	if numPlayers == 2 && snakes[0].onBoard() && snakes[1].onBoard() {
		if nx[0] == nx[1] && ny[0] == ny[1] {
			dies[0] = true
			dies[1] = true
		}
	}

	ateFood := -1
	for i := 0; i < numPlayers; i++ {
		if !snakes[i].onBoard() || dies[i] {
			continue
		}
		if uint8(nx[i]) == field.food.x && uint8(ny[i]) == field.food.y {
			ateFood = i
			break
		}
	}

	anyDied := false
	for i := 0; i < numPlayers; i++ {
		s := &snakes[i]
		if !s.onBoard() {
			continue
		}
		if dies[i] {
			killSnakeThisTick(i)
			anyDied = true
			continue
		}
		applyMove(i, nx[i], ny[i], i == ateFood)
	}

	fx := soloEffects()
	if ateFood >= 0 {
		field.placeFood(field.level)
		if fx != nil {
			fx.onAppleEaten() // after placeFood, so a new pickup steers clear of the new apple
		}
	}

	// After applyMove, so a shrink berry cuts the already-moved body.
	if s := &snakes[0]; fx != nil && s.onBoard() && !dies[0] && fx.collidesAt(nx[0], ny[0]) {
		fx.collect(s)
	}

	if anyDied {
		machine.Go(stateLostLife)
		return
	}

	if !field.swap.pending {
		threshold := nextLevelLength
		if numPlayers == 2 {
			threshold = nextLevelLength2P
		}
		for i := 0; i < numPlayers; i++ {
			if snakes[i].onBoard() && snakes[i].snakeLen > threshold {
				if numPlayers == 2 {
					triggerLevelUp2P()
				} else {
					triggerLevelUp()
				}
				break
			}
		}
	}
}

func applyMove(i, x, y int, ate bool) {
	s := &snakes[i]
	s.headIdx = (s.headIdx + 1) % ringSize
	s.body[s.headIdx] = cell{x: uint8(x), y: uint8(y)}
	field.occupied[y][x] = true
	s.snakeLen++

	if ate {
		s.growth += growPerEat
		s.score += applePoints()
		cartlib.PlaySound(soundEat, 0)
		updatePlayLcd(i)
	}

	if s.growth > 0 {
		s.growth--
	} else {
		s.dropTail()
	}
}

// killSnakeThisTick shows the lost heart now; resolveLostLife acts on it
// after the freeze.
func killSnakeThisTick(i int) {
	cartlib.PlaySound(soundLost, 0)
	snakes[i].lives--
	snakes[i].diedThisTick = true
	updatePlayLcd(i)
}
