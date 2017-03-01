package totp

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	duoapi "github.com/duosecurity/duo_api_golang"
	"github.com/duosecurity/duo_api_golang/authapi"
	"github.com/uva-its/gopherbot/bot"
)

var botHandler bot.Handler
var timeoutLock sync.RWMutex
var lastElevate map[string]time.Time
var auth *authapi.AuthApi

type timeoutType int

const (
	idle timeoutType = iota
	absolute
)

type config struct {
	TimeoutSeconds int
	tf64           float64
	TimeoutType    string // TimeoutType - one of idle, absolute
	tt             timeoutType
	DuoIKey        string
	DuoSKey        string
	DuoHost        string
	DuoUserString  string // DuoUserType - one of handle, email, emailUser
}

var cfg config

func authduo(r *bot.Robot, immediate bool, user string, res *authapi.PreauthResult) bool {
	dm := ""
	if r.Channel != "" {
		dm = " - I'll message you directly"
	}

	prompted := false
	var devnum, method int
	var msg []string
	var ret bot.RetVal
	var rep string
	var factor string

	if len(res.Response.Devices) > 1 {
		if immediate {
			r.Say("This command requires immediate elevation" + dm)
		} else {
			r.Say("This command requires elevation" + dm)
		}
		r.Pause(1)
		prompted = true

		msg = make([]string, 10)
		for d, dev := range res.Response.Devices {
			if d == 10 {
				break
			}
			switch dev.Type {
			case "phone":
				msg[d] = fmt.Sprintf("Device %d: %s - %s", d, dev.Type, dev.Number)
			case "token":
				msg[d] = fmt.Sprintf("Device %d: %s (%s)", d, dev.Type, dev.Name)
			}
		}
		r.Direct().Say(fmt.Sprintf("Duo devices:\n%s", strings.Join(msg, "\n")))
		r.Direct().Say("Which device # do you want to use?")
		rep, ret = r.Direct().WaitForReplyRegex(`\d`, 10)
		if ret != bot.Ok {
			r.Direct().Say("Try again? I need a single-digit device #")
			rep, ret = r.Direct().WaitForReplyRegex(`\d`, 10)
		}
		if ret != bot.Ok {
			return false
		}
		devnum, _ = strconv.Atoi(rep)
		if devnum < 0 || devnum >= len(res.Response.Devices) {
			r.Direct().Say("Invalid device number")
			return false
		}
	}
	autoProvided := false
	if len(res.Response.Devices) > 0 {
		for _, method := range res.Response.Devices[devnum].Capabilities {
			if method == "auto" {
				autoProvided = true
				break
			}
		}
	} else {
		r.Log(bot.Error, fmt.Sprintf("No devices returned for Duo user %s; auth response: %v", user, res))
		r.Direct().Say("There's a problem with your duo account, ask an admin to check the log")
		return false
	}
	if len(res.Response.Devices[devnum].Capabilities) == 1 || (autoProvided && len(res.Response.Devices[devnum].Capabilities) == 2) {
		factor = res.Response.Devices[devnum].Capabilities[0]
		ret = bot.Ok
	} else {
		if !prompted {
			if immediate {
				r.Say("This command requires immediate elevation" + dm)
			} else {
				r.Say("This command requires elevation" + dm)
			}
			r.Pause(1)
		}
		msg = make([]string, 10)
		for m, method := range res.Response.Devices[devnum].Capabilities {
			if m == 10 {
				break
			}
			msg[m] = fmt.Sprintf("Method %d: %s", m, method)
		}
		r.Direct().Say(fmt.Sprintf("Duo methods available for your device:\n%s", strings.Join(msg, "\n")))
		r.Direct().Say("Which method # do you want to use?")
		rep, ret = r.Direct().WaitForReplyRegex(`\d`, 10)
		if ret != bot.Ok {
			r.Direct().Say("Try again? I need a single-digit method #")
			rep, ret = r.Direct().WaitForReplyRegex(`\d`, 10)
		}
		method, _ = strconv.Atoi(rep)
		if method < 0 || method >= len(res.Response.Devices[devnum].Capabilities) {
			r.Direct().Say("Invalid method number")
			return false
		}
		factor = res.Response.Devices[devnum].Capabilities[method]
		if factor == "sms" {
			_, _ = auth.Auth(factor,
				authapi.AuthUsername(user),
				authapi.AuthDevice(res.Response.Devices[devnum].Device),
			)
			factor = "passcode"
		}
		if factor == "mobile_otp" {
			factor = "passcode"
		}
	}
	if ret == bot.Ok {
		nameattr := r.GetBotAttribute("name")
		botname := nameattr.Attribute
		if botname == "" {
			botname = "Gopherbot"
		} else {
			botname += " - Gopherbot"
		}
		var authres *authapi.AuthResult
		var err error
		switch factor {
		case "push":
			authres, err = auth.Auth(factor,
				authapi.AuthUsername(user),
				authapi.AuthDevice(res.Response.Devices[devnum].Device),
				authapi.AuthDisplayUsername(user),
				authapi.AuthType(botname),
			)
		case "passcode":
			r.Direct().Say("Ok, please enter a passcode to use")
			rep, ret = r.Direct().WaitForReplyRegex(`\d+`, 20)
			if ret != bot.Ok {
				r.Direct().Say("Try again? I need a short string of numbers")
				rep, ret = r.Direct().WaitForReplyRegex(`\d+`, 20)
			}
			if ret != bot.Ok {
				return false
			}
			authres, err = auth.Auth(factor,
				authapi.AuthUsername(user),
				authapi.AuthPasscode(rep),
			)
		default:
			authres, err = auth.Auth(factor,
				authapi.AuthUsername(user),
				authapi.AuthDevice(res.Response.Devices[devnum].Device),
			)
		}
		r.Log(bot.Debug, fmt.Sprintf("Auth response from duo: %v", authres))
		if err != nil {
			r.Log(bot.Error, fmt.Sprintf("Error during Duo auth for user %s (%s): %s", user, r.User, err))
			r.Direct().Say("Sorry, there was an error while, trying to authenticate you - ask an admin to check the log")
			return false
		}
		if authres.Response.Result != "allow" {
			r.Log(bot.Error, fmt.Sprintf("Duo auth failed for user %s (%s) - result: %s, status: %s, message: %s", user, r.User, authres.Response.Result, authres.Response.Status, authres.Response.Status_Msg))
			r.Direct().Say("Duo authentication failed")
			return false
		}
		return true
	}
	return false
}

func elevate(r *bot.Robot, immediate bool) bool {
	var duouser string

	switch cfg.DuoUserString {
	case "handle":
		duouser = r.User
	case "email":
		duouser = r.GetSenderAttribute("email").Attribute
	case "emailUser", "emailuser":
		mailattr := r.GetSenderAttribute("email")
		email := mailattr.Attribute
		duouser = strings.Split(email, "@")[0]
	default:
		r.Log(bot.Error, "No DuoUserString configured for Duo elevator plugin")
	}
	if len(duouser) == 0 {
		r.Log(bot.Error, fmt.Sprintf("Couldn't extract a Duo user name for %s with DuoUserString: %s", r.User, cfg.DuoUserString))
		r.Say("This command requires elevation and I couldn't determine your Duo username, sorry")
		return false
	}
	res, err := auth.Preauth(authapi.PreauthUsername(duouser))
	r.Log(bot.Debug, fmt.Sprintf("Preauth response for duo user %s: %v", duouser, res))
	if err != nil {
		r.Log(bot.Error, fmt.Sprintf("Duo preauthentication error for Duo user %s (%s): %s", duouser, r.User, err))
		r.Say("This command requires elevation, but there was an error during preauth")
		return false
	}
	if res.Response.Result == "deny" {
		r.Log(bot.Error, fmt.Sprintf("Received \"deny\" during Duo preauth for Duo user %s (%s)", duouser, r.User))
		r.Say("This command requires elevation, but I received a \"deny\" response during preauth")
		return false
	}

	allowed := false
	now := time.Now().UTC()
	if immediate {
		allowed = authduo(r, immediate, duouser, res)
	} else {
		timeoutLock.RLock()
		le, ok := lastElevate[r.User]
		timeoutLock.RUnlock()
		ask := false
		if ok {
			diff := now.Sub(le)
			if diff.Seconds() > cfg.tf64 {
				ask = true
			} else {
				allowed = true
			}
		} else {
			ask = true
		}
		if ask {
			allowed = authduo(r, immediate, duouser, res)
		}
	}
	if allowed && cfg.tt == idle {
		timeoutLock.Lock()
		lastElevate[r.User] = now
		timeoutLock.Unlock()
	}
	return allowed
}

func provider(r bot.Handler) bot.Elevate {
	botHandler = r
	botHandler.GetElevateConfig(&cfg)
	if cfg.TimeoutType == "absolute" {
		cfg.tt = absolute
	}
	cfg.tf64 = float64(cfg.TimeoutSeconds)
	duo := duoapi.NewDuoApi(cfg.DuoIKey, cfg.DuoSKey, cfg.DuoHost, "Gopherbot", duoapi.SetTimeout(10*time.Second))
	auth = authapi.NewAuthApi(*duo)
	return elevate
}

func init() {
	bot.RegisterElevator("duo", provider)
	lastElevate = make(map[string]time.Time)
}
