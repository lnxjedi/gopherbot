package totp

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"math/rand"
	"sync"
	"time"

	otp "github.com/dgryski/dgoogauth"
	"github.com/lnxjedi/gopherbot/bot"
)

var timeoutLock sync.RWMutex
var lastElevate = make(map[string]time.Time)
var random = rand.New(rand.NewSource(time.Now().UnixNano()))

type timeoutType int

const (
	idle timeoutType = iota
	absolute
)

type config struct {
	TimeoutSeconds int
	tf64           float64
	TimeoutType    string
	tt             timeoutType
}

var cfg config

func checkOTP(r *bot.Robot, code string) (bool, bot.TaskRetVal) {
	var userOTP otp.OTPConfig
	lock, exists, ret := r.CheckoutDatum(r.User, &userOTP, true)
	if ret != bot.Ok {
		r.CheckinDatum(r.User, lock)
		return false, bot.MechanismFail
	}
	if !exists {
		r.CheckinDatum(r.User, lock)
		return false, bot.MechanismFail
	}
	valid, err := userOTP.Authenticate(code)
	if err != nil {
		r.Log(bot.Error, "Problem authenticating launch code for user %s: %v", r.User, err)
		r.CheckinDatum(r.User, lock)
		return false, bot.MechanismFail
	}
	ret = r.UpdateDatum(r.User, lock, &userOTP)
	if ret != bot.Ok {
		r.Log(bot.Error, "Problem updating OTP for %s, failing", r.User)
		return false, bot.MechanismFail
	}
	return valid, bot.Success
}

func getcode(r *bot.Robot, immediate bool) (retval bot.TaskRetVal) {
	dm := ""
	if r.Channel != "" {
		dm = " - I'll message you directly"
	}
	if immediate {
		r.Say("This command requires immediate elevation" + dm)
	} else {
		r.Say("This command requires elevation" + dm)
	}
	r.Pause(1)
	rep, ret := r.Direct().PromptForReply("OTP", "Please provide your totp launch code")
	if ret != bot.Ok {
		rep, ret = r.Direct().PromptForReply("OTP", "Try again? I need a 6-digit launch code")
	}
	if ret == bot.Ok {
		ok, ret := checkOTP(r, rep)
		if ret != bot.Success {
			r.Direct().Say("There were technical issues validating your code, ask an administrator to check the log")
			return bot.MechanismFail
		}
		if ok {
			return bot.Success
		}
		r.Direct().Say("Invalid code")
		return bot.Fail
	}
	r.Log(bot.Error, "User \"%s\" failed to respond to TOTP token prompt", r.User)
	return bot.Fail
}

func elevate(r *bot.Robot, command string, args ...string) (retval bot.TaskRetVal) {
	switch command {
	case "send":
		var userOTP otp.OTPConfig
		updated := false
		lock, exists, ret := r.CheckoutDatum(r.User, &userOTP, true)
		if ret != bot.Ok {
			r.Say("Yikes! - Something went wrong with my brain, have an admin check my log")
			return
		}
		defer func() {
			if updated {
				ret = r.UpdateDatum(r.User, lock, &userOTP)
				if ret != bot.Ok {
					r.Log(bot.Error, "Couldn't save OTP config")
					r.Reply("Good grief, I'm having trouble remembering your launch codes - have somebody check my log")
				}
			} else {
				// Well-behaved plugins will always do a CheckinDatum when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(r.User, lock)
			}
		}()
		if exists {
			r.Reply("I've already sent you the launch codes, contact an administrator if you're having problems")
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
		r.CheckinDatum(r.User, lock)
		if ret = r.Email("Your launch codes - if you print this email, please chew it up and swallow it", &codeMail); ret != bot.Ok {
			r.Reply("There was a problem sending your launch codes, contact an administrator")
			return
		}
		lock, _, ret = r.CheckoutDatum(r.User, &userOTP, true)
		updated = true
		r.Reply("I've emailed your launch codes - please delete it promptly")
		return
	case "elevate":
		immediate := false
		switch args[0] {
		case "true", "True", "t", "T", "Yes", "yes", "Y":
			immediate = true
		}
		cfg := &config{}
		r.GetTaskConfig(&cfg)
		if cfg.TimeoutType == "absolute" {
			cfg.tt = absolute
		}
		cfg.tf64 = float64(cfg.TimeoutSeconds)
		now := time.Now().UTC()
		ask := false
		if immediate {
			retval = getcode(r, immediate)
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
				retval = getcode(r, immediate)
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
	return
}

func init() {
	bot.RegisterPlugin("totp", bot.PluginHandler{
		Handler: elevate,
		Config:  &config{},
	})
}
