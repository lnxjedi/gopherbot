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
- Keywords: [ "subscribe" ]
  Helptext:
  - "(bot), lua-subscribe - exercise Subscribe/Unsubscribe"
- Keywords: [ "prompt" ]
  Helptext:
  - "(bot), lua-prompts - exercise Prompt* methods (user/channel/thread variants)"
- Keywords: [ "memory" ]
  Helptext:
  - "(bot), lua-memory-seed/lua-memory-check/lua-memory-thread-check - exercise Remember*/Recall context behavior"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:lua-config)
  Command: configtest
- Regex: (?i:lua-http)
  Command: http
- Regex: (?i:lua-subscribe)
  Command: subscribe
- Regex: (?i:lua-prompts)
  Command: prompts
- Regex: (?i:lua-memory-seed)
  Command: memoryseed
- Regex: (?i:lua-memory-check)
  Command: memorycheck
- Regex: (?i:lua-memory-thread-check)
  Command: memorythreadcheck
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
local function showMemory(v)
  if v == nil or v == "" then
    return "<empty>"
  end
  return tostring(v)
end

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

function commands.subscribe(bot)
  local sub = bot:Subscribe()
  local unsub = bot:Unsubscribe()
  bot:Say("SUBSCRIBE FLOW: " .. tostring(sub) .. "/" .. tostring(unsub))
  return task.Normal
end

function commands.prompts(bot)
  local p1, r1 = bot:PromptForReply("SimpleString", "Codename check: pick a mission codename.")
  if r1 ~= ret.Ok then
    bot:Say("PROMPT FLOW FAILED 1:" .. tostring(r1))
    return task.Fail
  end
  local p2, r2 = bot:PromptThreadForReply("SimpleString", "Thread check: pick a favorite snack for launch.")
  if r2 ~= ret.Ok then
    bot:Say("PROMPT FLOW FAILED 2:" .. tostring(r2))
    return task.Fail
  end
  local p3, r3 = bot:PromptUserForReply("SimpleString", bot.user, "DM check: name a secret moon base.")
  if r3 ~= ret.Ok then
    bot:Say("PROMPT FLOW FAILED 3:" .. tostring(r3))
    return task.Fail
  end
  local p4, r4 = bot:PromptUserChannelForReply("SimpleString", bot.user, bot.channel, "Channel check: describe launch weather in two words.")
  if r4 ~= ret.Ok then
    bot:Say("PROMPT FLOW FAILED 4:" .. tostring(r4))
    return task.Fail
  end
  local p5, r5 = bot:PromptUserChannelThreadForReply("SimpleString", bot.user, bot.channel, bot.thread_id, "Thread rally: choose a backup call sign.")
  if r5 ~= ret.Ok then
    bot:Say("PROMPT FLOW FAILED 5:" .. tostring(r5))
    return task.Fail
  end
  bot:Say("PROMPT FLOW OK: " .. p1 .. " | " .. p2 .. " | " .. p3 .. " | " .. p4 .. " | " .. p5)
  return task.Normal
end

function commands.memoryseed(bot)
  bot:Remember("launch_snack", "saffron noodles", false)
  bot:Remember("launch_snack", "solar soup", true)
  bot:RememberContext("pad", "orbital-7")
  bot:RememberThread("thread_note", "delta thread", false)
  bot:RememberContextThread("mission", "aurora mission")
  bot:Say("MEMORY SEED: done")
  return task.Normal
end

function commands.memorycheck(bot)
  local localMem = bot:Recall("launch_snack", false)
  local sharedMem = bot:Recall("launch_snack", true)
  local ctx = bot:Recall("context:pad", false)
  local threadMem = bot:Recall("thread_note", false)
  local threadCtx = bot:Recall("context:mission", false)
  bot:Say("MEMORY CHECK: local=" .. showMemory(localMem) ..
    " shared=" .. showMemory(sharedMem) ..
    " ctx=" .. showMemory(ctx) ..
    " thread=" .. showMemory(threadMem) ..
    " threadctx=" .. showMemory(threadCtx))
  return task.Normal
end

function commands.memorythreadcheck(bot)
  local localMem = bot:Recall("launch_snack", false)
  local sharedMem = bot:Recall("launch_snack", true)
  local ctx = bot:Recall("context:pad", false)
  local threadMem = bot:Recall("thread_note", false)
  local threadCtx = bot:Recall("context:mission", false)
  bot:Say("MEMORY THREAD CHECK: local=" .. showMemory(localMem) ..
    " shared=" .. showMemory(sharedMem) ..
    " ctx=" .. showMemory(ctx) ..
    " thread=" .. showMemory(threadMem) ..
    " threadctx=" .. showMemory(threadCtx))
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
