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
-- Grab the first arg from the "arg" global table (i.e., what was passed in by
-- Gopherbot for a plugin command).
--------------------------------------------------------------------------------
local cmd = ""
if #arg > 0 then
    cmd = arg[1]
end

--------------------------------------------------------------------------------
-- Provide "configure" command
--------------------------------------------------------------------------------
if cmd == "init" then
    return taskNormal
elseif cmd == "configure" then
    return defaultConfig
end

--------------------------------------------------------------------------------
-- The rest of the commands
--------------------------------------------------------------------------------

if cmd == "lua" then
    -- Echo a random message from the config's "Replies" plus some environment checks
    local ret = robot:ReplyThread("Hello from Lua in a thread!", fmtFixed)
    robot:Say("My home is: " .. (os.getenv("GOPHER_HOME") or "unknown"))
    robot:Say("My plugin name is: " .. (arg[0] or "unknown"))

    -- Try direct message
    local directBot = robot:Direct()
    directBot:Say("Hi from a DM; your name is " .. robot.user)

    -- Try reading an array from config
    local configData = robot:GetTaskConfig()
    if configData["Replies"] then
        local reply = robot:RandomString(configData["Replies"])
        robot:Say("Random reply: " .. reply)
    end

    if ret == retOk then
        return taskNormal
    else
        return taskFail
    end

elseif cmd == "thread" then
    -- Demonstrate replying in a new thread
    local ret = robot:ReplyThread("Ok, let's chat here in a new thread")
    robot:SayThread("... note that you still have to mention me by name for now.")
    return taskNormal

elseif cmd == "askthread" then
    -- Prompt for user input in a thread
    local rep, rcode = robot:PromptThreadForReply("SimpleString", "Tell me something - anything!")
    if rcode == retOk then
        robot:SayThread("I hear what you're saying: '" .. rep .. "'")
    else
        robot:SayThread("Sorry, I'm not sure what you're trying to tell me. Maybe you used funny characters?")
    end
    return taskNormal

elseif cmd == "listen" then
    -- Demonstrate a DM-based prompt
    local dbot = robot:Direct()
    local rep, rcode = dbot:PromptForReply("SimpleString", "Ok, what do you want to tell me?")
    if rcode == retOk then
        dbot:Say("I hear what you're saying: '" .. rep .. "'")
    else
        robot:Say("Sorry, I'm not sure I caught that. Maybe you used funny characters?")
    end
    return taskNormal

elseif cmd == "remember" then
    -- ARGV: [ "slowly"? ], [ thing to remember ]
    -- Arg #2 might be "slowly"
    local speed = arg[2]
    local thing = arg[3]

    -- Check out "memory" read/write
    local retVal, data, token = robot:CheckoutDatum("memory", true)
    if retVal ~= retOk then
        robot:Say("Sorry, I'm having trouble checking out my memory.")
        return taskFail
    end

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
        robot:Say("That's already one of my fondest memories.")
        robot:CheckinDatum("memory", token)
    else
        if not data then
            data = {}
        end
        table.insert(data, thing)
        if speed == "slowly" then
            robot:Say("Ok, I'll remember \"" .. thing .. "\" ... but sloooowly")
            robot:Pause(4)
        else
            robot:Say("Ok, I'll remember \"" .. thing .. "\"")
        end
        local updRet = robot:UpdateDatum("memory", token, data)
        if updRet == retOk then
            if speed ~= "slowly" then
                robot:Say("committed to memory")
            end
        else
            if speed ~= "slowly" then
                robot:Say("Dang it, having problems with my memory")
            end
        end
    end
    return taskNormal

elseif cmd == "recall" then
    local which = arg[2] -- possibly a number
    local retVal, data, token = robot:CheckoutDatum("memory", false)
    if retVal ~= retOk then
        robot:Say("Sorry - trouble checking memory!")
        return taskFail
    end
    if data and #data > 0 then
        if which and which:len() > 0 then
            local idx = tonumber(which)
            if not idx or idx < 1 then
                robot:Say("I can't make out what you want me to recall.")
                robot:CheckinDatum("memory", token)
                return taskNormal
            end
            if idx > #data then
                robot:Say("I don't remember that many things!")
                robot:CheckinDatum("memory", token)
                return taskNormal
            end
            local item = data[idx]
            robot:CheckinDatum("memory", token)
            robot:Say(item)
        else
            -- If no index, list them all
            local reply = "Here's what I remember:\n"
            for i, mem in ipairs(data) do
                reply = reply .. i .. ": " .. mem .. "\n"
            end
            robot:CheckinDatum("memory", token)
            robot:Say(reply)
        end
    else
        robot:CheckinDatum("memory", token)
        robot:Say("Sorry - I don't remember anything!")
    end
    return taskNormal

elseif cmd == "forget" then
    local which = arg[2]
    local i = tonumber(which) or 0
    if i < 1 then
        robot:Say("I can't make out what you want me to forget.")
        return taskNormal
    end
    i = i - 1  -- zero-based index

    local retVal, data, token = robot:CheckoutDatum("memory", true)
    if retVal ~= retOk then
        robot:Say("Sorry - trouble checking memory!")
        return taskFail
    end

    if data and #data > 0 and data[i+1] then
        local item = data[i+1]
        robot:Say("Ok, I'll forget \"" .. item .. "\"")
        table.remove(data, i+1)
        local updRet = robot:UpdateDatum("memory", token, data)
        if updRet ~= retOk then
            robot:Say("Hmm, having trouble forgetting that item for real, sorry.")
        end
    else
        robot:CheckinDatum("memory", token)
        robot:Say("Gosh, I guess I never remembered that in the first place!")
    end
    return taskNormal

elseif cmd == "check" then
    local isAdmin = robot:CheckAdmin()
    if isAdmin then
        robot:Say("It looks like you're an administrator.")
    else
        robot:Say("Well, you're not an administrator.")
    end
    robot:Pause(1)
    robot:Say("Now I'll request elevation...")

    local success = robot:Elevate(true)
    if success then
        robot:Say("Everything looks good, mac!")
    else
        robot:Say("You failed to elevate, I'm calling the cops!")
    end
    robot:Log(logInfo, "Checked out " .. robot.user .. ", admin: " .. tostring(isAdmin) .. ", elevate check: " .. tostring(success))
    return taskNormal
end

-- If we reached this point, no recognized command => do nothing special.
return taskNormal
