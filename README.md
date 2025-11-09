# PlutoRCON (Black Ops 2 / T6 RCON Client for Go)

A lightweight, concurrency‑safe Go client for communicating with a T6 (Call of Duty: Black Ops II / Plutonium style) server over UDP RCON and query packets.

## Features
- Simple constructor `rcon.New(ip, port, password)`
- Authenticated RCON command execution with retry + adaptive read windows
- High‑level helpers:
  - `Status()`
  - `GetInfo()`(`ServerInfo`)
  - `GetStatus()` (`ServerStatusInfo`)
  - `GetDvar()` / `SetDvar()`
  - `Say()`, `Tell()`, `Kick()` 

- Thread‑safe (mutex around UDP conn usage)

## Installation
```bash
go get github.com/Yallamaztar/PlutoRCON
```

## Quick Start
```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/Yallamaztar/PlutoRCON/rcon"
)

func main() {
    rc, err := rcon.New(
        os.Getenv("RCON_IP"),
        os.Getenv("RCON_PORT"),
        os.Getenv("RCON_PASSWORD"),
    )

    if err != nil { log.Fatal(err) }
    defer rc.Close()

    status, err := rc.Status()
    if err != nil { log.Fatal(err) }
    fmt.Printf("Map: %s Players: %d\n", status.Map, len(status.Players))
    for _, p := range status.Players {
        fmt.Printf("#%d %s GUID:%s Ping:%v Score:%d\n", p.ClientNum, p.Name, p.GUID, p.Ping, p.Score)
    }
}
```

---

## API Overview
| Method | Purpose |
|--------|---------|
| `SendCommand(cmd, args, opts...)` | RCON send with options (retries, read timeouts) |
| `Status()` | Returns `*ServerStatus` (map + players) |
| `GetInfo()` | Returns `*ServerInfo` (ident / static-ish configuration) |
| `GetStatus()` | Returns `*ServerStatusInfo` (dynamic flags / gameplay settings) |
| `GetDvar(name)` | Retrieves a dvar value, stripping color codes, retrying if polluted |
| `SetDvar(name,value)` | Sets a dvar (auto‑quotes if needed) |
| `Say(message)` | Broadcast to all players |
| `Tell(clientNum,message)` | Private message to one player |
| `Kick(player,reason)` | Kick by name with reason |

### Dvar Retrieval Robustness
Some servers intermittently echo unrelated dvars (e.g. `sv_iw4madmin_in`). `GetDvar` transparently retries up to 3 attempts until it captures the correct value or returns an error


## Error Handling Patterns
Typical errors you should handle:
- Initialization: invalid port / missing password
- Network I/O: timeouts (consider wrapping calls with backoff if doing frequent polling)
- Parsing: if upstream format changes, methods return descriptive errors instead of panicking

## Example: Broadcast & Private Message
```go
if err := rc.Say("Server restart in 5 minutes"); err != nil { 
    log.Println("say error:", err) 
}

if err := rc.Tell(3, "Hi there!"); err != nil { 
    log.Println("tell error:", err) 
}
```

## Example: Dvar Operations
```go
maxClients, err := rc.GetDvar("sv_maxclients")
if err != nil { 
    log.Fatal(err)
}

fmt.Println(maxClients)

if err := rc.SetDvar("sv_hostname", "My Cool Server"); err != nil {
    log.Println("failed to set hostname:", err)
}
```

## Example: Server Info Snapshots
```go
info, err := rc.GetInfo()
if err != nil { 
    log.Fatal(err) 
}

fmt.Printf(
    "Host:%s Map:%s Clients:%d Gametype:%s\n", 
    info.Hostname, info.MapName, info.MaxClients, info.GameType
)

statInfo, err := rc.GetStatus()
if err != nil { 
    log.Fatal(err) 
}

fmt.Printf(
    "Status Map:%s Max:%d Game:%s Mod:%v Voice:%v\n", 
    statInfo.MapName, statInfo.SvMaxClients, statInfo.GameType, statInfo.ModEnabled, statInfo.SvVoice
)
```