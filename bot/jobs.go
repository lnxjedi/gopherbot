package bot

// stuff needed to run a job
type jobSpec struct {
	Job        string   // name of the job being scheduled
	Parameters []string // parameters for the scheduled job
}

// items in gopherbot.yaml
type scheduledJob struct {
	Schedule string // timespec for https://godoc.org/github.com/robfig/cron
	jobSpec
}

// stuff read in conf/jobs/<job>.yaml
type botJob struct {
	name               string         // from gopherbot.yaml Jobs: array
	jobID              string         // used to identify the job from function calls
	NameSpace          string         // jobs/plugins with same namespace share long-term memories; defaults to the 'name'
	Path               string         // path to executable, normally jobs/<job>.sh (or rb, py, etc.)
	Channel            string         // where job status updates are posted
	Notify             string         // user to notify on failure; job runs with this User
	SuccessStatus      bool           // whether to send "job ran ok" message to Channel
	NotifySuccess      bool           // whether to notify the Notify user on sucess
	RequiredParameters []string       // required in schedule, prompted to user for interactive
	HistoryFiles       int            // how many history files to keep
	Channels           []string       // Channels where users can run this job
	Users              []string       // Users who can manually trigger this job with 'run job <foo>'
	ReplyMatchers      []InputMatcher // jobs can ask questions, too - just not triggered by commands / messages
	NextJob            jobSpec        // job and params to run if this job exits 0; rudimentary pipeline support
}

type jobList struct {
	j       []*botJob
	nameMap map[string]int
	idMap   map[string]int
}
