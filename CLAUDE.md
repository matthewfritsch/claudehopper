# claudehopper-go

Go CLI for managing Claude Code configuration profiles via symlink switching.

## Architecture

- `main.go` — entry point, version injection via ldflags
- `cmd/` — thin Cobra command wrappers (no business logic)
- `internal/config/` — paths, config.json, manifest.go
- `internal/fs/` — atomic symlinks (renameio/v2), protected paths
- `internal/profile/` — all profile business logic (create, switch, share, tree, etc.)
- `internal/usage/` — usage.jsonl tracking
- `internal/updater/` — GitHub release checking with 24h TTL

## Conventions

- **stdlib testing only** — no testify. Fixtures in `testdata/` dirs.
- **os.Lstat** everywhere for symlink interrogation (never os.Stat).
- **AtomicSymlink** for all symlink operations (never os.Remove + os.Symlink).
- **Business logic in internal/, CLI wiring in cmd/** — cmd functions parse flags, call internal, format output.
- **Protected paths** match Python version exactly — see `internal/fs/protected.go`.
- **Format compatibility** — config.json and .hop-manifest.json must round-trip with the Python version.
- **Case-insensitive profile names** — normalized to lowercase.

## Testing

```bash
go test ./...              # quick
go test -v -race ./...     # full suite
```

## Building

```bash
make build                 # produces bin/claudehopper + bin/hop
make install               # installs to $GOPATH/bin
```
