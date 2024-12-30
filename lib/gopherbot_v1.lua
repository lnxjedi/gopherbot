-- gopherbot_v1.lua
-- This module defines constants and robot/bot methods required for Lua extensions.

-- 1. RetVal (Robot method return values)
local ret = {
    -- Connector Issues
    Ok = 0,                          -- robot.Ok
    UserNotFound = 1,                -- robot.UserNotFound
    ChannelNotFound = 2,             -- robot.ChannelNotFound
    AttributeNotFound = 3,           -- robot.AttributeNotFound
    FailedMessageSend = 4,           -- robot.FailedMessageSend
    FailedChannelJoin = 5,           -- robot.FailedChannelJoin

    -- Brain Maladies
    DatumNotFound = 6,               -- robot.DatumNotFound
    DatumLockExpired = 7,            -- robot.DatumLockExpired
    DataFormatError = 8,             -- robot.DataFormatError
    BrainFailed = 9,                 -- robot.BrainFailed
    InvalidDatumKey = 10,            -- robot.InvalidDatumKey

    -- GetTaskConfig
    InvalidConfigPointer = 11,       -- robot.InvalidConfigPointer
    ConfigUnmarshalError = 12,       -- robot.ConfigUnmarshalError
    NoConfigFound = 13,              -- robot.NoConfigFound

    -- PromptForReply
    RetryPrompt = 14,                -- robot.RetryPrompt
    ReplyNotMatched = 15,            -- robot.ReplyNotMatched
    UseDefaultValue = 16,            -- robot.UseDefaultValue
    TimeoutExpired = 17,             -- robot.TimeoutExpired
    Interrupted = 18,                -- robot.Interrupted
    MatcherNotFound = 19,            -- robot.MatcherNotFound

    -- Email
    NoUserEmail = 20,                -- robot.NoUserEmail
    NoBotEmail = 21,                 -- robot.NoBotEmail
    MailError = 22,                  -- robot.MailError

    -- Pipeline Errors
    TaskNotFound = 23,               -- robot.TaskNotFound
    MissingArguments = 24,           -- robot.MissingArguments
    InvalidStage = 25,               -- robot.InvalidStage
    InvalidTaskType = 26,            -- robot.InvalidTaskType
    CommandNotMatched = 27,          -- robot.CommandNotMatched
    TaskDisabled = 28,               -- robot.TaskDisabled
    PrivilegeViolation = 29,         -- robot.PrivilegeViolation

    -- General Failure
    Failed = 63,                     -- robot.Failed
}

-- Add a string method to ret
function ret:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownRetVal"
end

-- 2. TaskRetVal (Script return values)
local task = {
    Normal = 0,                 -- robot.Normal
    Fail = 1,                   -- robot.Fail
    MechanismFail = 2,          -- robot.MechanismFail
    ConfigurationError = 3,     -- robot.ConfigurationError
    PipelineAborted = 4,        -- robot.PipelineAborted
    RobotStopping = 5,          -- robot.RobotStopping
    NotFound = 6,               -- robot.NotFound
    Success = 7,                -- robot.Success
}

-- Add a string method to task
function task:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownTaskRetVal"
end

-- 3. LogLevel
local log = {
    Trace = 0,  -- robot.Trace
    Debug = 1,  -- robot.Debug
    Info = 2,   -- robot.Info
    Audit = 3,  -- robot.Audit
    Warn = 4,   -- robot.Warn
    Error = 5,  -- robot.Error
    Fatal = 6,  -- robot.Fatal
}

-- Add a string method to log
function log:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownLogLevel"
end

-- 4. MessageFormat
local fmt = {
    Raw = 0,      -- robot.Raw
    Fixed = 1,    -- robot.Fixed
    Variable = 2, -- robot.Variable
}

-- Add a string method to fmt
function fmt:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownMessageFormat"
end

-- 5. Protocol
local proto = {
    Slack = 0,     -- robot.Slack
    Rocket = 1,    -- robot.Rocket
    Terminal = 2,  -- robot.Terminal
    Test = 3,      -- robot.Test
    Null = 4,      -- robot.Null
}

-- Add a string method to proto
function proto:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownProtocol"
end

local robot = {}
function robot.New(...)
    local args = {...}
    local bot
    -- Go easy on script authors, allow robot:New() or robot.New()
    if type(args[1]) == "table" then -- called as robot:New()
        bot = args[2]
    else
        bot = args[1]
    end
    local gbot = bot or GBOT
    local newBot = {}
    newBot.gbot = gbot -- Keep a reference to the original Go gbot object
    newBot.user = gbot.user
    newBot.user_id = gbot.user_id
    newBot.channel = gbot.channel
    newBot.channel_id = gbot.channel_id
    newBot.thread_id = gbot.thread_id
    newBot.message_id = gbot.message_id
    newBot.protocol = gbot.protocol
    newBot.brain = gbot.brain
    newBot.type = "native"
    newBot.threaded_message = gbot.threaded_message

    -- Send*/Say*/Reply* Methods
    function newBot:SendChannelMessage(channel, message, format)
        return self.gbot:SendChannelMessage(channel, message, format)
    end

    function newBot:SendChannelThreadMessage(channel, thread, message, format)
        return self.gbot:SendChannelThreadMessage(channel, thread, message, format)
    end

    function newBot:SendUserMessage(user, message, format)
        return self.gbot:SendUserMessage(user, message, format)
    end

    function newBot:SendUserChannelMessage(user, channel, message, format)
        return self.gbot:SendUserChannelMessage(user, channel, message, format)
    end

    function newBot:SendUserChannelThreadMessage(user, channel, thread, message, format)
        return self.gbot:SendUserChannelThreadMessage(user, channel, thread, message, format)
    end

    function newBot:Say(message, format)
        -- Let the Go code handle whether channel is empty => DM, or channel => public message
        return self.gbot:Say(message, format)
    end

    function newBot:SayThread(message, format)
        return self.gbot:SayThread(message, format)
    end

    function newBot:Reply(message, format)
        return self.gbot:Reply(message, format)
    end

    function newBot:ReplyThread(message, format)
        return self.gbot:ReplyThread(message, format)
    end

    -- Robot Modifier Methods
    function newBot:Direct()
        local dbot = self.gbot:Direct()
        return robot.New(dbot)
    end

    function newBot:Fixed()
        local fbot = self.gbot:Fixed()
        return robot.New(fbot)
    end

    function newBot:Threaded()
        local tbot = self.gbot:Threaded()
        return robot.New(tbot)
    end

    function newBot:MessageFormat(fmt)
        local mbot = self.gbot:MessageFormat(fmt)
        return robot.New(mbot)
    end

    -- Prompting Methods
    function newBot:PromptForReply(regex_id, prompt, format)
        return self.gbot:PromptForReply(regex_id, prompt, format)
    end

    function newBot:PromptThreadForReply(regex_id, prompt, format)
        return self.gbot:PromptThreadForReply(regex_id, prompt, format)
    end

    function newBot:PromptUserForReply(regex_id, user, prompt, format)
        return self.gbot:PromptUserForReply(regex_id, user, prompt, format)
    end

    function newBot:PromptUserChannelForReply(regex_id, user, channel, prompt, format)
        return self.gbot:PromptUserChannelForReply(regex_id, user, channel, prompt, format)
    end

    function newBot:PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format)
        return self.gbot:PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format)
    end

    -- -------------------------------------------------------------------
    -- 6. Short Term Memory Methods
    -- -------------------------------------------------------------------

    -- bot:Remember(key, value, shared)
    function newBot:Remember(key, value, shared)
        return self.gbot:Remember(key, value, shared)
    end

    -- bot:RememberThread(key, value, shared)
    function newBot:RememberThread(key, value, shared)
        return self.gbot:RememberThread(key, value, shared)
    end

    -- bot:RememberContext(context, value)
    function newBot:RememberContext(context, value)
        return self.gbot:RememberContext(context, value)
    end

    -- bot:RememberContextThread(context, value)
    function newBot:RememberContextThread(context, value)
        return self.gbot:RememberContextThread(context, value)
    end

    -- bot:Recall(key, shared) -> string
    function newBot:Recall(key, shared)
        return self.gbot:Recall(key, shared)
    end

    -- -------------------------------------------------------------------
    -- 7. Pipeline Methods
    -- -------------------------------------------------------------------

    -- bot:GetParameter(name) -> string
    function newBot:GetParameter(name)
        return self.gbot:GetParameter(name)
    end

    -- bot:SetParameter(name, value) -> bool
    function newBot:SetParameter(name, value)
        return self.gbot:SetParameter(name, value)
    end

    -- bot:Exclusive(tag, queueTask) -> bool
    function newBot:Exclusive(tag, queueTask)
        return self.gbot:Exclusive(tag, queueTask)
    end

    -- bot:SpawnJob(name, arg1, arg2, ...) -> RetVal
    function newBot:SpawnJob(name, ...)
        return self.gbot:SpawnJob(name, ...)
    end

    -- bot:AddTask(name, arg1, arg2, ...) -> RetVal
    function newBot:AddTask(name, ...)
        return self.gbot:AddTask(name, ...)
    end

    -- bot:FinalTask(name, arg1, arg2, ...) -> RetVal
    function newBot:FinalTask(name, ...)
        return self.gbot:FinalTask(name, ...)
    end

    -- bot:FailTask(name, arg1, arg2, ...) -> RetVal
    function newBot:FailTask(name, ...)
        return self.gbot:FailTask(name, ...)
    end

    -- bot:AddJob(name, arg1, arg2, ...) -> RetVal
    function newBot:AddJob(name, ...)
        return self.gbot:AddJob(name, ...)
    end

    -- bot:AddCommand(pluginName, command) -> RetVal
    function newBot:AddCommand(pluginName, command)
        return self.gbot:AddCommand(pluginName, command)
    end

    -- bot:FinalCommand(pluginName, command) -> RetVal
    function newBot:FinalCommand(pluginName, command)
        return self.gbot:FinalCommand(pluginName, command)
    end

    -- bot:FailCommand(pluginName, command) -> RetVal
    function newBot:FailCommand(pluginName, command)
        return self.gbot:FailCommand(pluginName, command)
    end

    -- -------------------------------------------------------------------
    -- 8. Attribute Methods
    -- -------------------------------------------------------------------

    -- bot:GetBotAttribute(attr) -> (stringVal, retVal)
    function newBot:GetBotAttribute(attr)
        return self.gbot:GetBotAttribute(attr)
    end

    -- bot:GetUserAttribute(user, attr) -> (stringVal, retVal)
    function newBot:GetUserAttribute(user, attr)
        return self.gbot:GetUserAttribute(user, attr)
    end

    -- bot:GetSenderAttribute(attr) -> (stringVal, retVal)
    function newBot:GetSenderAttribute(attr)
        return self.gbot:GetSenderAttribute(attr)
    end

    -- -------------------------------------------------------------------
    -- 9. Long-Term Memory Methods
    -- -------------------------------------------------------------------

    -- bot:CheckoutDatum(key, rw) -> memory table, retVal
    function newBot:CheckoutDatum(key, rw)
        -- Call the underlying Go method
        local retVal, datum, token = self.gbot:CheckoutDatum(key, rw)

        -- Determine if the datum exists based on the return value and presence of data
        local exists = (retVal == ret.Ok and datum ~= nil)

        -- Create the memory table to return to Lua
        local memory = {
            key = key,
            exists = exists,
            datum = datum,       -- Initialize as empty table if nil
            token = token or "",       -- Initialize as empty string if nil
            retVal = retVal
        }

        return memory, retVal
    end

    -- bot:UpdateDatum(memory) -> retVal
    function newBot:UpdateDatum(memory)
        -- Validate that the memory table contains the necessary fields
        if not memory or not memory.key or not memory.token then
            error("UpdateDatum requires a memory table with 'key' and 'token' fields")
        end

        -- Call the underlying Go method with extracted fields
        local retVal = self.gbot:UpdateDatum(memory.key, memory.token, memory.datum)

        return retVal
    end

    -- bot:CheckinDatum(memory) -> retVal
    function newBot:CheckinDatum(memory)
        -- Validate that the memory table contains the necessary fields
        if not memory or not memory.key or not memory.token then
            error("CheckinDatum requires a memory table with 'key' and 'token' fields")
        end

        -- Call the underlying Go method with extracted fields
        local retVal = self.gbot:CheckinDatum(memory.key, memory.token)

        return retVal
end

    -- -------------------------------------------------------------------
    -- 10. Other Methods
    -- -------------------------------------------------------------------

    -- bot:GetTaskConfig() -> (table, retVal)
    function newBot:GetTaskConfig()
        return self.gbot:GetTaskConfig()
    end

    -- bot:RandomInt(n) -> number
    function newBot:RandomInt(n)
        return self.gbot:RandomInt(n)
    end

    -- bot:RandomString(array) -> string
    function newBot:RandomString(array)
        return self.gbot:RandomString(array)
    end

    -- bot:Pause(seconds) -> no return
    function newBot:Pause(seconds)
        return self.gbot:Pause(seconds)
    end

    -- bot:CheckAdmin() -> bool
    function newBot:CheckAdmin()
        return self.gbot:CheckAdmin()
    end

    -- bot:Elevate(immediate) -> bool
    function newBot:Elevate(immediate)
        return self.gbot:Elevate(immediate)
    end

    -- bot:Log(level, message) -> no return
    function newBot:Log(level, message)
        return self.gbot:Log(level, message)
    end

    return newBot
end

-- Create a function to return ret, task, log, fmt, proto, and the new robot
local function getExports()
  return robot, ret, task, log, fmt, proto
end

-- Return the function
return getExports
