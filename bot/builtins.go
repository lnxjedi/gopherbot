package bot

import (
	"fmt"
	"strings"

	otp "github.com/dgryski/dgoogauth"
)

// if help is more than tooLong lines long, send a private message
const tooLong = 7

// If this list doesn't match what's registered below,
// you're gonna have a bad time
var builtIns = []string{
	"builtInhelp",
	"builtInreload",
	"builtIndump",
}

func init() {
	RegisterPlugin("builtIndump", dump)
	RegisterPlugin("builtInhelp", help)
	RegisterPlugin("builtInreload", reload)
	RegisterPlugin("builtInlaunchcodes", launchCode)
}

/* builtin plugins, like help */

type OTPConfig otp.OTPConfig

func launchCode(bot Robot, command string, args ...string) {
	if command == "init" {
		return // ignore init
	}
	var userOTP OTPConfig
	updated := false
	lock, exists, err := bot.CheckoutDatum(bot.User, &userOTP, true)
	if err != nil {
		bot.Say("Yikes - something went wrong with my brain, have somebody check my log")
		return
	}
	defer func() {
		if updated {
			err := r.UpdateDatum(datumName, lock, lists)
			if err != nil {
				r.Log(bot.Error, fmt.Errorf("Saving OTP config: %v", err))
				r.Reply("Good grief. I'm having trouble remembering your launch codes - have somebody check my log")
			}
		} else {
			// Well-behaved plugins will always do a Checkin when the datum hasn't been updated,
			// in case there's another thread waiting.
			r.Checkin(datumName, lock)
		}
	}()
	switch command {
	case "sendCodes":
		if exists {
			bot.Reply("I've already sent you the launch codes, contact an administrator if you're having problems")
			return
		}
		user := bot.GetSenderAttribute("email")
		if len(user) == 0 {
			bot.Reply("Problem - I couldn't get your email address; check with my administrator")
			return
		}
		issuer := bot.GetBotAttribute("fullName")
		if len(issuer) == 0 {
			bot.Reply("Problem - I need to have a full name; check with my administrator")
			return
		}
		otpb := make([]byte, 10)
		random.Read(otpb)
		userOTP.Secret = base32.StdEncoding.EncodeToString(otpb)
		userOTP.WindowSize = 2
		otpuri := userOTP.ProvisionURIWithIssuer(user, issuer)
	}
}

func help(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "help" {
		b.lock.RLock()
		defer b.lock.RUnlock()

		var term, helpOutput string
		hasTerm := false
		helpLines := 0
		if len(args) == 1 && len(args[0]) > 0 {
			hasTerm = true
			term = args[0]
			b.Log(Trace, "Help requested for term", term)
		}

		for _, plugin := range b.plugins {
			b.Log(Trace, fmt.Sprintf("Checking help for plugin %s (term: %s)", plugin.Name, term))
			if !hasTerm { // if you ask for help without a term, you just get help for whatever commands are available to you
				if b.messageAppliesToPlugin(bot.User, bot.Channel, command, plugin) {
					for _, phelp := range plugin.Help {
						for _, helptext := range phelp.Helptext {
							helpOutput += helptext + string('\n')
							helpLines++
						}
					}
				}
			} else { // when there's a search term, give all help for that term, but add (channels: xxx) at the end
				for _, phelp := range plugin.Help {
					for _, keyword := range phelp.Keywords {
						if term == keyword {
							chantext := ""
							for _, pchan := range plugin.Channels {
								if bot.Channel != pchan {
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
								helpOutput += helptext + chantext + string('\n')
								helpLines++
							}
						}
					}
				}
			}
		}
		switch {
		case helpLines == 0:
			bot.Say("Sorry, bub - I got nothin' for ya'")
		case helpLines > tooLong:
			if len(bot.Channel) > 0 {
				bot.Reply("(the help for this channel was pretty long, so I sent you a private message)")
				helpOutput = "Help for channel: " + bot.Channel + "\n" + helpOutput
			}
			bot.SendUserMessage(bot.User, strings.TrimRight(helpOutput, "\n"))
		default:
			bot.Say(strings.TrimRight(helpOutput, "\n"))
		}
	}
}

func dump(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if !bot.CheckAdmin() {
		bot.Reply("Sorry, only an admin user can request that")
		return
	}
	switch command {
	case "robot":
		bot.Fixed().Say(fmt.Sprintf("%+v", bot))
	case "plugin":
		b.lock.RLock()
		defer b.lock.RUnlock()
		found := false
		for _, plugin := range b.plugins {
			if args[0] == plugin.Name {
				found = true
				bot.Fixed().Say(fmt.Sprintf("%+v", plugin))
				bot.Log(Info, fmt.Sprintf("Dump of plugin %s:\n%+v", args[0], plugin))
			}
		}
		if !found {
			bot.Say("Didn't find a plugin named " + args[0])
		}
	}
}

func reload(bot Robot, command string, args ...string) {
	// Get access to the underlying struct
	b := bot.robot
	if command == "reload" {
		if bot.CheckAdmin() {
			err := b.loadConfig()
			if err != nil {
				bot.Reply("Error encountered during reload, check the logs")
				b.Log(Error, fmt.Errorf("Reloading configuration, requested by %s: %v", bot.User, err))
				return
			}
			bot.Reply("Configuration reloaded successfully")
			b.Log(Info, "Configuration successfully reloaded after a request from:", bot.User)
		} else {
			bot.Reply("Sorry, only an admin user can request that")
		}
	}
}
