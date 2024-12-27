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
function robot.New()
    local newBot = {}
    newBot.BOT = BOT -- Keep a reference to the original Go BOT object
    newBot.user = BOT.user
    newBot.user_id = BOT.user_id
    newBot.channel = BOT.channel
    newBot.channel_id = BOT.channel_id
    newBot.thread_id = BOT.thread_id
    newBot.message_id = BOT.message_id
    newBot.protocol = BOT.protocol
    newBot.brain = BOT.brain
    newBot.threaded_message = BOT.threaded_message

    -- Send* methods (handling format with a formatted BOT object)

    function newBot:SendChannelMessage(channel, message, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        return fBOT:SendChannelMessage(channel, message)
    end

    function newBot:SendChannelThreadMessage(channel, thread, message, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        return fBOT:SendChannelThreadMessage(channel, thread, message)
    end

    function newBot:SendUserMessage(user, message, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        return fBOT:SendUserMessage(user, message)
    end

    function newBot:SendUserChannelMessage(user, channel, message, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        return fBOT:SendUserChannelMessage(user, channel, message)
    end

    function newBot:SendUserChannelThreadMessage(user, channel, thread, message, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        return fBOT:SendUserChannelThreadMessage(user, channel, thread, message)
    end

    -- Say, SayThread, Reply, and ReplyThread methods (convenience wrappers)

    function newBot:Say(message, format)
        if self.channel == "" then
            -- If channel is empty, send a user message
            return self:SendUserMessage(self.user, message, format)
        else
            -- Otherwise, send a channel/thread message
            local thread = ""
            if self.threaded_message then
                thread = self.thread_id
            end
            return self:SendChannelThreadMessage(self.channel, thread, message, format)
        end
    end

    function newBot:SayThread(message, format)
        if self.channel == "" then
            -- If channel is empty, send a user message
            return self:SendUserMessage(self.user, message, format)
        else
            -- Otherwise, send a channel/thread message with the current thread_id
            return self:SendChannelThreadMessage(self.channel, self.thread_id, message, format)
        end
    end

    function newBot:Reply(message, format)
        if self.channel == "" then
            -- If channel is empty, send a user message
            return self:SendUserMessage(self.user, message, format)
        else
            -- Otherwise, send a user/channel/thread message
            local thread = ""
            if self.threaded_message then
                thread = self.thread_id
            end
            return self:SendUserChannelThreadMessage(self.user, self.channel, thread, message, format)
        end
    end

    function newBot:ReplyThread(message, format)
        if self.channel == "" then
            -- If channel is empty, send a user message
            return self:SendUserMessage(self.user, message, format)
        else
            -- Otherwise, send a user/channel/thread message with the current thread_id
            return self:SendUserChannelThreadMessage(self.user, self.channel, self.thread_id, message, format)
        end
    end

    -- Prompting methods

    function newBot:PromptForReply(regex_id, prompt, format)
        local thread = ""
        if self.threaded_message then
            thread = self.thread_id
        end
        return self:PromptUserChannelThreadForReply(regex_id, self.user, self.channel, thread, prompt, format)
    end

    function newBot:PromptThreadForReply(regex_id, prompt, format)
        return self:PromptUserChannelThreadForReply(regex_id, self.user, self.channel, self.thread_id, prompt, format)
    end

    function newBot:PromptUserForReply(regex_id, prompt, format)
        return self:PromptUserChannelThreadForReply(regex_id, self.user, "", "", prompt, format)
    end

    function newBot:PromptUserChannelForReply(regex_id, prompt, format)
        return self:PromptUserChannelThreadForReply(regex_id, self.user, self.channel, "", prompt, format)
    end

    function newBot:PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format)
        local fBOT = self.BOT
        if format then
            fBOT = self.BOT:MessageFormat(format)
        end
        local args = { regex_id, user, channel, thread, prompt }
        local ret
        for i = 1, 3 do
            ret = fBOT:PromptUserChannelThreadForReply(unpack(args))
            if ret.RetVal ~= ret.RetryPrompt then
                return { Reply = ret.Reply, RetVal = ret.RetVal }
            end
        end
        if ret.RetVal == ret.RetryPrompt then
            return { Reply = ret.Reply, RetVal = ret.Interrupted }
        else
            return { Reply = ret.Reply, RetVal = ret.RetVal }
        end
    end

    -- Add a Pause method - call the Go API
    function newBot:Pause(seconds)
        self.BOT:Pause(seconds)
    end

    return newBot
end

-- Create a function to return ret, task, log, fmt, proto, and the new robot
local function getExports()
  return robot, ret, task, log, fmt, proto
end

-- Return the function
return getExports
