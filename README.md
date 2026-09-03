# go-dmg-packer

A cross-platform CLI that packs a finished folder into a macOS `.dmg`, with an
optional Finder layout that turns the mounted window into a drag-to-install
pair.

The disk-image format itself comes from
[`github.com/jetbrains/go-dmg-writer`](https://github.com/JetBrains/go-dmg-writer);
this repository holds only the CLI and the `.DS_Store` layout.

```console
$ dmg-packer -src ./build/MyApp -out MyApp.dmg -name MyApp -layout
wrote MyApp.dmg (72301184 bytes)
```

## Install

`go-dmg-writer` is a private module, so point the toolchain at it before
building:

```console
$ export GOPRIVATE='github.com/jetbrains/*'
$ go install github.com/jetbrains/go-dmg-packer/cmd/dmg-packer@latest
```

Or from a checkout:

```console
$ CGO_ENABLED=0 go build -o dmg-packer ./cmd/dmg-packer
```

No cgo, no `hdiutil`, no external tools: the binary builds and runs on macOS,
Linux, and Windows. Windows is a poor host in practice, because NTFS loses the
POSIX modes and symlinks that a signed bundle needs.

## Flags

| Flag      | Default        | Meaning                                                     |
| --------- | -------------- | ----------------------------------------------------------- |
| `-src`    | —              | Source folder. Required.                                    |
| `-out`    | —              | Output `.dmg` path. Required.                               |
| `-mode`   | `udzo`         | `udro` (uncompressed UDIF) or `udzo` (per-chunk zlib).      |
| `-name`   | `disk image`   | Volume name as it appears in Finder.                        |
| `-time`   | `0`            | Unix timestamp baked into the volume. `0` means now.        |
| `-layout` | `false`        | Write the Finder `.DS_Store` into `-src` before packing.    |

A missing `-src` or `-out` prints the usage and exits 2. Any other failure logs
and exits 1. On success the tool prints `wrote <path> (<n> bytes)` to stderr.

## Staging the source folder

The tool takes a *finished* folder. The writer copies the source tree verbatim
and formats the volume itself, so there is no seed image to supply — but it
also means the caller stages everything the mounted volume should show:

```
MyApp/                  <- pass this as -src
├── MyApp.app/          <- the signed bundle
├── Applications        <- symlink to /Applications
└── .background/        <- empty, or holding a window background image
```

Then `-layout` adds the `.DS_Store`. It requires exactly one `*.app` in the
volume root, because Finder keys each icon's position on the item's file name.

## Reproducible output

Pass a fixed `-time` and the output is byte-stable: the writer stamps every
catalog record with it, so source mtimes need no normalizing.

```console
$ dmg-packer -src ./vol -out a.dmg -time "$SOURCE_DATE_EPOCH"
$ dmg-packer -src ./vol -out b.dmg -time "$SOURCE_DATE_EPOCH"
$ cmp a.dmg b.dmg   # identical
```

## Limitations

Inherited from the writer:

- The volume is HFSX (case-sensitive), not classic HFS+.
- Only `udro` and `udzo`. No `ulfo`/LZFSE, no `udbz`/bzip2.
- No GUID partition map, so the image mounts but non-mounting parsers may
  reject it.
- No software licence agreement (SLA) resource, no journaling, no resource
  forks, no encryption.
- Hard links are duplicated, and symlink targets are stored verbatim.
- Write-only: it cannot read an existing `.dmg`.

Codesigning and notarization are the caller's job, after the image is written.

## Development

```console
$ go test ./...
$ go vet ./...
```

`internal/dsstore` carries the layout tests, which decode the `Bud1` container
that `Encode` produces and assert the icon positions and window records.

## Licence

Apache 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
