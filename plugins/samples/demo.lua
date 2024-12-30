-- demo.lua
-- A plugin for integration testing of Lua robot extensions, mirroring the Ruby demo.

local defaultConfig = [[
---
Help:
- Keywords: [ "lua" ]
  Helptext: [ "(bot), lua (me!) - prove that lua plugins work" ]
- Keywords: [ "listen" ]
  Helptext: [ "(bot), listen (to me?) - ask a question" ]
- Keywords: [ "thread" ]
  Helptext: [ "(bot), lua-thread - ask the robot to start a new thread" ]
- Keywords: [ "thread" ]
  Helptext: [ "(bot), lua-ask-thread - ask a question in a thread" ]
- Keywords: [ "remember", "memory" ]
  Helptext: [ "(bot), remember <anything> - prove the robot has a brain(tm)" ]
- Keywords: [ "recall", "memory" ]
  Helptext: [ "(bot), recall - list or recall a certain memory" ]
- Keywords: [ "forget", "memory" ]
  Helptext: [ "(bot), forget <#> - ask the robot to forget one of its remembered 'facts'" ]
- Keywords: [ "check" ]
  Helptext: [ "(bot), check me - get the bot to check you out" ]
CommandMatchers:
- Regex: (?i:lua( me)?!?)
  Command: lua
- Regex: (?i:lua-thread)
  Command: thread
- Regex: (?i:lua-ask-thread)
  Command: askthread
- Regex: (?i:listen( to me)?!?)
  Command: listen
- Regex: '(?i:remember(?: (slowly))? ([-\w .,!?:\/]+))'
  Command: remember
  Contexts: [ "", "item" ]
- Regex: (?i:recall ?([\d]+)?)
  Command: recall
- Regex: (?i:forget ([\d]{1,2}))
  Command: forget
- Regex: (?i:check me)
  Command: check
Config:
  Replies:
  - "Consider yourself lei'd"
  - "Waaaaaait a second... what do you mean by that?"
  - "I'll lua you, but not right now - I'll wait 'til you're least expecting it..."
  - "Crap, sorry - all out of lei"
]]

--------------------------------------------------------------------------------
-- Helper Functions
--------------------------------------------------------------------------------

local function extractEnv(envVarName)
  -- **Read /proc/self/environ and extract the specified environment variable**
  local envValue = "(not found)"
  local envFile = "/proc/self/environ"
  local envHandle = io.open(envFile, "r")
  if envHandle then
    local envContent = envHandle:read("*a")
    envHandle:close()
    -- Iterate over each key=value pair separated by null character
    for key, value in string.gmatch(envContent, "([^%z]+)=([^%z]*)") do
      if key == envVarName then
        envValue = value
        break
      end
    end
  else
    print("Failed to open " .. envFile)
  end
  return envValue
end

local function extractEnvBash(envVarName)
  -- **Execute 'bash -c "echo $envVarName"' and display its output**
  local envVarHandle = io.popen('bash -c "echo $' .. envVarName .. '"')
  local envVarOutput = "" -- Initialize the variable to prevent nil access
  if envVarHandle then
    local result = envVarHandle:read("*a")
    envVarHandle:close()
    -- Trim any trailing whitespace or newline characters
    envVarOutput = result:gsub("%s+", "")
  end
  if envVarOutput == "" then
    return "(not found)"
  end
  return envVarOutput
end

-- Require the constants module (concise form)
local gopherbot_v1 = require "gopherbot_v1"
local robot, ret, task, log, fmt, proto = require "gopherbot_v1" ()

local bot = robot.New()

--------------------------------------------------------------------------------
-- Command Functions
--------------------------------------------------------------------------------

local function luaCommand(bot)
  -- Start by replying in a thread with fixed formatting
  local retThread = bot:ReplyThread("Hello from Lua in a thread!", fmt.Fixed)

  -- Gather environment info
  local home = bot:GetParameter("GOPHER_HOME") or "unknown"
  local pluginName = arg[0] or "unknown"

  -- Call our attribute methods
  local user = bot.user
  local userID = bot.user_id
  local channel = bot.channel
  local channelID = bot.channel_id
  local threadID = bot.thread_id
  local isThreaded = bot.threaded_message

  -- **Execute the 'whoami' system command to get the current system user**
  local systemUser = "unknown"
  local handle = io.popen("whoami")
  if handle then
    local result = handle:read("*a")
    handle:close()
    -- Trim any trailing whitespace or newline characters
    systemUser = result:gsub("%s+", "")
  else
    bot:Say("Failed to execute 'whoami' command.")
  end

  -- Combine them into one line, including the system user
  bot:Say(string.format(
    "Home: %s | Plugin: %s | User: %s (%s) | Channel: %s (%s) | ThreadID: %s | Threaded: %s | System User: %s",
    home, pluginName, user, userID, channel, channelID, threadID,
    tostring(isThreaded), systemUser))

  -- **Read /proc/self/environ and extract GOPHER_CUSTOM_REPOSITORY**
  local gopherRepo = extractEnv("GOPHER_CUSTOM_REPOSITORY")

  -- **Say the GOPHER_CUSTOM_REPOSITORY value**
  bot:Say("GOPHER_CUSTOM_REPOSITORY is set to: " .. gopherRepo)

  local pathOutput = extractEnvBash("PATH")

  -- **Say the PATH value**
  bot:Say("bash PATH is set to: " .. pathOutput)

  -- **Execute 'bash -c "echo $GOPHER_CUSTOM_REPOSITORY"' and display its output**
  local gopherRepoOutput = extractEnvBash("GOPHER_CUSTOM_REPOSITORY")

  -- **Say the GOPHER_CUSTOM_REPOSITORY value from bash**
  bot:Say("GOPHER_CUSTOM_REPOSITORY (from bash) is set to: " ..
    gopherRepoOutput)

  -- Show the DM usage
  local directBot = bot:Direct()
  directBot:Say("Hi from a DM; your name is " .. user)

  -- Demonstrate GetSenderAttribute (e.g., "email")
  local senderEmail, senderRet = bot:GetSenderAttribute("email")
  if senderRet == ret.Ok and senderEmail ~= "" then
    bot:Say("I have your email attribute as: " .. senderEmail)
  end

  -- Demonstrate GetBotAttribute (e.g., "name")
  local botName, botRet = bot:GetBotAttribute("name")
  if botRet == ret.Ok and botName ~= "" then
    bot:Say("My bot name is: " .. botName)
  end

  -- Now try reading array from config
  local configData, retCfg = bot:GetTaskConfig()
  if retCfg ~= ret.Ok then
    bot:Say("I wasn't able to find my configuration")
  else
    if configData["Replies"] then
      local reply = bot:RandomString(configData["Replies"])
      bot:Say("Random reply: " .. reply)
    end
  end

  -- Decide final return based on config retrieval
  if retThread == ret.Ok then
    return task.Normal
  elseif retThread == ret.Failed then
    return task.Fail
  else
    return task.MechanismFail
  end
end

local function threadCommand(bot)
  -- Demonstrate replying in a new thread
  local retThread = bot:ReplyThread("Ok, let's chat here in a new thread")
  bot:SayThread("... note that you still have to mention me by name for now.")
  return task.Normal
end

local function askThreadCommand(bot)
  -- Prompt for user input in a thread
  bot:Say("I'm going to ask you a question...")
  local rep, rcode = bot:PromptThreadForReply("SimpleString",
    "Tell me something - anything!")
  if rcode == ret.Ok then
    bot:SayThread("I hear what you're saying: '" .. rep .. "'")
  else
    bot:SayThread(
      "Sorry, I'm not sure what you're trying to tell me. Maybe you used funny characters?")
  end
  return task.Normal
end

local function listenCommand(bot)
  -- Demonstrate a DM-based prompt
  local dbot = bot:Direct()
  local rep, rcode = dbot:PromptForReply("SimpleString",
    "Ok, what do you want to tell me?")
  if rcode == ret.Ok then
    dbot:Say("I hear what you're saying: '" .. rep .. "'")
  else
    bot:Say("Sorry, I'm not sure I caught that. Maybe you used funny characters?")
  end
  return task.Normal
end

local function rememberCommand(bot)
  -- ARGV: [ "slowly"? ], [ thing to remember ]
  -- Arg #2 might be "slowly"
  local speed = arg[2]
  local thing = arg[3]

  -- Check out "memory" read/write
  local memory, retVal = bot:CheckoutDatum("memory", true)
  if retVal ~= ret.Ok then
    bot:Say("Sorry, I'm having trouble checking out my memory.")
    return task.Fail
  end

  if not memory.exists then
    memory.datum = {}
  end

  local data = memory.datum

  local found = false
  if data and #data > 0 then
    -- data is an array. Let's see if we already have `thing`.
    for i, mem in ipairs(data) do
      if mem == thing then
        found = true
        break
      end
    end
  end

  if found then
    bot:Say("That's already one of my fondest memories.")
    bot:CheckinDatum(memory)
  else
    if not data then data = {} end
    table.insert(data, thing)
    if speed == "slowly" then
      bot:Say("Ok, I'll remember \"" .. thing .. "\" ... but sloooowly")
      bot:Pause(4)
    else
      bot:Say("Ok, I'll remember \"" .. thing .. "\"")
    end
    local updRet = bot:UpdateDatum(memory)
    if updRet == ret.Ok then
      if speed ~= "slowly" then bot:Say("committed to memory") end
    else
      if speed ~= "slowly" then
        bot:Say("Dang it, having problems with my memory")
      end
    end
  end
  return task.Normal
end

local function recallCommand(bot)
  local which = arg[2] -- possibly a number
  local memory, retVal = bot:CheckoutDatum("memory", false)
  if retVal ~= ret.Ok then
    bot:Say("Sorry - trouble checking memory!")
    return task.Fail
  end
  local data = memory.datum
  if data and #data > 0 then
    if which and #which > 0 then
      local idx = tonumber(which)
      if not idx or idx < 1 then
        bot:Say("I can't make out what you want me to recall.")
        bot:CheckinDatum(memory)
        return task.Normal
      end
      if idx > #data then
        bot:Say("I don't remember that many things!")
        bot:CheckinDatum(memory)
        return task.Normal
      end
      local item = data[idx]
      bot:CheckinDatum(memory)
      bot:Say(item)
    else
      -- If no index, list them all
      local reply = "Here's what I remember:\n"
      for i, mem in ipairs(data) do
        reply = reply .. i .. ": " .. mem .. "\n"
      end
      bot:CheckinDatum(memory)
      bot:Say(reply)
    end
  else
    bot:CheckinDatum(memory)
    bot:Say("Sorry - I don't remember anything!")
  end
  return task.Normal
end

local function forgetCommand(bot)
  local which = arg[2]
  local i = tonumber(which) or 0
  if i < 1 then
    bot:Say("I can't make out what you want me to forget.")
    return task.Normal
  end
  i = i - 1 -- zero-based index

  local memory, retVal = bot:CheckoutDatum("memory", true)
  if retVal ~= ret.Ok then
    bot:Say("Sorry - trouble checking memory!")
    return task.Fail
  end
  local data = memory.datum

  if data and #data > 0 and data[i + 1] then
    local item = data[i + 1]
    bot:Say("Ok, I'll forget \"" .. item .. "\"")
    table.remove(data, i + 1)
    local updRet = bot:UpdateDatum(memory)
    if updRet ~= ret.Ok then
      bot:Say("Hmm, having trouble forgetting that item for real, sorry.")
    end
  else
    bot:CheckinDatum(memory)
    bot:Say("Gosh, I guess I never remembered that in the first place!")
  end
  return task.Normal
end

local function checkCommand(bot)
  local isAdmin = bot:CheckAdmin()
  if isAdmin then
    bot:Say("It looks like you're an administrator.")
  else
    bot:Say("Well, you're not an administrator.")
  end
  bot:Pause(1)
  bot:Say("Now I'll request elevation...")

  local success = bot:Elevate(true)
  if success then
    bot:Say("Everything looks good, mac!")
  else
    bot:Say("You failed to elevate, I'm calling the cops!")
  end
  bot:Log(log.Info,
    "Checked out " .. bot:User() .. ", admin: " .. tostring(isAdmin) ..
      ", elevate check: " .. tostring(success))
  return task.Normal
end

--------------------------------------------------------------------------------
-- Command Dispatch Table
--------------------------------------------------------------------------------

local commands = {
  lua = luaCommand,
  thread = threadCommand,
  askthread = askThreadCommand,
  listen = listenCommand,
  remember = rememberCommand,
  recall = recallCommand,
  forget = forgetCommand,
  check = checkCommand,
}

--------------------------------------------------------------------------------
-- Main Execution
--------------------------------------------------------------------------------

local cmd = ""
if #arg > 0 then cmd = arg[1] end

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
    bot:Log(log.Error, "Lua plugin received unknown command: " .. tostring(cmd))
    return task.Fail
  end
end
