# Backtrack

A chat server for retro/older consoles (and modern desktop clients), using a lightweight binary TCP protocol designed to be easy to implement on constrained hardware.

The server is written in Go. Clients can be written in anything (like C for homebrew consoles, or Go/Wails for desktop) as long as they can connect over raw TCP.

## Why binary instead of JSON?

Older consoles have limited memory and CPU. Parsing JSON in C requires a library, dynamic allocation, and wastes bandwidth on syntax characters (`{`, `"`, `:`, etc.). Our binary protocol is just raw bytes with known offsets — a C client can parse a message with a few `recv()` calls and zero allocations.

## Architecture

- **One global chat room** — all connected users see all messages.
- **Hub-based Event Loop** — all chat state and routing is managed by a central lock-free `Hub` (`server/hub.go`) that processes events sequentially via channels, preventing race conditions without needing mutexes.
- **Per-user Goroutines** — each connection gets a goroutine for reading incoming TCP frames and a dedicated `writeLoop` for sending them.
- **Non-blocking broadcasts** — a slow client gets disconnected instead of freezing the Hub or other users.
- **Ping/pong keepalive** — dead connections are detected and cleaned up automatically.

## Protocol

Every message follows this

- **type**: 1 byte — what kind of message this is
- **length**: 2 bytes — big-endian, how many payload bytes follow
- **payload**: variable — depends on message type

### Client → Server

| Type | Name | Payload |

| `0x01` | Identify | `username_len(1) + username + avatar_id(2)` |
| `0x02` | Chat Send | `body_len(2) + body` |
| `0x03` | Set Presence | `status(1)` |
| `0x04` | Pong | *(no payload)* |

### Server → Client

| Type | Name | Payload |

| `0x80` | Welcome | `user_id(4)` |
| `0x81` | Chat Recv | `user_id(4) + body_len(2) + body` |
| `0x82` | User Joined | `user_id(4) + avatar_id(2) + username_len(1) + username` |
| `0x83` | User Left | `user_id(4)` |
| `0x84` | User Info | `user_id(4) + avatar_id(2) + status(1) + username_len(1) + username` |
| `0x85` | Presence Update | `user_id(4) + status(1)` |
| `0x86` | Ping | *(no payload)* |
| `0xFF` | Error | `msg_len(2) + msg` |

### Presence Statuses

| Value | Meaning |

| `0x00` | Online |
| `0x01` | Away |
| `0x02` | Busy |
| `0x03` | Offline |


### Avatar IDs

Avatars are referenced by a `uint16` ID. The server stores and broadcasts the ID — it's up to each client to map IDs to actual sprites/images. This keeps the protocol lightweight (2 bytes instead of sending image data).

## Project Structure

```
backtrack/
├── cmd/
│   └── server/main.go      # Server entrypoint — parses flags, starts the server
├── desktop/                # Wails Desktop Application (Modern UI Client)
├── protocol/               # Binary wire format — framing, builders, parsers
├── server/
│   ├── server.go           # Chat server — TCP connections & lifecycle
│   └── hub.go              # Lock-free event router for chat state
└── go.mod
```

## Building & Running

### Server

```bash
# build server
go build -o bin/server ./cmd/server

# run server (default :9000)
./bin/server

# run directly with go
go run ./cmd/server/main.go -addr :9000
```

### Desktop Client (Wails)

```bash
cd desktop

# run client in development mode (hot-reloading)
wails dev

# build client for production
wails build
```

## Roadmap

- [ ] Console clients (C — PSP, Wii, DS, etc.)
- [ ] Channels / rooms
- [ ] Message history / persistence
- [ ] User authentication
- [ ] Tests
