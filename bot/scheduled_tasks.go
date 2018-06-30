package bot

import (
	"fmt"
	"sync"

	"github.com/robfig/cron"
)

var taskRunner *cron.Cron
var schedMutex sync.Mutex

func scheduleTasks() {
	schedMutex.Lock()
	if taskRunner != nil {
		taskRunner.Stop()
	}
	taskRunner = cron.New()
	robot.RLock()
	scheduled := robot.scheduledTasks
	robot.RUnlock()
	for _, st := range scheduled {
		Log(Info, fmt.Sprintf("Scheduling job '%s' with schedule: %s", st.taskSpec.Name, st.Schedule))
		taskRunner.AddFunc(st.Schedule, func() { runScheduledTask(st.taskSpec) })
	}
	taskRunner.Start()
	schedMutex.Unlock()
}

func runScheduledTask(ts taskSpec) {
	currentTasks.RLock()
	tasks := taskList{
		currentTasks.t,
		currentTasks.nameMap,
		currentTasks.idMap,
		currentTasks.nameSpaces,
		sync.RWMutex{},
	}
	currentTasks.RUnlock()
	t := tasks.getTaskByName(ts.Name)
	if t == nil {
		Log(Error, fmt.Sprintf("Task not found when running scheduled task: %s", ts.Name))
		return
	}
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin && len(ts.Command) == 0 {
		Log(Error, fmt.Sprintf("Empty 'Command' when running scheduled task '%s' of type plugin", ts.Name))
		return
	}

	// Create the botContext to carry state through the pipeline.
	// runPipeline will take care of registerActive()
	bot := &botContext{
		User:                 task.User,
		Channel:              task.Channel,
		tasks:                tasks,
		isCommand:            isPlugin,
		directMsg:            false,
		bypassSecurityChecks: true, // scheduled jobs don't get authorization / elevation checks
		environment:          make(map[string]string),
	}
	var command string
	args := make([]string, 0, 0)
	if isPlugin {
		command = ts.Command
		args = append(args, ts.Arguments...)
	} else {
		command = "run"
		for _, p := range ts.Parameters {
			bot.environment[p.Name] = p.Value
		}
	}
	Log(Debug, fmt.Sprintf("Starting scheduled task: %s", task.name))
	bot.runPipeline(t, false, scheduled, command, args...)
}
