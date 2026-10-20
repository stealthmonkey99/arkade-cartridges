# Game manuals

Every cartridge ships a player-facing manual: `cartridges/<slug>/MANUAL.md`.
Manuals are for **players**, not engineers — someone standing at the
cabinet (or dropping an `.rcart` on the web player) who may never have seen
the game before. Writing or updating one? The format and the upkeep rules
are in [MANUAL_FORMAT.md](MANUAL_FORMAT.md).

## The library

| | Game | Players | One-liner |
|---|---|---|---|
| <img src="../cartridges/life/manual-icon.png" width="32" height="32"> | [Life](../cartridges/life/MANUAL.md) | 1 | Conway's cellular zoo runs itself. Sit back and watch the gliders. |
| <img src="../cartridges/snake/manual-icon.png" width="32" height="32"> | [Snake](../cartridges/snake/MANUAL.md) | 1–2 | Eat apples, grow long, and never meet your own tail. |

## Shared controls background

Manuals name cabinet buttons by **color** — that's how players describe
them at the panel:

| Button | Color | Web player key | Dev console key |
|---|---|---|---|
| A | 🟢 green | `Z` (also `Enter`/`Space`) | `i` |
| B | 🔴 red | `X` (also `Esc`) | `o` |
| C | 🟡 yellow | `C` | `k` |
| D | 🔵 blue | `V` | `l` |
| Aux1/Aux2 | ⚪ white | — | — |

The white Aux buttons have **no keyboard equivalent** — anything bound to
them is reachable only at the real cabinet. Manuals write them as
`⚪ Aux1` / `⚪ Aux2`.

The joystick is 8-way; arrows (or WASD on the web player) stand in for it
on a keyboard. **B always backs out** — hold of that fact is engine-wide:
in a game it opens the quit/confirm path, on a menu it leaves. Each manual
only documents what the game DOES with the controls, and links here
instead of re-explaining key equivalents.

**Entering your initials** works the same way in every game: the stick
picks letters, 🟢 A confirms one and moves along, 🟡 C rubs the last one
out, and 🔴 B leaves — because B always backs out, including here.

Player colors follow the sticks: **P1 is red, P2 is blue**.
