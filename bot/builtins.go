package bot

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	otp "github.com/dgryski/dgoogauth"
	"github.com/ghodss/yaml"
)

// if help is more than tooLong lines long, send a private message
const tooLong = 14

// Size of QR code
const qrsize = 400

// If this list doesn't match what's registered below,
// you're gonna have a bad time.
var builtIns = []string{
	"builtInhelp",
	"builtInadmin",
	"builtIndump",
	"builtInlaunchcodes",
	"builtInlogging",
}

func init() {
	RegisterPlugin("builtIndump", PluginHandler{DefaultConfig: dumpConfig, Handler: dump})
	RegisterPlugin("builtInhelp", PluginHandler{DefaultConfig: helpConfig, Handler: help})
	RegisterPlugin("builtInadmin", PluginHandler{DefaultConfig: adminConfig, Handler: admin})
	RegisterPlugin("builtInlaunchcodes", PluginHandler{DefaultConfig: launchCodesConfig, Handler: launchCode})
	RegisterPlugin("builtInlogging", PluginHandler{DefaultConfig: logConfig, Handler: logging})
}

/* builtin plugins, like help */

func launchCode(bot *Robot, command string, args ...string) (retval PlugRetVal) {
	if command == "init" {
		return Success // ignore init
	}
	var userOTP otp.OTPConfig
	otpKey := "elevator-totp:" + bot.User
	updated := false
	lock, exists, ret := checkoutDatum(otpKey, &userOTP, true)
	if ret != Ok {
		bot.Say("Yikes - something went wrong with my brain, have somebody check my log")
		return
	}
	defer func() {
		if updated {
			ret = updateDatum(otpKey, lock, &userOTP)
			if ret != Ok {
				Log(Error, "Couldn't save OTP config")
				bot.Reply("Good grief, I'm having trouble remembering your launch codes - have somebody check my log")
			}
		} else {
			// Well-behaved plugins will always do a CheckinDatum when the datum hasn't been updated,
			// in case there's another thread waiting.
			checkinDatum(otpKey, lock)
		}
	}()
	switch command {
	case "send":
		if exists {
			bot.Reply("I've already sent you the launch codes, contact an administrator if you're having problems")
			return
		}
		otpb := make([]byte, 10)
		random.Read(otpb)
		userOTP.Secret = base32.StdEncoding.EncodeToString(otpb)
		userOTP.WindowSize = 2
		userOTP.DisallowReuse = []int{}
		var codeMail bytes.Buffer
		fmt.Fprintf(&codeMail, "For your authenticator:\n%s\n", userOTP.Secret)
		// Sending email takes longer than the timeout, so we check it in and check
		// out again after.
		checkinDatum(otpKey, lock)
		if ret = bot.Email("Your launch codes - if you print this email, please chew it up and swallow it", &codeMail); ret != Ok {
			bot.Reply("There was a problem sending your launch codes, contact an administrator")
			return
		}
		lock, _, ret = checkoutDatum(otpKey, &userOTP, true)
		updated = true
		bot.Reply("I've emailed your launch codes - please delete it promptly")
	}
	return
}

func help(bot *Robot, command string, args ...string) (retval PlugRetVal) {
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
			msg = append(msg, fmt.Sprintf("My local configuration directory is: %s", robot.localPath))
		}
		msg = append(msg, fmt.Sprintf("My software version is: Gopherbot %s, commit: %s", Version, commit))
		if alias != 0 {
			msg = append(msg, fmt.Sprintf("My alias is: %s", string(alias)))
		}
		msg = append(msg, fmt.Sprintf("The administrators for this robot are: %s", admins))
		bot.Say(strings.Join(msg, "\n"))
	}
	if command == "help" {
		robot.RLock()
		defer robot.RUnlock()

		var term, helpOutput string
		botSub := `(bot)`
		hasTerm := false
		lineSeparator := "\n\n"

		if len(args) == 1 && len(args[0]) > 0 {
			hasTerm = true
			term = args[0]
			if term == "help" {
				Log(Trace, "Help requested for help, returning")
				return
			}
			Log(Trace, "Help requested for term", term)
		}

		helpLines := make([]string, 0, tooLong)
		pluginlist.RLock()
		plugins := pluginlist.p
		pluginlist.RUnlock()
		for _, plugin := range plugins {
			Log(Trace, fmt.Sprintf("Checking help for plugin %s (term: %s)", plugin.name, term))
			if !hasTerm { // if you ask for help without a term, you just get help for whatever commands are available to you
				if messageAppliesToPlugin(bot.User, bot.Channel, plugin) {
					for _, phelp := range plugin.Help {
						for _, helptext := range phelp.Helptext {
							if len(phelp.Keywords) > 0 && phelp.Keywords[0] == "*" {
								// * signifies help that should be prepended
								newSize := tooLong
								if len(helpLines) > newSize {
									newSize += len(helpLines)
								}
								prepend := make([]string, 1, newSize)
								prepend[0] = strings.Replace(helptext, botSub, robot.name, -1)
								helpLines = append(prepend, helpLines...)
							} else {
								helpLines = append(helpLines, strings.Replace(helptext, botSub, robot.name, -1))
							}
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
								for _, pchan := range plugin.Channels {
									if len(chantext) == 0 {
										chantext += " (channels: " + pchan
									} else {
										chantext += ", " + pchan
									}
								}
							}
							if len(chantext) != 0 {
								chantext += ")"
							}
							for _, helptext := range phelp.Helptext {
								helpLines = append(helpLines, strings.Replace(helptext, botSub, robot.name, -1)+chantext)
							}
						}
					}
				}
			}
		}
		if hasTerm {
			helpOutput = "Command(s) matching keyword: " + term + "\n" + strings.Join(helpLines, lineSeparator)
		}
		switch {
		case len(helpLines) == 0:
			bot.Say("Sorry, bub - I got nothin' for ya'")
		case len(helpLines) > tooLong:
			if len(bot.Channel) > 0 {
				bot.Reply("(the help output was pretty long, so I sent you a private message)")
				if !hasTerm {
					helpOutput = "Command(s) available in channel: " + bot.Channel + "\n" + strings.Join(helpLines, lineSeparator)
				}
			} else {
				if !hasTerm {
					helpOutput = "Command(s) available:" + "\n" + strings.Join(helpLines, lineSeparator)
				}
			}
			bot.SendUserMessage(bot.User, helpOutput)
		default:
			if !hasTerm {
				helpOutput = "Command(s) available:" + "\n" + strings.Join(helpLines, lineSeparator)
			}
			bot.Say(helpOutput)
		}
	}
	return
}

func dump(bot *Robot, command string, args ...string) (retval PlugRetVal) {
	if command == "init" {
		return // ignore init
	}
	pluginlist.RLock()
	plugins := pluginlist.p
	pluginlist.RUnlock()
	robot.RLock()
	defer robot.RUnlock()
	switch command {
	case "robot":
		c, _ := yaml.Marshal(config)
		bot.Fixed().Say(fmt.Sprintf("Here's how I've been configured, irrespective of interactive changes:\n%s", c))
	case "plugdefault":
		if plug, ok := pluginHandlers[args[0]]; ok {
			bot.Fixed().Say(fmt.Sprintf("Here's the default configuration for \"%s\":\n%s", args[0], plug.DefaultConfig))
		} else { // look for an external plugin
			found := false
			for _, plugin := range plugins {
				if args[0] == plugin.name && plugin.pluginType == plugExternal {
					found = true
					if cfg, err := getExtDefCfg(plugin); err == nil {
						bot.Fixed().Say(fmt.Sprintf("Here's the default configuration for \"%s\":\n%s", args[0], *cfg))
					} else {
						bot.Say("I had a problem looking that up - somebody should check my logs")
					}
				}
			}
			if !found {
				bot.Say("Didn't find a plugin named " + args[0])
			}
		}
	case "plugin":
		found := false
		for _, plugin := range plugins {
			if args[0] == plugin.name {
				found = true
				c, _ := yaml.Marshal(plugin)
				bot.Fixed().Say(fmt.Sprintf("%s", c))
			}
		}
		if !found {
			bot.Say("Didn't find a plugin named " + args[0])
		}
	case "list":
		plist := make([]string, 0, len(plugins))
		for _, plugin := range plugins {
			plist = append(plist, plugin.name)
		}
		bot.Say(fmt.Sprintf("Here are the plugins I have configured:\n%s", strings.Join(plist, ", ")))
	}
	return
}

var byebye = []string{
	"Sayonara!",
	"Adios",
	"Hasta la vista!",
	"Later gator!",
}

func logging(bot *Robot, command string, args ...string) (retval PlugRetVal) {
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

func admin(bot *Robot, command string, args ...string) (retval PlugRetVal) {
	if command == "init" {
		return // ignore init
	}
	if !bot.CheckAdmin() {
		bot.Reply("Sorry, only an admin user can request that")
		return
	}
	switch command {
	case "reload":
		err := loadConfig()
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
		panic("Abort command issued")
	case "quit":
		pluginsRunning.Done()
		pluginsRunning.Lock()
		pluginsRunning.count--
		pluginsRunning.shuttingDown = true
		if pluginsRunning.count > 0 {
			runningCount := pluginsRunning.count
			pluginsRunning.Unlock()
			bot.Say(fmt.Sprintf("There are still %d plugins running; I'll exit when they all complete, or you can issue an \"abort\" command", runningCount))
		} else {
			pluginsRunning.Unlock()
		}
		// Wait for all plugins to stop running
		pluginsRunning.Wait()
		bot.Reply(bot.RandomString(byebye))
		// Stop the brain after it finishes any current task
		brainQuit()
		Log(Info, "Exiting on administrator command")
		// How long does it _actually_ take for the message to go out?
		time.Sleep(time.Second)
		close(finish)
	}
	return
}
