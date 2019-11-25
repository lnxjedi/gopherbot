package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// GetRepoData returns the contents of configPath/conf/repodata.yaml, or an
// empty map/dict/hash. Mainly for GopherCI, Methods for Python and Ruby will
// retrieve it. Returns nil and logs an error if the calling task isn't a job,
// or the namespace has already been extended.
func (r *Robot) GetRepoData() map[string]json.RawMessage {
	c := r.getContext()
	t, p, j := getTask(c.currentTask)
	if j == nil && p == nil {
		r.Log(robot.Error, "GetRepoData called by non-job/plugin task '%s'", t.name)
		return nil
	}
	if len(c.nsExtension) > 0 {
		r.Log(robot.Error, "GetRepoData called with namespace extended: '%s'", c.nsExtension)
		return nil
	}
	confLock.RLock()
	defer confLock.RUnlock()
	return repodata
}

// ExtendNamespace is for CI/CD applications to support building multiple
// repositories from a single triggered job. When ExtendNamespace is called,
// all future long-term memory lookups are prefixed with the extended
// namespace, and a new history is started for the extended namespace.
// It is an error to call ExtendNamespace twice in a single job pipeline, or
// outside of a running job. The histories argument is interpreted as the
// number of histories to keep for the extended namespace, or -1 to inherit
// from the parent job.
// Arguments:
// ext (extension) => "<repository>/<branch>", where repository is listed in
//   repositories.yaml
// histories => number of histories to keep
func (r *Robot) ExtendNamespace(ext string, histories int) bool {
	if strings.ContainsRune(ext, ':') {
		r.Log(robot.Error, "Invalid namespace extension contains ':'")
		return false
	}
	c := r.getContext()
	if c.stage != primaryTasks {
		r.Log(robot.Error, "ExtendNamespace called after pipeline end")
		return false
	}
	if len(c.jobName) == 0 {
		r.Log(robot.Error, "ExtendNamespace called with no job in progress")
		return false
	}
	if len(c.nsExtension) > 0 {
		r.Log(robot.Error, "ExtendNamespace called after namespace already extended")
		return false
	}
	cmp := strings.Split(ext, "/")
	repo := strings.Join(cmp[0:len(cmp)-1], "/")
	branch := cmp[len(cmp)-1]
	repository, exists := c.repositories[repo]
	if !exists {
		r.Log(robot.Error, "Repository '%s' not found in repositories.yaml (missing branch in call to ExtendNamespace?)", repo)
		return false
	}
	r.Log(robot.Debug, "Extending namespace for job '%s': %s (branch %s)", c.jobName, repo, branch)
	c.nsExtension = ext
	c.environment["GOPHER_NAMESPACE_EXTENDED"] = repo

	jk := histPrefix + c.jobName
	var pjh jobHistory
	jtok, _, jret := checkoutDatum(jk, &pjh, true)
	if jret != robot.Ok {
		r.Log(robot.Error, "Problem checking out '%s', unable to record extended namespace '%s'", jk, ext)
	} else {
		xn := make(map[string]bool)
		for _, v := range pjh.ExtendedNamespaces {
			xn[v] = true
		}
		xn[ext] = true
		pjh.ExtendedNamespaces = make([]string, len(xn))
		i := 0
		for k := range xn {
			pjh.ExtendedNamespaces[i] = k
			i++
		}
		ret := updateDatum(jk, jtok, pjh)
		if ret != robot.Ok {
			r.Log(robot.Error, "Problem updating '%s', unable to record extended namespace '%s'", jk, ext)
		}
	}

	var nh int
	if histories != -1 {
		nh = histories
	} else {
		j := c.tasks.getTaskByName(c.jobName)
		_, _, job := getTask(j)
		nh = job.HistoryLogs
	}
	var jh jobHistory
	rememberRuns := nh
	if rememberRuns == 0 {
		rememberRuns = 1
	}
	key := histPrefix + c.jobName + ":" + ext
	tok, _, ret := checkoutDatum(key, &jh, true)
	if ret != robot.Ok {
		Log(robot.Error, "Error checking out '%s', no history will be remembered for '%s'", key, c.pipeName)
	} else {
		var start time.Time
		if c.timeZone != nil {
			start = time.Now().In(c.timeZone)
		} else {
			start = time.Now()
		}
		c.runIndex = jh.NextIndex
		c.environment["GOPHER_RUN_INDEX"] = fmt.Sprintf("%d", c.runIndex)
		hist := historyLog{
			LogIndex:   c.runIndex,
			CreateTime: start.Format("Mon Jan 2 15:04:05 MST 2006"),
		}
		jh.NextIndex++
		jh.Histories = append(jh.Histories, hist)
		l := len(jh.Histories)
		if l > rememberRuns {
			jh.Histories = jh.Histories[l-rememberRuns:]
		}
		ret := updateDatum(key, tok, jh)
		if ret != robot.Ok {
			Log(robot.Error, "Error updating '%s', no history will be remembered for '%s'", key, c.pipeName)
		} else {
			if nh > 0 && c.history != nil {
				hspec := c.pipeName + ":" + ext
				pipeHistory, err := c.history.NewHistory(hspec, hist.LogIndex, nh)
				if err != nil {
					Log(robot.Error, "Error starting history for '%s', no history will be recorded: %v", c.pipeName, err)
				} else {
					if c.logger != nil {
						c.logger.Section("close log", fmt.Sprintf("Job '%s' extended namespace: '%s'; starting new log on next task", c.jobName, ext))
					}
					c.logger = pipeHistory
					c.logger.Section("new log", fmt.Sprintf("Extended log created by job '%s'", c.jobName))
					r.Log(robot.Debug, "Started new history for job '%s' with namespace '%s'", c.jobName, ext)
					r.Channel = c.jobChannel
					var link string
					if url, ok := c.history.GetHistoryURL(hspec, hist.LogIndex); ok {
						link = fmt.Sprintf(" (link: %s)", url)
					}
					r.Say("Job '%s' extended namespace: %s:%s, run %d%s", c.jobName, c.jobName, ext, c.runIndex, link)
				}
			} else {
				if c.history == nil {
					Log(robot.Warn, "Error starting history, no history provider available")
				}
			}
		}
	}
	for _, param := range repository.Parameters {
		name := param.Name
		value := param.Value
		_, exists := c.environment[name]
		if !exists {
			c.environment[name] = value
		}
	}
	// Populate the environment with encrypted parameters for this repository.
	// Task secrets are populated in runtasks.go/callTask. Note that repo+branch
	// is checked before the repo, so the more specific parameters take
	// precedence.
	cryptKey.RLock()
	initialized := cryptKey.initialized
	ckey := cryptKey.key
	cryptKey.RUnlock()
	if initialized {
		// check repo/branch (ext) first
		repEnv, exists := c.storedEnv.RepositoryParams[ext]
		if exists {
			if initialized {
				for name, encvalue := range repEnv {
					_, exists := c.environment[name]
					if !exists {
						value, err := decrypt(encvalue, ckey)
						if err != nil {
							Log(robot.Error, "Error decrypting '%s' for repository/branch '%s': %v", name, ext, err)
							break
						}
						c.environment[name] = string(value)
					}
				}
			}
		}
		// ... then check repo
		repEnv, exists = c.storedEnv.RepositoryParams[repo]
		if exists {
			if initialized {
				for name, encvalue := range repEnv {
					_, exists := c.environment[name]
					if !exists {
						value, err := decrypt(encvalue, ckey)
						if err != nil {
							Log(robot.Error, "Error decrypting '%s' for repository '%s': %v", name, repo, err)
							break
						}
						c.environment[name] = string(value)
					}
				}
			}
		}
	}
	return true
}

// pipeTask does all the real work of adding tasks to pipelines or spawning
// new tasks.
func (r *Robot) pipeTask(pflavor pipeAddFlavor, ptype pipeAddType, name string, args ...string) robot.RetVal {
	c := r.getContext()
	if c.stage != primaryTasks {
		task, _, _ := getTask(c.currentTask)
		r.Log(robot.Error, "request to modify pipeline outside of initial pipeline in task '%s'", task.name)
		return robot.InvalidStage
	}
	t := c.tasks.getTaskByName(name)
	if t == nil {
		task, _, _ := getTask(c.currentTask)
		r.Log(robot.Error, "task '%s' not found updating pipeline from task '%s'", name, task.name)
		return robot.TaskNotFound
	}
	task, plugin, job := getTask(t)
	isPlugin := plugin != nil
	isJob := job != nil
	if task.Disabled {
		r.Log(robot.Error, "attempt to add disabled task '%s' to pipeline", name)
		return robot.TaskDisabled
	}
	if ptype == typePlugin && !isPlugin {
		r.Log(robot.Error, "adding command to pipeline - not a plugin: %s", name)
		return robot.InvalidTaskType
	}
	if ptype == typeJob && !isJob {
		r.Log(robot.Error, "adding job to pipeline - not a job: %s", name)
		return robot.InvalidTaskType
	}
	if ptype == typeTask && (isJob || isPlugin) {
		r.Log(robot.Error, "adding task to pipeline - not a task: %s", name)
		return robot.InvalidTaskType
	}
	var command string
	var cmdargs []string
	if isPlugin {
		if len(args) == 0 {
			r.Log(robot.Error, "added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		if len(args[0]) == 0 {
			r.Log(robot.Error, "added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		cmsg := args[0]
		c.debugT(t, fmt.Sprintf("Checking %d command matchers against pipe command: '%s'", len(plugin.CommandMatchers), cmsg), false)
		matched := false
		for _, matcher := range plugin.CommandMatchers {
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindAllStringSubmatch(cmsg, -1)
			if matches != nil {
				c.debugT(t, fmt.Sprintf("Matched command regex '%s', command: %s", matcher.Regex, matcher.Command), false)
				matched = true
				Log(robot.Trace, "pipeline command '%s' matches '%s'", cmsg, matcher.Command)
				command = matcher.Command
				cmdargs = matches[0][1:]
				break
			} else {
				c.debugT(t, fmt.Sprintf("Not matched: %s", matcher.Regex), false)
			}
		}
		if !matched {
			r.Log(robot.Error, "Command '%s' didn't match any CommandMatchers while adding plugin '%s' to pipeline", cmsg, name)
			return robot.CommandNotMatched
		}
	} else {
		command = "run"
		cmdargs = args
	}
	ts := TaskSpec{
		Name:      name,
		Command:   command,
		Arguments: cmdargs,
		task:      t,
	}
	argstr := strings.Join(args, " ")
	r.Log(robot.Debug, "Adding pipeline task %s/%s: %s %s", pflavor, ptype, name, argstr)
	switch pflavor {
	case flavorAdd:
		c.nextTasks = append(c.nextTasks, ts)
	case flavorFinal:
		// Final tasks are FILO/LIFO (run in reverse order of being added)
		c.finalTasks = append([]TaskSpec{ts}, c.finalTasks...)
	case flavorFail:
		c.failTasks = append(c.failTasks, ts)
	case flavorSpawn:
		sb := c.clone()
		go sb.startPipeline(nil, t, spawnedTask, command, args...)
	}
	return robot.Ok
}

// SpawnJob creates a new botContext in a new goroutine to run a
// job. It's primary use is for CI/CD applications where a single
// triggered job may want to spawn several jobs when e.g. a dependency for
// multiple projects is updated.
func (r *Robot) SpawnJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorSpawn, typeJob, name, args...)
}

// AddTask puts another task (job or plugin) in the queue for the pipeline. Unlike other
// CI/CD tools, gopherbot pipelines are code generated, not configured; it is,
// however, trivial to write code that reads an arbitrary configuration file
// and uses AddTask to generate a pipeline. When the task is a plugin, cmdargs
// should be a command followed by arguments. For jobs, cmdargs are just
// arguments passed to the job.
func (r *Robot) AddTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeTask, name, args...)
}

// FinalTask adds a task that always runs when the pipeline ends, whether
// it succeeded or failed. This can be used to ensure that cleanup tasks like
// terminating a VM or stopping the ssh-agent will run, regardless of whether
// the pipeline failed.
// Note that unlike other tasks, final tasks are run in reverse of the order
// they're added.
func (r *Robot) FinalTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFinal, typeTask, name, args...)
}

// FailTask adds a task that runs only if the pipeline fails. This can be used
// to e.g. notify a user / channel on failure.
func (r *Robot) FailTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFail, typeTask, name, args...)
}

// AddJob puts another job in the queue for the pipeline. The added job
// will run in a new separate context, and when it completes the current
// pipeline will resume if the job succeeded.
func (r *Robot) AddJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeJob, name, args...)
}

// AddCommand adds a plugin command to the pipeline. The command string
// argument should match a CommandMatcher for the given plugin.
func (r *Robot) AddCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorAdd, typePlugin, plugname, command)
}

// FinalCommand adds a plugin command that always runs when a pipeline
// ends, for e.g. emailing the job history. The command string
// argument should match a CommandMatcher for the given plugin.
func (r *Robot) FinalCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFinal, typePlugin, plugname, command)
}

// FailCommand adds a plugin command that runs whenever a pipeline fails,
// for e.g. emailing the job history. The command string
// argument should match a CommandMatcher for the given plugin.
func (r *Robot) FailCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFail, typePlugin, plugname, command)
}
