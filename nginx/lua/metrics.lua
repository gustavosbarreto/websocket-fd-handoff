local prometheus = require("prometheus").init("prometheus_metrics")

local mem_gauge = prometheus:gauge(
    "nginx_lua_rss_kb", 
    "Resident Set Size (RSS) in KB for the Nginx worker process"
)

local heap_gauge = prometheus:gauge(
    "nginx_lua_heap_kb", 
    "Memory used by LuaJIT heap (collectgarbage)"
)

local function get_rss_kb()
    local f = io.open("/proc/self/status", "r")
    if not f then return nil end

    for line in f:lines() do
        local name, value = line:match("^(%S+):%s+(%d+)")
        if name == "VmRSS" then
            f:close()
            return tonumber(value)
        end
    end
    f:close()
    return nil
end

return function()
    local rss = get_rss_kb()
    if rss then
        mem_gauge:set(rss)
    end

    local heap = collectgarbage("count") -- KB
    heap_gauge:set(math.floor(heap))
end
