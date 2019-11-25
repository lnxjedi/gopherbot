package duo

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	duoapi "github.com/duosecurity/duo_api_golang"
	"github.com/duosecurity/duo_api_golang/authapi"
	"github.com/lnxjedi/gopherbot/robot"
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

func authduo(r robot.Robot, immediate bool, user string, res *authapi.PreauthResult) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	dm := ""
	if m.Channel != "" {
		dm = " - I'll message you directly"
	}

	prompted := false
	remembered := false
	// values too big to be valid
	devnum := 1024
	method := 1024
	var msg []string
	var ret robot.RetVal
	var rep string
	var factor, memtype string
	var err error

	var duoDefs duoDefMap

	_, exists, ret := r.CheckoutDatum(datumName, &duoDefs, false)
	if ret == robot.Ok && exists {
		duoDefConfig, ok := duoDefs[m.User]
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
		r.Direct().Say("Duo devices:\n%s", strings.Join(msg, "\n"))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which device # do you want to use?")
		if ret != robot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit device #")
		}
		if ret != robot.Ok {
			r.Log(robot.Error, "User \"%s\" failed to respond to duo elevation prompt", m.User)
			return robot.Fail
		}
		devnum, _ = strconv.Atoi(rep)
		if devnum < 0 || devnum >= len(res.Response.Devices) {
			r.Direct().Say("Invalid device number")
			r.Log(robot.Error, "Invalid duo device # response from user \"%s\"", m.User)
			return robot.Fail
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
		r.Log(robot.Error, "No devices returned for Duo user %s; auth response: %v", user, res)
		r.Direct().Say("There's a problem with your duo account, ask an admin to check the log")
		return robot.MechanismFail
	}
	if len(res.Response.Devices[devnum].Capabilities) == 1 || (autoProvided && len(res.Response.Devices[devnum].Capabilities) == 2) {
		factor = res.Response.Devices[devnum].Capabilities[0]
		ret = robot.Ok
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
		r.Direct().Say("Duo methods available for your device:\n%s", strings.Join(msg, "\n"))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which method # do you want to use?")
		if ret != robot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit method #")
		}
		if ret != robot.Ok {
			r.Log(robot.Error, "User \"%s\" failed to respond to duo elevation prompt", m.User)
			return robot.Fail
		}
		method, _ = strconv.Atoi(rep)
		if method < 0 || method >= len(res.Response.Devices[devnum].Capabilities) {
			r.Direct().Say("Invalid method number")
			r.Log(robot.Error, "Invalid duo method # response from user \"%s\"", m.User)
			return robot.Fail
		}
	} else {
		if method >= len(res.Response.Devices[devnum].Capabilities) {
			method = 0
		}
	}
	if !prompted {
		if remembered {
			if immediate {
				r.Say("This command requires immediate elevation - using the last device and method %s", memtype)
			} else {
				r.Say("This command requires elevation - using the last device and method %s", memtype)
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
	r.Log(robot.Debug, "Attempting duo auth for device %v, factor %s", res.Response.Devices[devnum].Device, factor)
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
		if ret != robot.Ok {
			rep, ret = r.Direct().PromptForReply("multiDigit", "Try again? I need a short string of numbers")
		}
		if ret != robot.Ok {
			r.Log(robot.Error, "User \"%s\" failed to respond to duo elevation prompt", m.User)
			return robot.Fail
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
	r.Log(robot.Debug, "Auth response from duo: %v", authres)
	if err != nil {
		r.Log(robot.Error, "Error during Duo auth for user %s (%s): %s", user, m.User, err)
		r.Say("Sorry, there was an error while trying to authenticate you - ask an admin to check the log")
		return robot.MechanismFail
	}
	if authres.Response.Result != "allow" {
		r.Log(robot.Error, "Duo auth failed for user %s (%s) - result: %s, status: %s, message: %s", user, m.User, authres.Response.Result, authres.Response.Status, authres.Response.Status_Msg)
		r.Say("Duo authentication failed")
		return robot.Fail
	}
	r.Remember(memoryKey, fmt.Sprintf("%d,%d", devnum, method))
	return robot.Success
}

func configure(r robot.Robot, user string, res *authapi.PreauthResult) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	if m.Channel != "" {
		r.Say("Ok, I'll message you directly to get your default configuration")
	}

	var duoDefConfig duoDefault
	var duoDefs duoDefMap
	var devnum, method int
	var msg []string
	var ret robot.RetVal
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
		r.Direct().Say("Duo devices:\n%s", strings.Join(msg, "\n"))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which device # do you want to use?")
		if ret != robot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit device #")
		}
		if ret != robot.Ok {
			r.Log(robot.Error, "User \"%s\" failed to respond to duo configure prompt", m.User)
			return robot.Fail
		}
		devnum, _ = strconv.Atoi(rep)
		if devnum < 0 || devnum >= len(res.Response.Devices) {
			r.Direct().Say("Invalid device number")
			r.Log(robot.Error, "Invalid duo device # response from user \"%s\"", m.User)
			return robot.Fail
		}
		duoDefConfig.device = devnum
		prompted = true
	} else {
		r.Log(robot.Error, "No devices returned for Duo user %s; auth response: %v", user, res)
		r.Direct().Say("There's a problem with your duo account, ask an admin to check the log")
		return robot.MechanismFail
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
		r.Direct().Say("Duo methods available for your device:\n%s", strings.Join(msg, "\n"))
		rep, ret = r.Direct().PromptForReply("singleDigit", "Which method # do you want to use?")
		if ret != robot.Ok {
			rep, ret = r.Direct().PromptForReply("singleDigit", "Try again? I need a single-digit method #")
		}
		if ret != robot.Ok {
			r.Log(robot.Error, "User \"%s\" failed to respond to duo configure prompt", m.User)
			return robot.Fail
		}
		method, _ = strconv.Atoi(rep)
		if method < 0 || method >= len(res.Response.Devices[devnum].Capabilities) {
			r.Direct().Say("Invalid method number")
			r.Log(robot.Error, "Invalid duo method # response from user \"%s\"", m.User)
			return robot.Fail
		}
		prompted = true
	}
	if !prompted {
		r.Say("Only one device and method available, not storing")
		return robot.Normal
	}
	duoDefConfig.method = method

	tok, exists, ret := r.CheckoutDatum(datumName, &duoDefs, true)
	if ret == robot.Ok {
		if !exists {
			duoDefs = make(map[string]duoDefault)
		}
		duoDefs[m.User] = duoDefConfig
		r.UpdateDatum(datumName, tok, duoDefs)
		r.Reply("Your duo default configuration has been set")
		return robot.Normal
	}
	r.Log(robot.Error, "Error storing user duo config: %d", ret)
	return robot.Fail
}

func duocommands(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
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
	r.GetTaskConfig(&cfg)
	if cfg.TimeoutType == "absolute" {
		cfg.tt = absolute
	}
	cfg.tf64 = float64(cfg.TimeoutSeconds)
	if len(cfg.DuoIKey) == 0 {
		cfg.DuoIKey = r.GetSecret("IKEY")
	}
	if len(cfg.DuoSKey) == 0 {
		cfg.DuoSKey = r.GetSecret("SKEY")
	}
	if len(cfg.DuoHost) == 0 {
		cfg.DuoHost = r.GetSecret("HOST")
	}
	for _, s := range []string{cfg.DuoIKey, cfg.DuoSKey, cfg.DuoHost} {
		if len(s) == 0 {
			r.Log(robot.Error, "Missing Duo IKey, SKey or Host; not configured or in Environment")
		}
	}
	duo := duoapi.NewDuoApi(cfg.DuoIKey, cfg.DuoSKey, cfg.DuoHost, "Gopherbot", duoapi.SetTimeout(10*time.Second))
	auth = authapi.NewAuthApi(*duo)
	var duouser string

	switch cfg.DuoUserString {
	case "handle":
		duouser = m.User
	case "email":
		duouser = r.GetSenderAttribute("email").Attribute
	case "emailUser", "emailuser":
		mailattr := r.GetSenderAttribute("email")
		email := mailattr.Attribute
		duouser = strings.Split(email, "@")[0]
	default:
		r.Log(robot.Error, "No DuoUserString configured for Duo elevator plugin")
		return robot.ConfigurationError
	}
	if len(duouser) == 0 {
		r.Log(robot.Error, "Couldn't extract a Duo user name for %s with DuoUserString: %s", m.User, cfg.DuoUserString)
		return robot.MechanismFail
	}
	res, err := auth.Preauth(authapi.PreauthUsername(duouser))
	r.Log(robot.Debug, "Preauth response for duo user %s: %v", duouser, res)
	if err != nil {
		r.Log(robot.Error, "Duo preauthentication error for Duo user %s (%s): %s", duouser, m.User, err)
		return robot.MechanismFail
	}
	if res.Response.Result == "deny" {
		r.Log(robot.Error, "Received \"deny\" during Duo preauth for Duo user %s (%s)", duouser, m.User)
		return robot.Fail
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
		le, ok := lastElevate[m.User]
		timeoutLock.RUnlock()
		if ok {
			diff := now.Sub(le)
			if diff.Seconds() > cfg.tf64 {
				ask = true
			} else {
				retval = robot.Success
			}
		} else {
			ask = true
		}
		if ask {
			retval = authduo(r, immediate, duouser, res)
		}
	}
	if retval == robot.Success && cfg.tt == idle {
		timeoutLock.Lock()
		lastElevate[m.User] = now
		timeoutLock.Unlock()
	} else if retval == robot.Success && ask && cfg.tt == absolute {
		timeoutLock.Lock()
		lastElevate[m.User] = now
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
#  DuoIKey: <YourIKey> # ... or set in DUO_IKEY
#  DuoSKey: <YourSKey> # ... or set in DUO_SKEY
#  DuoHost: <YourDuoHost> # ... or set in DUO_HOST
  DuoUserString: emailUser
`

var duohandler = robot.PluginHandler{
	DefaultConfig: defaultConfig,
	Handler:       duocommands,
	Config:        &config{},
}
