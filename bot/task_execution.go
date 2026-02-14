package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

type taskExecutionRunner interface {
	runTask(w *worker, t interface{}, command string, args ...string) (errString string, retval robot.TaskRetVal)
}

type inProcessTaskRunner struct{}
type externalProcessTaskRunner struct{}

func (inProcessTaskRunner) runTask(w *worker, t interface{}, command string, args ...string) (string, robot.TaskRetVal) {
	return w.callTask(t, command, args...)
}

func (externalProcessTaskRunner) runTask(w *worker, t interface{}, command string, args ...string) (string, robot.TaskRetVal) {
	opts := taskCallOptions{
		externalExecutableProcess: true,
	}
	return w.callTaskWithOptions(opts, t, command, args...)
}

func isExternalInterpreterTask(task *Task) bool {
	path := strings.ToLower(task.Path)
	return strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".lua") || strings.HasSuffix(path, ".js")
}

func (w *worker) selectTaskExecutionRunner(t interface{}) taskExecutionRunner {
	task, _, _ := getTask(t)
	// Invariant: tasks implemented in bot/* (taskGo) always execute in-process.
	if task.taskType == taskGo {
		return inProcessTaskRunner{}
	}
	// Interpreter-backed tasks remain in-process for now; external executables
	// run in a child process.
	if task.taskType == taskExternal && !isExternalInterpreterTask(task) {
		return externalProcessTaskRunner{}
	}
	return inProcessTaskRunner{}
}

func (w *worker) executeTask(t interface{}, command string, args ...string) (string, robot.TaskRetVal) {
	runner := w.selectTaskExecutionRunner(t)
	return runner.runTask(w, t, command, args...)
}
