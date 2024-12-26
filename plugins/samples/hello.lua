-- hello.lua
-- Lua plugin "Hello World" and boilerplate

-- By convention, plugins provide their own help and regular expressions
-- for matching commands. Other configuration, like which channel a plugin
-- is active in, is normally configured by the individual robot.
local defaultConfig = [[
---
Help:
- Keywords: [ "lua" ]
  Helptext: [ "(bot), hello lua - trigger lua hello world" ]
CommandMatchers:
- Regex: (?i:hello lua)
  Command: lua
]]

-- Require the constants module
ret, task, log, fmt, proto = require "gopherbot_constants" ()

local cmd = ""
if #arg > 0 then cmd = arg[1] end

-- Command dispatch table
local commands = {
    lua = function(bot)
        local retVal = bot:Say("Hello, Lua World!")
        if retVal == ret.Ok then
            return task.Normal
        else
            return task.Fail
        end
    end,
    -- Add more commands here
}

if cmd == "init" then
    return task.Normal
elseif cmd == "configure" then
    return defaultConfig
else
    -- robot isn't available during "configure", so we initialize bot here.
    local bot = robot:New()

    local commandFunc = commands[cmd]
    if commandFunc then
        return commandFunc(bot)
    else
        bot:Log(log.Error,"Lua plugin received unknown command: "..tostring(cmd))
        return task.Fail
    end
end
