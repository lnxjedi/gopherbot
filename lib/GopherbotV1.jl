# /opt/gopherbot/lib/GopherbotV1.jl

module GopherbotV1

using JSON
using HTTP
using Random

# =============================
# Module-Level Variables
# =============================

# Ref to store the caller_id once it's read
const global_caller_id = Ref{String}("")

# =============================
# Struct Definitions
# =============================

"""
    Attribute(attr::String, ret::Int)

Represents an attribute returned by Gopherbot.
"""
struct Attribute
    attr::String
    ret::Int
end

"""
    Reply(reply::String, ret::Int)

Represents a reply returned by Gopherbot.
"""
struct Reply
    reply::String
    ret::Int
end

"""
    Memory(key::String, lock_token::String, exists::Bool, datum::String, ret::Int)

Represents a memory object returned by Gopherbot.
"""
struct Memory
    key::String
    lock_token::String
    exists::Bool
    datum::String
    ret::Int
end

"""
    Robot(channel::String, channel_id::String, message_id::String, thread_id::String,
          threaded_message::Bool, user::String, user_id::String, caller_id::String,
          protocol::String, brain::String, format::String, prng::MersenneTwister)

Represents the Robot instance initialized with environment variables.
"""
mutable struct Robot
    channel::String
    channel_id::String
    message_id::String
    thread_id::String
    threaded_message::Bool
    user::String
    user_id::String
    caller_id::String
    protocol::String
    brain::String
    format::String
    prng::MersenneTwister
end

# =============================
# Constants
# =============================

# Return values for robot method calls
const Ok = 0
const UserNotFound = 1
const ChannelNotFound = 2
const AttributeNotFound = 3
const FailedMessageSend = 4
const FailedChannelJoin = 5
const DatumNotFound = 6
const DatumLockExpired = 7
const DataFormatError = 8
const BrainFailed = 9
const InvalidDatumKey = 10
const InvalidConfigPointer = 11
const ConfigUnmarshalError = 12
const NoConfigFound = 13
const RetryPrompt = 14
const ReplyNotMatched = 15
const UseDefaultValue = 16
const TimeoutExpired = 17
const Interrupted = 18
const MatcherNotFound = 19
const NoUserEmail = 20
const NoBotEmail = 21
const MailError = 22
const TaskNotFound = 23
const MissingArguments = 24
const InvalidStage = 25
const InvalidTaskType = 26
const CommandNotMatched = 27
const TaskDisabled = 28
const PrivilegeViolation = 29
const Failed = 63

# Plugin return values / exit codes
const Normal = 0
const Fail = 1
const MechanismFail = 2
const ConfigurationError = 3
const NotFound = 6
const Success = 7

# =============================
# Constructor Functions
# =============================

"""
    initialize_robot() -> Robot

Initializes a Robot instance by reading environment variables.
Handles secure retrieval of caller_id from stdin if required.
"""
function initialize_robot()::Robot
    # Check if global_caller_id has already been set
    if isempty(global_caller_id[])  # [] dereferences the Ref
        plugin_id = get(ENV, "GOPHER_CALLER_ID", "")
        if plugin_id == "stdin"
            # Read the caller_id from stdin
            println("Please enter the caller ID:")
            input = readline(stdin)
            global_caller_id[] = strip(input)  # Store in module-level variable

            # Update the environment variable to indicate consumption
            ENV["GOPHER_CALLER_ID"] = "read"
        else
            global_caller_id[] = plugin_id
        end
    end

    channel = get(ENV, "GOPHER_CHANNEL", "")
    channel_id = get(ENV, "GOPHER_CHANNEL_ID", "")
    message_id = get(ENV, "GOPHER_MESSAGE_ID", "")
    thread_id = get(ENV, "GOPHER_THREAD_ID", "")
    threaded_message_str = get(ENV, "GOPHER_THREADED_MESSAGE", "")
    threaded_message = threaded_message_str == "true"
    user = get(ENV, "GOPHER_USER", "")
    user_id = get(ENV, "GOPHER_USER_ID", "")
    protocol = get(ENV, "GOPHER_PROTOCOL", "")
    brain = get(ENV, "GOPHER_BRAIN", "")
    format = ""
    prng = MersenneTwister()
    return Robot(channel, channel_id, message_id, thread_id, threaded_message,
                 user, user_id, global_caller_id[], protocol, brain, format, prng)
end

# =============================
# Helper Functions
# =============================

"""
    send_command(robot::Robot, funcname::String, args::Dict{String, Any}=Dict(), format::String="") -> Dict

Sends a command to the Gopherbot engine via HTTP POST and returns the parsed JSON response.
"""
using HTTP
using JSON

function send_command(robot::Robot, funcname::String, args::Dict{String, Any}=Dict(), format::String="")::Dict
    if isempty(format)
        format = robot.format
    end
    payload = Dict(
        "FuncName" => funcname,
        "Format" => format,
        "FuncArgs" => args
    )
    json_payload = JSON.json(payload)
    url = "$(get(ENV, "GOPHER_HTTP_POST", "http://localhost:8000"))/json"

    try
        headers = [
            "Content-Type" => "application/json",
            "X-Caller-ID" => robot.caller_id
        ]
        response = HTTP.post(url, headers, json_payload)
        if response.status == 200
            return JSON.parse(String(response.body))
        else
            error("Gopherbot returned status code $(response.status)")
        end
    catch e
        error("Failed to send command: $e")
    end
end

# =============================
# Core Methods
# =============================

"""
    check_admin(robot::Robot) -> Bool

Checks if the user is an admin.
"""
function check_admin(robot::Robot)::Bool
    response = send_command(robot, "CheckAdmin")
    return get(response, "Boolean", false)
end

"""
    subscribe(robot::Robot) -> Bool

Subscribes the user to updates.
"""
function subscribe(robot::Robot)::Bool
    response = send_command(robot, "Subscribe")
    return get(response, "Boolean", false)
end

"""
    unsubscribe(robot::Robot) -> Bool

Unsubscribes the user from updates.
"""
function unsubscribe(robot::Robot)::Bool
    response = send_command(robot, "Unsubscribe")
    return get(response, "Boolean", false)
end

"""
    elevate(robot::Robot, immediate::Bool=false) -> Bool

Elevates the user's privileges.
"""
function elevate(robot::Robot, immediate::Bool=false)::Bool
    args = Dict{String, Any}("Immediate" => immediate)
    response = send_command(robot, "Elevate", args)
    return get(response, "Boolean", false)
end

"""
    pause_robot(robot::Robot, seconds::Float64)

Pauses execution for the specified number of seconds.
"""
function pause_robot(robot::Robot, seconds::Float64)
    sleep(seconds)
end

"""
    random_string(robot::Robot, sarr::Vector{String}) -> String

Returns a random string from the provided array.
"""
function random_string(robot::Robot, sarr::Vector{String})::String
    return sarr[rand(1:length(sarr))]
end

"""
    random_int(robot::Robot, i::Int) -> Int

Returns a random integer between 1 and `i`.
"""
function random_int(robot::Robot, i::Int)::Int
    return rand(1:i)
end

"""
    spawn_job(robot::Robot, name::String, args::Vector{String}) -> Int

Spawns a new job with the given name and arguments.
"""
function spawn_job(robot::Robot, name::String, args::Vector{String})::Int
    args_dict = Dict{String, Any}("Name" => name, "CmdArgs" => args)
    response = send_command(robot, "SpawnJob", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    add_job(robot::Robot, name::String, args::Vector{String}) -> Int

Adds a new job with the given name and arguments.
"""
function add_job(robot::Robot, name::String, args::Vector{String})::Int
    args_dict = Dict{String, Any}("Name" => name, "CmdArgs" => args)
    response = send_command(robot, "AddJob", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    add_task(robot::Robot, name::String, args::Vector{String}) -> Int

Adds a new task with the given name and arguments.
"""
function add_task(robot::Robot, name::String, args::Vector{String})::Int
    args_dict = Dict{String, Any}("Name" => name, "CmdArgs" => args)
    response = send_command(robot, "AddTask", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    final_task(robot::Robot, name::String, args::Vector{String}) -> Int

Finalizes a task with the given name and arguments.
"""
function final_task(robot::Robot, name::String, args::Vector{String})::Int
    args_dict = Dict{String, Any}("Name" => name, "CmdArgs" => args)
    response = send_command(robot, "FinalTask", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    fail_task(robot::Robot, name::String, args::Vector{String}) -> Int

Fails a task with the given name and arguments.
"""
function fail_task(robot::Robot, name::String, args::Vector{String})::Int
    args_dict = Dict{String, Any}("Name" => name, "CmdArgs" => args)
    response = send_command(robot, "FailTask", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    add_command(robot::Robot, plugin::String, cmd::String) -> Int

Adds a new command to a plugin.
"""
function add_command(robot::Robot, plugin::String, cmd::String)::Int
    args_dict = Dict{String, Any}("Plugin" => plugin, "Command" => cmd)
    response = send_command(robot, "AddCommand", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    final_command(robot::Robot, plugin::String, cmd::String) -> Int

Finalizes a command in a plugin.
"""
function final_command(robot::Robot, plugin::String, cmd::String)::Int
    args_dict = Dict{String, Any}("Plugin" => plugin, "Command" => cmd)
    response = send_command(robot, "FinalCommand", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    fail_command(robot::Robot, plugin::String, cmd::String) -> Int

Fails a command in a plugin.
"""
function fail_command(robot::Robot, plugin::String, cmd::String)::Int
    args_dict = Dict{String, Any}("Plugin" => plugin, "Command" => cmd)
    response = send_command(robot, "FailCommand", args_dict)
    return get(response, "RetVal", Fail)
end

"""
    set_parameter(robot::Robot, name::String, value::String) -> Bool

Sets a parameter with the given name and value.
"""
function set_parameter(robot::Robot, name::String, value::String)::Bool
    args = Dict{String, Any}("Name" => name, "Value" => value)
    response = send_command(robot, "SetParameter", args)
    return get(response, "Boolean", false)
end

"""
    get_parameter(robot::Robot, name::String) -> String

Gets a parameter from the pipeline.
"""
function get_parameter(robot::Robot, name::String)::String
    args = Dict{String, Any}("Parameter" => name)
    response = send_command(robot, "GetParameter", args)
    return get(response, "StrVal", "")
end

"""
    exclusive(robot::Robot, tag::String, queue_task::Bool=false) -> Bool

Sets an exclusive tag for the robot.
"""
function exclusive(robot::Robot, tag::String, queue_task::Bool=false)::Bool
    args = Dict{String, Any}("Tag" => tag, "QueueTask" => queue_task)
    response = send_command(robot, "Exclusive", args)
    return get(response, "Boolean", false)
end

"""
    set_working_directory(robot::Robot, path::String) -> Bool

Sets the working directory for the robot.
"""
function set_working_directory(robot::Robot, path::String)::Bool
    args = Dict{String, Any}("Path" => path)
    response = send_command(robot, "SetWorkingDirectory", args)
    return get(response, "Boolean", false)
end

# =============================
# Memory Management
# =============================

"""
    checkout_datum(robot::Robot, key::String, rw::Bool) -> Memory

Checks out a datum with the given key and read/write flag.
"""
function checkout_datum(robot::Robot, key::String, rw::Bool)::Memory
    args = Dict{String, Any}("Key" => key, "RW" => rw)
    response = send_command(robot, "CheckoutDatum", args)
    return Memory(
        key,
        get(response, "LockToken", ""),
        get(response, "Exists", false),
        get(response, "Datum", ""),
        get(response, "RetVal", Fail)
    )
end

"""
    checkin_datum(robot::Robot, m::Memory) -> Int

Checks in a previously checked out datum.
"""
function checkin_datum(robot::Robot, m::Memory)::Int
    args = Dict{String, Any}("Key" => m.key, "Token" => m.lock_token)
    send_command(robot, "CheckinDatum", args)
    return Ok
end

"""
    update_datum(robot::Robot, m::Memory) -> Int

Updates a previously checked out datum.
"""
function update_datum(robot::Robot, m::Memory)::Int
    args = Dict{String, Any}("Key" => m.key, "Token" => m.lock_token, "Datum" => m.datum)
    response = send_command(robot, "UpdateDatum", args)
    return get(response, "RetVal", Fail)
end

# =============================
# Memory Operations
# =============================

"""
    remember(robot::Robot, k::String, v::String, shared::Bool=false) -> Int

Remembers a key-value pair.
"""
function remember(robot::Robot, k::String, v::String, shared::Bool=false)::Int
    funcname = robot.threaded_message ? "RememberThread" : "Remember"
    args = Dict{String, Any}("Key" => k, "Value" => v, "Shared" => shared)
    response = send_command(robot, funcname, args)
    return get(response, "RetVal", Fail)
end

"""
    remember_context(robot::Robot, c::String, v::String) -> Int

Remembers context-specific key-value pair.
"""
function remember_context(robot::Robot, c::String, v::String)::Int
    return remember(robot, "context:" * c, v, false)
end

"""
    remember_thread(robot::Robot, k::String, v::String, shared::Bool=false) -> Int

Remembers a key-value pair in a threaded context.
"""
function remember_thread(robot::Robot, k::String, v::String, shared::Bool=false)::Int
    args = Dict{String, Any}("Key" => k, "Value" => v, "Shared" => shared)
    response = send_command(robot, "RememberThread", args)
    return get(response, "RetVal", Fail)
end

"""
    remember_context_thread(robot::Robot, c::String, v::String) -> Int

Remembers context-specific key-value pair in a threaded context.
"""
function remember_context_thread(robot::Robot, c::String, v::String)::Int
    return remember_thread(robot, "context:" * c, v, false)
end

"""
    recall(robot::Robot, memory::String, shared::Bool=false) -> String

Recalls the value associated with a memory key.
"""
function recall(robot::Robot, memory::String, shared::Bool=false)::String
    args = Dict{String, Any}("Key" => memory, "Shared" => shared)
    response = send_command(robot, "Recall", args)
    return get(response, "StrVal", "")
end

# =============================
# Attribute Retrieval
# =============================

"""
    get_sender_attribute(robot::Robot, attr::String) -> Attribute

Retrieves an attribute of the message sender.
"""
function get_sender_attribute(robot::Robot, attr::String)::Attribute
    args = Dict{String, Any}("Attribute" => attr)
    response = send_command(robot, "GetSenderAttribute", args)
    return Attribute(get(response, "Attribute", ""), get(response, "RetVal", Fail))
end

"""
    get_user_attribute(robot::Robot, user::String, attr::String) -> Attribute

Retrieves an attribute of a specific user.
"""
function get_user_attribute(robot::Robot, user::String, attr::String)::Attribute
    args = Dict{String, Any}("User" => user, "Attribute" => attr)
    response = send_command(robot, "GetUserAttribute", args)
    return Attribute(get(response, "Attribute", ""), get(response, "RetVal", Fail))
end

"""
    get_bot_attribute(robot::Robot, attr::String) -> Attribute

Retrieves an attribute of the bot itself.
"""
function get_bot_attribute(robot::Robot, attr::String)::Attribute
    args = Dict{String, Any}("Attribute" => attr)
    response = send_command(robot, "GetBotAttribute", args)
    return Attribute(get(response, "Attribute", ""), get(response, "RetVal", Fail))
end

# =============================
# Logging
# =============================

"""
    log_message(robot::Robot, level::String, message::String) -> Int

Logs a message with the specified level.
"""
function log_message(robot::Robot, level::String, message::String)::Int
    args = Dict{String, Any}("Level" => level, "Message" => message)
    send_command(robot, "Log", args)
    return Ok
end

# =============================
# Messaging Functions
# =============================

"""
    send_channel_message(robot::Robot, channel::String, message::String, format::String="") -> Int

Sends a message to a specific channel.
"""
function send_channel_message(robot::Robot, channel::String, message::String, format::String="")::Int
    return send_channel_thread_message(robot, channel, "", message, format)
end

"""
    send_channel_thread_message(robot::Robot, channel::String, thread::String, message::String, format::String="") -> Int

Sends a threaded message to a specific channel.
"""
function send_channel_thread_message(robot::Robot, channel::String, thread::String, message::String, format::String="")::Int
    args = Dict{String, Any}("Channel" => channel, "Thread" => thread, "Message" => message)
    response = send_command(robot, "SendChannelThreadMessage", args, format)
    return get(response, "RetVal", Fail)
end

"""
    send_user_message(robot::Robot, user::String, message::String, format::String="") -> Int

Sends a message to a specific user.
"""
function send_user_message(robot::Robot, user::String, message::String, format::String="")::Int
    args = Dict{String, Any}("User" => user, "Message" => message)
    response = send_command(robot, "SendUserMessage", args, format)
    return get(response, "RetVal", Fail)
end

"""
    send_user_channel_message(robot::Robot, user::String, channel::String, message::String, format::String="") -> Int

Sends a message to a specific user within a channel.
"""
function send_user_channel_message(robot::Robot, user::String, channel::String, message::String, format::String="")::Int
    return send_user_channel_thread_message(robot, user, channel, "", message, format)
end

"""
    send_user_channel_thread_message(robot::Robot, user::String, channel::String, thread::String, message::String, format::String="") -> Int

Sends a threaded message to a specific user within a channel.
"""
function send_user_channel_thread_message(robot::Robot, user::String, channel::String, thread::String, message::String, format::String="")::Int
    args = Dict{String, Any}("User" => user, "Channel" => channel, "Thread" => thread, "Message" => message)
    response = send_command(robot, "SendUserChannelThreadMessage", args, format)
    return get(response, "RetVal", Fail)
end

"""
    say(robot::Robot, message::String, format::String="") -> Int

Sends a message either to a user or channel based on the robot's context.
"""
function say(robot::Robot, message::String, format::String="")::Int
    if isempty(robot.channel)
        return send_user_message(robot, robot.user, message, format)
    else
        thread = robot.threaded_message ? robot.thread_id : ""
        return send_channel_thread_message(robot, robot.channel, thread, message, format)
    end
end

"""
    say_thread(robot::Robot, message::String, format::String="") -> Int

Sends a threaded message either to a user or channel based on the robot's context.
"""
function say_thread(robot::Robot, message::String, format::String="")::Int
    if isempty(robot.channel)
        return send_user_message(robot, robot.user, message, format)
    else
        return send_channel_thread_message(robot, robot.channel, robot.thread_id, message, format)
    end
end

"""
    reply(robot::Robot, message::String, format::String="") -> Int

Replies to a message either to a user or channel based on the robot's context.
"""
function reply(robot::Robot, message::String, format::String="")::Int
    if isempty(robot.channel)
        return send_user_message(robot, robot.user, message, format)
    else
        thread = robot.threaded_message ? robot.thread_id : ""
        return send_user_channel_thread_message(robot, robot.user, robot.channel, thread, message, format)
    end
end

"""
    reply_thread(robot::Robot, message::String, format::String="") -> Int

Replies to a threaded message either to a user or channel based on the robot's context.
"""
function reply_thread(robot::Robot, message::String, format::String="")::Int
    if isempty(robot.channel)
        return send_user_message(robot, robot.user, message, format)
    else
        return send_user_channel_thread_message(robot, robot.user, robot.channel, robot.thread_id, message, format)
    end
end

# =============================
# Prompt Functions
# =============================

"""
    prompt_for_reply(robot::Robot, regex_id::String, prompt::String) -> Reply

Prompts the user for a reply based on a regex pattern.
"""
function prompt_for_reply(robot::Robot, regex_id::String, prompt::String)::Reply
    thread = robot.threaded_message ? robot.thread_id : ""
    return prompt_user_channel_thread_for_reply(robot, regex_id, robot.user, robot.channel, thread, prompt)
end

"""
    prompt_thread_for_reply(robot::Robot, regex_id::String, prompt::String) -> Reply

Prompts the user for a reply in a threaded context based on a regex pattern.
"""
function prompt_thread_for_reply(robot::Robot, regex_id::String, prompt::String)::Reply
    return prompt_user_channel_thread_for_reply(robot, regex_id, robot.user, robot.channel, robot.thread_id, prompt)
end

"""
    prompt_user_for_reply(robot::Robot, regex_id::String, prompt::String) -> Reply

Prompts a specific user for a reply based on a regex pattern.
"""
function prompt_user_for_reply(robot::Robot, regex_id::String, prompt::String)::Reply
    return prompt_user_channel_thread_for_reply(robot, regex_id, robot.user, "", "", prompt)
end

"""
    prompt_user_channel_for_reply(robot::Robot, regex_id::String, user::String, channel::String, prompt::String) -> Reply

Prompts a specific user in a specific channel for a reply based on a regex pattern.
"""
function prompt_user_channel_for_reply(robot::Robot, regex_id::String, user::String, channel::String, prompt::String)::Reply
    return prompt_user_channel_thread_for_reply(robot, regex_id, user, channel, "", prompt)
end

"""
    prompt_user_channel_thread_for_reply(robot::Robot, regex_id::String, user::String, channel::String, thread::String, prompt::String) -> Reply

Internal function to handle prompting the user for a reply.
"""
function prompt_user_channel_thread_for_reply(robot::Robot, regex_id::String, user::String, channel::String, thread::String, prompt::String)::Reply
    for _ in 1:3
        args = Dict{String, Any}("RegexID" => regex_id, "User" => user, "Channel" => channel, "Thread" => thread, "Prompt" => prompt)
        response = send_command(robot, "PromptUserChannelThreadForReply", args)
        if get(response, "RetVal", RetryPrompt) == RetryPrompt
            continue
        end
        return Reply(get(response, "Reply", ""), get(response, "RetVal", Fail))
    end
    if get(response, "RetVal", RetryPrompt) == RetryPrompt
        return Reply("", Interrupted)
    else
        return Reply(get(response, "Reply", ""), get(response, "RetVal", Fail))
    end
end

# =============================
# Subclass Definitions
# =============================

# Define DirectBot as a subtype of Robot
mutable struct DirectBot <: Robot
    # Inherits all fields from Robot
end

"""
    DirectBot() -> DirectBot

Creates a new DirectBot instance with modified channel settings.
"""
function DirectBot()::DirectBot
    parent_robot = initialize_robot()
    new_direct_bot = DirectBot(
        "",                      # channel
        "",                      # channel_id
        parent_robot.message_id, # message_id
        "",                      # thread_id
        false,                   # threaded_message
        parent_robot.user,       # user
        parent_robot.user_id,    # user_id
        parent_robot.caller_id,  # caller_id
        parent_robot.protocol,   # protocol
        parent_robot.brain,      # brain
        parent_robot.format,     # format
        parent_robot.prng        # prng
    )
    return new_direct_bot
end

# Similarly, define ThreadedBot
mutable struct ThreadedBot <: Robot
    # Inherits all fields from Robot
end

"""
    ThreadedBot() -> ThreadedBot

Creates a new ThreadedBot instance with threaded message settings.
"""
function ThreadedBot()::ThreadedBot
    parent_robot = initialize_robot()
    threaded_message = parent_robot.channel != "" ? true : false
    new_threaded_bot = ThreadedBot(
        parent_robot.channel,       # channel
        parent_robot.channel_id,    # channel_id
        parent_robot.message_id,    # message_id
        parent_robot.thread_id,     # thread_id
        threaded_message,           # threaded_message
        parent_robot.user,          # user
        parent_robot.user_id,       # user_id
        parent_robot.caller_id,     # caller_id
        parent_robot.protocol,      # protocol
        parent_robot.brain,         # brain
        parent_robot.format,        # format
        parent_robot.prng           # prng
    )
    return new_threaded_bot
end

# Define FormattedBot
mutable struct FormattedBot <: Robot
    # Inherits all fields from Robot
end

"""
    FormattedBot(format::String) -> FormattedBot

Creates a new FormattedBot instance with a specified message format.
"""
function FormattedBot(format::String)::FormattedBot
    parent_robot = initialize_robot()
    new_formatted_bot = FormattedBot(
        parent_robot.channel,        # channel
        parent_robot.channel_id,     # channel_id
        parent_robot.message_id,     # message_id
        parent_robot.thread_id,      # thread_id
        parent_robot.threaded_message, # threaded_message
        parent_robot.user,           # user
        parent_robot.user_id,        # user_id
        parent_robot.caller_id,      # caller_id
        parent_robot.protocol,       # protocol
        parent_robot.brain,          # brain
        format,                      # format
        parent_robot.prng            # prng
    )
    return new_formatted_bot
end

# =============================
# End of Module
# =============================

end # module
