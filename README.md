# Årkade Cartridges

The game cartridges for [Årkade](https://github.com/stealthmonkey99/arkade),
an engine for a 64×64 LED arcade cabinet. A cartridge is a TinyGo to
WebAssembly module that runs unmodified on the cabinet and in the browser.

## The cartridges

| Cartridge | |
|---|---|
| [Life](cartridges/life) | Conway's cellular zoo runs itself. Sit back and watch the gliders. |
| [Snake](cartridges/snake) | Eat apples, grow long, and never meet your own tail. |

Each has a player manual (`MANUAL.md` in its folder); they are collected in
[`docs/MANUALS.md`](docs/MANUALS.md).

## Play

- **In the browser** at [arkade.build](https://arkade.build), or drag a
  built cartridge (`cartridges/life/life.wasm`) onto that page.
- **On your own machine**, clone the engine beside this repo and point it
  at the cartridges:

  ```sh
  git clone https://github.com/stealthmonkey99/arkade
  git clone https://github.com/stealthmonkey99/arkade-cartridges
  cd arkade && make build && ./arkade --cartridges '../arkade-cartridges/cartridges/*'
  ```

- **On a cabinet**: from that engine checkout, pack a cartridge's `.rcart`
  bundle (`make rcart DIR=../arkade-cartridges/cartridges/life`) and copy
  it into the cabinet's `data/cartridges/` directory. The engine repo
  covers building the cabinet.

The sibling checkout is only needed to run the engine or pack an `.rcart`;
`make test` here fetches the engine's tagged modules on its own.

## Build and test

`make test` needs Go 1.23.4 or newer, the version the modules declare.
Rebuilding a `.wasm` needs TinyGo 0.42.0 and a Go it supports; CI uses
Go 1.27. The built `.wasm` is committed, so the tests never need TinyGo:
they drive it through `carttest`, the engine's fake host, and compare
frames against golden files.

| Target | Does |
|---|---|
| `make test` | vet and test every cartridge, then the manual lint |
| `make cart-<slug>` | regenerate `levels_gen.go` and `manifest_gen.go`, then TinyGo and the wasmfix post-pass |
| `make build` | `cart-<slug>` for every cartridge |
| `make fmt-check` | fail on any gofmt-dirty file |
| `make license-check` | fail on a source file without its SPDX line |
| `make manual-icons` | regenerate each `manual-icon.png` from the cartridge's icon |

The layout: each cartridge is its own Go module under `cartridges/<slug>/`;
[`docs/`](docs) holds the manuals index and the manual format;
`tools/fleetlint` checks every manual against its manifest; `scripts/`
holds the licence check and the one-off publish step.

## Assets

Art and sound that a lab can author live in each cartridge's `assets/`,
and the file is the source: open it in spritelab, soundlab or levellab at
[arkade.build/dev/tools](https://arkade.build/dev/tools), edit it, and
save it back over the file. A sprite is an `.arkadesprite.png` sheet of
named cels that `Init` lifts with `s.LoadSheet` and `Sheet.Cel` (Snake's
`sprites.go`); the manifest's `icon` and `logo` are sheets too. A sound is
an `.arkadesound.wav` whose envelope carries its recipe, so soundlab
re-renders it byte for byte.

A level pack is an `.arkadelevels` text file that the build compiles into
`levels_gen.go`. It sits in `assets/` with the rest but is left out of the
`.rcart`, since nothing reads it at run time.

Code draws only what is computed per frame: the snake, the apple, the
grid, text set in the SDK's fonts.

## Writing your own

The SDK (`cartlib`, `ui`, `fsm`, `levels`), the `carttest` harness and the
authoring guides live in the [arkade](https://github.com/stealthmonkey99/arkade)
repo. Start from its `sdk/template` with
[getting started](https://github.com/stealthmonkey99/arkade/blob/main/sdk/docs/getting-started.md)
and [GAME_AUTHORING.md](https://github.com/stealthmonkey99/arkade/blob/main/docs/GAME_AUTHORING.md),
and read the cartridges here as worked examples: Life is the simplest,
Snake uses everything. [CONTRIBUTING.md](CONTRIBUTING.md) covers sending
one here.

## Licence

GPL-2.0-or-later, every file unless it says otherwise. The licence text is
in [`LICENSE`](LICENSE) and the application notice in [`COPYING`](COPYING).
