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
	botCfg.RLock()
	scheduled := botCfg.scheduledTasks
	tz := botCfg.timeZone
	botCfg.RUnlock()
	if tz != nil {
		Log(Info, fmt.Sprintf("Scheduling tasks in TimeZone: %s", tz))
		taskRunner = cron.NewWithLocation(tz)
	} else {
		Log(Info, "Scheduling tasks in system default timezone")
		taskRunner = cron.New()
	}
	currentTasks.Lock()
	tasks := taskList{
		currentTasks.t,
		currentTasks.nameMap,
		currentTasks.idMap,
		currentTasks.nameSpaces,
	}
	currentTasks.Unlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()
	for _, st := range scheduled {
		t := tasks.getTaskByName(st.Name)
		if t == nil {
			Log(Error, fmt.Sprintf("Task not found when scheduling task: %s", st.Name))
			continue
		}
		task, _, _ := getTask(t)
		if task.Disabled {
			Log(Error, fmt.Sprintf("Not scheduling disabled task '%s'; reason: %s", st.Name, task.reason))
			continue
		}
		// NOTE: bare tasks ALWAYS have zero-length Channel, and can't be scheduled by design;
		// TODO: A task would only need a Channel to be scheduled
		if len(task.Channel) == 0 {
			Log(Error, fmt.Sprintf("Not scheduling task '%s'; zero-length Channel", st.Name))
			continue
		}
		Log(Info, fmt.Sprintf("Scheduling job '%s' with schedule: %s", st.Name, st.Schedule))
		taskRunner.AddFunc(st.Schedule, func() { runScheduledTask(t, st.taskSpec, tasks, repolist) })
	}
	taskRunner.Start()
	schedMutex.Unlock()
}

func runScheduledTask(t interface{}, ts taskSpec, tasks taskList, repolist map[string]repository) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin && len(ts.Command) == 0 {
		Log(Error, fmt.Sprintf("Empty 'Command' when running scheduled task '%s' of type plugin", ts.Name))
		return
	}

	// Create the botContext to carry state through the pipeline.
	// startPipeline will take care of registerActive()
	bot := &botContext{
		Channel:          task.Channel,
		tasks:            tasks,
		repositories:     repolist,
		isCommand:        isPlugin,
		directMsg:        false,
		automaticTask:    true, // scheduled jobs don't get authorization / elevation checks
		workingDirectory: botCfg.workSpace,
		environment:      make(map[string]string),
	}
	var command string
	if isPlugin {
		command = ts.Command
	} else {
		command = "run"
	}
	Log(Debug, fmt.Sprintf("Starting scheduled task: %s", task.name))
	bot.startPipeline(nil, t, scheduled, command, ts.Arguments...)
}
