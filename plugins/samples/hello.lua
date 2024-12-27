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

-- Require the robot and all constants
gopherbot_v1 = require "gopherbot_v1"
robot, ret, task, log, fmt, proto = gopherbot_v1()

local cmd = arg[1] or ""
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
    local bot = robot:New()

    local commandFunc = commands[cmd]
    if commandFunc then
        return commandFunc(bot)
    else
        bot:Log(log.Error,"Lua plugin received unknown command: "..tostring(cmd))
        return task.Fail
    end
end
