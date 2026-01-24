-- luafull.lua
-- Comprehensive Lua integration test plugin (messaging + config + RandomString).

local defaultConfig = [[
---
Help:
- Keywords: [ "say" ]
  Helptext:
  - "(bot), say everything - full test of Say*/Reply*/Send* methods"
- Keywords: [ "config" ]
  Helptext:
  - "(bot), lua-config - exercise GetTaskConfig + RandomString"
- Keywords: [ "http" ]
  Helptext:
  - "(bot), lua-http - exercise HTTP GET/POST/PUT"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:lua-config)
  Command: configtest
- Regex: (?i:lua-http)
  Command: http
AllowedHiddenCommands:
- sendmsg
Config:
  Openings:
  - "Not completely random 1"
  - "Not completely random 2"
]]

local gopherbot = require "gopherbot_v1"
local ret, task, log, fmt, proto, Robot =
    gopherbot.ret,
    gopherbot.task,
    gopherbot.log,
    gopherbot.fmt,
    gopherbot.proto,
    gopherbot.Robot

local commands = {}

function commands.sendmsg(bot)
  bot:Say("Regular Say")
  bot:SayThread("SayThread, yeah")
  bot:Reply("Regular Reply")
  bot:ReplyThread("Reply in thread, yo")
  bot:SendChannelMessage(bot.channel, "Sending to the channel: " .. bot.channel)
  bot:SendUserMessage(bot.user, "Sending this message to user: " .. bot.user)
  bot:SendUserChannelMessage(bot.user, bot.channel, "Sending to user '" .. bot.user .. "' in channel: " .. bot.channel)
  bot:SendChannelThreadMessage(bot.channel, bot.thread_id, "Sending to channel '" .. bot.channel .. "' in thread: " .. bot.thread_id)
  bot:SendUserChannelThreadMessage(bot.user, bot.channel, bot.thread_id, "Sending to user '" .. bot.user .. "' in channel '" .. bot.channel .. "' in thread: " .. bot.thread_id)
  return task.Normal
end

function commands.configtest(bot)
  local cfg, retVal = bot:GetTaskConfig()
  if retVal ~= ret.Ok then
    bot:Say("No config available")
    return task.Fail
  end
  local openings = cfg.Openings or {}
  bot:Say(bot:RandomString(openings))
  return task.Normal
end

function commands.http(bot)
  local cfg, retVal = bot:GetTaskConfig()
  if retVal ~= ret.Ok then
    bot:Say("No config available")
    return task.Fail
  end
  local baseURL = cfg.HttpBaseURL
  if not baseURL or baseURL == "" then
    bot:Say("No http base url configured")
    return task.Fail
  end
  local http = require("gopherbot_http")
  local client, err = http.create_client{
    base_url = baseURL,
    timeout_ms = 5000,
    throw_on_http_error = true,
  }
  if err then
    bot:Say("HTTP client error")
    return task.Fail
  end
  local getRes, err = client:get_json("/json/get")
  if err then
    bot:Say("HTTP GET failed")
    return task.Fail
  end
  bot:Say("HTTP GET ok: " .. tostring(getRes.method))
  local postRes, err = client:post_json("/json/post", { value = "alpha" })
  if err then
    bot:Say("HTTP POST failed")
    return task.Fail
  end
  bot:Say("HTTP POST ok: " .. tostring(postRes.value))
  local putRes, err = client:put_json("/json/put", { value = "bravo" })
  if err then
    bot:Say("HTTP PUT failed")
    return task.Fail
  end
  bot:Say("HTTP PUT ok: " .. tostring(putRes.value))
  local _, err = client:get_json("/json/error")
  if err and err.status then
    bot:Say("HTTP ERROR ok: " .. tostring(err.status))
  else
    bot:Say("HTTP ERROR unexpected")
  end
  local _, err = client:get_json("/json/slow", { timeout_ms = 50 })
  if err then
    bot:Say("HTTP TIMEOUT ok")
  else
    bot:Say("HTTP TIMEOUT unexpected")
  end
  return task.Normal
end

local cmd = arg and arg[1] or ""

if cmd == "init" then
  return task.Normal
elseif cmd == "configure" then
  return defaultConfig
else
  local bot = Robot:new()
  local handler = commands[cmd]
  if handler then
    return handler(bot)
  else
    bot:Log(log.Error, "Lua plugin received unknown command: " .. tostring(cmd))
    return task.Fail
  end
end
