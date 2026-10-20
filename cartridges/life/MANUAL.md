<img src="manual-icon.png" width="128" height="128" align="left" alt="Life icon">

# Life

**1 player · ambient** · Conway's cellular zoo runs itself. Sit back and
watch the gliders.

<br clear="all">

## How to play

Life is Conway's Game of Life, running on its own across the full screen.
The cartridge boots straight into a running world: nothing to press,
nothing to wait through. Each fresh world is a weighted draw from classic
patterns: the acorn and r-pentomino (slow-burning "methuselahs"), a Gosper
glider gun firing from a corner, fleets of gliders or lightweight
spaceships in random orientations, a diehard, a pentadecathlon
"heartbeat" beside an r-pentomino, or a random soup in the middle.

Every world eventually ends. It settles into a still life, locks into an
oscillator, or dies out, and then a new one is drawn.

Press 🟢 A any time to open the settings menu: how the field is drawn
(STYLE), how fast it runs (SPEED, a slider), hand-drawing your own seed
(DRAW), or a fresh random world (RESEED). Closing the menu resumes exactly
where it left off.

The LCD shows the generation count and the number of live cells, labeled
Gen and Pop. At the faster speeds they refresh about four times a second
rather than every generation; a world that dies out is reported the
instant it happens. While a menu or the DRAW editor is open, the LCD names
that screen instead.

## Controls

| Input | Does |
|---|---|
| 🕹️ Stick | In the menu: up/down moves the highlighted row, left/right changes STYLE or SPEED. In DRAW: moves the cursor |
| 🟢 A | Opens the menu; in it, steps the highlighted setting or runs the highlighted action; in DRAW, toggles the cell under the cursor |
| 🔴 B | Quits at once from the running field; in a menu or DRAW, backs out one level instead |
| 🟡 C | DRAW only: runs the drawing as the field's new world |

[Keyboard equivalents](../../docs/MANUALS.md#shared-controls-background)

## Rules

- A live cell with 2 or 3 live neighbors (of its 8) survives, otherwise it
  dies; a dead cell with exactly 3 live neighbors is born.
- Edges are hard walls, not wraparound: border cells have fewer
  neighbors, so patterns that drift off the side fizzle out.
- A world is finished when a generation repeats one from the last 40, or
  after 1200 generations. The finished pattern fades out over about a
  second and a half before the next world, unless it died out completely,
  which is replaced right away.
- **STYLE** loops through three looks. CLASSIC draws live cells white on
  black. GLOW leaves a fading green trail behind recently live cells, as
  the cabinet's screensaver does. HEAT flashes a newborn cell bright green
  and eases it to white, and flashes a dying cell bright red before it
  dims out through darkening red.
- **SPEED** is a 9-stop slider that stops at both ends: 800, 500, 300,
  200 and 133 ms per generation, **100 ms (NORMAL, center)**, then 66, 40
  and 20 ms.
- **DRAW** freezes the field and opens a blank or previously saved canvas
  with a blinking cursor that wraps at every edge. Every drawing you run
  is saved; the last 10 are offered back as canvases, newest first.
- STYLE, SPEED and the saved drawings are remembered across reboots.

## Tips

- Leave an acorn or r-pentomino running: they look modest but often take
  hundreds of generations to burn out.
- HEAT makes a glider gun mesmerizing: each new cell flashes green and
  every cell left behind flashes red as it dies.
- Slide SPEED all the way down to follow a pattern generation by
  generation, or all the way up to watch a methuselah race to its end.
- A hand-drawn glider gun or a dense scribble in DRAW usually outlives the
  random soup; save a few favorites.

---
*Manual for version 1.3.0 · [game library](../../docs/MANUALS.md)*
