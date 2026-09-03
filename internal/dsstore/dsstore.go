// Package dsstore composes the Finder `.DS_Store` that styles the installer
// window of a disk image.
//
// The format itself is handled by `github.com/leaanthony/dmg/dsstore`. This
// package only holds the layout: the window rectangle, the icon view
// options, and where the two icons sit. Finder keys an icon's position on
// the item's file name, so the layout cannot be bundle-name-agnostic on its
// own and [Encode] takes the name.
//
// "Write-once" like the rest of this tool: [Encode] composes one complete
// file. There is no support for reading or editing an existing `.DS_Store`.
package dsstore

import (
	"bytes"
	"fmt"
	"strings"

	upstream "github.com/leaanthony/dmg/dsstore"
)

// The drag-to-install pair, in points from the window's top-left corner.
// Finder reads an `Iloc` position as the icon's centre, so these are centres
// too. The values mirror `makedmg.py`, the macOS-host path this replaces.
const (
	AppIconX = 110
	AppIconY = 167

	AliasIconX = 340
	AliasIconY = 167
)

// AliasName is the `/Applications` symlink's name in the volume root, and so
// the name its icon position is keyed on.
const AliasName = "Applications"

// The installer window's rectangle on screen.
const (
	windowTop    = 100
	windowLeft   = 400
	windowBottom = 396
	windowRight  = 855
)

// The icon view options Finder stores in the `icvp` record. Sizes are in
// points.
const (
	iconSize    = 100
	textSize    = 12
	gridSpacing = 100
)

// sidebarWidth is the `fwsw` record. The window hides its sidebar, so this is
// the width Finder restores if the viewer turns the sidebar back on.
const sidebarWidth = 170

// hiddenNames are the items Finder parks outside the window so they stay
// invisible when the viewer has hidden files turned on. They are named even
// though a read-only volume has no `.fseventsd` or `.Trashes`, because
// `hdiutil` names them and an unused record is cheap.
var hiddenNames = []string{".background", ".DS_Store", ".fseventsd", ".Trashes"}

// hiddenIconY is the row the parked icons sit on, and hiddenIconX the first
// column. Each subsequent item steps right by hiddenIconStep, well past the
// window's right edge.
const (
	hiddenIconX    = 560
	hiddenIconY    = 170
	hiddenIconStep = 50
)

// Encode returns the `.DS_Store` bytes that style a volume root holding
// appEntry, the application bundle's name including its `.app` extension.
func Encode(appEntry string) ([]byte, error) {
	if appEntry == "" {
		return nil, fmt.Errorf("dsstore: the application bundle name is empty")
	}
	if !strings.HasSuffix(appEntry, ".app") {
		return nil, fmt.Errorf("dsstore: %q is not an `.app` bundle name", appEntry)
	}

	s := upstream.New()
	// Write the legacy `fwi0` record and no `bwsp` record. Finder prefers `bwsp`, the modern window plist, whenever
	// it is present. `bwsp` holds the window frame in Cocoa coordinates, which start at the bottom left of the
	// screen, and the frame includes the title bar. A rectangle baked in at build time therefore puts the window at
	// a height that depends on the screen, and it leaves the icon area 50 points too short. The two icons then touch
	// the bottom edge. `fwi0` holds the content rectangle from the top left of the screen, so it gives the same
	// window on every screen. It is also what `makedmg.py` writes, the macOS-host path this tool replaces.
	//
	// Do not add `upstream.Store.SetWindowSettings` back. That method writes `bwsp` only.
	s.AddRecord(upstream.NewFwi0Record(windowTop, windowLeft, windowBottom, windowRight, upstream.FCC("icnv")))
	s.AddRecord(upstream.NewFwswRecord(sidebarWidth))
	s.AddRecord(upstream.NewFwvhRecord(windowBottom - windowTop))
	s.SetViewStyle(upstream.FCC("icnv"))
	s.SetIconViewOptions(upstream.IconViewOptions{
		IconSize:      iconSize,
		TextSize:      textSize,
		GridSpace:     gridSpacing,
		LabelOnBottom: true,
		// The window shows one bundle and one alias, so an icon preview is the app icon and item info is noise.
		ShowIconPreview: true,
		ShowItemInfo:    false,
		// A plain white ground. `/.background/` is staged but empty, so a picture record would point at nothing.
		BackgroundType:  upstream.BackgroundColor,
		BackgroundRed:   1,
		BackgroundGreen: 1,
		BackgroundBlue:  1,
	})

	s.SetIconPosition(appEntry, AppIconX, AppIconY)
	s.SetIconPosition(AliasName, AliasIconX, AliasIconY)
	for i, name := range hiddenNames {
		s.SetIconPosition(name, hiddenIconX+i*hiddenIconStep, hiddenIconY)
	}
	s.EnsureDefaults()

	var buf bytes.Buffer
	if err := s.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("dsstore: encode: %w", err)
	}
	return buf.Bytes(), nil
}
