<img src="manual-icon.png" width="128" height="128" align="left" alt="Snake icon">

# Snake

**1–2 players · versus** · Eat apples, grow long, and never meet your own
tail.

<br clear="all">

## How to play

You steer a snake around a walled arena, eating apples to grow longer. In
**1 player** you're the lone green snake: hit a wall or your own tail and
you lose one of your **3 lives**. Every so often a **pickup** blinks onto
the field for a temporary boost.

In **2 players** it's one shared arena and one shared apple: **P1 is red,
P2 is light blue**, and the apple turns green to stand out against both.
A wall, either snake's body, or a head-on crash (which costs you both)
loses a life, and whichever snake still has lives when the other runs out
wins at once.

Grow long enough and the arena swaps to its next wall layout, cycling
through 6. Your lives, level, score and high score are on the LCD in
front of your seat, not on screen.

## Controls

| Input | Does |
|---|---|
| 🕹️ Stick | Turn (can't reverse straight into yourself) |
| 🟢 A | Confirm — picks the title menu item, locks in a name-entry letter, continues past game over |
| 🔴 B | Quit — press once for the confirm screen, then press-and-**hold B** ~1 second; it blinks and exits when you let go. Any other button resumes. Works the same during name entry, so a slip can't cost you a placement |
| 🟡 C | Name entry only: erases the letter you just entered |

Second player: pick **2 PLAYERS** on the title menu; P2 plays from the
other seat.
[Keyboard equivalents](../../docs/MANUALS.md#shared-controls-background)

## Rules

- You start at **3 lives** and **4** cells long; each apple eaten adds
  **+3** length.
- 1 player: the level advances once your snake passes **100** cells long.
- 2 players: the level advances once either snake passes **30** cells
  long — lower than solo, since sharing an arena tends to end runs
  sooner.
- A lost life freezes the arena briefly, then respawns you at your same
  length from a random spot clear of the current walls.
- **Pickups (1 player only)**: one shows up roughly every 8-15 apples and
  sits for **8 seconds**, blinking over its last 2.5s. Collecting one
  shows its label on screen and replaces any running effect; the LCD
  counts the effect down, and the arena border takes its colour, blinking
  over the last ~3s.
  - 🔴 **Chili** ("2X SPEED"): double speed for 20s, **+25** points.
  - 🔵 **Ice cube** ("1/2 SPEED"): half speed for 20s, **+10** points.
  - 🟣 **Shrink berry** ("CUT SHORT"): cuts about a third of your length
    at once, **+10** points.
  - ⚪ **Ghost berry** ("INVINCIBLE"): pass through your own tail for 20s,
    **+10** points. Walls still kill.
  - 🟡 **Gold apple** ("3X POINTS"): triple points per apple for 20s.

## Scoring

- **+3** points per apple eaten.
- 1 player: every finished run is offered to the shared top-10 table.
  2 players: only the winner's score is.
- A score good enough to place gets name entry right after game over.
  Quitting from it with the B hold abandons the placement unsubmitted.

## Tips

- Turning is perpendicular only — you can't reverse into yourself, so
  line up a turn a beat early instead of at the last cell.
- A lost life keeps your length, only your position resets — a late hit
  in a long run costs far less than an early one.
- In 2P, don't sprint straight at the apple when your rival's closing
  from the far side — that's exactly how head-to-heads happen. Angle in
  from the side instead.
- A pickup you don't want (say, ice cube) lasts only until the next one
  you collect; effects never stack.

---
*Manual for version 1.5.1 · [game library](../../docs/MANUALS.md)*
