# Contributing to Årkade Cartridges

Thanks for being here. This repo holds the game cartridges that ship with
[Årkade](https://github.com/stealthmonkey99/arkade), a real arcade cabinet
in a real living room that happens to be open source. It is a personal
project, not one with a governance model, so the deal is up front.

## The fork-first deal

- **Fork freely.** GPL-2.0-or-later means the code is yours to take, remix
  and build your own cabinet's library on. A fork needs no permission and no
  upstream plan.
- **Open an issue before a large PR.** Five minutes of talking is the
  difference between "merged" and "beautiful work that doesn't fit the
  cabinet."
- **Review is slow and merging is discretionary.** PRs may sit for weeks, and
  some good ones won't merge. That is what "personal project" means, and a
  fork is always a fine home for your version.
- **Fixes and polish are the sweet spot.** You played a cartridge, something
  bugged you, you fixed it: bug fixes, balance and feel, manual corrections.
- **This repository is a published mirror.** Development happens in a
  private upstream repository that also holds the cabinet's unreleased
  games; each release lands here as one commit. A merged PR is applied
  there with you as the author and appears here in the next release
  commit, which is when the PR gets closed with a pointer to it. Issues
  and Discussions are read here directly.

## Building a cartridge

A cartridge is a TinyGo to wasm module with its own `go.mod`, living under
`cartridges/<slug>/`. It runs unmodified on the cabinet and in the browser.
The toolchain versions are in the [README](README.md#build-and-test).

The SDK modules come from the engine repo:
`github.com/stealthmonkey99/arkade/sdk/{cartlib,ui,fsm,levels}`. The
authoring guides, the SDK reference and the starter template (`sdk/template`)
live there too. Start from `sdk/template` with
[`docs/GAME_AUTHORING.md`](https://github.com/stealthmonkey99/arkade/blob/main/docs/GAME_AUTHORING.md)
and the [ABI reference](https://github.com/stealthmonkey99/arkade/blob/main/docs/CARTRIDGE.md),
and read `cartridges/life` (simplest) and `cartridges/snake` (everything)
here as worked examples.

Build and test from the repo root (a new cartridge joins `CARTS` in the
Makefile):

```sh
make test          # vet and test every cartridge
make cart-life     # rebuild cartridges/life/life.wasm (TinyGo, then wasmfix)
make fmt-check license-check
```

The Makefile is the one place the TinyGo flags live. The built `.wasm` is
committed, so the tests never need TinyGo; one test per
cartridge checks that the checked-in binary matches its source, on the
host that built it (CI skips the byte comparison and runs the goldens
against a rebuild instead). Tests drive
the real wasm through `carttest`, the engine repo's fake host, and compare
frames pixel for pixel against golden files.

The cartridges' `go.mod` files carry local `replace` directives so the
repo builds beside a sibling engine checkout. `scripts/release-deps.sh [tag]`
(default `v0.1.0`) is the maintainer's one-off publish step: it deletes those
directives, pins the engine requires to the tag and runs `go mod tidy`. A
fresh clone of the published repo already has tagged versions and needs none
of this.

## Constraints that apply to every cartridge

1. **Button semantics.** B is back and exit only; D is interact, and the D
   light is lit exactly when pressing D does something. Hold-B exit works from
   every state. Visual style is free, but interactions are uniform so the
   cabinet always feels like one machine.
2. **Determinism.** Time comes only from `dtMicros`, randomness only from
   `RandomU64`, and no map iteration order may reach a frame. Gameplay must
   be reproducible from a seed plus an input log.
3. **Goldens are the oracle.** A cleanup that moves a golden frame is a
   behaviour change. Intentional changes regenerate goldens
   (`UPDATE_GOLDENS=1 go test ./...`) and say why in the PR.

## Proposing a new cartridge

Open an issue first, using the game idea template: the pitch, player modes
and tags, and where it stands (idea, prototype, playable). A cartridge is a
long-term commitment for a reviewer, so agreeing on fit before you build
saves everyone a lot of evenings.

## What a cartridge needs to be accepted

In brief, a cartridge is ready for this repo when:

- Golden and determinism tests pass and cover its main screens and a scripted
  run.
- `manifest.json` is complete: slug, title, author, version, description,
  tags, players, audience, icon and logo.
- A player manual (`MANUAL.md`, in the [manual format](docs/MANUAL_FORMAT.md))
  is present and matches the controls.
- Every asset and idea is original or properly licensed: no ripped sprites,
  sounds, music or trademarked names.
- It has been played through on a real cabinet, with the 64×64 panel and the
  physical buttons.

The engine repo's `tools/preflight` scanner checks most of the mechanical
parts. From a sibling engine checkout:

```sh
ARKADE_CARTRIDGE_ROOTS=../arkade-cartridges/cartridges go run ./tools/preflight <slug>
```

## Code style

Follow the conventions in the engine repo's
[CONTRIBUTING.md](https://github.com/stealthmonkey99/arkade/blob/main/CONTRIBUTING.md):
the tree stays `gofmt`-clean, and comments are scarce, saved for the
non-obvious why.

## Licensing of contributions

By submitting a PR you agree your contribution is licensed under
**GPL-2.0-or-later**, same as the project (inbound = outbound). No CLA.
