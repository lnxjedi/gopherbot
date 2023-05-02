module bot

using Random
using HTTP
using JSON

export Robot

struct Robot
    channel::Union{String, Nothing}
    thread_id::Union{String, Nothing}
    threaded_message::Union{String, Nothing}
    user::Union{String, Nothing}
    plugin_id::Union{String, Nothing}
    format::String
    protocol::Union{String, Nothing}
end

function new()::Robot
    Random.seed!()  # Seed the random number generator
    Robot(
        get(ENV, "GOPHER_CHANNEL", ""),
        get(ENV, "GOPHER_THREAD_ID", ""),
        get(ENV, "GOPHER_THREADED_MESSAGE", ""),
        get(ENV, "GOPHER_USER", ""),
        get(ENV, "GOPHER_CALLER_ID", ""),
        "",
        get(ENV, "GOPHER_PROTOCOL", "")
    )
end

function Say(robot::Robot, message::String)
    println(nameof(var"#self#"))
    println(message)
end

end
