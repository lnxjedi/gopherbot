package bot

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/lnxjedi/gopherbot/robot"
)

// if help is more than tooLong lines long, send a private message
const tooLong = 14

// Cut off for listing channels after help text
const tooManyChannels = 4

// Size of QR code
const qrsize = 400

func init() {
	RegisterPlugin("builtin-dmadmin", robot.PluginHandler{Handler: dmadmin})
	RegisterPlugin("builtin-help", robot.PluginHandler{Handler: help})
	RegisterPlugin("builtin-admin", robot.PluginHandler{Handler: admin})
	RegisterPlugin("builtin-logging", robot.PluginHandler{Handler: logging})
}

/* builtin plugins, like help */

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
			msg = append(msg, fmt.Sprintf("My install directory is: %s", installPath))
			lp := "(not set)"
			if len(configPath) > 0 {
				lp = configPath
			}
			msg = append(msg, fmt.Sprintf("My configuration directory is: %s", lp))
		}
		msg = append(msg, fmt.Sprintf("My software version is: Gopherbot %s, commit: %s", botVersion.Version, botVersion.Commit))
		msg = append(msg, fmt.Sprintf("The administrators for this robot are: %s", admins))
		adminContact := r.GetBotAttribute("contact")
		if len(adminContact.Attribute) > 0 {
			msg = append(msg, fmt.Sprintf("The administrative contact for this robot is: %s", adminContact))
		}
		r.Say(strings.Join(msg, "\n"))
	}
	if command == "help" {
		botname := r.cfg.botinfo.UserName
		tasks := r.tasks
		var term, helpOutput string
		botSub := `(bot)`
		hasKeyword := false
		lineSeparator := "\n\n"

		if len(args) == 1 && len(args[0]) > 0 {
			hasKeyword = true
			term = args[0]
			Log(robot.Trace, "Help requested for term", term)
		}

		// Nothing we need will ever change for a worker.
		w := getLockedWorker(r.tid)
		w.Unlock()
		helpLines := make([]string, 0, tooLong)
		for _, t := range tasks.t[1:] {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			// If a keyword was supplied, give help for all matching commands with channels;
			// without a keyword, show help for all commands available in the channel.
			if !w.pluginAvailable(task, hasKeyword, true) {
				continue
			}
			Log(robot.Trace, "Checking help for plugin %s (term: %s)", task.name, term)
			if !hasKeyword { // if you ask for help without a term, you just get help for whatever commands are available to you
				for _, phelp := range plugin.Help {
					for _, helptext := range phelp.Helptext {
						if len(phelp.Keywords) > 0 && phelp.Keywords[0] == "*" {
							// * signifies help that should be prepended
							newSize := tooLong
							if len(helpLines) > newSize {
								newSize += len(helpLines)
							}
							prepend := make([]string, 1, newSize)
							prepend[0] = strings.Replace(helptext, botSub, botname, -1)
							helpLines = append(prepend, helpLines...)
						} else {
							helpLines = append(helpLines, strings.Replace(helptext, botSub, botname, -1))
						}
					}
				}
			} else { // when there's a search term, give all help for that term, but add (channels: xxx) at the end
				for _, phelp := range plugin.Help {
					for _, keyword := range phelp.Keywords {
						if term == keyword {
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
								helpLines = append(helpLines, strings.Replace(helptext, botSub, botname, -1)+chantext)
							}
						}
					}
				}
			}
		}
		if hasKeyword {
			helpOutput = "Command(s) matching keyword: " + term + "\n" + strings.Join(helpLines, lineSeparator)
		}
		switch {
		case len(helpLines) == 0:
			// Unless builtins are disabled or reconfigured, 'ping' is available in all channels
			r.Say("Sorry, I didn't find any commands matching your keyword")
		case len(helpLines) > tooLong:
			if !w.directMsg {
				r.Reply("(the help output was pretty long, so I sent you a private message)")
				if !hasKeyword {
					helpOutput = "Command(s) available in channel: " + r.Channel + "\n" + strings.Join(helpLines, lineSeparator)
				}
			} else {
				if !hasKeyword {
					helpOutput = "Command(s) available in this channel:\n" + strings.Join(helpLines, lineSeparator)
				}
			}
			r.SendUserMessage(r.User, helpOutput)
		default:
			if !hasKeyword {
				helpOutput = "Command(s) available in this channel:\n" + strings.Join(helpLines, lineSeparator)
			}
			r.Say(helpOutput)
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
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test {
			r.Say("This command is only valid with the 'terminal' connector")
			return
		}
		confLock.RLock()
		c, _ := yaml.Marshal(config)
		confLock.RUnlock()
		r.Fixed().Say("Here's how I've been configured, irrespective of interactive changes:\n%s", c)
	case "dumpplugdefault":
		if plug, ok := pluginHandlers[args[0]]; ok {
			r.Fixed().Say("Here's the default configuration for \"%s\":\n%s", args[0], plug.DefaultConfig)
		} else { // look for an external plugin
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
						if cfg, err := getExtDefCfg(plugin.Task); err == nil {
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
		}
	case "dumpplugin":
		if r.Protocol != robot.Terminal && r.Protocol != robot.Test {
			r.Say("This command is only valid with the 'terminal' connector")
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

func admin(m robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	r := m.(Robot)
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
		r.Log(robot.Info, "Configuration successfully reloaded by a request from: %s", r.User)
	case "abort":
		buf := make([]byte, 32768)
		runtime.Stack(buf, true)
		log.Printf("%s", buf)
		time.Sleep(2 * time.Second)
		panic("Abort command issued")
	case "debug":
		tname := args[0]
		if !identifierRe.MatchString(tname) {
			r.Say("Invalid task name '%s', doesn't match regexp: '%s' (task can't load)", tname, identifierRe.String())
			return
		}
		t := r.tasks.getTaskByName(tname)
		if t == nil {
			r.Say("Task '%s' not found", tname)
			return
		}
		task, _, _ := getTask(t)
		if task.Disabled {
			r.Say("That task is disabled, fix and reload; reason: %s", task.reason)
			return
		}
		verbose := false
		if len(args[1]) > 0 {
			verbose = true
		}
		Log(robot.Debug, "Enabling debugging for %s, verbose: %v", tname, verbose)
		pd := &debuggingTask{
			taskID:  task.name,
			name:    tname,
			verbose: verbose,
		}
		taskDebug.Lock()
		taskDebug.p[task.name] = pd
		taskDebug.Unlock()
		r.Say("Debugging enabled for %s (verbose: %v)", args[0], verbose)
	case "stop":
		taskDebug.Lock()
		taskDebug.p = make(map[string]*debuggingTask)
		taskDebug.Unlock()
		r.Say("Debugging disabled")
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
				r.Say("There are still %d plugins running; I'll exit when they all complete, or you can issue an \"abort\" command", runningCount)
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
