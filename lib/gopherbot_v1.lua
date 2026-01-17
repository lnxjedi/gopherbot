--------------------------------------------------------------------------------
-- gopherbot_v1.lua
--
-- This module defines constants and a "Robot" class for Lua-based Gopherbot
-- plugins. It aims to provide an idiomatic Lua OOP style, plus EmmyLua-style
-- annotations for a better experience in VSCode or other Lua IDEs.
--------------------------------------------------------------------------------

---@class GopherbotExports
---@field ret table<string, number>   # Return value constants
---@field task table<string, number>  # Script/pipeline task return constants
---@field log table<string, number>   # Log level constants
---@field fmt table<string, number>   # Message format constants
---@field proto table<string, number> # Protocol constants
---@field Robot RobotClass           # The Robot "class" constructor

local M = {} -- Our module table

--------------------------------------------------------------------------------
-- 1. RetVal (Robot method return values)
--------------------------------------------------------------------------------

M.ret = {
    -- Connector Issues
    Ok = 0,
    UserNotFound = 1,
    ChannelNotFound = 2,
    AttributeNotFound = 3,
    FailedMessageSend = 4,
    FailedChannelJoin = 5,

    -- Brain Maladies
    DatumNotFound = 6,
    DatumLockExpired = 7,
    DataFormatError = 8,
    BrainFailed = 9,
    InvalidDatumKey = 10,

    -- GetTaskConfig
    InvalidConfigPointer = 11,
    ConfigUnmarshalError = 12,
    NoConfigFound = 13,

    -- PromptForReply
    RetryPrompt = 14,
    ReplyNotMatched = 15,
    UseDefaultValue = 16,
    TimeoutExpired = 17,
    Interrupted = 18,
    MatcherNotFound = 19,

    -- Email
    NoUserEmail = 20,
    NoBotEmail = 21,
    MailError = 22,

    -- Pipeline Errors
    TaskNotFound = 23,
    MissingArguments = 24,
    InvalidStage = 25,
    InvalidTaskType = 26,
    CommandNotMatched = 27,
    TaskDisabled = 28,
    PrivilegeViolation = 29,

    -- General Failure
    Failed = 63,
}

---Convert a ret constant to its string name.
---@param val number
---@return string
function M.ret:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownRetVal"
end

--------------------------------------------------------------------------------
-- 2. TaskRetVal (Script return values)
--------------------------------------------------------------------------------

M.task = {
    Normal = 0,
    Fail = 1,
    MechanismFail = 2,
    ConfigurationError = 3,
    PipelineAborted = 4,
    RobotStopping = 5,
    NotFound = 6,
    Success = 7,
}

---Convert a task constant to its string name.
---@param val number
---@return string
function M.task:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownTaskRetVal"
end

--------------------------------------------------------------------------------
-- 3. LogLevel
--------------------------------------------------------------------------------

M.log = {
    Trace = 0,
    Debug = 1,
    Info = 2,
    Audit = 3,
    Warn = 4,
    Error = 5,
    Fatal = 6,
}

---Convert a log constant to its string name.
---@param val number
---@return string
function M.log:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownLogLevel"
end

--------------------------------------------------------------------------------
-- 4. MessageFormat
--------------------------------------------------------------------------------

M.fmt = {
    Raw = 0,
    Fixed = 1,
    Variable = 2,
}

---Convert a fmt constant to its string name.
---@param val number
---@return string
function M.fmt:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownMessageFormat"
end

--------------------------------------------------------------------------------
-- 5. Protocol
--------------------------------------------------------------------------------

M.proto = {
    Slack = 0,
    Rocket = 1,
    Terminal = 2,
    Test = 3,
    Null = 4,
}

---Convert a protocol constant to its string name.
---@param val number
---@return string
function M.proto:string(val)
    for k, v in pairs(self) do
        if v == val then
            return k
        end
    end
    return "UnknownProtocol"
end

--------------------------------------------------------------------------------
-- Robot Class Definition
--------------------------------------------------------------------------------

---@class Robot
---@field gbot any            # Underlying Go "gbot" object
---@field user string
---@field user_id string
---@field channel string
---@field channel_id string
---@field thread_id string
---@field message_id string
---@field protocol number
---@field brain string|any
---@field type string
---@field threaded_message boolean
local Robot = {}
Robot.__index = Robot

--------------------------------------------------------------------------------
-- Constructor
--------------------------------------------------------------------------------

---Create a new Lua-level Robot instance wrapping a GBOT or the global `GBOT`.
---@param gbot? any Optional underlying Go robot object
---@return Robot
function Robot:new(gbot)
    local actualGBot = gbot or GBOT
    if not actualGBot then
        error("No valid bot object provided, and no global GBOT available.")
    end

    local o = {
        gbot = actualGBot,
        user = actualGBot.user,
        user_id = actualGBot.user_id,
        channel = actualGBot.channel,
        channel_id = actualGBot.channel_id,
        thread_id = actualGBot.thread_id,
        message_id = actualGBot.message_id,
        protocol = actualGBot.protocol,
        brain = actualGBot.brain,
        type = "native",
        threaded_message = actualGBot.threaded_message
    }
    setmetatable(o, self)
    return o
end

--------------------------------------------------------------------------------
-- Robot Methods (Message Sending, Prompting, etc.)
--------------------------------------------------------------------------------

---Send a channel message.
---@param channel string
---@param message string
---@param format? number
---@return number retVal
function Robot:SendChannelMessage(channel, message, format)
    return self.gbot:SendChannelMessage(channel, message, format)
end

---Send a thread message in a channel.
---@param channel string
---@param thread string
---@param message string
---@param format? number
---@return number retVal
function Robot:SendChannelThreadMessage(channel, thread, message, format)
    return self.gbot:SendChannelThreadMessage(channel, thread, message, format)
end

---Send a direct user message.
---@param user string
---@param message string
---@param format? number
---@return number retVal
function Robot:SendUserMessage(user, message, format)
    return self.gbot:SendUserMessage(user, message, format)
end

---Send a message to a user within a specific channel.
---@param user string
---@param channel string
---@param message string
---@param format? number
---@return number retVal
function Robot:SendUserChannelMessage(user, channel, message, format)
    return self.gbot:SendUserChannelMessage(user, channel, message, format)
end

---Send a message to a user within a channel thread.
---@param user string
---@param channel string
---@param thread string
---@param message string
---@param format? number
---@return number retVal
function Robot:SendUserChannelThreadMessage(user, channel, thread, message, format)
    return self.gbot:SendUserChannelThreadMessage(user, channel, thread, message, format)
end

---Send a message in the current context (channel or DM).
---@param message string
---@param format? number
---@return number retVal
function Robot:Say(message, format)
    return self.gbot:Say(message, format)
end

---Send a threaded message in the current channel thread.
---@param message string
---@param format? number
---@return number retVal
function Robot:SayThread(message, format)
    return self.gbot:SayThread(message, format)
end

---Reply to the current message (channel or DM).
---@param message string
---@param format? number
---@return number retVal
function Robot:Reply(message, format)
    return self.gbot:Reply(message, format)
end

---Reply in the same thread as the current message.
---@param message string
---@param format? number
---@return number retVal
function Robot:ReplyThread(message, format)
    return self.gbot:ReplyThread(message, format)
end

--------------------------------------------------------------------------------
-- Robot Modifier Methods (Direct, Fixed, Threaded, MessageFormat)
--------------------------------------------------------------------------------

---Return a Robot configured to send direct (DM) messages by default.
---@return Robot
function Robot:Direct()
    local dbot = self.gbot:Direct()
    return Robot:new(dbot)
end

---Return a Robot configured for fixed-format messages.
---@return Robot
function Robot:Fixed()
    local fbot = self.gbot:Fixed()
    return Robot:new(fbot)
end

---Return a Robot configured for threaded messages by default.
---@return Robot
function Robot:Threaded()
    local tbot = self.gbot:Threaded()
    return Robot:new(tbot)
end

---Return a Robot with a specified message format (e.g. fmt.Raw, fmt.Fixed, fmt.Variable).
---@param fmtVal number
---@return Robot
function Robot:MessageFormat(fmtVal)
    local mbot = self.gbot:MessageFormat(fmtVal)
    return Robot:new(mbot)
end

--------------------------------------------------------------------------------
-- Prompting Methods
--------------------------------------------------------------------------------

---Prompt for a reply matching a regex.
---@param regex_id string
---@param prompt string
---@param format? number
---@return string reply
---@return number retVal
function Robot:PromptForReply(regex_id, prompt, format)
    return self.gbot:PromptForReply(regex_id, prompt, format)
end

---Prompt in a thread for a reply matching a regex.
---@param regex_id string
---@param prompt string
---@param format? number
---@return string reply
---@return number retVal
function Robot:PromptThreadForReply(regex_id, prompt, format)
    return self.gbot:PromptThreadForReply(regex_id, prompt, format)
end

---Prompt a specific user for a reply matching a regex.
---@param regex_id string
---@param user string
---@param prompt string
---@param format? number
---@return string reply
---@return number retVal
function Robot:PromptUserForReply(regex_id, user, prompt, format)
    return self.gbot:PromptUserForReply(regex_id, user, prompt, format)
end

---Prompt a user in a channel for a reply matching a regex.
---@param regex_id string
---@param user string
---@param channel string
---@param prompt string
---@param format? number
---@return string reply
---@return number retVal
function Robot:PromptUserChannelForReply(regex_id, user, channel, prompt, format)
    return self.gbot:PromptUserChannelForReply(regex_id, user, channel, prompt, format)
end

---Prompt a user in a channel thread for a reply matching a regex.
---@param regex_id string
---@param user string
---@param channel string
---@param thread string
---@param prompt string
---@param format? number
---@return string reply
---@return number retVal
function Robot:PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format)
    return self.gbot:PromptUserChannelThreadForReply(regex_id, user, channel, thread, prompt, format)
end

--------------------------------------------------------------------------------
-- Short-Term Memory Methods
--------------------------------------------------------------------------------

---Remember a key-value pair in short-term memory.
---@param key string
---@param value string
---@param shared? boolean
function Robot:Remember(key, value, shared)
    self.gbot:Remember(key, value, shared)
end

---Remember a key-value pair in a thread's short-term memory.
---@param key string
---@param value string
---@param shared? boolean
function Robot:RememberThread(key, value, shared)
    self.gbot:RememberThread(key, value, shared)
end

---Remember a context value (for referencing "it" in subsequent commands).
---@param context string
---@param value string
function Robot:RememberContext(context, value)
    self.gbot:RememberContext(context, value)
end

---Remember a context value in a thread's short-term memory.
---@param context string
---@param value string
function Robot:RememberContextThread(context, value)
    self.gbot:RememberContextThread(context, value)
end

---Recall a value from short-term memory.
---@param key string
---@param shared? boolean
---@return string
function Robot:Recall(key, shared)
    return self.gbot:Recall(key, shared)
end

--------------------------------------------------------------------------------
-- Pipeline Methods
--------------------------------------------------------------------------------

---Get a pipeline parameter.
---@param name string
---@return string
function Robot:GetParameter(name)
    return self.gbot:GetParameter(name)
end

---Set a pipeline parameter.
---@param name string
---@param value any
---@return boolean
function Robot:SetParameter(name, value)
    return self.gbot:SetParameter(name, value)
end

---Obtain exclusive access to a resource for this task.
---@param tag string
---@param queueTask boolean
---@return boolean
function Robot:Exclusive(tag, queueTask)
    return self.gbot:Exclusive(tag, queueTask)
end

---Spawn a new job in the pipeline.
---@param name string
---@vararg any
---@return number retVal
function Robot:SpawnJob(name, ...)
    return self.gbot:SpawnJob(name, ...)
end

---Add a new task to the pipeline.
---@param name string
---@vararg any
---@return number retVal
function Robot:AddTask(name, ...)
    return self.gbot:AddTask(name, ...)
end

---Add a final task to the pipeline.
---@param name string
---@vararg any
---@return number retVal
function Robot:FinalTask(name, ...)
    return self.gbot:FinalTask(name, ...)
end

---Add a fail task to the pipeline.
---@param name string
---@vararg any
---@return number retVal
function Robot:FailTask(name, ...)
    return self.gbot:FailTask(name, ...)
end

---Add a new job to the pipeline (alias for spawning).
---@param name string
---@vararg any
---@return number retVal
function Robot:AddJob(name, ...)
    return self.gbot:AddJob(name, ...)
end

---Add a command to the pipeline.
---@param pluginName string
---@param command string
---@return number retVal
function Robot:AddCommand(pluginName, command)
    return self.gbot:AddCommand(pluginName, command)
end

---Add a final command to the pipeline.
---@param pluginName string
---@param command string
---@return number retVal
function Robot:FinalCommand(pluginName, command)
    return self.gbot:FinalCommand(pluginName, command)
end

---Add a fail command to the pipeline.
---@param pluginName string
---@param command string
---@return number retVal
function Robot:FailCommand(pluginName, command)
    return self.gbot:FailCommand(pluginName, command)
end

--------------------------------------------------------------------------------
-- Attribute Methods
--------------------------------------------------------------------------------

---Get a bot attribute.
---@param attr string
---@return string attribute
---@return number retVal
function Robot:GetBotAttribute(attr)
    return self.gbot:GetBotAttribute(attr)
end

---Get a user attribute.
---@param user string
---@param attr string
---@return string attribute
---@return number retVal
function Robot:GetUserAttribute(user, attr)
    return self.gbot:GetUserAttribute(user, attr)
end

---Get an attribute of the message sender.
---@param attr string
---@return string attribute
---@return number retVal
function Robot:GetSenderAttribute(attr)
    return self.gbot:GetSenderAttribute(attr)
end

--------------------------------------------------------------------------------
-- Long-Term Memory Methods
--------------------------------------------------------------------------------

-- Class definition for memories
---@class MemoryObject
---@field key string The key passed to CheckoutDatum
---@field exists boolean True if the datum was found
---@field datum table The actual stored data (could be any Lua table)
---@field token string The lock token, returned only for read/write checkouts
---@field retVal number The underlying ret.* code (e.g. ret.Ok, etc.)

---Check out a datum from long-term memory.
---@param key string
---@param rw? boolean
---@return MemoryObject
---@return number retVal
function Robot:CheckoutDatum(key, rw)
    local retVal, datum, token = self.gbot:CheckoutDatum(key, rw)
    local exists = (retVal == M.ret.Ok and datum ~= nil)
    local memory = {
        key = key,
        exists = exists,
        datum = datum or {},
        token = token or "",
        retVal = retVal
    }
    return memory, retVal
end

---Update a previously checked-out datum in long-term memory.
---@param memory MemoryObject
---@return number retVal
function Robot:UpdateDatum(memory)
    if not memory or not memory.key or not memory.token then
        error("UpdateDatum requires a table with 'key' and 'token'")
    end
    local retVal = self.gbot:UpdateDatum(memory.key, memory.token, memory.datum)
    return retVal
end

---Check in a previously checked-out datum.
---@param memory MemoryObject
function Robot:CheckinDatum(memory)
    if not memory or not memory.key or not memory.token then
        error("CheckinDatum requires a table with 'key' and 'token'")
    end
    self.gbot:CheckinDatum(memory.key, memory.token)
end

--------------------------------------------------------------------------------
-- Other Methods
--------------------------------------------------------------------------------

---Get the current task configuration.
---@return table config
---@return number retVal
function Robot:GetTaskConfig()
    return self.gbot:GetTaskConfig()
end

---Generate a random integer in [0, n-1].
---@param n number
---@return number
function Robot:RandomInt(n)
    return self.gbot:RandomInt(n)
end

---Select a random string from an array of strings.
---@param array string[]
---@return string
function Robot:RandomString(array)
    return self.gbot:RandomString(array)
end

---Pause execution for the specified number of seconds.
---@param seconds number
function Robot:Pause(seconds)
    self.gbot:Pause(seconds)
end

---Check if the current user has administrative privileges.
---@return boolean
function Robot:CheckAdmin()
    return self.gbot:CheckAdmin()
end

---Elevate the current user's privileges (e.g., require 2FA).
---@param immediate? boolean
---@return boolean
function Robot:Elevate(immediate)
    return self.gbot:Elevate(immediate)
end

---Log a message at the specified log level.
---@param level number
---@param message string
function Robot:Log(level, message)
    return self.gbot:Log(level, message)
end

--------------------------------------------------------------------------------
-- HTTP Client
--------------------------------------------------------------------------------

---@class HttpClient
---@field get fun(self: HttpClient, path: string, options: table|nil, callback: function)
---@field post fun(self: HttpClient, path: string, body: string, options: table|nil, callback: function)
---@field put fun(self: HttpClient, path: string, body: string, options: table|nil, callback: function)
---@field delete fun(self: HttpClient, path: string, options: table|nil, callback: function)
local HttpClient = {}

---@class HttpModule
---@field new fun(baseURI: string, options: table):HttpClient
local http = {}

---
-- The http global table provides a simple HTTP client for making web requests.
--
-- Usage:
--   local http = require("http") -- if not globally available
--   local client = http.new("https://api.example.com", {
--     headers = { ["Authorization"] = "Bearer mytoken" },
--     timeout = 10
--   })
--
--   client:get("/data", nil, function(resp, err)
--     if err then
--       bot:Log(log.Error, "HTTP request failed: " .. err)
--       return
--     end
--     bot:Say("Got response: " .. resp.body)
--   end)
--

--------------------------------------------------------------------------------
-- Module Exports
--------------------------------------------------------------------------------

-- We export:
--   - M.ret, M.task, M.log, M.fmt, M.proto
--   - M.Robot => the Robot class table
--   - M.New   => a helper that mimics old usage: local bot = robot.New(...)
--------------------------------------------------------------------------------

M.Robot = Robot

---Return the module table.
return M
