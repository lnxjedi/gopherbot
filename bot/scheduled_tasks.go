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
		Log(Debug, fmt.Sprintf("Scheduling job '%s' with schedule: %s", st.taskSpec.Name, st.Schedule))
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
		sync.RWMutex{},
	}
	currentTasks.RUnlock()
	robot.RLock()
	proto := setProtocol(robot.protocol)
	robot.RUnlock()
	t := tasks.getTaskByName(ts.Name)
	if t == nil {
		Log(Error, "Task not found when running scheduled task: %s", ts.Name)
	}
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil

	// Create the botContext to carry state through the pipeline.
	// runPipeline will take care of registerActive()
	bot := &botContext{
		User:        task.User,
		Channel:     task.Channel,
		Protocol:    proto,
		tasks:       tasks,
		isCommand:   isPlugin,
		directMsg:   false,
		environment: make(map[string]string),
	}
	m := &InputMatcher{
		Command: "run",
	}
	args := make([]string, 0, 0)
	if isPlugin {
		args = append(args, ts.Arguments...)
	} else {
		for _, p := range ts.Parameters {
			bot.environment[p.Name] = p.Value
		}
	}
	Log(Debug, fmt.Sprintf("Starting scheduled task: %s", task.name))
	bot.runPipeline(t, false, m, args...)
}
