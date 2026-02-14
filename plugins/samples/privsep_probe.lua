-- privsep_probe.lua - manual privilege-separation probe for built-in Lua plugins.
--
-- Sample robot.yaml entries:
--
-- ExternalPlugins:
--   "privsep-lua-priv":
--     Description: Manual privsep probe (built-in lua, privileged)
--     Path: plugins/samples/privsep_probe.lua
--     Privileged: true
--   "privsep-lua-unpriv":
--     Description: Manual privsep probe (built-in lua, unprivileged)
--     Path: plugins/samples/privsep_probe.lua
--     Privileged: false
--
-- Sample conf/plugins overlays:
--   conf/plugins/privsep-lua-priv.yaml.sample
--   conf/plugins/privsep-lua-unpriv.yaml.sample
--
-- Usage:
--   - Enable one or both plugin entries above
--   - Copy the sample overlay(s) to .yaml, adjust channels if needed
--   - Keep this script non-executable so it runs via built-in Lua
--   - Run:
--       ;privsep lua priv
--       ;privsep lua unpriv

local gopherbot = require("gopherbot_v1")
local task = gopherbot.task
local Robot = gopherbot.Robot

local default_config = [[
---
Help:
- Keywords: [ "privsep", "lua" ]
  Helptext:
  - "(bot), privsep lua priv - run built-in lua privsep probe in #general"
  - "(bot), privsep lua unpriv - run built-in lua privsep probe in #random"
CommandMatchers: []
]]

local sensitive_keys = {
  "GOPHER_ENCRYPTION_KEY",
  "GOPHER_DEPLOY_KEY",
  "GOPHER_HOST_KEYS",
}

local command = arg[1]
if command == "configure" then
  print(default_config)
  return task.Normal
end

if command == "probe" then
  local bot = Robot:new()
  local home = os.getenv("GOPHER_HOME") or ""
  local env_path = ".env"
  if home ~= "" then
    env_path = home .. "/.env"
  end

  local envread = "deny"
  local errdetail = ""
  local f, ferr = io.open(env_path, "r")
  if f then
    local first = f:read("*l") or ""
    envread = "ok(" .. tostring(string.len(first)) .. "b)"
    f:close()
  else
    errdetail = tostring(ferr)
  end

  local parts = {
    "probe=builtin-lua",
    "channel=" .. tostring(bot.channel),
    "user=" .. tostring(bot.user),
    "envread=" .. envread,
  }
  if errdetail ~= "" then
    table.insert(parts, "error=" .. errdetail)
  end

  for _, key in ipairs(sensitive_keys) do
    local value = os.getenv(key)
    if value == nil or value == "" then
      table.insert(parts, key .. "=unset")
    else
      table.insert(parts, key .. "=set")
    end
  end

  bot:SayThread("PRIVSEP_PROBE_RESULT " .. table.concat(parts, " | "))
end

return task.Normal
