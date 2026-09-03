// Command dmg-packer builds a macOS disk image out of a finished folder.
// It is a CLI over [dmg.DMG.Create] in `github.com/jetbrains/go-dmg-writer`.
//
// Usage:
//
//	dmg-packer -src ./build/MyApp -out MyApp.dmg -mode udzo -name MyApp
//
// The tool takes a finished folder: [dmg.DMG.Create] copies a source tree
// verbatim, so a `/Applications` symlink or a `/.background/` folder is
// staged by the caller. It also formats the volume itself, so there is no
// seed image to supply.
//
// With -layout the tool adds the Finder `.DS_Store` that turns the window
// into a drag-to-install pair. See [dsstore].
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jetbrains/go-dmg-packer/internal/dsstore"
	"github.com/jetbrains/go-dmg-writer"
)

func main() {
	var (
		srcFlag    = flag.String("src", "", "source folder (required)")
		outFlag    = flag.String("out", "", "output .dmg path (required)")
		modeFlag   = flag.String("mode", "udzo", "output mode: udro | udzo")
		nameFlag   = flag.String("name", "disk image", "volume name as it appears in Finder")
		timeFlag   = flag.Int64("time", 0, "Unix timestamp baked into the volume (0 = now)")
		layoutFlag = flag.Bool("layout", false, "add the Finder `.DS_Store` that lays out the installer window")
	)
	flag.Parse()
	if *srcFlag == "" || *outFlag == "" {
		flag.Usage()
		os.Exit(2)
	}

	var mode dmg.Mode
	switch strings.ToLower(*modeFlag) {
	case "udro":
		mode = dmg.ModeReadOnly
	case "udzo":
		mode = dmg.ModeReadOnlyCompressed
	default:
		log.Fatalf("unknown -mode %q (want udro or udzo)", *modeFlag)
	}

	// A fixed time makes the output byte-stable. The writer stamps every catalog record with it, so the caller does
	// not have to normalize the source mtimes.
	var when time.Time
	if *timeFlag != 0 {
		when = time.Unix(*timeFlag, 0).UTC()
	}

	if *layoutFlag {
		if err := writeDSStore(*srcFlag); err != nil {
			log.Fatalf("dmg-packer: %v", err)
		}
	}

	d := &dmg.DMG{
		VolumeName: *nameFlag,
		Time:       when,
		// `hdiutil`'s default. HFS+ reads owner 99 as "no specific owner", so the volume mounts with the same
		// permissions on any host. The struct's zero value is the literal root UID, which is NOT what we want.
		OwnerID: dmg.OwnerIDUnset,
		GroupID: dmg.OwnerIDUnset,
	}
	if err := d.Create(*srcFlag, *outFlag, mode); err != nil {
		log.Fatalf("dmg create: %v", err)
	}
	st, _ := os.Stat(*outFlag)
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", *outFlag, st.Size())
}

// writeDSStore puts the Finder layout in the volume root, keyed on the bundle it finds there. The file has to exist
// before [dmg.DMG.Create] copies the tree, because the writer takes the source verbatim.
func writeDSStore(src string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read the source folder: %w", err)
	}
	var bundles []string
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			bundles = append(bundles, e.Name())
		}
	}
	if len(bundles) != 1 {
		return fmt.Errorf("the source folder %s holds %d `.app` bundles, want exactly 1: %v", src, len(bundles), bundles)
	}
	b, err := dsstore.Encode(bundles[0])
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(src, ".DS_Store"), b, 0o644)
}
