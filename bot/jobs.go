package bot

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// stuff needed to run a job
type jobSpec struct {
	Job        string   // name of the job being scheduled
	Parameters []string // parameters for the scheduled job
}

// Job parameters are provided to jobs as environment variables
type jobParameter struct {
	Name, Value string
}

// items in gopherbot.yaml
type scheduledJob struct {
	Schedule string // timespec for https://godoc.org/github.com/robfig/cron
	jobSpec
}

// stuff read in conf/jobs/<job>.yaml
type botJob struct {
	jobPath            string   // path to executable, normally jobs/<job>.sh (or rb, py, etc.)
	Channel            string   // where job status updates are posted
	Notify             string   // user to notify on failure; job runs with this User
	SuccessStatus      bool     // whether to send "job ran ok" message to Channel
	NotifySuccess      bool     // whether to notify the Notify user on sucess
	RequiredParameters []string // required in schedule, prompted to user for interactive
	HistoryFiles       int      // how many history files to keep
	Channels           []string // Channels where users can run this job
	Users              []string // Users who can manually trigger this job with 'run job <foo>'
	NextJob            jobSpec  // job and params to run if this job exits 0; rudimentary pipeline support
	botCaller
}

type jobList struct {
	j       []*botJob
	nameMap map[string]int
	idMap   map[string]int
}

// Global persistent map of job name to unique ID
var jobNameIDmap = struct {
	m map[string]string
	sync.Mutex
}{
	make(map[string]string),
	sync.Mutex{},
}

func getJobID(job string) string {
	jobNameIDmap.Lock()
	callerID, ok := jobNameIDmap.m[job]
	if ok {
		jobNameIDmap.Unlock()
		return callerID
	} else {
		// Generate a random id
		p := make([]byte, 16)
		rand.Read(p)
		callerID = fmt.Sprintf("%x", p)
		jobNameIDmap.m[job] = callerID
		jobNameIDmap.Unlock()
		return callerID
	}
}
