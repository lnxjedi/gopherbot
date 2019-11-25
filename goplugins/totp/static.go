// +build !module

package totp

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("goplugins/totp.so")
	bot.RegisterPlugin("totp", totphandler)
}
