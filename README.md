# WebSocket FD Handoff (OpenResty + Go)

This project demonstrates an efficient architecture where WebSocket connections are accepted by OpenResty (Nginx with Lua), and the raw socket file descriptors (FDs) are transferred to a Go process using `SCM_RIGHTS`. The Go backend takes over the full lifecycle of the WebSocket communication — all with minimal memory overhead in Nginx.

---

## 🧐 Motivation

Traditional WebSocket handling with `proxy_pass` causes Nginx to hold onto all WebSocket connections, leading to high memory usage.

This architecture allows:

- Nginx to perform the WebSocket HTTP upgrade
- Lua/FFI to extract the underlying file descriptor
- The FD to be transferred to another process (Go) via Unix domain socket using `SCM_RIGHTS`
- Nginx to safely detach from the connection (with `dup2()` and dummy socket)
- Go to fully manage WebSocket frames using `gobwas/ws`

---

## 📁 Project Structure

```
.
├── Dockerfile
├── entrypoint.sh
├── go.mod / go.sum       # Go dependencies
├── main.go               # Go backend to receive and handle WebSocket FDs
├── nginx.conf            # OpenResty config for /ws
└── lua/
    ├── fd_manager.lua    # send_fd + replace_with_dummy
    ├── ffi_defs.lua      # All FFI C definitions
    └── ws_handler.lua    # Logic to extract, send, and detach FD
```

---

## 🔧 How it works

1. Client connects to `/ws`
2. `resty.websocket.server` accepts and upgrades
3. Lua extracts the `fd` from the internal Nginx struct using `ffi`
4. FD is sent over a Unix socket to the Go process (`sendmsg + SCM_RIGHTS`)
5. Lua replaces the original FD in Nginx with a dummy TCP socket (`dup2`) to prevent shutdown/close errors
6. Go receives the FD, reconstructs `net.Conn`, and uses `gobwas/ws` to handle frames

---

## 🥪 Memory usage comparison (15,000 connections)

| Architecture        | Nginx Memory | Go Memory |
|---------------------|--------------|-----------|
| proxy_pass          | 219MB        | —         |
| SCM_RIGHTS (this)   | 8.1MB        | 219MB     |

✅ Massive Nginx memory reduction by offloading the WebSocket connections.

---

## 🚀 How to run

```bash
docker build -t websocket-fd-handoff .
docker run --rm -it --network host websocket-fd-handoff
```

---

## 🔒 Permissions

To allow Nginx (running as `nobody`) to connect to the Unix socket:

- The Go process calls `os.Chown(path, -1, gid_of_nobody)`
- Then sets permissions to `0660` using `os.Chmod`
- This allows group-based access control

---

## 🔖 Requirements

- Linux (for `SCM_RIGHTS`, `dup2`, and Unix sockets)
- Go ≥ 1.24
- OpenResty or Nginx with Lua module

---

## 📌 Credits

- Uses [`gobwas/ws`](https://github.com/gobwas/ws) for fast, low-level WebSocket frame handling
- Thanks to `LuaJIT` and `OpenResty` for giving us raw FD access in userland

---

## 🧩️ Future ideas

- Integration into ShellHub SSH tunnel flow

---
