local gopherbot = require "gopherbot_v1"

local ret = gopherbot.ret
local task = gopherbot.task
local Robot = gopherbot.Robot

local default_config = [[
---
Commands:
  - Regex: (?i:lua local demo)(?: (.*))?
    Command: demo
  - Regex: (?i:lua local prompt)
    Command: prompt
  - Regex: (?i:lua local memory)
    Command: memory
  - Regex: (?i:lua local config)
    Command: config
]]

local handlers = {}

function handlers.demo(bot)
  local who = bot:GetSenderAttribute("fullName")
  local cat_color = bot:GetParameter("CAT_COLOR")
  bot:Say("Lua demo for " .. tostring(who) .. " with " .. tostring(cat_color) .. " cat syntax.")
  bot:Reply("Command arg[2] was " .. tostring(arg[2] or "<none>"))
  bot:Fixed():SendChannelMessage(bot.channel, "fixed-width channel message from Lua")
  return task.Normal
end

function handlers.prompt(bot)
  local name, prompt_ret = bot:PromptForReply("SimpleString", "Name of cat?")
  if prompt_ret ~= ret.Ok then
    bot:Say("Prompt failed: " .. ret:string(prompt_ret))
    return task.Fail
  end
  bot:Say("Lua heard cat name: " .. name)
  return task.Normal
end

function handlers.memory(bot)
  bot:Remember("last_cat", "Garfield", false)
  bot:Remember("team_cat", "Pixel", true)
  local local_cat = bot:Recall("last_cat", false)
  local team_cat = bot:Recall("team_cat", true)
  bot:Say("Lua memory local=" .. tostring(local_cat) .. " shared=" .. tostring(team_cat))

  local profile, checkout_ret = bot:CheckoutDatum("cat_profile", true)
  if checkout_ret == ret.Ok then
    if not profile.exists or type(profile.datum) ~= "table" then
      profile.datum = { name = "Garfield", snacks = {} }
    end
    profile.datum.name = "Garfield"
    profile.datum.snacks = profile.datum.snacks or {}
    table.insert(profile.datum.snacks, "lasagna")
    bot:UpdateDatum(profile)
  end
  return task.Normal
end

function handlers.config(bot)
  local cfg, cfg_ret = bot:GetTaskConfig()
  if cfg_ret ~= ret.Ok then
    bot:Say("Lua config unavailable: " .. ret:string(cfg_ret))
    return task.Fail
  end
  bot:Say("Lua config opening: " .. tostring(cfg.Openings[1]))
  bot:SetParameter("LUA_LOCAL_DEMO", "ok")
  bot:AddTask("next-task", "from-lua")
  return task.Normal
end

local cmd = arg and arg[1] or ""
if cmd == "_configure" then
  return default_config
end
if cmd == "_init" then
  return task.Normal
end

local bot = Robot:new()
local handler = handlers[cmd]
if not handler then
  bot:Log(gopherbot.log.Error, "unknown Lua local command: " .. tostring(cmd))
  return task.Fail
end
return handler(bot)
