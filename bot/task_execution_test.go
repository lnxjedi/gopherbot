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

func TestSelectTaskExecutionRunnerDefaultsExternalToInProcessInSlice1(t *testing.T) {
	w := &worker{}
	externalTask := &Task{name: "external-script", taskType: taskExternal}
	externalPlugin := &Plugin{Task: &Task{name: "external-plugin", taskType: taskExternal}}
	externalJob := &Job{Task: &Task{name: "external-job", taskType: taskExternal}}

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
			if _, ok := runner.(inProcessTaskRunner); !ok {
				t.Fatalf("selectTaskExecutionRunner() runner type = %T, want inProcessTaskRunner", runner)
			}
		})
	}
}
