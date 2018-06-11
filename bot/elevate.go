package bot

import "fmt"

const technicalElevError = "Sorry, elevation failed due to a problem with the elevation service"
const configElevError = "Sorry, elevation failed due to a configuration error"

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

func (bot *Robot) elevate(task *botTask, immediate bool) (retval TaskRetVal) {
	// TODO: set
	robot.RLock()
	defaultElevator := robot.defaultElevator
	robot.RUnlock()
	if task.Elevator == "" && defaultElevator == "" {
		Log(Audit, fmt.Sprintf("Task '%s' requires elevation, but no elevator configured", task.name))
		bot.Say(configElevError)
		emit(ElevNoRunMisconfigured)
		return ConfigurationError
	}
	elevator := defaultElevator
	if task.Elevator != "" {
		elevator = task.Elevator
	}
	_, ePlug, _ := currentTasks.getTaskByName(elevator)
	if ePlug != nil {
		immedString := "true"
		if !immediate {
			immedString = "false"
		}
		elevRet := bot.callTask(ePlug.botTask, "elevate", immedString)
		if elevRet == Success {
			Log(Audit, fmt.Sprintf("Elevation succeeded by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, bot.User, task.name, bot.Channel))
			emit(ElevRanSuccess)
			return Success
		}
		if elevRet == Fail {
			Log(Audit, fmt.Sprintf("Elevation FAILED by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, bot.User, task.name, bot.Channel))
			bot.Say("Sorry, this command requires elevation")
			emit(ElevRanFail)
			return Fail
		}
		if elevRet == MechanismFail {
			Log(Audit, fmt.Sprintf("Elevator plugin '%s' mechanism failure while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, bot.User, task.name, bot.Channel))
			bot.Say(technicalElevError)
			emit(ElevRanMechanismFailed)
			return MechanismFail
		}
		if elevRet == Normal {
			Log(Audit, fmt.Sprintf("Elevator plugin '%s' returned 'Normal' (0) instead of 'Success' (1), failing elevation in '%s' for task '%s' in channel '%s'", ePlug.name, bot.User, task.name, bot.Channel))
			bot.Say(technicalElevError)
			emit(ElevRanFailNormal)
			return MechanismFail
		}
		Log(Audit, fmt.Sprintf("Elevator plugin '%s' exit code %d while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, retval, bot.User, task.name, bot.Channel))
		bot.Say(technicalElevError)
		emit(ElevRanFailOther)
		return MechanismFail
	}
	Log(Audit, fmt.Sprintf("Elevator plugin '%s' not found while elevating user '%s' for task '%s' in channel '%s'", task.Elevator, bot.User, task.name, bot.Channel))
	bot.Say(technicalElevError)
	emit(ElevNoRunNotFound)
	return ConfigurationError
}

// Check for a configured Elevator and check elevation
func (bot *Robot) checkElevation(t interface{}, command string) (retval TaskRetVal) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	immediate := false
	elevationRequired := false
	if isPlugin && len(plugin.ElevateImmediateCommands) > 0 {
		for _, i := range plugin.ElevateImmediateCommands {
			if command == i {
				elevationRequired = true
				immediate = true
				break
			}
		}
	}
	if isPlugin && !elevationRequired && len(plugin.ElevatedCommands) > 0 {
		for _, i := range plugin.ElevatedCommands {
			if command == i {
				elevationRequired = true
				break
			}
		}
	}
	if !isPlugin {
		if len(task.Elevator) > 0 {
			elevationRequired = true
		}
	}
	if !elevationRequired {
		return Success
	}
	retval = bot.elevate(task, immediate)
	if retval == Success {
		return Success
	}
	Log(Error, fmt.Sprintf("Elevation failed for task '%s', command: '%s'", task.name, command))
	return Fail
}
