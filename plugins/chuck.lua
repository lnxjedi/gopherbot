-- chuck.lua
-- Lua version of the Chuck Norris plugin using gluahttp.

local defaultConfig = [[
---
MessageMatchers:
  - Regex: (?i:\bchuck norris\b)
    Command: chuck
Config:
  Openings:
  - "Chuck Norris?!?! He's AWESOME!!!"
  - "Oh cool, you like Chuck Norris, too?"
  - "Speaking of Chuck Norris - "
  - "Hey, I know EVERYTHING about Chuck Norris!"
  - "I'm a HUUUUGE Chuck Norris fan!"
  - "Not meaning to eavesdrop or anything, but are we talking about CHUCK NORRIS ?!?"
  - "Oh yeah, Chuck Norris! The man, the myth, the legend."
]]

local gopherbot = require "gopherbot_v1"
local ret, task, log, fmt, proto, Robot =
    gopherbot.ret,
    gopherbot.task,
    gopherbot.log,
    gopherbot.fmt,
    gopherbot.proto,
    gopherbot.Robot

local function handleChuck(bot)
  local cfg, retVal = bot:GetTaskConfig()
  if retVal ~= ret.Ok then
    bot:Say("Uh-oh, I wasn't able to find any configuration")
    return task.Normal
  end
  local openings = cfg.Openings or {}
  local opening = bot:RandomString(openings)

  local http = require("http")
  local json = require("json")
  local response, err = http.get("https://api.chucknorris.io/jokes/random", {
    headers = { Accept = "application/json" },
    timeout = "10s",
  })
  if err then
    bot:Say("I tried to fetch a Chuck Norris joke but something broke.")
    return task.Normal
  end

  if response.status_code >= 400 then
    bot:Log(log.Error, "chuck.lua HTTP error: HTTP " .. tostring(response.status_code))
    bot:Say("I tried to fetch a Chuck Norris joke but something broke.")
    return task.Normal
  end

  local data, decodeErr = json.decode(response.body)
  if decodeErr then
    bot:Log(log.Error, "chuck.lua JSON error: " .. tostring(decodeErr))
    bot:Say("I tried to fetch a Chuck Norris joke but something broke.")
    return task.Normal
  end

  bot:Say(opening .. " Did you know ...?")
  bot:Pause(2)
  if data and data.value then
    bot:Say(data.value)
  else
    bot:Say("Chuck Norris is too awesome to describe right now.")
  end
  return task.Normal
end

local cmd = arg and arg[1] or ""

if cmd == "configure" then
  return defaultConfig
elseif cmd == "chuck" then
  local bot = Robot:new()
  return handleChuck(bot)
else
  return task.Fail
end
