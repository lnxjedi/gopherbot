package bot

import (
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/robfig/cron"
)

var taskRunner *cron.Cron
var schedMutex sync.Mutex

func scheduleTasks() {
	schedMutex.Lock()
	if taskRunner != nil {
		taskRunner.Stop()
	}
	currentCfg.RLock()
	scheduled := currentCfg.ScheduledJobs
	tz := currentCfg.timeZone
	currentCfg.RUnlock()
	if tz != nil {
		Log(robot.Info, "Scheduling tasks in TimeZone: %s", tz)
		taskRunner = cron.NewWithLocation(tz)
	} else {
		Log(robot.Info, "Scheduling tasks in system default timezone")
		taskRunner = cron.New()
	}
	currentCfg.RLock()
	cfg := currentCfg.configuration
	tasks := currentCfg.taskList
	currentCfg.RUnlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()
	for _, st := range scheduled {
		t := tasks.getTaskByName(st.Name)
		if t == nil {
			Log(robot.Error, "Task not found when scheduling task: %s", st.Name)
			continue
		}
		task, _, job := getTask(t)
		if job == nil {
			Log(robot.Error, "Ignoring '%s' in ScheduledJobs: not a job", st.Name)
			continue
		}
		if task.Disabled {
			Log(robot.Error, "Not scheduling disabled job '%s'; reason: %s", st.Name, task.reason)
			continue
		}
		if len(task.Channel) == 0 {
			Log(robot.Error, "Not scheduling job '%s'; zero-length Channel", st.Name)
			continue
		}
		ts := st.TaskSpec
		Log(robot.Info, "Scheduling job '%s', args '%v' with schedule: %s", ts.Name, ts.Arguments, st.Schedule)
		taskRunner.AddFunc(st.Schedule, func() { runScheduledTask(t, ts, cfg, tasks, repolist) })
	}
	taskRunner.Start()
	schedMutex.Unlock()
}

func runScheduledTask(t interface{}, ts TaskSpec, cfg *configuration, tasks *taskList, repolist map[string]robot.Repository) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin && len(ts.Command) == 0 {
		Log(robot.Error, "Empty 'Command' when running scheduled task '%s' of type plugin", ts.Name)
		return
	}

	// Create the pipeContext to carry state through the pipeline.
	// startPipeline will take care of registerActive()
	w := &worker{
		Channel:       task.Channel,
		cfg:           cfg,
		tasks:         tasks,
		repositories:  repolist,
		isCommand:     isPlugin,
		directMsg:     false,
		automaticTask: true, // scheduled jobs don't get authorization / elevation checks
	}
	var command string
	if isPlugin {
		command = ts.Command
	} else {
		command = "run"
	}
	Log(robot.Info, "Starting scheduled task: %s", task.name)
	w.startPipeline(nil, t, scheduled, command, ts.Arguments...)
}
