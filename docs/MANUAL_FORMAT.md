# Writing a game manual

Every cartridge's `MANUAL.md` follows one format and is kept current with
the game; this page is both rules. The player-facing index is
[MANUALS.md](MANUALS.md).

## The format

Every `MANUAL.md` follows this exact skeleton, top to bottom. Sections
marked *(optional)* may be dropped when they'd be empty; nothing may be
reordered or renamed, and no extra top-level sections may be added.

```markdown
<img src="manual-icon.png" width="128" height="128" align="left" alt="<Name> icon">

# <Name>

**<players line — e.g. "1–2 players · versus">** · <one-line pitch,
ideally the manifest description>

<br clear="all">

## How to play

<2–4 short paragraphs: the objective, the core loop, and how a
game/match ends. Written for someone who has never seen the game.>

## Controls

| Input | Does |
|---|---|
| 🕹️ Stick | <what it does> |
| 🟢 A | <what it does> |
| 🔴 B | Quit / back |
| ... | <only rows that DO something in this game> |

<For 2P games: one line on how the second player joins/starts.>
[Keyboard equivalents](../../docs/MANUALS.md#shared-controls-background)

## Rules

<Bulleted, concrete. Timers, lives, win conditions, level advancement,
power-ups — with real numbers where a player would notice them
(fuse seconds, lives count), no source constants or code names.>

## Scoring *(optional — drop for games with no score table)*

<What earns points (real numbers), what gets submitted to the top-10
table, and when name entry appears.>

## Tips *(optional)*

<3–5 genuinely useful tactics, not padding.>

---
*Manual for version <manifest version> · [game library](../../docs/MANUALS.md)*
```

Format rules the skeleton doesn't show:

- **Header art**: always `manual-icon.png` (the 8×8 icon baked to 128×128
  by `make manual-icons`; a raw `<img width="128">` of the 8×8 file
  blurs). Title and pitch sit to its right via `align="left"`; the
  `<br clear="all">` closes the float.
- **The LCD is part of the UI** — if the game shows score/stats/prompts
  on the per-seat 16×2 LCD, say so in How to play or Rules; players
  otherwise never think to look down.
- **Numbers over adjectives**: "3 lives", "90-second rounds", "+100 per
  knockout" — never "several", "a while", "big bonus".
- **No spoilers rule of thumb**: mechanics yes, secrets sparingly —
  e.g. a manual may say a boss exists, not chart where it spawns.
- Keep it to roughly one screen of scrolling (≤ ~80 lines). A manual
  nobody reads to the end documents nothing.

## Keeping manuals current

The manual lives next to the game code so gameplay changes and manual
updates land in the same commit. Concretely:

1. **Any change a player can feel updates MANUAL.md in the same commit**:
   controls, rules, timers, scoring, modes, power-ups, win conditions.
   Renders, refactors and perf work don't.
2. **Gameplay changes bump the manifest version** (patch for tuning,
   minor for new rules/modes), and the manual's footer must quote the
   same version string. `TestManualsCurrent`
   (`tools/fleetlint/manuals_test.go`, run by `make test` and CI) enforces
   footer == manifest, so a version bump without a manual pass fails.
   Fix it by rereading the manual against the change, not by editing the
   footer alone.
3. **New cartridge = manual before ship**. It's on the authoring
   checklist in the engine repo's
   [docs/GAME_AUTHORING_AI.md](https://github.com/stealthmonkey99/arkade/blob/main/docs/GAME_AUTHORING_AI.md);
   the same test fails on any cartridge directory that has a
   manifest.json but no MANUAL.md.
4. **UX fixes flow back**: when a change touches a button binding or a
   cursor behavior, re-check the Controls table and Tips — they're the
   sections that rot fastest.
5. **Regenerate `manual-icon.png`** (`make manual-icons`) whenever a
   cartridge's `assets/icon.arkadesprite.png` changes; the test fails
   when the two disagree.

## Adding a game to the index

One table row in [MANUALS.md](MANUALS.md)'s library (alphabetical), one
manual. The test checks every cartridge's manual is linked from there, so
forgetting the row is a failure, not a silent gap.
