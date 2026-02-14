package bot

import "testing"

func TestSelectTaskExecutionRunnerAlwaysInProcessForGo(t *testing.T) {
	w := &worker{}
	goTask := &Task{name: "builtin-admin", taskType: taskGo}
	goPlugin := &Plugin{Task: &Task{name: "builtin-help", taskType: taskGo}}
	goJob := &Job{Task: &Task{name: "go-bootstrap", taskType: taskGo}}

	cases := []struct {
		name string
		task interface{}
	}{
		{name: "go task", task: goTask},
		{name: "go plugin", task: goPlugin},
		{name: "go job", task: goJob},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := w.selectTaskExecutionRunner(tc.task)
			if _, ok := runner.(inProcessTaskRunner); !ok {
				t.Fatalf("selectTaskExecutionRunner() runner type = %T, want inProcessTaskRunner", runner)
			}
		})
	}
}

func TestSelectTaskExecutionRunnerUsesProcessForExternalExecutables(t *testing.T) {
	w := &worker{}
	externalTask := &Task{name: "external-script", taskType: taskExternal, Path: "tasks/thing.sh"}
	externalPlugin := &Plugin{Task: &Task{name: "external-plugin", taskType: taskExternal, Path: "plugins/thing.py"}}
	externalJob := &Job{Task: &Task{name: "external-job", taskType: taskExternal, Path: "jobs/thing.rb"}}

	cases := []struct {
		name string
		task interface{}
	}{
		{name: "external task", task: externalTask},
		{name: "external plugin", task: externalPlugin},
		{name: "external job", task: externalJob},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := w.selectTaskExecutionRunner(tc.task)
			if _, ok := runner.(externalProcessTaskRunner); !ok {
				t.Fatalf("selectTaskExecutionRunner() runner type = %T, want externalProcessTaskRunner", runner)
			}
		})
	}
}

func TestSelectTaskExecutionRunnerKeepsInterpretersInProcess(t *testing.T) {
	w := &worker{}
	luaTask := &Task{name: "lua-task", taskType: taskExternal, Path: "plugins/thing.lua"}
	jsTask := &Task{name: "js-task", taskType: taskExternal, Path: "plugins/thing.js"}
	goTask := &Task{name: "yaegi-task", taskType: taskExternal, Path: "plugins/thing.go"}

	cases := []struct {
		name string
		task interface{}
	}{
		{name: "lua", task: luaTask},
		{name: "js", task: jsTask},
		{name: "go", task: goTask},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := w.selectTaskExecutionRunner(tc.task)
			if _, ok := runner.(inProcessTaskRunner); !ok {
				t.Fatalf("selectTaskExecutionRunner() runner type = %T, want inProcessTaskRunner", runner)
			}
		})
	}
}
