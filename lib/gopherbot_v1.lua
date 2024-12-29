-- gopherbot_constants.lua
-- This module defines constants from the Go 'robot' package for use in Lua scripts.

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
    TaskNotFound = 23,                -- robot.TaskNotFound
    MissingArguments = 24,            -- robot.MissingArguments
    InvalidStage = 25,                -- robot.InvalidStage
    InvalidTaskType = 26,             -- robot.InvalidTaskType
    CommandNotMatched = 27,           -- robot.CommandNotMatched
    TaskDisabled = 28,                -- robot.TaskDisabled
    PrivilegeViolation = 29,          -- robot.PrivilegeViolation

    -- General Failure
    Failed = 63,                       -- robot.Failed
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
function robot.New(bot)
    gbot = bot or GBOT
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

    -- For the "Send*" methods, we still just proxy to the Go methods directly:
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

    -- Robot modifier methods
    function newBot:Direct()
        dbot = self.gbot:Direct()
        return robot.New(dbot)
    end

    function newBot:Fixed()
        fbot = self.gbot:Fixed()
        return robot.New(fbot)
    end

    function newBot:Threaded()
        tbot = self.gbot:Threaded()
        return robot.New(tbot)
    end

    function newBot:MessageFormat(fmt)
        mbot = self.gbot:MessageFormat(fmt)
        return robot.New(mbot)
    end

    -- Prompting methods

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

    -- Add a Pause method - call the Go API
    function newBot:Pause(seconds)
        self.gbot:Pause(seconds)
    end

    return newBot
end

-- Create a function to return ret, task, log, fmt, proto, and the new robot
local function getExports()
  return robot, ret, task, log, fmt, proto
end

-- Return the function
return getExports
