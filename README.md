# btrman

`btrman` is a modern Bubble Tea terminal UI for browsing Linux manual pages. It keeps the official `man` database as the source of truth while adding fast command discovery, section navigation, in-page search, related-page links, recents, and snippet copying.

## Status

V1 implementation in progress.

## Requirements

- Go 1.24.5 or newer
- Linux manual page tooling available on `PATH` (`man` and preferably `apropos`/`man -k`)

The project is intentionally vendorless: dependencies are resolved with Go modules through `go.mod` and `go.sum`; no `vendor/` directory is used.

## Build

```sh
go build ./cmd/btrman
```

Or with Nix:

```sh
nix build .#
```

## Run

```sh
# Open fuzzy command search
go run ./cmd/btrman

# Open a specific page
go run ./cmd/btrman ls
go run ./cmd/btrman 'printf(1)'
go run ./cmd/btrman 1 printf
```

Or with Nix:

```sh
nix run .#
nix run .# -- ls
nix run .# -- 'printf(1)'
```

## Development shell

```sh
nix develop
```

The Nix development shell includes Go tooling, Linux manual-page tooling, and clipboard helpers.

## Test

```sh
go test ./...
```

## Keybindings

| Key | Action |
| --- | --- |
| `q`, `ctrl+c` | Quit |
| `j`/`k`, arrows | Scroll document or move list selection |
| `pgup`/`pgdn`, `home`/`end` | Fast document navigation |
| `[` / `]`, `shift+tab` / `tab` | Previous/next section |
| `o` | Open command search |
| `/` | In-page search |
| `n` / `N` | Next/previous in-page match |
| `r` or `g` | Related pages from detected man references |
| `h` | Recent history |
| `backspace` | Go back to the previous page |
| `y` | Copy current line |
| `Y` | Copy current paragraph/block |
| `esc` | Close the current overlay/input |

## Architecture

- `cmd/btrman`: CLI entrypoint
- `internal/app`: Bubble Tea application model and views
- `internal/manual`: system `man` provider and output cleanup
- `internal/document`: document parsing, section detection, related links
- `internal/search`: fuzzy page search and in-page search
- `internal/history`: XDG recent-history persistence
- `internal/clipboard`: copy abstraction
