package bot

import "github.com/lnxjedi/gopherbot/robot"

const technicalElevError = "Sorry, elevation failed due to a problem with the elevation service"
const configElevError = "Sorry, elevation failed due to a configuration error"

// Elevator plugins provide an elevate method for checking if the user
// can run a privileged command.

func (r Robot) elevate(task *Task, immediate bool) (retval robot.TaskRetVal) {
	defaultElevator := r.cfg.defaultElevator
	if task.Elevator == "" && defaultElevator == "" {
		Log(robot.Audit, "Task '%s' requires elevation, but no elevator configured", task.name)
		r.Say(configElevError)
		emit(ElevNoRunMisconfigured)
		return robot.ConfigurationError
	}
	elevator := defaultElevator
	if task.Elevator != "" {
		elevator = task.Elevator
	}
	_, ePlug, _ := getTask(r.tasks.getTaskByName(elevator))
	if ePlug != nil {
		immedString := "true"
		if !immediate {
			immedString = "false"
		}
		w := getLockedWorker(r.tid)
		w.Unlock()
		_, elevRet := w.callTask(ePlug, "elevate", immedString)
		elevated := elevRet == robot.Success
		w.Lock()
		w.elevated = elevated
		w.currentTask = r.currentTask
		w.Unlock()
		if elevated {
			Log(robot.Audit, "Elevation succeeded by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, r.User, task.name, r.Channel)
			emit(ElevRanSuccess)
			return robot.Success
		}
		if elevRet == robot.Fail {
			Log(robot.Audit, "Elevation FAILED by elevator '%s', user '%s', task '%s' in channel '%s'", ePlug.name, r.User, task.name, r.Channel)
			r.Say("Sorry, this command requires elevation")
			emit(ElevRanFail)
			return robot.Fail
		}
		if elevRet == robot.MechanismFail {
			Log(robot.Audit, "Elevator plugin '%s' mechanism failure while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, r.User, task.name, r.Channel)
			r.Say(technicalElevError)
			emit(ElevRanMechanismFailed)
			return robot.MechanismFail
		}
		if elevRet == robot.Normal {
			Log(robot.Audit, "Elevator plugin '%s' returned 'Normal' (0) instead of 'Success' (1), failing elevation in '%s' for task '%s' in channel '%s'", ePlug.name, r.User, task.name, r.Channel)
			r.Say(technicalElevError)
			emit(ElevRanFailNormal)
			return robot.MechanismFail
		}
		Log(robot.Audit, "Elevator plugin '%s' exit code %d while elevating user '%s' for task '%s' in channel '%s'", ePlug.name, retval, r.User, task.name, r.Channel)
		r.Say(technicalElevError)
		emit(ElevRanFailOther)
		return robot.MechanismFail
	}
	Log(robot.Audit, "Elevator plugin '%s' not found while elevating user '%s' for task '%s' in channel '%s'", task.Elevator, r.User, task.name, r.Channel)
	r.Say(technicalElevError)
	emit(ElevNoRunNotFound)
	return robot.ConfigurationError
}

// Check for a configured Elevator and check elevation
func (r Robot) checkElevation(t interface{}, command string) (retval robot.TaskRetVal, required bool) {
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
		return robot.Success, false
	}
	retval = r.elevate(task, immediate)
	if retval == robot.Success {
		return robot.Success, true
	}
	Log(robot.Error, "Elevation failed for task '%s', command: '%s'", task.name, command)
	return robot.Fail, true
}
