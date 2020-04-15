package bot

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/robot"
)

// GetRepoData returns the contents of configPath/conf/repodata.yaml, or an
// empty map/dict/hash. Mainly for GopherCI, Methods for Python and Ruby will
// retrieve it. Returns nil and logs an error if the calling task isn't a job,
// or the namespace has already been extended.
func (r Robot) GetRepoData() map[string]robot.Repository {
	t, p, j := getTask(r.currentTask)
	if j == nil && p == nil {
		r.Log(robot.Error, "GetRepoData called by non-job/plugin task '%s'", t.name)
		return nil
	}
	if len(r.nsExtension) > 0 {
		r.Log(robot.Error, "GetRepoData called with namespace extended: '%s'", r.nsExtension)
		return nil
	}
	export := make(map[string]robot.Repository)
	for r, d := range r.repositories {
		e := robot.Repository{
			Type:         d.Type,
			CloneURL:     d.CloneURL,
			Dependencies: d.Dependencies,
			KeepHistory:  d.KeepHistory,
			Parameters:   []robot.Parameter{},
		}
		export[r] = e
	}
	return export
}

// ExtendNamespace is for CI/CD applications to support building multiple
// repositories from a single triggered job. When ExtendNamespace is called,
// all future long-term memory lookups are prefixed with the extended
// namespace, and a new history is started for the extended namespace,
// including the branch (needed to differentiate histories for differing
// branches).
// It is an error to call ExtendNamespace twice in a single job pipeline, or
// outside of a running job. The histories argument is interpreted as the
// number of histories to keep for the extended namespace, or -1 to inherit
// from the parent job. The jobName must match the repository Type, to protect
// secret parameters stored in repositories.yaml.
// Arguments:
// ext (extension) => "<repository>/<branch>", where repository is listed in
//   repositories.yaml
// histories => number of histories to keep
func (r Robot) ExtendNamespace(ext string, histories int) bool {
	if strings.ContainsRune(ext, ':') {
		r.Log(robot.Error, "Invalid namespace extension contains ':'")
		return false
	}
	if r.stage != primaryTasks {
		r.Log(robot.Error, "ExtendNamespace called after pipeline end")
		return false
	}
	if len(r.jobName) == 0 {
		r.Log(robot.Error, "ExtendNamespace called with no job in progress")
		return false
	}
	if len(r.nsExtension) > 0 {
		r.Log(robot.Error, "ExtendNamespace called after namespace already extended")
		return false
	}
	cmp := strings.Split(ext, "/")
	repo := strings.Join(cmp[0:len(cmp)-1], "/")
	branch := cmp[len(cmp)-1]
	repository, exists := r.repositories[repo]
	if !exists {
		r.Log(robot.Error, "Repository '%s' not found in repositories.yaml (missing branch in call to ExtendNamespace?)", repo)
		return false
	}
	if r.jobName != repository.Type {
		r.Log(robot.Error, "space called with jobName(%s) != repository Type(%s)", r.jobName, repository.Type)
		return false
	}
	r.Log(robot.Debug, "Extending namespace for job '%s': %s (branch %s)", r.jobName, repo, branch)
	w := getLockedWorker(r.tid)
	w.nsExtension = ext
	// old, deprecated, TODO: remove me someday
	w.environment["GOPHER_NAMESPACE_EXTENDED"] = repo
	// new hotness
	w.environment["GOPHER_REPOSITORY"] = repo
	jobLogger := w.logger
	wid := w.id
	eid := w.eid
	w.Unlock()

	jk := histPrefix + r.jobName
	var pjh pipeHistory
	jtok, _, jret := checkoutDatum(jk, &pjh, true)
	if jret != robot.Ok {
		r.Log(robot.Error, "Problem checking out '%s', unable to record extended namespace '%s'", jk, ext)
	} else {
		if len(pjh.ExtendedNamespaces) == 0 {
			pjh.ExtendedNamespaces = []string{ext}
			ret := updateDatum(jk, jtok, pjh)
			if ret != robot.Ok {
				r.Log(robot.Error, "Problem updating '%s', unable to record extended namespace '%s'", jk, ext)
			}
		} else {
			found := false
			for _, ns := range pjh.ExtendedNamespaces {
				if ns == ext {
					found = true
					break
				}
			}
			if !found {
				pjh.ExtendedNamespaces = append(pjh.ExtendedNamespaces, ext)
				ret := updateDatum(jk, jtok, pjh)
				if ret != robot.Ok {
					r.Log(robot.Error, "Problem updating '%s', unable to record extended namespace '%s'", jk, ext)
				}
			} else {
				checkinDatum(jk, jtok)
			}
		}
	}

	tag := r.jobName + ":" + repo
	var nh int
	if histories != -1 {
		nh = histories
	} else {
		j := r.tasks.getTaskByName(r.jobName)
		_, _, job := getTask(j)
		nh = job.HistoryLogs
	}
	pipeHistory, link, ref, idx := newLogger(tag, eid, branch, wid, nh)
	w.section("close log", fmt.Sprintf("Job '%s' extended namespace: '%s'; starting new log on next task", r.jobName, ext))
	jobLogger.Close()
	jobLogger.Finalize()
	w.Lock()
	w.histName = tag
	w.runIndex = idx
	// support old method using AddCommand for now
	w.environment["GOPHER_RUN_INDEX"] = fmt.Sprintf("%d", idx)
	if nh > 0 {
		if len(link) > 0 {
			w.environment["GOPHER_LOG_LINK"] = link
		} else {
			delete(w.environment, "GOPHER_LOG_LINK")
		}
		if len(ref) > 0 {
			w.environment["GOPHER_LOG_REF"] = ref
		} else {
			delete(w.environment, "GOPHER_LOG_REF")
		}
	}
	// Question: should repository parameters override job parameters, but _not_
	// parameters set with SetParameter?
	for _, param := range repository.Parameters {
		name := param.Name
		value := param.Value
		w.environment[name] = value
	}
	w.logger = pipeHistory
	w.Unlock()
	w.section("new log", fmt.Sprintf("Extended log created by job '%s'", r.jobName))
	r.Log(robot.Debug, "Started new history for job '%s' with namespace '%s'", r.jobName, ext)
	return true
}

// pipeTask does all the real work of adding tasks to pipelines or spawning
// new tasks.
func (r Robot) pipeTask(pflavor pipeAddFlavor, ptype pipeAddType, name string, args ...string) robot.RetVal {
	if r.stage != primaryTasks {
		task, _, _ := getTask(r.currentTask)
		r.Log(robot.Error, "request to modify pipeline outside of initial pipeline in task '%s'", task.name)
		return robot.InvalidStage
	}
	t := r.tasks.getTaskByName(name)
	if t == nil {
		task, _, _ := getTask(r.currentTask)
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
		r.Log(robot.Error, "Adding job to pipeline - not a job: %s", name)
		return robot.InvalidTaskType
	}
	if ptype == typeTask && (isJob || isPlugin) {
		r.Log(robot.Error, "Adding task to pipeline - not a task: %s", name)
		return robot.InvalidTaskType
	}
	if !r.privileged {
		if isJob && job.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged job '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
		if isPlugin && plugin.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged plugin '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
		if !isJob && !isPlugin && task.Privileged {
			r.Log(robot.Error, "PrivilegeViolation adding privileged task '%s' to unprivileged pipeline", name)
			return robot.PrivilegeViolation
		}
	}
	var command string
	var cmdargs []string
	if isPlugin {
		if len(args) == 0 {
			r.Log(robot.Error, "Added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		if len(args[0]) == 0 {
			r.Log(robot.Error, "Added plugin '%s' to pipeline with no command", name)
			return robot.MissingArguments
		}
		cmsg := args[0]
		debugT(t, fmt.Sprintf("Checking %d command matchers against pipe command: '%s'", len(plugin.CommandMatchers), cmsg), false)
		matched := false
		for _, matcher := range plugin.CommandMatchers {
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindAllStringSubmatch(cmsg, -1)
			if matches != nil {
				debugT(t, fmt.Sprintf("Matched command regex '%s', command: %s", matcher.Regex, matcher.Command), false)
				matched = true
				Log(robot.Trace, "pipeline command '%s' matches '%s'", cmsg, matcher.Command)
				command = matcher.Command
				cmdargs = matches[0][1:]
				break
			} else {
				debugT(t, fmt.Sprintf("Not matched: %s", matcher.Regex), false)
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
	w := getLockedWorker(r.tid)
	switch pflavor {
	case flavorAdd:
		w.nextTasks = append(w.nextTasks, ts)
		w.Unlock()
	case flavorFinal:
		// Final tasks are FILO/LIFO (run in reverse order of being added)
		w.finalTasks = append([]TaskSpec{ts}, w.finalTasks...)
		w.Unlock()
	case flavorFail:
		w.failTasks = append(w.failTasks, ts)
		w.Unlock()
	case flavorSpawn:
		w.Unlock()
		sb := w.clone()
		go sb.startPipeline(nil, t, spawnedTask, command, args...)
	}
	return robot.Ok
}

// SpawnJob creates a new pipeContext in a new goroutine to run a
// job. It's primary use is for CI/CD applications where a single
// triggered job may want to spawn several jobs when e.g. a dependency for
// multiple projects is updated.
func (r Robot) SpawnJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorSpawn, typeJob, name, args...)
}

// AddTask puts another task (job or plugin) in the queue for the pipeline. Unlike other
// CI/CD tools, gopherbot pipelines are code generated, not configured; it is,
// however, trivial to write code that reads an arbitrary configuration file
// and uses AddTask to generate a pipeline. When the task is a plugin, cmdargs
// should be a command followed by arguments. For jobs, cmdargs are just
// arguments passed to the job.
func (r Robot) AddTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeTask, name, args...)
}

// FinalTask adds a task that always runs when the pipeline ends, whether
// it succeeded or failed. This can be used to ensure that cleanup tasks like
// terminating a VM or stopping the ssh-agent will run, regardless of whether
// the pipeline failed.
// Note that unlike other tasks, final tasks are run in reverse of the order
// they're added.
func (r Robot) FinalTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFinal, typeTask, name, args...)
}

// FailTask adds a task that runs only if the pipeline fails. This can be used
// to e.g. notify a user / channel on failure.
func (r Robot) FailTask(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorFail, typeTask, name, args...)
}

// AddJob puts another job in the queue for the pipeline. The added job
// will run in a new separate context, and when it completes the current
// pipeline will resume if the job succeeded.
func (r Robot) AddJob(name string, args ...string) robot.RetVal {
	return r.pipeTask(flavorAdd, typeJob, name, args...)
}

// AddCommand adds a plugin command to the pipeline. The command string
// argument should match a CommandMatcher for the given plugin.
func (r Robot) AddCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorAdd, typePlugin, plugname, command)
}

// FinalCommand adds a plugin command that always runs when a pipeline
// ends, for e.g. emailing the job history. The command string
// argument should match a CommandMatcher for the given plugin.
func (r Robot) FinalCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFinal, typePlugin, plugname, command)
}

// FailCommand adds a plugin command that runs whenever a pipeline fails,
// for e.g. emailing the job history. The command string
// argument should match a CommandMatcher for the given plugin.
func (r Robot) FailCommand(plugname, command string) robot.RetVal {
	return r.pipeTask(flavorFail, typePlugin, plugname, command)
}
