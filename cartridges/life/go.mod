module arkade-life-cartridge

go 1.23.4

require (
	github.com/stealthmonkey99/arkade v0.1.0
	github.com/stealthmonkey99/arkade/sdk/cartlib v0.1.0
	github.com/stealthmonkey99/arkade/sdk/fsm v0.1.0
	github.com/stealthmonkey99/arkade/sdk/levels v0.1.0
	github.com/stealthmonkey99/arkade/sdk/ui v0.1.0
)

require (
	github.com/gopxl/beep/v2 v2.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tetratelabs/wazero v1.10.1 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/term v0.28.0 // indirect
)

replace (
	github.com/stealthmonkey99/arkade => ../../../arkade
	github.com/stealthmonkey99/arkade/sdk/cartlib => ../../../arkade/sdk/cartlib
	github.com/stealthmonkey99/arkade/sdk/fsm => ../../../arkade/sdk/fsm
	github.com/stealthmonkey99/arkade/sdk/levels => ../../../arkade/sdk/levels
	github.com/stealthmonkey99/arkade/sdk/ui => ../../../arkade/sdk/ui
)
