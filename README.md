# btrman

`btrman` is a terminal UI for browsing Linux manual pages. It keeps the official `man` database as the source of truth while adding fast command discovery, section navigation, in-page search, related-page links, recents, and snippet copying.

## Status

Currently being developed, as of now it requires the man program (provided by the man-db package) 

## Requirements

- Go 1.24.5 or newer
- Linux manual page tooling available on `PATH` (`man` binary)

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
btrman

# Open a specific page
btrman ls
btrman 'printf(1)'
btrman 1 printf
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
