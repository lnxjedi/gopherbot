package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// pipeTask does all the real work of adding tasks to pipelines or spawning
// new tasks.
func (r Robot) pipeTask(pflavor pipeAddFlavor, ptype pipeAddType, name string, args ...string) robot.RetVal {
	if r.stage != primaryTasks {
		task, _, _ := getTask(r.currentTask)
		r.Log(robot.Error, "Request to modify pipeline outside of initial pipeline in task '%s'", task.name)
		return robot.InvalidStage
	}
	t := r.tasks.getTaskByName(name)
	if t == nil {
		task, _, _ := getTask(r.currentTask)
		r.Log(robot.Error, "Task '%s' not found updating pipeline from task '%s'", name, task.name)
		return robot.TaskNotFound
	}
	task, plugin, job := getTask(t)
	isPlugin := plugin != nil
	isJob := job != nil
	if task.Disabled {
		r.Log(robot.Error, "Attempt to add disabled task '%s' to pipeline", name)
		return robot.TaskDisabled
	}
	if ptype == typePlugin && !isPlugin {
		r.Log(robot.Error, "Adding command to pipeline - not a plugin: %s", name)
		return robot.InvalidTaskType
	}
	if ptype == typeJob && !isJob {
		r.Log(robot.Error, "Adding job to pipeline - not a job: %s", name)
		return robot.InvalidTaskType
	}
	if ptype == typeTask && (isJob || isPlugin) {
		r.Log(robot.Error, "Adding task to pipeline - not a task: %s", name)
		return robot.InvalidTaskType
	}
	if !r.privileged {
		if isJob && job.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged job '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
		if isPlugin && plugin.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged plugin '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
		if !isJob && !isPlugin && task.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged task '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
	}
	var command string
	var cmdargs []string
	if isPlugin {
		if len(args) == 0 {
			r.Log(robot.Error, "Added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		if len(args[0]) == 0 {
			r.Log(robot.Error, "Added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		cmsg := args[0]
		matched := false
		for _, matcher := range plugin.CommandMatchers {
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindAllStringSubmatch(cmsg, -1)
			if matches != nil {
				matched = true
				Log(robot.Trace, "Pipeline command '%s' matches '%s'", cmsg, matcher.Command)
				command = matcher.Command
				cmdargs = matches[0][1:]
				break
			}
		}
		if !matched {
			r.Log(robot.Error, "Command '%s' didn't match any CommandMatchers while adding plugin '%s' to pipeline", cmsg, name)
			return robot.CommandNotMatched
		}
	} else {
		command = "run"
		cmdargs = args
	}
	ts := TaskSpec{
		Name:      name,
		Command:   command,
		Arguments: cmdargs,
		task:      t,
	}
	argstr := strings.Join(args, " ")
	r.Log(robot.Debug, "Adding pipeline task %s/%s: %s %s", pflavor, ptype, name, argstr)
	w := getLockedWorker(r.tid)
	switch pflavor {
	case flavorAdd:
		w.nextTasks = append(w.nextTasks, ts)
		w.Unlock()
	case flavorFinal:
		// Final tasks are FILO/LIFO (run in reverse order of being added)
		w.finalTasks = append([]TaskSpec{ts}, w.finalTasks...)
		w.Unlock()
	case flavorFail:
		w.failTasks = append(w.failTasks, ts)
		w.Unlock()
	case flavorSpawn:
		w.Unlock()
		sb := w.clone()
		go sb.startPipeline(nil, t, spawnedTask, command, args...)
	}
	return robot.Ok
}

// see robot/robot.go
func (r Robot) SpawnJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorSpawn, typeJob, name, args...)
}

// see robot/robot.go
func (r Robot) AddTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeTask, name, args...)
}

// see robot/robot.go
func (r Robot) FinalTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFinal, typeTask, name, args...)
}

// see robot/robot.go
func (r Robot) FailTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFail, typeTask, name, args...)
}

// see robot/robot.go
func (r Robot) AddJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeJob, name, args...)
}

// see robot/robot.go
func (r Robot) AddCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorAdd, typePlugin, plugname, command)
}

// see robot/robot.go
func (r Robot) FinalCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFinal, typePlugin, plugname, command)
}

// see robot/robot.go
func (r Robot) FailCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFail, typePlugin, plugname, command)
}
