-- welcome.lua - let the default robot greet the user

local robot = require("gopherbot_v1")
local bot = robot.Robot:new()

local command = arg[1]

if command == "configure" then
    return ""  -- No default configuration
end

-- Welcome messaging for SSH now runs through the welcome-join trigger job.
-- Keep init quiet so startup doesn't emit chat lines before a user connects.
if command == "init" then
    return robot.task.Normal
end

return robot.task.Normal
