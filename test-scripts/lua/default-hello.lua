local gopherbot = require "gopherbot_v1"

local task = gopherbot.task
local Robot = gopherbot.Robot

local default_config = [[
---
Commands:
  - Regex: (?i:lua default hello)
    Command: hello
]]

local cmd = arg and arg[1] or ""

if cmd == "_configure" then
  return default_config
end

if cmd == "_init" then
  return task.Normal
end

local bot = Robot:new()

if cmd == "hello" then
  local bot_name = bot:GetBotAttribute("fullName")
  local sender = bot:GetSenderAttribute("fullName")
  bot:Say("Lua says hello from " .. tostring(bot_name) .. " to " .. tostring(sender) .. " in #" .. tostring(bot.channel) .. ".")
  bot:Reply("This script only needs the installed default fixture.")
  return task.Normal
end

bot:Log(gopherbot.log.Error, "unknown Lua default command: " .. tostring(cmd))
return task.Fail
