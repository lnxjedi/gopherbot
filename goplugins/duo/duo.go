package duo

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

var timeoutLock sync.RWMutex
var lastElevate = make(map[string]time.Time)
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

type duoDefault struct {
	device, method int
}

type duoDefMap map[string]duoDefault

var cfg config

const memoryKey = "duoOpts"
const datumName = "duoOpts"

func authduo(r *bot.Robot, immediate bool, user string, res *authapi.PreauthResult) (retval bot.PlugRetVal) {
	dm := ""
	if r.Channel != "" {
		dm = " - I'll message you directly"
	}

	prompted := false
	remembered := false
	// values too big to be valid
	devnum := 1024
	method := 1024
	var msg []string
	var ret bot.RetVal
	var rep string
	var factor, memtype string
	var err error

	var duoDefs duoDefMap

	_, exists, ret := r.CheckoutDatum(datumName, &duoDefs, false)
	if ret == bot.Ok && exists {
		duoDefConfig, ok := duoDefs[r.User]
		if ok {
			devnum = duoDefConfig.device
			method = duoDefConfig.method
			remembered = true
			memtype = "configured"
		}
	}

	if !remembered {
		rememberedOpts := r.Recall(memoryKey)
		if rememberedOpts != "" {
			v := strings.Split(rememberedOpts, ",")
			devnum, _ = strconv.Atoi(v[0])
			method, _ = strconv.Atoi(v[1])
			remembered = true
			memtype = "selected"
		}
	}

	if len(res.Response.Devices) == 1 {
		devnum = 0
	} else if len(res.Response.Devices) > 1 && devnum >= len(res.Response.Devices) {
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
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which device # do you want to use?")
		if ret != bot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit device #")
		}
		if ret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("User \"%s\" failed to respond to duo elevation prompt", r.User))
			return bot.Fail
		}
		devnum, _ = strconv.Atoi(rep)
		if devnum < 0 || devnum >= len(res.Response.Devices) {
			r.Direct().Say("Invalid device number")
			r.Log(bot.Error, fmt.Sprintf("Invalid duo device # response from user \"%s\"", r.User))
			return bot.Fail
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
		return bot.MechanismFail
	}
	if len(res.Response.Devices[devnum].Capabilities) == 1 || (autoProvided && len(res.Response.Devices[devnum].Capabilities) == 2) {
		factor = res.Response.Devices[devnum].Capabilities[0]
		ret = bot.Ok
	} else if method >= len(res.Response.Devices[devnum].Capabilities) {
		if !prompted {
			if immediate {
				r.Say("This command requires immediate elevation" + dm)
			} else {
				r.Say("This command requires elevation" + dm)
			}
			r.Pause(1)
			prompted = true
		}
		msg = make([]string, 10)
		for m, method := range res.Response.Devices[devnum].Capabilities {
			if m == 10 {
				break
			}
			msg[m] = fmt.Sprintf("Method %d: %s", m, method)
		}
		r.Direct().Say(fmt.Sprintf("Duo methods available for your device:\n%s", strings.Join(msg, "\n")))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which method # do you want to use?")
		if ret != bot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit method #")
		}
		if ret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("User \"%s\" failed to respond to duo elevation prompt", r.User))
			return bot.Fail
		}
		method, _ = strconv.Atoi(rep)
		if method < 0 || method >= len(res.Response.Devices[devnum].Capabilities) {
			r.Direct().Say("Invalid method number")
			r.Log(bot.Error, fmt.Sprintf("Invalid duo method # response from user \"%s\"", r.User))
			return bot.Fail
		}
	} else {
		if method >= len(res.Response.Devices[devnum].Capabilities) {
			method = 0
		}
	}
	if !prompted {
		if remembered {
			if immediate {
				r.Say(fmt.Sprintf("This command requires immediate elevation - using the last device and method %s", memtype))
			} else {
				r.Say(fmt.Sprintf("This command requires elevation - using the last device and method %s", memtype))
			}
		} else {
			if immediate {
				r.Say("This command requires immediate elevation - requesting additional authentication")
			} else {
				r.Say("This command requires elevation - requesting additional authentication")
			}
		}
	}
	if factor == "" {
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
	nameattr := r.GetBotAttribute("name")
	botname := nameattr.Attribute
	if botname == "" {
		botname = "Gopherbot"
	} else {
		botname += " - Gopherbot"
	}
	var authres *authapi.AuthResult
	r.Log(bot.Debug, fmt.Sprintf("Attempting duo auth for device %v, factor %s", res.Response.Devices[devnum].Device, factor))
	switch factor {
	case "push":
		authres, err = auth.Auth(factor,
			authapi.AuthUsername(user),
			authapi.AuthDevice(res.Response.Devices[devnum].Device),
			authapi.AuthDisplayUsername(user),
			authapi.AuthType(botname),
		)
	case "passcode":
		if !prompted {
			if immediate {
				r.Say("This command requires immediate elevation" + dm)
			} else {
				r.Say("This command requires elevation" + dm)
			}
			r.Pause(1)
		}
		rep, ret = r.Direct().PromptForReply("multiDigit", "Please enter a passcode to use")
		if ret != bot.Ok {
			rep, ret = r.Direct().PromptForReply("multiDigit", "Try again? I need a short string of numbers")
		}
		if ret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("User \"%s\" failed to respond to duo elevation prompt", r.User))
			return bot.Fail
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
		r.Say("Sorry, there was an error while trying to authenticate you - ask an admin to check the log")
		return bot.MechanismFail
	}
	if authres.Response.Result != "allow" {
		r.Log(bot.Error, fmt.Sprintf("Duo auth failed for user %s (%s) - result: %s, status: %s, message: %s", user, r.User, authres.Response.Result, authres.Response.Status, authres.Response.Status_Msg))
		r.Say("Duo authentication failed")
		return bot.Fail
	}
	r.Remember(memoryKey, fmt.Sprintf("%d,%d", devnum, method))
	return bot.Success
}

func configure(r *bot.Robot, user string, res *authapi.PreauthResult) (retval bot.PlugRetVal) {
	if r.Channel != "" {
		r.Say("Ok, I'll message your directly to get your default configuration")
	}

	var duoDefConfig duoDefault
	var duoDefs duoDefMap
	var devnum, method int
	var msg []string
	var ret bot.RetVal
	var rep string
	prompted := false

	if len(res.Response.Devices) == 1 {
		duoDefConfig.device = 0
	} else if len(res.Response.Devices) > 1 {
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
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which device # do you want to use?")
		if ret != bot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit device #")
		}
		if ret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("User \"%s\" failed to respond to duo configure prompt", r.User))
			return bot.Fail
		}
		devnum, _ = strconv.Atoi(rep)
		if devnum < 0 || devnum >= len(res.Response.Devices) {
			r.Direct().Say("Invalid device number")
			r.Log(bot.Error, fmt.Sprintf("Invalid duo device # response from user \"%s\"", r.User))
			return bot.Fail
		}
		duoDefConfig.device = devnum
		prompted = true
	} else {
		r.Log(bot.Error, fmt.Sprintf("No devices returned for Duo user %s; auth response: %v", user, res))
		r.Direct().Say("There's a problem with your duo account, ask an admin to check the log")
		return bot.MechanismFail
	}
	autoProvided := false
	for _, method := range res.Response.Devices[devnum].Capabilities {
		if method == "auto" {
			autoProvided = true
			break
		}
	}
	if !(len(res.Response.Devices[devnum].Capabilities) == 1 || (autoProvided && len(res.Response.Devices[devnum].Capabilities) == 2)) {
		msg = make([]string, 10)
		for m, method := range res.Response.Devices[devnum].Capabilities {
			if m == 10 {
				break
			}
			msg[m] = fmt.Sprintf("Method %d: %s", m, method)
		}
		r.Direct().Say(fmt.Sprintf("Duo methods available for your device:\n%s", strings.Join(msg, "\n")))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which method # do you want to use?")
		if ret != bot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit method #")
		}
		if ret != bot.Ok {
			r.Log(bot.Error, fmt.Sprintf("User \"%s\" failed to respond to duo configure prompt", r.User))
			return bot.Fail
		}
		method, _ = strconv.Atoi(rep)
		if method < 0 || method >= len(res.Response.Devices[devnum].Capabilities) {
			r.Direct().Say("Invalid method number")
			r.Log(bot.Error, fmt.Sprintf("Invalid duo method # response from user \"%s\"", r.User))
			return bot.Fail
		}
		prompted = true
	}
	if !prompted {
		r.Say("Only one device and method available, not storing")
		return bot.Normal
	}
	duoDefConfig.method = method

	tok, exists, ret := r.CheckoutDatum(datumName, &duoDefs, true)
	if ret == bot.Ok {
		if !exists {
			duoDefs = make(map[string]duoDefault)
		}
		duoDefs[r.User] = duoDefConfig
		r.UpdateDatum(datumName, tok, duoDefs)
		r.Reply("Your duo default configuration has been set")
		return bot.Normal
	}
	r.Log(bot.Error, fmt.Sprintf("Error storing user duo config: %d"), ret)
	return bot.Fail
}

func duocommands(r *bot.Robot, command string, args ...string) (retval bot.PlugRetVal) {
	if command != "elevate" && command != "duoconf" {
		return
	}
	immediate := false
	if len(args) > 0 {
		switch args[0] {
		case "true", "True", "t", "T", "Yes", "yes", "Y":
			immediate = true
		}
	}
	cfg := &config{}
	r.GetPluginConfig(&cfg)
	if cfg.TimeoutType == "absolute" {
		cfg.tt = absolute
	}
	cfg.tf64 = float64(cfg.TimeoutSeconds)
	duo := duoapi.NewDuoApi(cfg.DuoIKey, cfg.DuoSKey, cfg.DuoHost, "Gopherbot", duoapi.SetTimeout(10*time.Second))
	auth = authapi.NewAuthApi(*duo)
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
		return bot.ConfigurationError
	}
	if len(duouser) == 0 {
		r.Log(bot.Error, fmt.Sprintf("Couldn't extract a Duo user name for %s with DuoUserString: %s", r.User, cfg.DuoUserString))
		return bot.MechanismFail
	}
	res, err := auth.Preauth(authapi.PreauthUsername(duouser))
	r.Log(bot.Debug, fmt.Sprintf("Preauth response for duo user %s: %v", duouser, res))
	if err != nil {
		r.Log(bot.Error, fmt.Sprintf("Duo preauthentication error for Duo user %s (%s): %s", duouser, r.User, err))
		return bot.MechanismFail
	}
	if res.Response.Result == "deny" {
		r.Log(bot.Error, fmt.Sprintf("Received \"deny\" during Duo preauth for Duo user %s (%s)", duouser, r.User))
		return bot.Fail
	}
	if command == "duoconf" {
		return configure(r, duouser, res)
	}

	now := time.Now().UTC()
	ask := false
	if immediate {
		retval = authduo(r, immediate, duouser, res)
	} else {
		timeoutLock.RLock()
		le, ok := lastElevate[r.User]
		timeoutLock.RUnlock()
		if ok {
			diff := now.Sub(le)
			if diff.Seconds() > cfg.tf64 {
				ask = true
			} else {
				retval = bot.Success
			}
		} else {
			ask = true
		}
		if ask {
			retval = authduo(r, immediate, duouser, res)
		}
	}
	if retval == bot.Success && cfg.tt == idle {
		timeoutLock.Lock()
		lastElevate[r.User] = now
		timeoutLock.Unlock()
	} else if retval == bot.Success && ask && cfg.tt == absolute {
		timeoutLock.Lock()
		lastElevate[r.User] = now
		timeoutLock.Unlock()
	}
	return
}

const defaultConfig = `
AllChannels: true
Help:
- Keywords: [ "duo" ]
  Helptext: [ "(bot), configure duo - remember a duo device and method to always use" ]
ReplyMatchers:
- Label: singleDigit
  Regex: '\d'
- Label: multiDigit
  Regex: '\d+'
CommandMatchers:
- Command: duoconf
  Regex: (?i:config(?:ure)? duo)
Config:
  TimeoutSeconds: 7200
  TimeoutType: idle # or absolute
#  DuoIKey: <YourIKey>
#  DuoSKey: <YourSKey>
#  DuoHost: <YourDuoHost>
  DuoUserString: emailUser
`

func init() {
	bot.RegisterPlugin("duo", bot.PluginHandler{
		DefaultConfig: defaultConfig,
		Handler:       duocommands,
		Config:        &config{},
	})
}
