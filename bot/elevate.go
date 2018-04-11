package bot

import "fmt"

const technicalElevError = "Sorry, elevation failed due to a problem with the elevation service"
const configElevError = "Sorry, elevation failed due to a configuration error"

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

func (bot *Robot) elevate(plugins []*Plugin, plugin *Plugin, immediate bool) (retval PlugRetVal) {
	robot.RLock()
	defaultElevator := robot.defaultElevator
	robot.RUnlock()
	if plugin.Elevator == "" && defaultElevator == "" {
		Log(Error, fmt.Sprintf("Plugin \"%s\" requires elevation, but no elevator configured", plugin.name))
		bot.Say(configElevError)
		emit(ElevNoRunMisconfigured)
		return ConfigurationError
	}
	elevator := defaultElevator
	if plugin.Elevator != "" {
		elevator = plugin.Elevator
	}
	for _, ePlug := range plugins {
		if elevator == ePlug.name {
			if !pluginAvailable(bot.User, bot.Channel, ePlug, false) {
				Log(Error, fmt.Sprintf("Elevation plugin \"%s\" not available while elevating user \"%s\" for plugin \"%s\" in channel \"%s\"", ePlug.name, bot.User, plugin.name, bot.Channel))
				bot.Say(configElevError)
				emit(ElevNoRunNotAvailable)
				return ConfigurationError
			}
			immedString := "true"
			if !immediate {
				immedString = "false"
			}
			elevRet := callPlugin(bot, ePlug, false, false, "elevate", immedString)
			if elevRet == Success {
				emit(ElevRanSuccess)
				return Success
			}
			if elevRet == Fail {
				Log(Warn, fmt.Sprintf("Elevation failed by elevator \"%s\", user \"%s\", plugin \"%s\" in channel \"%s\"", ePlug.name, bot.User, plugin.name, bot.Channel))
				bot.Say("Sorry, this command requires elevation")
				emit(ElevRanFail)
				return Fail
			}
			if elevRet == MechanismFail {
				Log(Error, fmt.Sprintf("Elevator plugin \"%s\" mechanism failure while elevating user \"%s\" for plugin \"%s\" in channel \"%s\"", ePlug.name, bot.User, plugin.name, bot.Channel))
				bot.Say(technicalElevError)
				emit(ElevRanMechanismFailed)
				return MechanismFail
			}
			if elevRet == Normal {
				Log(Error, fmt.Sprintf("Elevator plugin \"%s\" returned 'Normal' (0) instead of 'Success' (1), failing elevation in \"%s\" for plugin \"%s\" in channel \"%s\"", ePlug.name, bot.User, plugin.name, bot.Channel))
				bot.Say(technicalElevError)
				emit(ElevRanFailNormal)
				return MechanismFail
			}
			Log(Error, fmt.Sprintf("Elevator plugin \"%s\" exit code %d while elevating user \"%s\" for plugin \"%s\" in channel \"%s\"", ePlug.name, retval, bot.User, plugin.name, bot.Channel))
			bot.Say(technicalElevError)
			emit(ElevRanFailOther)
			return MechanismFail
		}
	}
	Log(Error, fmt.Sprintf("Elevator plugin \"%s\" not found while elevating user \"%s\" for plugin \"%s\" in channel \"%s\"", plugin.Elevator, bot.User, plugin.name, bot.Channel))
	bot.Say(technicalElevError)
	emit(ElevNoRunNotFound)
	return ConfigurationError
}

// Check for a configured Elevator and check elevation
func (bot *Robot) checkElevation(plugins []*Plugin, plugin *Plugin, command string) (retval PlugRetVal) {
	immediate := false
	elevationRequired := false
	if len(plugin.ElevateImmediateCommands) > 0 {
		for _, i := range plugin.ElevateImmediateCommands {
			if command == i {
				elevationRequired = true
				immediate = true
				break
			}
		}
	}
	if !elevationRequired && len(plugin.ElevatedCommands) > 0 {
		for _, i := range plugin.ElevatedCommands {
			if command == i {
				elevationRequired = true
				break
			}
		}
	}
	if !elevationRequired {
		return Success
	}
	retval = bot.elevate(plugins, plugin, immediate)
	if retval == Success {
		return Success
	}
	Log(Error, fmt.Sprintf("Elevation failed for plugin \"%s\", command: \"%s\"", plugin.name, command))
	return Fail
}
