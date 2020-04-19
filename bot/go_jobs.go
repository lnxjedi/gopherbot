package bot

import (
	"strings"

	"github.com/lnxjedi/robot"
)

func pausenotifies(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	notifies := make(map[string][]string)
	pausedJobs.Lock()
	for job, user := range pausedJobs.jobs {
		joblist, ok := notifies[user]
		if ok {
			joblist = append(joblist, job)
		} else {
			joblist = []string{job}
		}
		notifies[user] = joblist
	}
	pausedJobs.Unlock()
	for user, jobs := range notifies {
		ret := m.SendUserMessage(user, "Reminder, you have paused jobs: %s", strings.Join(jobs, ", "))
		if ret != robot.Ok {
			m.Log(robot.Error, "Failed sending reminder to user %s about paused jobs: %s", user, strings.Join(jobs, ", "))
		}
	}
	return
}

func init() {
	RegisterJob("pause-notifies", robot.JobHandler{Handler: pausenotifies})
}
