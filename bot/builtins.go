package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

// Cut off for listing channels after help text
const tooManyChannels = 4

func init() {
	robot.RegisterPlugin("builtin-fallback", robot.PluginHandler{Handler: fallback})
	robot.RegisterPlugin("builtin-dmadmin", robot.PluginHandler{Handler: dmadmin})
	robot.RegisterPlugin("builtin-help", robot.PluginHandler{Handler: help})
	robot.RegisterPlugin("builtin-admin", robot.PluginHandler{Handler: admin})
	robot.RegisterPlugin("builtin-logging", robot.PluginHandler{Handler: logging})
}

func defaultHelp() []string {
	return []string{
		"(alias) help <keyword> - get help for the provided <keyword>",
		"(alias) help-all - help for all commands available in this channel, including global commands",
	}
}

/* builtin plugins, like help */

func fallback(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	botAlias := r.GetBotAttribute("alias").String()
	if command == "catchall" {
		channelName := r.GetMessage().Channel
		if len(channelName) > 0 {
			r.SayThread("No command matched in channel '%s'; try '%shelp'", channelName, botAlias)
		} else {
			r.Say("Command not found; try your command in a channel, or use '%shelp'", botAlias)
		}
	}
	return
}

var botRegex = regexp.MustCompile(`^([^(]*)\(bot\)(,?) *`)
var aliasRegex = regexp.MustCompile(`^\(alias\) *`)

func (r Robot) formatHelpLine(input string) (ret string) {
	w := getLockedWorker(r.tid)
	w.Unlock()
	botName := r.cfg.botinfo.UserName
	botAlias := string(r.cfg.alias)
	if len(botName) == 0 && len(botAlias) == 0 {
		ret = input
	} else {
		if botRegex.MatchString(input) {
			if len(botName) > 0 {
				ret = botRegex.ReplaceAllString(input, "${1}"+botName+"${2} ")
				w.Log(robot.Debug, "Sending '%s' to FormatHelp", ret)
			} else {
				ret = botRegex.ReplaceAllString(input, botAlias)
			}
		} else if aliasRegex.MatchString(input) {
			if len(botAlias) > 0 {
				ret = aliasRegex.ReplaceAllString(input, botAlias)
			} else {
				ret = aliasRegex.ReplaceAllString(input, botName+", ")
			}
		}
	}
	conn := getConnectorForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))
	if conn == nil {
		return ret
	}
	return conn.FormatHelp(ret)
}

func help(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	if command == "info" {
		admins := strings.Join(r.cfg.adminUsers, ", ")
		aliasCh := r.cfg.alias
		name := r.cfg.botinfo.UserName
		if len(name) == 0 {
			name = "(unknown)"
		}
		ID := r.cfg.botinfo.UserID
		if len(ID) == 0 {
			ID = "(unknown)"
		}
		var alias string
		if aliasCh == 0 {
			alias = "(not set)"
		} else {
			alias = string(aliasCh)
		}
		channelID, _ := handle.ExtractID(r.ProtocolChannel)
		msg := make([]string, 0, 7)
		msg = append(msg, "Here's some information about me and my running environment:")
		msg = append(msg, fmt.Sprintf("The hostname for the server I'm running on is: %s", hostName))
		msg = append(msg, fmt.Sprintf("My name is '%s', alias '%s', and my %s internal ID is '%s'", name, alias, r.Protocol, ID))
		msg = append(msg, fmt.Sprintf("This is channel '%s', %s internal ID: %s", r.Channel, r.Protocol, channelID))
		if r.CheckAdmin() {
			msg = append(msg, fmt.Sprintf("The gopherbot install directory is: %s", installPath))
			msg = append(msg, fmt.Sprintf("My home directory ($GOPHER_HOME) is: %s", homePath))
			if custom, ok := lookupEnv("GOPHER_CUSTOM_REPOSITORY"); ok {
				msg = append(msg, fmt.Sprintf("My git repository is: %s", custom))
			}
		}
		msg = append(msg, fmt.Sprintf("My software version is: Gopherbot %s, commit: %s", botVersion.Version, botVersion.Commit))
		msg = append(msg, fmt.Sprintf("The administrators for this robot are: %s", admins))
		adminContact := r.GetBotAttribute("contact")
		if len(adminContact.Attribute) > 0 {
			msg = append(msg, fmt.Sprintf("The administrative contact for this robot is: %s", adminContact))
		}
		r.MessageFormat(robot.Variable).SayThread(strings.Join(msg, "\n"))
	}
	if command == "help" || command == "help-all" {
		tasks := r.tasks
		var term, helpOutput string
		hasKeyword := false
		lineSeparator := "\n\n"

		if len(args) == 1 && len(args[0]) > 0 {
			hasKeyword = true
			term = strings.ToLower(args[0])
			Log(robot.Trace, "Help requested for term '%s'", term)
		}

		// Nothing we need will ever change for a worker.
		w := getLockedWorker(r.tid)
		w.Unlock()
		helpLines := make([]string, 0, 14)
		if command == "help" {
			if !hasKeyword {
				conn := getConnectorForProtocol(protocolFromIncoming(r.Incoming, r.Protocol))
				var defaultHelpLines []string
				if conn != nil {
					defaultHelpLines = conn.DefaultHelp()
				}
				if len(defaultHelpLines) == 0 {
					defaultHelpLines = defaultHelp()
				}
				for _, line := range defaultHelpLines {
					helpLines = append(helpLines, r.formatHelpLine(line))
				}
			}
		}
		want_specific := command == "help" || hasKeyword
		for _, t := range tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			// If a keyword was supplied, give help for all matching commands with channels;
			// without a keyword, show help for all commands available in the channel.
			available, specific := w.pluginAvailable(task, hasKeyword, true)
			if !available {
				continue
			}
			if want_specific && !specific {
				continue
			}
			Log(robot.Trace, "Checking help for plugin %s (term: %s)", task.name, term)
			if !hasKeyword { // if you ask for help without a term, you just get help for whatever commands are available to you
				for _, phelp := range plugin.Help {
					for _, helptext := range phelp.Helptext {
						if len(phelp.Keywords) > 0 && phelp.Keywords[0] == "*" {
							// * signifies help that should be prepended
							prepend := make([]string, 1, len(helpLines)+1)
							prepend[0] = r.formatHelpLine(helptext)
							helpLines = append(prepend, helpLines...)
						} else {
							helpLines = append(helpLines, r.formatHelpLine(helptext))
						}
					}
				}
			} else { // when there's a search term, give all help for that term, but add (channels: xxx) at the end
				for _, phelp := range plugin.Help {
					for _, keyword := range phelp.Keywords {
						if term == strings.ToLower(keyword) {
							chantext := ""
							if task.DirectOnly {
								// Look: the right paren gets added below
								chantext = " (direct message only"
							} else {
								if len(task.Channels) > tooManyChannels {
									chantext += " (channels: (many) "
								} else {
									for _, pchan := range task.Channels {
										if len(chantext) == 0 {
											chantext += " (channels: " + pchan
										} else {
											chantext += ", " + pchan
										}
									}
								}
							}
							if len(chantext) != 0 {
								chantext += ")"
							}
							for _, helptext := range phelp.Helptext {
								helpLines = append(helpLines, r.formatHelpLine(helptext)+chantext)
							}
						}
					}
				}
			}
		}
		if len(helpLines) == 0 {
			// Unless builtins are disabled or reconfigured, 'ping' is available in all channels
			if r.Incoming.ThreadedMessage {
				r.Reply("Sorry, I didn't find any commands matching your keyword")
			} else {
				r.SayThread("Sorry, I didn't find any commands matching your keyword")
			}
		} else {
			if hasKeyword {
				helpOutput = "Command(s) matching keyword: " + term + "\n" + strings.Join(helpLines, lineSeparator)
			} else {
				helpOutput = "Command(s) available in this channel:\n" + strings.Join(helpLines, lineSeparator)
			}
			if r.Incoming.ThreadedMessage {
				r.Reply(helpOutput)
			} else {
				r.SayThread(helpOutput)
			}
		}
	}
	return
}

func dmadmin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if command == "init" {
		return // ignore init
	}
	switch command {
	case "dumprobot":
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test && r.Protocol != robot.SSH {
			r.Say("This command is only valid with the 'terminal' or 'ssh' connector")
			return
		}
		confLock.RLock()
		c, _ := yaml.Marshal(config)
		confLock.RUnlock()
		r.Fixed().Say("Here's how I've been configured, irrespective of interactive changes:\n%s", c)
	case "dumpplugdefault":
		found := false
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if args[0] == task.name {
				if plugin == nil {
					r.Say("No default configuration available for task type 'job'")
					return
				}
				if plugin.taskType == taskExternal {
					found = true
					if cfg, err := getDefCfg(t); err == nil {
						r.Fixed().Say("Here's the default configuration for \"%s\":\n%s", args[0], *cfg)
					} else {
						r.Say("I had a problem looking that up - somebody should check my logs")
					}
				}
			}
		}
		if !found {
			r.Say("Didn't find a plugin named " + args[0])
		}
	case "dumpplugin":
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test && r.Protocol != robot.SSH {
			r.Say("This command is only valid with the 'terminal' or 'ssh' connector")
			return
		}
		found := false
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if args[0] == task.name {
				if plugin == nil {
					r.Say("Task '%s' is a job, not a plugin", task.name)
					return
				}
				found = true
				c, _ := yaml.Marshal(plugin)
				r.Fixed().Say("%s", c)
			}
		}
		if !found {
			r.Say("Didn't find a plugin named " + args[0])
		}
	case "listplugins":
		joiner := ", "
		message := "Here are the plugins I have configured:\n%s"
		wantDisabled := false
		if len(args[0]) > 0 {
			wantDisabled = true
			joiner = "\n"
			message = "Here's a list of all disabled plugins:\n%s"
		}
		plist := make([]string, 0, len(r.tasks.t))
		for _, t := range r.tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			ptext := task.name
			if wantDisabled {
				if task.Disabled {
					ptext += "; reason: " + task.reason
					plist = append(plist, ptext)
				}
			} else {
				if task.Disabled {
					ptext += " (disabled)"
				}
				plist = append(plist, ptext)
			}
		}
		if len(plist) > 0 {
			r.Say(message, strings.Join(plist, joiner))
		} else { // note because of builtin plugins, plist is ALWAYS > 0 if disabled wasn't specified
			r.Say("There are no disabled plugins")
		}
	}
	return
}

var byebye = []string{
	"Sayonara!",
	"Adios",
	"Hasta la vista!",
	"Later gator!",
}

var rightback = []string{
	"Back in a flash!",
	"Be right back!",
	"You won't even have time to miss me...",
}

func logging(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	switch command {
	case "init":
		return
	case "level":
		setLogLevel(logStrToLevel(args[0]))
		r.Say("I've adjusted the log level to %s", args[0])
		Log(robot.Info, "User %s changed logging level to %s", r.User, args[0])
	case "show":
		page := 0
		if len(args) == 1 {
			page, _ = strconv.Atoi(args[0])
		}
		lines, wrap := logPage(page)
		if wrap {
			r.Say("(warning: value too large for pages, wrapped past beginning of log)")
		}
		r.Fixed().Say(strings.Join(lines, ""))
	case "showlevel":
		l := getLogLevel()
		r.Say("My current logging level is: %s", logLevelToStr(l))
	case "setlines":
		l, _ := strconv.Atoi(args[0])
		set := setLogPageLines(l)
		r.Say("Lines per page of log output set to: %d", set)
	}
	return
}

type psList struct {
	pslines []string
	wids    []int
}

func (p *psList) Len() int {
	return len(p.pslines)
}

func (p *psList) Swap(i, j int) {
	p.pslines[i], p.pslines[j] = p.pslines[j], p.pslines[i]
	p.wids[i], p.wids[j] = p.wids[j], p.wids[i]
}

func (p *psList) Less(i, j int) bool {
	return p.wids[i] < p.wids[j]
}

func admin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	switch command {
	case "reload":
		err := loadConfig(false)
		if err != nil {
			r.Reply("Error encountered during reload:")
			r.Fixed().Say("%v", err)
			Log(robot.Error, "Reloading configuration, requested by %s: %v", r.User, err)
			return
		}
		r.Reply("Configuration reloaded successfully")
		w.Log(robot.Info, "Configuration successfully reloaded by a request from: %s", r.User)
	case "protocollist":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		statuses := listConnectorProtocolStatus()
		if len(statuses) == 0 {
			r.Say("No protocol runtimes are configured")
			return
		}
		lines := make([]string, 0, len(statuses)+1)
		lines = append(lines, "Protocol runtime status:")
		for _, status := range statuses {
			line := fmt.Sprintf("%s (%s): %s", status.protocol, status.role, status.state)
			if status.err != "" {
				line += " (" + status.err + ")"
			}
			lines = append(lines, line)
		}
		r.Say(strings.Join(lines, "\n"))
	case "protocolstart":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-start <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := startSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to start protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Started protocol '%s'", protocol)
	case "protocolstop":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-stop <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := stopSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to stop protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Stopped protocol '%s'", protocol)
	case "protocolrestart":
		sourceProtocol := protocolFromIncoming(r.Incoming, r.Protocol)
		primaryProtocol, _ := getRuntimePrimaryProtocol()
		if !isPrimaryProtocolSource(sourceProtocol) {
			r.Say("This command is only available from the primary protocol '%s'", primaryProtocol)
			return
		}
		if len(args) == 0 || len(strings.TrimSpace(args[0])) == 0 {
			r.Say("Usage: protocol-restart <protocol>")
			return
		}
		protocol := normalizeProtocolName(args[0])
		if err := restartSecondaryConnectorRuntime(protocol); err != nil {
			r.Say("Unable to restart protocol '%s': %v", protocol, err)
			return
		}
		r.Say("Restarted protocol '%s'", protocol)
	case "abort":
		buf := make([]byte, 32768)
		runtime.Stack(buf, true)
		log.Printf("%s", buf)
		time.Sleep(2 * time.Second)
		panic("Abort command issued")
	case "ps":
		// wid pwid pid Go|Ext plugin|task|job
		psl := &psList{
			pslines: []string{
				"WID    PWID  PID   G/E TYPE   PIPENAME         TASK             PLUG-COMMAND ARGS",
			},
			wids: []int{-1},
		}
		activePipelines.Lock()
		if len(activePipelines.i) == 1 {
			activePipelines.Unlock()
			r.Say("No pipelines running")
			return
		}
		for widx, worker := range activePipelines.i {
			pipename := worker.pipeName
			worker.Lock()
			wid := strconv.Itoa(widx)
			pwid := ""
			if worker._parent != nil {
				pwid = strconv.Itoa(worker._parent.id)
			}
			pid := ""
			if worker.osCmd != nil {
				pid = strconv.Itoa(worker.osCmd.Process.Pid)
				wid = wid + "*"
			}
			class := worker.taskClass
			ttype := worker.taskType
			tname := worker.taskName
			command := worker.plugCommand
			args := strings.Join(worker.taskArgs, " ")
			worker.Unlock()
			if pipename == "builtin-admin" && command == "ps" {
				continue
			}
			psline := fmt.Sprintf("%6.6s %5.5s %5.5s %-3.3s %-6.6s %-16.16s %-16.16s %-12.12s %s", wid, pwid, pid, class, ttype, pipename, tname, command, args)
			psl.pslines = append(psl.pslines, psline)
			psl.wids = append(psl.wids, widx)
		}
		activePipelines.Unlock()
		sort.Sort(psl)
		r.Fixed().Say(strings.Join(psl.pslines, "\n"))
	case "kill":
		if len(args) == 0 {
			r.Say("Usage: kill <wid>")
			return
		}
		wid := args[0]
		widx, err := strconv.ParseInt(wid, 10, 0)
		if err != nil {
			r.Say("Couldn't convert '%s' to an int", wid)
			return
		}
		activePipelines.Lock()
		worker, ok := activePipelines.i[int(widx)]
		activePipelines.Unlock()
		if !ok {
			r.Say("Pipeline %s not found", wid)
			return
		}
		var pid int
		var activeTaskTID int
		var rpcCancel context.CancelFunc
		worker.Lock()
		if worker.osCmd != nil {
			pid = worker.osCmd.Process.Pid
		}
		activeTaskTID = worker.activeTaskTID
		rpcCancel = worker.rpcCancel
		worker.Unlock()
		if rpcCancel != nil {
			rpcCancel()
		}
		_ = interruptReplyWaitersForTask(activeTaskTID)
		if pid == 0 {
			r.Say("No active process found for pipeline")
			return
		}
		raiseThreadPriv(fmt.Sprintf("killing process %d", pid))
		if err := unix.Kill(-pid, unix.SIGKILL); err != nil {
			r.Say("Unable to kill pid %d: %v", pid, err)
			return
		}
		r.Say("Killed pid %d", pid)
	case "pause":
		name := args[0]
		notfound := "I don't have a job configured with that name"
		t := r.tasks.getTaskByName(name)
		if t == nil {
			r.Say(notfound)
			return
		}
		_, _, job := getTask(t)
		if job == nil {
			r.Say(notfound)
			return
		}
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		_, ok := pausedJobs.jobs[name]
		if ok {
			r.Say("That job has already been paused")
			return
		}
		m := r.GetMessage()
		pausedJobs.jobs[name] = m.User
		r.Say("Ok, I'll stop running '%s' as a scheduled task", name)
		return
	case "resume":
		name := args[0]
		t := r.tasks.getTaskByName(name)
		_, _, job := getTask(t)
		if job == nil {
			r.Say("I don't have a job configured with that name")
		}
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		_, ok := pausedJobs.jobs[name]
		if !ok {
			r.Say("That job isn't paused")
			return
		}
		delete(pausedJobs.jobs, name)
		r.Say("Ok, I'll resume running '%s' as a scheduled task", name)
		return
	case "pauselist":
		pausedJobs.Lock()
		defer pausedJobs.Unlock()
		if len(pausedJobs.jobs) == 0 {
			r.Say("There are no paused jobs")
			return
		}
		jl := make([]string, 0, len(pausedJobs.jobs))
		for job := range pausedJobs.jobs {
			jl = append(jl, job)
		}
		sort.Strings(jl)
		r.Say("These jobs are paused: %s", strings.Join(jl, ", "))
	case "chanlog":
		lchan := r.Channel
		if len(args) > 0 && len(args[0]) > 0 {
			lchan = args[0]
		}
		if len(lchan) == 0 {
			lchan = "dm"
		}
		fname := lchan + "-channel.log"
		cfile, err := os.Create(fname)
		if err != nil {
			r.Say("Sorry, there was a problem creating the log file")
			Log(robot.Error, "Creating '%s': %v", fname, err)
			return
		}
		clog := log.New(cfile, "", log.LstdFlags)
		chanLoggers.Lock()
		chanLoggers.channels[lchan] = clog
		chanLoggers.Unlock()
		r.Say("Ok, I'll start logging all messages in channel '%s' to '%s'", lchan, fname)
	case "stopchanlog":
		chanLoggers.Lock()
		chanLoggers.channels = make(map[string]*log.Logger)
		chanLoggers.Unlock()
		r.Say("Ok, I've stopped all channel logs")
	case "quit", "restart":
		state.Lock()
		if state.shuttingDown {
			state.Unlock()
			Log(robot.Warn, "Received administrator `quit` while shutdown in progress")
			return
		}
		state.shuttingDown = true
		restart := command == "restart"
		if restart {
			state.restart = true
		}
		proto := r.cfg.protocol
		// NOTE: THIS plugin is definitely running, but will end soon!
		if state.pipelinesRunning > 1 {
			runningCount := state.pipelinesRunning - 1
			state.Unlock()
			if proto != "test" {
				r.Say("There are still %d pipelines running; I'll %s when they all complete, or you can issue an \"abort\" command", runningCount, command)
			}
		} else {
			state.Unlock()
			if proto != "test" {
				if restart {
					r.Reply(r.RandomString(rightback))
				} else {
					r.Reply(r.RandomString(byebye))
				}
				// How long does it _actually_ take for the message to go out?
				time.Sleep(time.Second)
			}
		}
		Log(robot.Info, "Exiting on administrator 'quit|restart' command")
		go stop()
	}
	return
}
