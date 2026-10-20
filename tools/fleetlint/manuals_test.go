// SPDX-License-Identifier: GPL-2.0-or-later

package fleetlint

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// manifestDoc is the part of a cartridge manifest this lint reads.
type manifestDoc struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// cartridgeRoots lists the cartridges directories to lint: the one beside
// this tool, or ARKADE_CARTRIDGE_ROOTS (comma-separated) to lint several
// trees in one run. Each root's index is <root>/../docs/MANUALS.md.
func cartridgeRoots() []string {
	if env := strings.TrimSpace(os.Getenv("ARKADE_CARTRIDGE_ROOTS")); env != "" {
		var out []string
		for _, p := range strings.Split(env, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{"../../cartridges"}
}

// TestManualsCurrent checks that every cartridge ships a MANUAL.md whose
// title matches the manifest, whose footer quotes the manifest's current
// version, whose header uses manual-icon.png (the icon's current upscale),
// and which docs/MANUALS.md links. On a version mismatch, reread the manual
// against your change before updating the footer.
func TestManualsCurrent(t *testing.T) {
	checked := 0
	for _, cartsDir := range cartridgeRoots() {
		checked += lintManuals(t, cartsDir)
	}
	if checked == 0 {
		t.Fatalf("found no cartridges under %s — wrong directory?", strings.Join(cartridgeRoots(), ", "))
	}
}

// lintManuals checks every cartridge under cartsDir and returns how many it
// checked.
func lintManuals(t *testing.T, cartsDir string) int {
	t.Helper()
	indexPath := filepath.Join(cartsDir, "..", "docs", "MANUALS.md")

	indexRaw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read %s: %v", indexPath, err)
	}
	index := string(indexRaw)

	entries, err := os.ReadDir(cartsDir)
	if err != nil {
		t.Fatalf("read %s: %v", cartsDir, err)
	}
	checked := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		manifestPath := filepath.Join(cartsDir, slug, "manifest.json")
		manifestRaw, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // not a cartridge directory
		}
		var manifest manifestDoc
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			t.Errorf("%s: unparseable manifest.json: %v", slug, err)
			continue
		}
		if manifest.Title == "" {
			t.Errorf("%s: manifest has no title", slug)
			continue
		}
		checked++

		manualPath := filepath.Join(cartsDir, slug, "MANUAL.md")
		manualRaw, err := os.ReadFile(manualPath)
		if err != nil {
			t.Errorf("%s: no MANUAL.md — every cartridge ships a player manual (see the manuals index)", slug)
			continue
		}
		manual := string(manualRaw)

		if !strings.Contains(manual, "# "+manifest.Title) {
			t.Errorf("%s: MANUAL.md has no '# %s' title matching the manifest title", slug, manifest.Title)
		}
		wantFooter := "*Manual for version " + manifest.Version
		if !strings.Contains(manual, wantFooter) {
			t.Errorf("%s: MANUAL.md footer doesn't quote manifest version %s — a gameplay change bumped the version? Reread the manual against the change, then update the footer (docs/MANUAL_FORMAT.md, keeping manuals current, rule 2)", slug, manifest.Version)
		}
		if !strings.Contains(manual, `src="manual-icon.png"`) {
			t.Errorf("%s: MANUAL.md header doesn't use manual-icon.png", slug)
		}
		if err := manualIconCurrent(filepath.Join(cartsDir, slug)); err != nil {
			t.Errorf("%s: %v — regenerate with `make manual-icons`", slug, err)
		}
		if !strings.Contains(index, "cartridges/"+slug+"/MANUAL.md") {
			t.Errorf("%s: not linked from %s's library table", slug, indexPath)
		}
	}
	return checked
}

// manualIconScale is tools/manualicon's upscale.
const manualIconScale = 16

// manualIconCurrent checks manual-icon.png against the icon it is baked
// from, pixel for pixel, since the PNG encoder's bytes are not stable.
func manualIconCurrent(dir string) error {
	icon, err := decodePNG(filepath.Join(dir, "manual-icon.png"))
	if err != nil {
		return fmt.Errorf("manual-icon.png: %w", err)
	}
	src, err := decodePNG(filepath.Join(dir, "assets", "icon.arkadesprite.png"))
	if errors.Is(err, fs.ErrNotExist) {
		src, err = decodePNG(filepath.Join(dir, "assets", "icon.png"))
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	sb, ib := src.Bounds(), icon.Bounds()
	if ib.Dx() != sb.Dx()*manualIconScale || ib.Dy() != sb.Dy()*manualIconScale {
		return fmt.Errorf("manual-icon.png is %dx%d, not the %dx%d icon at %dx", ib.Dx(), ib.Dy(), sb.Dx(), sb.Dy(), manualIconScale)
	}
	for y := 0; y < ib.Dy(); y++ {
		for x := 0; x < ib.Dx(); x++ {
			want := color.NRGBAModel.Convert(src.At(sb.Min.X+x/manualIconScale, sb.Min.Y+y/manualIconScale))
			if got := color.NRGBAModel.Convert(icon.At(ib.Min.X+x, ib.Min.Y+y)); got != want {
				return fmt.Errorf("manual-icon.png differs from the icon at (%d, %d)", x, y)
			}
		}
	}
	return nil
}

func decodePNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
