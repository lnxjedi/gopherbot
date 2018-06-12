package bot

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
)

// if help is more than tooLong lines long, send a private message
const tooLong = 14

// Cut off for listing channels after help text
const tooManyChannels = 4

// Size of QR code
const qrsize = 400

// If this list doesn't match what's registered below,
// you're gonna have a bad time.
// var builtIns = []string{
// 	"builtInhelp",
// 	"builtInadmin",
// 	"builtIndump",
// 	"builtInlogging",
// }

func init() {
	RegisterPlugin("builtIndump", PluginHandler{DefaultConfig: dumpConfig, Handler: dump})
	RegisterPlugin("builtInhelp", PluginHandler{DefaultConfig: helpConfig, Handler: help})
	RegisterPlugin("builtInadmin", PluginHandler{DefaultConfig: adminConfig, Handler: admin})
	RegisterPlugin("builtInlogging", PluginHandler{DefaultConfig: logConfig, Handler: logging})
}

/* builtin plugins, like help */

func help(bot *Robot, command string, args ...string) (retval TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	if command == "info" {
		robot.RLock()
		admins := strings.Join(robot.adminUsers, ", ")
		alias := robot.alias
		robot.RUnlock()
		msg := make([]string, 0, 7)
		msg = append(msg, "Here's some information about my running environment:")
		msg = append(msg, fmt.Sprintf("The hostname for the server I'm running on is: %s", hostName))
		if bot.CheckAdmin() {
			msg = append(msg, fmt.Sprintf("My install directory is: %s", robot.installPath))
			lp := "(not set)"
			if len(robot.configPath) > 0 {
				lp = robot.configPath
			}
			msg = append(msg, fmt.Sprintf("My configuration directory is: %s", lp))
		}
		msg = append(msg, fmt.Sprintf("My software version is: Gopherbot %s, commit: %s", botVersion.Version, botVersion.Commit))
		if alias != 0 {
			msg = append(msg, fmt.Sprintf("My alias is: %s", string(alias)))
		}
		msg = append(msg, fmt.Sprintf("The administrators for this robot are: %s", admins))
		bot.Say(strings.Join(msg, "\n"))
	}
	if command == "help" {
		robot.RLock()
		botname := robot.name
		robot.RUnlock()

		var term, helpOutput string
		botSub := `(bot)`
		hasKeyword := false
		lineSeparator := "\n\n"

		if len(args) == 1 && len(args[0]) > 0 {
			hasKeyword = true
			term = args[0]
			Log(Trace, "Help requested for term", term)
		}

		helpLines := make([]string, 0, tooLong)
		c := bot.getContext()
		for _, t := range c.tasks.t {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			// If a keyword was supplied, give help for all matching commands with channels;
			// without a keyword, show help for all commands available in the channel.
			if !bot.getContext().taskAvailable(task, hasKeyword, true) {
				continue
			}
			Log(Trace, fmt.Sprintf("Checking help for plugin %s (term: %s)", plugin.name, term))
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
							if plugin.DirectOnly {
								// Look: the right paren gets added below
								chantext = " (direct message only"
							} else {
								if len(plugin.Channels) > tooManyChannels {
									chantext += "(channels: (many) "
								} else {
									for _, pchan := range plugin.Channels {
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
			bot.Say("Sorry, I didn't find any commands matching your keyword")
		case len(helpLines) > tooLong:
			if len(bot.Channel) > 0 {
				bot.Reply("(the help output was pretty long, so I sent you a private message)")
				if !hasKeyword {
					helpOutput = "Command(s) available in channel: " + bot.Channel + "\n" + strings.Join(helpLines, lineSeparator)
				}
			} else {
				if !hasKeyword {
					helpOutput = "Command(s) available in this channel:\n" + strings.Join(helpLines, lineSeparator)
				}
			}
			bot.SendUserMessage(bot.User, helpOutput)
		default:
			if !hasKeyword {
				helpOutput = "Command(s) available in this channel:\n" + strings.Join(helpLines, lineSeparator)
			}
			bot.Say(helpOutput)
		}
	}
	return
}

func dump(bot *Robot, command string, args ...string) (retval TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	switch command {
	case "robot":
		robot.RLock()
		c, _ := yaml.Marshal(config)
		robot.RUnlock()
		bot.Fixed().Say(fmt.Sprintf("Here's how I've been configured, irrespective of interactive changes:\n%s", c))
	case "plugdefault":
		if plug, ok := pluginHandlers[args[0]]; ok {
			bot.Fixed().Say(fmt.Sprintf("Here's the default configuration for \"%s\":\n%s", args[0], plug.DefaultConfig))
		} else { // look for an external plugin
			found := false
			c := bot.getContext()
			for _, t := range c.tasks.t {
				task, plugin, _ := getTask(t)
				if args[0] == task.name {
					if plugin == nil {
						bot.Say("No default configuration available for task type 'job'")
						return
					}
					if plugin.pluginType == plugExternal {
						found = true
						if cfg, err := getExtDefCfg(plugin.botTask); err == nil {
							bot.Fixed().Say(fmt.Sprintf("Here's the default configuration for \"%s\":\n%s", args[0], *cfg))
						} else {
							bot.Say("I had a problem looking that up - somebody should check my logs")
						}
					}
				}
			}
			if !found {
				bot.Say("Didn't find a plugin named " + args[0])
			}
		}
	case "plugin":
		found := false
		c := bot.getContext()
		for _, t := range c.tasks.t {
			task, plugin, _ := getTask(t)
			if args[0] == task.name {
				if plugin == nil {
					bot.Say(fmt.Sprintf("Task '%s' is a job, not a plugin", task.name))
					return
				}
				found = true
				c, _ := yaml.Marshal(plugin)
				bot.Fixed().Say(fmt.Sprintf("%s", c))
			}
		}
		if !found {
			bot.Say("Didn't find a plugin named " + args[0])
		}
	case "list":
		joiner := ", "
		message := "Here are the plugins I have configured:\n%s"
		wantDisabled := false
		if len(args[0]) > 0 {
			wantDisabled = true
			joiner = "\n"
			message = "Here's a list of all disabled plugins:\n%s"
		}
		c := bot.getContext()
		plist := make([]string, 0, len(c.tasks.t))
		for _, t := range c.tasks.t {
			task, plugin, _ := getTask(t)
			if plugin == nil {
				continue
			}
			ptext := task.name
			if wantDisabled {
				if plugin.Disabled {
					ptext += "; reason: " + plugin.reason
					plist = append(plist, ptext)
				}
			} else {
				if plugin.Disabled {
					ptext += " (disabled)"
				}
				plist = append(plist, ptext)
			}
		}
		if len(plist) > 0 {
			bot.Say(fmt.Sprintf(message, strings.Join(plist, joiner)))
		} else { // note because of builtin plugins, plist is ALWAYS > 0 if disabled wasn't specified
			bot.Say("There are no disabled plugins")
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

func logging(bot *Robot, command string, args ...string) (retval TaskRetVal) {
	switch command {
	case "init":
		return
	case "level":
		setLogLevel(logStrToLevel(args[0]))
		bot.Say(fmt.Sprintf("I've adjusted the log level to %s", args[0]))
		Log(Info, fmt.Sprintf("User %s changed logging level to %s", bot.User, args[0]))
	case "show":
		page := 0
		if len(args) == 1 {
			page, _ = strconv.Atoi(args[0])
		}
		lines, wrap := logPage(page)
		if wrap {
			bot.Say("(warning: value too large for pages, wrapped past beginning of log)")
		}
		bot.Fixed().Say(strings.Join(lines, ""))
	case "showlevel":
		l := getLogLevel()
		bot.Say(fmt.Sprintf("My current logging level is: %s", logLevelToStr(l)))
	case "setlines":
		l, _ := strconv.Atoi(args[0])
		set := setLogPageLines(l)
		bot.Say(fmt.Sprintf("Lines per page of log output set to: %d", set))
	}
	return
}

func admin(bot *Robot, command string, args ...string) (retval TaskRetVal) {
	if command == "init" {
		return // ignore init
	}
	if !bot.CheckAdmin() {
		bot.Reply("Sorry, only an admin user can request that")
		return
	}
	switch command {
	case "reload":
		err := bot.getContext().loadConfig()
		if err != nil {
			bot.Reply("Error encountered during reload, check the logs")
			Log(Error, fmt.Errorf("Reloading configuration, requested by %s: %v", bot.User, err))
			return
		}
		bot.Reply("Configuration reloaded successfully")
		Log(Info, "Configuration successfully reloaded by a request from:", bot.User)
	case "abort":
		buf := make([]byte, 32768)
		runtime.Stack(buf, true)
		log.Printf("%s", buf)
		time.Sleep(2 * time.Second)
		panic("Abort command issued")
	case "debug":
		pname := args[0]
		if !taskNameRe.MatchString(pname) {
			bot.Say(fmt.Sprintf("Invalid plugin name '%s', doesn't match regexp: '%s' (plugin can't load)", pname, taskNameRe.String()))
			return
		}
		c := bot.getContext()
		_, plugin, _ := c.tasks.getTaskByName(pname)
		if plugin == nil {
			bot.Say("I don't have any plugins with that name configured")
			return
		}
		if plugin.Disabled {
			bot.Say(fmt.Sprintf("That plugin is disabled, fix and reload; reason: %s", plugin.reason))
			return
		}
		verbose := false
		if len(args[1]) > 0 {
			verbose = true
		}
		Log(Debug, fmt.Sprintf("Enabling debugging for %s (%s), verbose: %v", pname, plugin.taskID, verbose))
		pd := &debuggingPlug{
			taskID:  plugin.taskID,
			name:    pname,
			user:    bot.User,
			verbose: verbose,
		}
		plugDebug.Lock()
		plugDebug.p[plugin.taskID] = pd
		plugDebug.u[bot.User] = pd
		plugDebug.Unlock()
		bot.Say(fmt.Sprintf("Debugging enabled for %s (verbose: %v)", args[0], verbose))
	case "stop":
		plugDebug.Lock()
		pd, ok := plugDebug.u[bot.User]
		if ok {
			delete(plugDebug.p, pd.taskID)
			delete(plugDebug.u, bot.User)
		}
		plugDebug.Unlock()
		bot.Say("Debugging disabled")
	case "quit":
		robot.Lock()
		if robot.shuttingDown {
			robot.Unlock()
			Log(Warn, "Received administrator `quit` while shutdown in progress")
			return
		}
		robot.shuttingDown = true
		proto := robot.protocol
		// NOTE: THIS plugin is definitely running, but will end soon!
		if robot.pluginsRunning > 1 {
			runningCount := robot.pluginsRunning - 1
			robot.Unlock()
			if proto != "test" {
				bot.Say(fmt.Sprintf("There are still %d plugins running; I'll exit when they all complete, or you can issue an \"abort\" command", runningCount))
			}
		} else {
			robot.Unlock()
			if proto != "test" {
				bot.Reply(bot.RandomString(byebye))
				// How long does it _actually_ take for the message to go out?
				time.Sleep(time.Second)
			}
		}
		Log(Info, "Exiting on administrator 'quit' command")
		go stop()
	}
	return
}
