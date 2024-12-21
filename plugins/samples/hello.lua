-- hello.lua
-- A trivial script that says hello and returns taskNormal,
-- unless robot:Say fails (returning something other than retOk).

-- If we have at least one argument, check if it's "init" or "configure".
if #args > 0 then
    local cmd = args[1]
    if cmd == "init" or cmd == "configure" then
        -- Return taskNormal immediately.
        return taskNormal
    end
end

-- Call robot:Say and check the return code
local ret = robot:Say("Hello, world from Lua!")
if ret == retOk then
    return taskNormal
else
    return taskFail
end
