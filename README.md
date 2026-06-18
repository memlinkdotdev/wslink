# wslink

**Single-binary TCP bridge between Windows and WSL.** Zero-config, no dependencies, no installation. Statically linked Go binary.

## Problem

WSL2 has its own virtual network. TCP services on `localhost` inside WSL are not always reachable from Windows (and vice versa) - depends on WSL version, distro state, and corporate proxies. The Windows tooling (`netsh interface portproxy`, `wsl --list`) requires admin, gets out of sync, and breaks on reboot.

## Solution

`wslink` is a single Go binary that runs on **either side** and proxies TCP traffic:

```
Windows side:        wslink forward 4444           listens 127.0.0.1:4444
                                                       |
                                                       +--> WSL distro IP:4444
                                                          (auto-detected)

WSL side:            wslink forward 4444           listens 127.0.0.1:4444
                                                       |
                                                       +--> Windows host IP:4444
                                                          (auto-detected from /etc/resolv.conf)
```

Direct TCP proxy - no `netsh`, no `iptables`, no admin, no leftover state. Press Ctrl-C and it's gone.

## Install

### One-line installer

**Windows**
```powershell
irm https://raw.githubusercontent.com/pyrofast/wslink/main/install.ps1 | iex
```

**Linux**
```bash
curl -fsSL https://raw.githubusercontent.com/pyrofast/wslink/main/install.sh | bash
```

### Manual

Grab a binary from the [latest release](https://github.com/pyrofast/wslink/releases):

| OS      | Arch    | Binary                                |
| ------- | ------- | ------------------------------------- |
| Windows | amd64   | `wslink-windows-amd64.zip`            |
| Windows | arm64   | `wslink-windows-arm64.zip`            |
| Linux   | amd64   | `wslink-linux-amd64.tar.gz`           |
| Linux   | arm64   | `wslink-linux-arm64.tar.gz`           |

## Usage

```bash
# Auto-detect target
wslink forward 4444

# From Windows: specify WSL distro
wslink forward 4444 --wsl-name Ubuntu

# From WSL: specify Windows host IP
wslink forward 4444 --windows-host 172.20.0.1

# Skip auto-detect: target host:port directly
wslink forward 4444 --connect 192.168.1.5:4444

# Bind to specific address
wslink forward 4444 --listen 127.0.0.1
```

## Flags

```
wslink forward <port> [flags]

  --connect <host:port>    Target directly (skip auto-detect)
  --listen <addr>          Listen address (default 127.0.0.1)
  --wsl-name <distro>      WSL distro name (Windows only)
  --windows-host <ip>      Windows host IP (WSL only)
```

## Requirements

| Requirement | Details                                  |
| ----------- | ---------------------------------------- |
| OS          | Windows 10/11 with WSL2, or WSL2/Linux   |
| Runtime     | Static binary, no dependencies           |
| Elevation   | None (raw TCP, no kernel hooks)          |

## Architecture

```
wslink (Go, statically linked)
  |-- Auto-detect target
  |     Windows -> wsl.exe --list + hostname -I
  |     WSL     -> /etc/resolv.conf nameserver
  |-- TCP listener (net.Listen)
  |-- Per-connection goroutine
        |-- io.Copy bidirectional proxy
```

One goroutine per connection, two `io.Copy` goroutines for the bidirectional pipe. Releases are 1-2 MB statically linked binaries - no runtime, no CGO, no libc.
