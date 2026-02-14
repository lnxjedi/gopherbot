package bot

import "github.com/lnxjedi/gopherbot/robot"

type taskExecutionRunner interface {
	runTask(w *worker, t interface{}, command string, args ...string) (errString string, retval robot.TaskRetVal)
}

type inProcessTaskRunner struct{}

func (inProcessTaskRunner) runTask(w *worker, t interface{}, command string, args ...string) (string, robot.TaskRetVal) {
	return w.callTask(t, command, args...)
}

func (w *worker) selectTaskExecutionRunner(t interface{}) taskExecutionRunner {
	task, _, _ := getTask(t)
	// Invariant: tasks implemented in bot/* (taskGo) always execute in-process.
	if task.taskType == taskGo {
		return inProcessTaskRunner{}
	}
	// Slice 1 foundation: non-Go tasks still execute in-process until process
	// runners and parent/child RPC are implemented.
	return inProcessTaskRunner{}
}

func (w *worker) executeTask(t interface{}, command string, args ...string) (string, robot.TaskRetVal) {
	runner := w.selectTaskExecutionRunner(t)
	return runner.runTask(w, t, command, args...)
}
