package bot

import (
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/robfig/cron"
)

var taskRunner *cron.Cron
var schedMutex sync.Mutex

var pausedJobs = struct {
	jobs map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

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
		if st.Schedule != "@init" {
			Log(robot.Info, "Scheduling job '%s', args '%v' with schedule: %s", ts.Name, ts.Arguments, st.Schedule)
			taskRunner.AddFunc(st.Schedule, func() { runScheduledTask(t, ts, cfg, tasks, false) })
		}
	}
	taskRunner.Start()
	schedMutex.Unlock()
}

// initJobs - run init jobs that might be required by external plugins
func initJobs() {
	currentCfg.RLock()
	scheduled := currentCfg.ScheduledJobs
	cfg := currentCfg.configuration
	tasks := currentCfg.taskList
	currentCfg.RUnlock()
	for _, st := range scheduled {
		ts := st.TaskSpec
		if st.Schedule == "@init" {
			t := tasks.getTaskByName(st.Name)
			if t == nil {
				Log(robot.Error, "Task not found when running init job: %s", st.Name)
				continue
			}
			task, _, job := getTask(t)
			if job == nil {
				Log(robot.Error, "Ignoring '%s' while running init jobs: not a job", st.Name)
				continue
			}
			if task.Disabled {
				Log(robot.Error, "Ignoring disabled job '%s' while running init jobs; reason: %s", st.Name, task.reason)
				continue
			}
			runScheduledTask(t, ts, cfg, tasks, true)
		}
	}
}

func runScheduledTask(t interface{}, ts TaskSpec, cfg *configuration, tasks *taskList, isInitJob bool) {
	task, _, _ := getTask(t)
	currentCfg.RLock()
	protocol := currentCfg.protocol
	currentCfg.RUnlock()
	pausedJobs.Lock()
	if user, ok := pausedJobs.jobs[task.name]; ok {
		Log(robot.Debug, "Skipping run of job '%s' paused by user '%s'", task.name, user)
		pausedJobs.Unlock()
		return
	}
	pausedJobs.Unlock()

	// Create the pipeContext to carry state through the pipeline.
	// startPipeline will take care of registerActive()
	w := &worker{
		Channel:       task.Channel,
		Protocol:      getProtocol(protocol),
		Incoming:      &robot.ConnectorMessage{},
		cfg:           cfg,
		id:            getWorkerID(),
		tasks:         tasks,
		automaticTask: true, // scheduled jobs don't get authorization / elevation checks
	}
	if isInitJob {
		Log(robot.Debug, "Starting init job: %s", task.name)
	} else {
		Log(robot.Debug, "Starting scheduled job: %s", task.name)
	}
	jobtype := scheduled
	if isInitJob {
		jobtype = initJob
	}
	w.startPipeline(nil, t, jobtype, "run", ts.Arguments...)
}
