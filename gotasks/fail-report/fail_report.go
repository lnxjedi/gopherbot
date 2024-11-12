package failreport

import (
	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterTask("fail-report", true, robot.TaskHandler{
		Handler: failReportTask,
	})
}

func failReportTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	pipename := r.GetParameter("GOPHER_PIPE_NAME")
	r.Say("Pipeline failed: " + pipename)
	return robot.Normal
}
