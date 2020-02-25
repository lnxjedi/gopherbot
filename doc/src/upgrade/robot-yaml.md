# Main Configuration File Rename

To make naming more consistent, **Gopherbot** v2 and the standard robot now expect to find `conf/robot.yaml` for the robot's main configuration file. It should still fall back to the old `conf/gopherbot.yaml` if `robot.yaml` isn't present, but for maximum forward-compatibility the old `gopherbot.yaml` should be renamed.