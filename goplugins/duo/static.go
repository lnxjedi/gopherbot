// +build !module

package duo

import "github.com/lnxjedi/gopherbot/bot"

func init() {
	bot.RegisterPreload("goplugins/duo.so")
	bot.RegisterPlugin("duo", duohandler)
}
