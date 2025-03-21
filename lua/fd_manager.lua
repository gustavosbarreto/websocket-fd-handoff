local ffi = require "ffi_defs"
local C = ffi.C

local M = {}

--- Sends a file descriptor over a Unix domain socket using SCM_RIGHTS
---@param path string
---@param fd number
---@return boolean, string|nil
function M.send_fd(path, fd)
    local sock = C.socket(C.AF_UNIX, C.SOCK_STREAM, 0)
    if sock < 0 then
        return false, "failed to create unix socket"
    end

    local addr = ffi.new("struct sockaddr_un")
    addr.sun_family = C.AF_UNIX
    ffi.copy(addr.sun_path, path, #path)

    if C.connect(sock, ffi.cast("struct sockaddr*", addr), ffi.sizeof(addr)) < 0 then
        local msg = ffi.string(C.strerror(ffi.errno()))
        C.close(sock)
        return false, "failed to connect to unix socket: " .. msg
    end

    local control_len = ffi.sizeof("struct cmsghdr") + ffi.sizeof("int")
    local control = ffi.new("char[?]", control_len)
    local cmsg = ffi.cast("struct cmsghdr*", control)
    cmsg.cmsg_len = control_len
    cmsg.cmsg_level = C.SOL_SOCKET
    cmsg.cmsg_type = C.SCM_RIGHTS

    ffi.cast("int*", control + ffi.sizeof("struct cmsghdr"))[0] = fd

    local buf = ffi.new("char[1]", "X")
    local iov = ffi.new("struct iovec[1]")
    iov[0].iov_base = buf
    iov[0].iov_len = 1

    local msg = ffi.new("struct msghdr")
    msg.msg_iov = iov
    msg.msg_iovlen = 1
    msg.msg_control = control
    msg.msg_controllen = control_len

    if C.sendmsg(sock, msg, 0) < 0 then
        C.close(sock)
        return false, "failed to send fd"
    end

    C.close(sock)
    return true
end

--- Replaces the given file descriptor with a dummy AF_INET socket
---@param fd number
---@return boolean, string|nil
function M.replace_with_dummy(fd)
    local dummy = C.socket(C.AF_INET, C.SOCK_STREAM, 0)
    if dummy == -1 then
        return false, "failed to create dummy socket: " .. ffi.errno()
    end

    if C.dup2(dummy, fd) == -1 then
        C.close(dummy)
        return false, "dup2 failed: " .. ffi.errno()
    end

    C.close(dummy)
    return true
end

return M
