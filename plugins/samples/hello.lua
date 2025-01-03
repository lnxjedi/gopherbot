--------------------------------------------------------------------------------
-- hello.lua
--
-- A "Hello World" plugin for Gopherbot in Lua, using idiomatic Lua structures:
--   - Local constants/variables for plugin config
--   - A command dispatch table
--   - An "init/configure" pattern plus dispatch for other commands
--   - EmmyLua-style annotations for a better IDE experience
--------------------------------------------------------------------------------

-- Load our gopherbot library module (gopherbot_v1.lua).
local gopherbot = require "gopherbot_v1"

local ret, task, log, fmt, proto, Robot =
    gopherbot.ret,   -- returns a table of API return-value constants
    gopherbot.task,  -- returns a table of task return-value constants
    gopherbot.log,   -- log levels
    gopherbot.fmt,   -- message formats
    gopherbot.proto, -- chat protocols
    gopherbot.Robot  -- the Robot class for Robot:new()

--- Default YAML configuration for this plugin
local defaultConfig = [[
---
Help:
  - Keywords: [ "lua" ]
    Helptext: [ "(bot), hello lua - trigger lua hello world" ]
CommandMatchers:
  - Regex: (?i:hello lua)
    Command: lua
]]

--------------------------------------------------------------------------------
-- Command Dispatch
--------------------------------------------------------------------------------

-- We define a Lua table mapping command names to handler functions.
-- Each handler receives a Robot instance and must return a `task.*` constant.
local commands = {}

--- Handler for the "lua" command.
---@param bot Robot
---@return number taskRetVal
function commands.lua(bot)
  local sendRet = bot:Say("Hello, Lua World!")
  if sendRet == ret.Ok then
    return task.Normal
  else
    bot:Log(log.Error, "Failed sending 'Hello, Lua World!'")
    return task.Fail
  end
end

--------------------------------------------------------------------------------
-- Main Plugin Logic
--------------------------------------------------------------------------------

-- Gopherbot calls this script with arg[1] set to a command like:
--   "init", "configure", or a user-defined command ("lua" in this case).
---@type string
local cmd = arg and arg[1] or ""

-- Handle init and configure first. If it’s not one of those, dispatch.
if cmd == "init" then
  -- Perform any plugin initialization (if needed).
  return task.Normal

elseif cmd == "configure" then
  -- Return our YAML config so Gopherbot can incorporate it.
  return defaultConfig

else
  -- For other commands, create a new Robot instance and dispatch.
  local bot = Robot:new()
  local handler = commands[cmd]

  if handler then
    -- Execute the corresponding handler function.
    return handler(bot)
  else
    -- Unknown command: log an error and fail the task.
    bot:Log(log.Error, "Lua plugin received unknown command: " .. tostring(cmd))
    return task.Fail
  end
end
