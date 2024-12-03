#!/usr/bin/env julia

using GopherbotV1

# Initialize the Robot object
robot = GopherbotV1.initialize_robot()

# Define the configure function to output YAML
function configure()
    yaml = raw"""
---
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <something> - tell the bot to say <something>" ]
CommandMatchers:
- Command: "echo"
  Regex: '(?i:echo ([^\n]*))'
AllowedHiddenCommands:
- echo
"""
    println(yaml)
end

# Parse command-line arguments
if length(ARGS) < 1
    GopherbotV1.reply(robot, "No command provided.", "")
else
    command = ARGS[1]
    args = ARGS[2:end]

    if command == "configure"
        configure()
        exit(0)  # Exit after handling configure
    elseif command == "echo"
        if isempty(args)
            GopherbotV1.reply(robot, "No message provided for echo.", "")
        else
            message = join(args, " ")
            GopherbotV1.say(robot, message)
        end
    else
        GopherbotV1.say(robot, "Unknown command: $command")
    end
end
