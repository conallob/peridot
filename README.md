# peridot

Cache-aware, multi-source wallpaper rotation daemon.

[![Build](https://github.com/conallob/peridot/actions/workflows/ci.yml/badge.svg)](https://github.com/conallob/peridot/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/conallob/peridot/branch/main/graph/badge.svg)](https://codecov.io/gh/conallob/peridot)
[![Go Reference](https://pkg.go.dev/badge/github.com/conallob/peridot.svg)](https://pkg.go.dev/github.com/conallob/peridot)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow?logo=buy-me-a-coffee)](https://www.buymeacoffee.com/conallob)

## Overview

macOS's native `wallpaper.agent` has no cache bounds and eagerly caches entire
content folders. On a large Google Drive folder this can balloon to tens of
gigabytes of cached imagery (96GB has been observed in the wild) with no way to
constrain it. peridot replaces that agent with a well-engineered daemon that has
explicit cache policies, pluggable content sources, and three client interfaces.

## Features

- **Bounded cache** — enforce both a maximum size (e.g. `2GB`) and a maximum
  item count, with selectable eviction policies (LRU, FIFO, random).
- **Pluggable sources** — Local directories, Digital Blasphemy, Google Drive,
  and arbitrary URL endpoints, each with its own weight in the rotation.
- **Three client interfaces** — a `peridot` CLI, a companion Swift menubar app,
  and an MCP server for agent/automation integration.
- **OS-agnostic core** — all OS-specific behaviour lives behind interfaces, so
  the scheduler, cache, and source logic are portable and testable.

## Architecture

peridot is split into a daemon and its clients:

- **`peridotd`** — the long-running daemon. Owns the scheduler, the bounded
  cache, and the content sources. Listens on a Unix domain socket.
- **`peridot`** — the CLI client. Sends length-prefixed protobuf commands to the
  daemon over the Unix socket.
- **Menubar app** — a native Swift menubar client that speaks the same IPC.
- **MCP server** — exposes daemon operations as MCP tools (`peridot mcp`).

Clients and daemon communicate over a Unix socket using length-prefixed protobuf
frames (`internal/ipc`). All OS-specific operations (setting wallpaper, system
events, credential storage, service lifecycle) are abstracted behind the
`internal/platform` interfaces and selected at build time.

## Quick start

Install via Homebrew (recommended):

```sh
brew install conallob/tap/peridot
```

Or build from source:

```sh
make build
```

Install and start the daemon:

```sh
peridot daemon install
peridot daemon start
```

### macOS quarantine note

Binaries downloaded directly (not via Homebrew) may be blocked by Gatekeeper.
To clear the quarantine flag:

```sh
xattr -d com.apple.quarantine $(which peridot)
xattr -d com.apple.quarantine $(which peridotd)
```

## Configuration

peridot reads a TOML config (default `~/.peridot/config.toml`). A full example
lives in [`contrib/config.example.toml`](contrib/config.example.toml). A minimal
config:

```toml
[scheduler]
interval = "30m"
shuffle = true
change_on = ["timer", "wake"]

[cache]
max_size = "2GB"
max_items = 500
eviction_policy = "lru"

[[sources]]
id = "pictures"
type = "local"
display_name = "My Pictures"
weight = 1.0
path = "/home/user/Pictures/wallpapers"
watch = true
```

## CLI usage

| Command            | Description                                        |
| ------------------ | -------------------------------------------------- |
| `peridot next`     | Advance to the next wallpaper.                      |
| `peridot prev`     | Go back to the previous wallpaper.                 |
| `peridot pause`    | Pause automatic rotation.                          |
| `peridot resume`   | Resume automatic rotation.                          |
| `peridot status`   | Show daemon and cache status.                      |
| `peridot history`  | Show recently displayed wallpapers.               |
| `peridot sources`  | List configured content sources.                  |
| `peridot fetch`    | Fetch new items from a source.                    |
| `peridot config`   | Validate, compile, dump, diff, or reload config.  |
| `peridot daemon`   | Install, start, stop, restart, or status.         |
| `peridot stats`    | Show wallpaper display statistics (Atuin-style).  |
| `peridot mcp`      | Run the MCP server.                               |

## Platform support

| Platform | Status         |
| -------- | -------------- |
| macOS    | Full support   |
| Linux    | Stub / planned |
| Windows  | Stub / planned |

## Contributing

OS-specific behaviour is isolated behind the interfaces in
`internal/platform` (`Display`, `Events`, `Credential`, `Daemon`). Each backend
registers itself from a build-tagged `init()` via `platform.Register`. To add a
new OS backend, implement those interfaces in a build-tagged file and register
the implementation; the portable core (scheduler, cache, sources) needs no
changes. Unsupported operations should return `platform.ErrNotImplemented`.

## License

See the [LICENSE](LICENSE) file.
