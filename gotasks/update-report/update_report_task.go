package updatereport

import (
	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterTask("update-report", true, robot.TaskHandler{
		Handler: updateReportTask,
	})
}

func updateReportTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	branch := r.GetParameter("GOPHER_CUSTOM_BRANCH")
	r.Say("Sucessfully updated robot's git repository on branch '%s'", branch)
	return robot.Normal
}
