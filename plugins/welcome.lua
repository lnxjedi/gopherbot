-- welcome.lua - let the default robot greet the user

local robot = require("gopherbot_v1")
local bot = robot.Robot:new()

local command = arg[1]

if command == "configure" then
    return ""  -- No default configuration
end

-- Note that this plugin is only active when unconfigured and proto == terminal.
if command == "init" then
    bot:Pause(1)

    local name = bot:GetBotAttribute("name")
    bot:SendChannelMessage("general", "*******")
    bot:SendChannelMessage("general", "Welcome to the *Gopherbot* terminal connector. Since no " ..
        "configuration was detected, you're connected to '" .. name .. "', the default robot.")

    bot:Pause(2)

    local alias = bot:GetBotAttribute("alias")
    bot:SendChannelMessage("general", "If you've started the robot by mistake, just hit ctrl-D " ..
        "to exit and try 'gopherbot --help'; otherwise feel free to play around with the default robot - " ..
        "you can start by typing 'help'. If you'd like to start configuring a new robot, " ..
        "type: '" .. alias .. "setup slack'.")

    return robot.task.Normal
end

return robot.task.Normal
