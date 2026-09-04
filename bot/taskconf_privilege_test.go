package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func boolPointer(value bool) *bool {
	return &value
}

func TestLoadTaskConfigAppliesExplicitGoTaskPrivilegeOverride(t *testing.T) {
	originalTasks := taskHandlers
	originalPlugins := pluginHandlers
	originalJobs := jobHandlers
	currentCfg.Lock()
	originalList := currentCfg.taskList
	currentCfg.Unlock()
	t.Cleanup(func() {
		taskHandlers = originalTasks
		pluginHandlers = originalPlugins
		jobHandlers = originalJobs
		currentCfg.Lock()
		currentCfg.taskList = originalList
		currentCfg.Unlock()
	})

	tests := []struct {
		name       string
		registered bool
		configured *bool
		want       bool
	}{
		{name: "explicit false overrides privileged registration", registered: true, configured: boolPointer(false), want: false},
		{name: "explicit true overrides unprivileged registration", registered: false, configured: boolPointer(true), want: true},
		{name: "omission preserves registration default", registered: true, configured: nil, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			const taskName = "configurable-go-task"
			registeredTask := &Task{name: taskName, taskType: taskGo, Privileged: tc.registered}
			baseList := &taskList{
				t:             []interface{}{struct{}{}},
				nameMap:       make(map[string]int),
				idMap:         make(map[string]int),
				uuidTriggers:  make(map[string]interface{}),
				nameSpaces:    make(map[string]ParameterSet),
				parameterSets: make(map[string]ParameterSet),
			}
			baseList.addTask(registeredTask)

			taskHandlers = map[string]robot.TaskHandler{taskName: {}}
			pluginHandlers = make(map[string]robot.PluginHandler)
			jobHandlers = make(map[string]robot.JobHandler)
			currentCfg.Lock()
			currentCfg.taskList = baseList
			currentCfg.Unlock()

			processed := &configuration{goTasks: []TaskSettings{{Name: taskName, Privileged: tc.configured}}}
			loaded, err := loadTaskConfig(processed, true)
			if err != nil {
				t.Fatalf("loadTaskConfig() error = %v", err)
			}
			task, _, _ := getTask(loaded.getTaskByName(taskName))
			if task.Privileged != tc.want {
				t.Fatalf("loaded task Privileged = %t, want %t", task.Privileged, tc.want)
			}
		})
	}
}
