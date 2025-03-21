local ffi = require "ffi_defs"
local C = ffi.C

local fd_manager = require "fd_manager"

local M = {}

--- Extracts the file descriptor from a resty.websocket.server socket
---@param sock table
---@return number|string fd or error
local function get_socket_fd(sock)
    if not sock or not sock[1] then
        return nil, "invalid websocket object"
    end

    local u = ffi.cast("ngx_http_lua_socket_tcp_upstream_s*", sock[1])
    if not u or not u.peer or not u.peer.connection then
        return nil, "invalid upstream socket structure"
    end

    return tonumber(u.peer.connection.fd)
end

--- Handles the process of extracting, sending, and detaching the WebSocket fd
---@param sock table
function M.handle(sock)
    local fd, err = get_socket_fd(sock)
    if not fd then
        ngx.log(ngx.ERR, "failed to extract socket fd: ", err)
        return
    end

    ngx.log(ngx.INFO, "socket fd extracted: ", fd)

    local ok, send_err = fd_manager.send_fd("/tmp/websocket.sock", fd)
    if not ok then
        ngx.log(ngx.ERR, "failed to send fd: ", send_err)
        return
    end

    ngx.log(ngx.INFO, "fd sent successfully")

    local replaced, repl_err = fd_manager.replace_with_dummy(fd)
    if not replaced then
        ngx.log(ngx.ERR, "failed to replace fd with dummy socket: ", repl_err)
        return
    end

    ngx.log(ngx.INFO, "fd replaced with dummy successfully")
end

return M
